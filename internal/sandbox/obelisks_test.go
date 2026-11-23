package sandbox

import (
	"os"
	"slices"
	"testing"
)

func obeliskFixture(t *testing.T) (World, int64) {
	t.Helper()
	w, id, _ := pyramidFixture(t)
	w.MapExtent = 64
	w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool { return b.ID == id || b.Kind == "workcamp" })
	delete(w.TradePolicies, "stone")
	// The test supplier records granite production into actual yard inventory.
	// The original allows the citywide stock to be consumed at placement.
	left := 10000
	for y := 35; y <= 43 && left > 0; y += 4 {
		for x := 12; x < 40 && left > 0; x += 4 {
			yard := addGoodsBuilding(t, &w, "storageyard", x, y)
			n := min(left, 800)
			w.ProduceResource(yard, "granite", n)
			left -= n
		}
	}
	w.refresh()
	if dir := os.Getenv("NILE_OBELISK_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "obelisk-prepared.nile")
	}
	id = addGoodsBuilding(t, &w, "smallobelisk", 23, 25)
	return w, id
}
func TestObeliskAtomicOriginalUpfrontGranite(t *testing.T) {
	w := NewTown()
	w.BudgetEnabled = false
	before := w.Hash()
	if e := w.Build("smallobelisk", 23, 25); e == nil || w.Hash() != before {
		t.Fatal("obelisk placed or mutated without10000granite")
	}
	w, id := obeliskFixture(t)
	if w.monumentStored("granite") != 0 || w.GoodsConsumed["granite"] != 10000 || w.building(id).MonumentMaterials["granite"] != 10000 {
		t.Fatal("upfront granite not conserved")
	}
	if e := w.Validate(); e != nil {
		t.Fatal(e)
	}
	before = w.Hash()
	if e := w.Build("largeobelisk", 28, 24); e == nil || w.Hash() != before {
		t.Fatal("allowed second unfinished obelisk")
	}
	for kind, amount := range map[string]int{"smallobelisk": 10000, "largeobelisk": 20000} {
		p := monumentPlans[kind]
		w, h := dimensions(kind)
		want := 3
		if kind == "largeobelisk" {
			want = 5
		}
		if p.UpfrontAmount != amount || w != want || h != want {
			t.Fatal("wrong original obelisk dimensions/materials")
		}
	}
}
func TestObeliskPhysicalCarpentersAndMasons(t *testing.T) {
	w, id := obeliskFixture(t)
	seen := map[string]bool{}
	for range 12000 {
		if e := w.Step(1); e != nil {
			t.Fatal(e)
		}
		if w.Tick%16 == 0 {
			if e := w.Validate(); e != nil {
				t.Fatalf("tick%d: %v", w.Tick, e)
			}
		}
		for _, a := range w.Walkers {
			if a.Role == "stonehauler" {
				t.Fatal("obelisk used a Work Camp sledge")
			}
			if a.Role == "builder" && a.Stage == "work" {
				seen[w.building(a.Origin).Kind] = true
			}
		}
		if monumentComplete(*w.building(id)) {
			break
		}
	}
	b := w.building(id)
	if !monumentComplete(*b) || b.ConstructionDone != 10 || b.MonumentMaterials["timber"] != 1600 || !seen["carpenters"] || !seen["stonemasons"] {
		t.Fatalf("incomplete original obelisk: %s %v", b.Stage, b.MonumentMaterials)
	}
	if dir := os.Getenv("NILE_OBELISK_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "obelisk-complete.nile")
	}
	t.Logf("ten original obelisk jobs completed at tick%d", w.Tick)
}

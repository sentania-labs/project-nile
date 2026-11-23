package sandbox

import (
	"os"
	"slices"
	"testing"
)

func templeMonumentFixture(t *testing.T, kind string) (World, int64) {
	t.Helper()
	w, id, _ := pyramidFixture(t)
	w.MapExtent = 64
	w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool { return b.ID == id })
	w.Roads = slices.DeleteFunc(w.Roads, func(p Point) bool { return p.X > 22 && p.Y >= 20 })
	delete(w.TradePolicies, "stone")
	left := monumentPlans[kind].UpfrontAmount
	for y := 44; y <= 60 && left > 0; y += 4 {
		for x := 12; x <= 48 && left > 0; x += 4 {
			yard := addGoodsBuilding(t, &w, "storageyard", x, y)
			n := min(left, 800)
			w.ProduceResource(yard, "sandstone", n)
			left -= n
		}
	}
	if left != 0 {
		t.Fatal("fixture did not store the placement sandstone")
	}
	stageY := 28
	if kind == "suntemple" {
		stageY = 30
	}
	for y := 29; y <= stageY; y++ {
		if e := w.Build("road", 22, y); e != nil {
			t.Fatal(e)
		}
	}
	for x := 23; x <= 26; x++ {
		if e := w.Build("road", x, stageY); e != nil {
			t.Fatal(e)
		}
	}
	if e := w.SetTradePolicy("sandstone", "import", 800); e != nil {
		t.Fatal(e)
	}
	w.refresh()
	if dir := os.Getenv("NILE_TEMPLE_MONUMENT_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, kind+"-prepared.nile")
	}
	id = addGoodsBuilding(t, &w, kind, 24, 20)
	return w, id
}
func TestSunTempleOriginalCompositeAndMaterialTotals(t *testing.T) {
	w, id := templeMonumentFixture(t, "suntemple")
	b := w.building(id)
	if b.Width != 11 || b.Height != 10 || len(b.MonumentParts) != 5 || len(b.tiles()) != 135 || !b.contains(Point{30, 36}) || b.contains(Point{24, 36}) {
		t.Fatal("sun temple footprint differs from original100-part table")
	}
	if b.MonumentRequiredMaterials["sandstone"] != 38000 || b.MonumentMaterials["sandstone"] != 22000 {
		t.Fatal("wrong sandstone placement/later supply split")
	}
	if e := w.Validate(); e != nil {
		t.Fatal(e)
	}
	before := w.Hash()
	if e := w.Build("road", 30, 36); e == nil || w.Hash() != before {
		t.Fatal("road entered temple corridor")
	}
}
func TestTempleMonumentsPhysicalSandstoneAndGuilds(t *testing.T) {
	for _, kind := range []string{"suntemple", "smallmausoleum"} {
		t.Run(kind, func(t *testing.T) {
			w, id := templeMonumentFixture(t, kind)
			seen := map[string]bool{}
			deliveries := map[int64]bool{}
			for range 40000 {
				if e := w.Step(1); e != nil {
					t.Fatal(e)
				}
				if w.Tick%64 == 0 {
					if e := w.Validate(); e != nil {
						t.Fatalf("tick%d phase%s: %v", w.Tick, w.building(id).Stage, e)
					}
				}
				for _, a := range w.Walkers {
					if a.Role == "builder" && a.Stage == "work" {
						seen[w.building(a.Origin).Kind] = true
					}
					if a.Role == "stonehauler" && a.Stage == "deliver" {
						if a.Resource != "sandstone" || a.Cargo != 400 {
							t.Fatal("wrong material haul")
						}
						deliveries[a.ID] = true
					}
				}
				if monumentComplete(*w.building(id)) {
					break
				}
			}
			want := 49600
			timber := 1600
			loads := 64
			if kind == "suntemple" {
				want = 38000
				timber = 3200
				loads = 40
			}
			b := w.building(id)
			if !monumentComplete(*b) || b.MonumentMaterials["sandstone"] != want || b.MonumentMaterials["timber"] != timber || len(deliveries) != loads {
				t.Fatalf("unfinished%s tick%d stage%s installed%v sledges%d", kind, w.Tick, b.Stage, b.MonumentMaterials, len(deliveries))
			}
			for _, guild := range []string{"workcamp", "carpenters", "stonemasons"} {
				if !seen[guild] {
					t.Fatal("missing physical guild", guild)
				}
			}
			if w.GoodsConsumed["sandstone"] != int64(want) {
				t.Fatal("sandstone was lost or duplicated")
			}
			if dir := os.Getenv("NILE_TEMPLE_MONUMENT_DIR"); dir != "" {
				saveMonumentFixture(t, w, dir, kind+"-complete.nile")
			}
			t.Logf("%s completed with original materials at tick%d", kind, w.Tick)
		})
	}
}

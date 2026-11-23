package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func pyramidFixture(t *testing.T) (World, int64, int64) {
	t.Helper()
	w := NewTown()
	w.BudgetEnabled = false
	w.Treasury = 25000
	w.HazardsEnabled = false
	for _, y := range []int{4, 7, 10, 14} {
		for x := 11; x <= 27; x++ {
			if e := w.Build("road", x, y); e != nil {
				t.Fatal(e)
			}
		}
	}
	for _, y := range []int{3, 6, 9} {
		for x := 11; x <= 16; x++ {
			id := addGoodsBuilding(t, &w, "hut", x, y)
			w.building(id).Pop = 5
			w.Arrivals += 5
		}
	}
	for y := 15; y <= 28; y++ {
		if e := w.Build("road", 22, y); e != nil {
			t.Fatal(e)
		}
	}
	for x := 23; x <= 26; x++ {
		if e := w.Build("road", x, 28); e != nil {
			t.Fatal(e)
		}
	}
	addGoodsBuilding(t, &w, "workcamp", 12, 15)
	addGoodsBuilding(t, &w, "stonemasons", 12, 12)
	addGoodsBuilding(t, &w, "carpenters", 16, 12)
	yard := addGoodsBuilding(t, &w, "storageyard", 17, 15)
	site := int64(0)
	if e := w.OpenTrade(tradeRouteID); e != nil {
		t.Fatal(e)
	}
	if e := w.SetTradePolicy("stone", "import", 800); e != nil {
		t.Fatal(e)
	}
	if e := w.SetTradePolicy("timber", "import", 400); e != nil {
		t.Fatal(e)
	}
	w.refresh()
	if dir := os.Getenv("NILE_MONUMENT_FIXTURE_DIR"); dir != "" {
		os.MkdirAll(dir, 0755)
		s, e := Open(filepath.Join(dir, "prepared.nile"))
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Save(w); e != nil {
			t.Fatal(e)
		}
		s.Close()
	}
	site = addGoodsBuilding(t, &w, "smallstepped", 23, 20)
	if e := w.Validate(); e != nil {
		t.Fatal(e)
	}
	return w, site, yard
}
func TestSmallSteppedOriginalRequirementsAndStaging(t *testing.T) {
	w, id, _ := pyramidFixture(t)
	b := w.building(id)
	if b.Width != 8 || b.Height != 8 || constructionCost(b.Kind) != 0 || b.ConstructionRequired != 24000 || !b.StagingAccess {
		t.Fatalf("wrong original footprint or requirement: %+v", *b)
	}
	if b.roadAccess(Point{22, 20}) || !b.roadAccess(Point{25, 28}) {
		t.Fatal("arbitrary edge substituted for staging")
	}
	if e := w.Build("clear", 25, 28); e != nil {
		t.Fatal(e)
	}
	runValid(t, &w, 64)
	if w.building(id).MonumentPhase != 0 || w.building(id).MonumentWork != 0 {
		t.Fatal("site advanced without staging")
	}
}
func TestPyramidPhysicalGuildsStoneRampsAndReplay(t *testing.T) {
	w, id, _ := pyramidFixture(t)
	seen := map[string]bool{}
	var replay *World
	store, e := Open(filepath.Join(t.TempDir(), "pyramid.nile"))
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	for range 22000 {
		if e = w.Step(1); e != nil {
			t.Fatal(e)
		}
		if w.Tick%8 == 0 {
			if e = w.Validate(); e != nil {
				t.Fatalf("tick%d phase%s: %v", w.Tick, w.building(id).Stage, e)
			}
		}
		for _, a := range w.Walkers {
			if a.Role == "builder" {
				seen[w.building(a.Origin).Kind+"/"+a.Stage] = true
			}
			if a.Role == "stonehauler" {
				seen["sledge/"+a.Stage] = true
				if a.Stage == "deliver" && replay == nil {
					if e = store.Save(w); e != nil {
						t.Fatal(e)
					}
					r, err := store.Load()
					if err != nil {
						t.Fatal(err)
					}
					replay = &r
				}
			}
		}
		if replay != nil && replay.Tick < w.Tick {
			if e = replay.Step(1); e != nil {
				t.Fatal(e)
			}
			if replay.Hash() != w.Hash() {
				t.Fatalf("in-flight replay diverged at%d", w.Tick)
			}
		}
		if w.building(id).Stage == "complete" {
			break
		}
	}
	b := w.building(id)
	if b.Stage != "complete" || b.ConstructionDone != 24000 || b.MonumentMaterials["stone"] != 24000 || b.MonumentMaterials["timber"] != 400 {
		t.Fatalf("pyramid unfinished at%d: %+v materials%v workers%v", w.Tick, *b, b.MonumentMaterials, seen)
	}
	for _, s := range []string{"workcamp/work", "stonemasons/work", "carpenters/work", "sledge/collect", "sledge/deliver", "sledge/return"} {
		if !seen[s] {
			t.Fatal("missing physical activity", s)
		}
	}
	if w.ResourceImported["stone"] < 24000 || w.GoodsConsumed["stone"] != 24000 || w.GoodsConsumed["timber"] != 400 {
		t.Fatal("imported materials not conserved")
	}
	runValid(t, &w, 100)
	if dir := os.Getenv("NILE_MONUMENT_FIXTURE_DIR"); dir != "" {
		os.MkdirAll(dir, 0755)
		s, err := Open(filepath.Join(dir, "completed.nile"))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		if err = s.Save(w); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("completed verified 240-block pyramid at tick%d, imports%v, roles%v", w.Tick, w.ResourceImported, seen)
}
func TestPyramidCannotBuildWithoutPairedMasonsOrSuppliedCarpenter(t *testing.T) {
	w, id, yard := pyramidFixture(t)
	w.TradePolicies = map[string]TradePolicy{}
	w.ProduceResource(yard, "stone", 800)
	runValid(t, &w, 450)
	b := w.building(id)
	if b.Stage != "course 1" || b.ConstructionDone > 800 {
		t.Fatalf("unexpected no-import progress: %+v", *b)
	}
	// An arrived mason alone cannot turn a sledge into a completed paired section.
	b.MonumentWork = 0
	w.DepositResource(id, "stone", 400)
	w.RecordResourceProduction("stone", 400)
	for i := range w.Walkers {
		a := &w.Walkers[i]
		if a.Role == "builder" && w.building(a.Origin).Kind == "stonemasons" {
			a.Stage = "return"
			break
		}
	}
	before := b.ConstructionDone
	w.constructMonument(b)
	if b.MonumentWork != 0 || b.ConstructionDone != before {
		t.Fatal("one mason built a paired section")
	}
}

package sandbox

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMonumentOriginalFamilyQuantities(t *testing.T) {
	cases := []struct {
		kind                           string
		side, stone, bricks, limestone int
	}{
		{"smallstepped", 8, 24000, 0, 0}, {"mediumstepped", 12, 67200, 0, 0}, {"largestepped", 16, 144000, 0, 0}, {"complexstepped", 20, 264000, 0, 0}, {"grandstepped", 24, 436800, 0, 0},
		{"smallpyramid", 8, 24000, 0, 19200}, {"mediumpyramid", 12, 67200, 0, 43200}, {"largepyramid", 16, 144000, 0, 76800}, {"complexpyramid", 20, 264000, 0, 120000}, {"grandpyramid", 24, 436800, 0, 172800},
		{"smallbent", 8, 21600, 0, 16800}, {"mediumbent", 12, 55200, 0, 33600},
		{"smallbrick", 8, 0, 24000, 19200}, {"mediumbrick", 12, 0, 67200, 43200}, {"largebrick", 16, 0, 144000, 76800}, {"complexbrick", 20, 0, 264000, 120000}, {"grandbrick", 24, 0, 436800, 172800},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			total := map[string]int{}
			for _, p := range monumentPlans[c.kind].Phases {
				total[p.Resource] += p.Stone
			}
			width, height := dimensions(c.kind)
			if width != c.side || height != c.side || total["stone"] != c.stone || total["bricks"] != c.bricks || total["limestone"] != c.limestone {
				t.Fatalf("dimensions%d,%d materials%v", width, height, total)
			}
		})
	}
}
func TestGrandComplexCompositeFootprintAndAtomicPlacement(t *testing.T) {
	w := NewValley()
	w.BudgetEnabled = false
	// 24x24 core, mortuary and causeway, then east-facing shore temple.
	if e := w.Build("grandpyramid", 5, 26); e != nil {
		t.Fatal(e)
	}
	b := w.Buildings[len(w.Buildings)-1]
	if len(b.MonumentParts) != 3 || b.MonumentParts[2].Kind != "valley_temple" || b.MonumentParts[2].X != 33 {
		t.Fatalf("parts%+v", b.MonumentParts)
	}
	if !b.contains(Point{34, 38}) || b.contains(Point{30, 30}) {
		t.Fatal("composite occupancy does not match physical parts")
	}
	if e := w.Validate(); e != nil {
		t.Fatal(e)
	}
	if e := w.Build("road", 31, 38); e == nil {
		t.Fatal("road crossed causeway")
	}
	saved := w.Hash()
	if e := w.Build("complexpyramid", 50, 30); e == nil || saved != w.Hash() {
		t.Fatal("invalid bank caused partial placement")
	}
}
func TestBrickCorePhysicalSledgesCasingAndPolishing(t *testing.T) {
	w, id, yard := pyramidFixture(t)
	b := w.building(id)
	b.Kind = "smallbrick"
	// A separate bricklayers guild shares the starter's y14 road.
	addGoodsBuilding(t, &w, "bricklayers", 19, 12)
	delete(w.TradePolicies, "stone")
	if e := w.SetTradePolicy("limestone", "import", 800); e != nil {
		t.Fatal(e)
	}
	seenBrick, seenLime, polishing := false, false, false
	w.refresh()
	for range 30000 {
		// This test's brickworks supplier produces into its storage fixture;
		// every subsequent400-unit trip still reserves, walks, loads and unloads.
		if free := w.ResourceFreeSpace(yard, "bricks"); free >= 400 {
			w.ProduceResource(yard, "bricks", 400)
		}
		if e := w.Step(1); e != nil {
			t.Fatal(e)
		}
		for _, a := range w.Walkers {
			if a.Role == "stonehauler" && a.Cargo == 400 {
				seenBrick = seenBrick || a.Resource == "bricks"
				seenLime = seenLime || a.Resource == "limestone"
			}
		}
		if w.Tick%32 == 0 {
			if e := w.Validate(); e != nil {
				t.Fatalf("tick%d: %v", w.Tick, e)
			}
		}
		b = w.building(id)
		if b.Stage == "polishing" {
			polishing = true
			if monumentComplete(*b) {
				t.Fatal("completed before polishing")
			}
		}
		if monumentComplete(*b) {
			break
		}
	}
	if !monumentComplete(*b) || !seenBrick || !seenLime || !polishing || b.MonumentMaterials["bricks"] != 24000 || b.MonumentMaterials["limestone"] != 19200 {
		t.Fatalf("incomplete chain: %s %v", b.Stage, b.MonumentMaterials)
	}
}
func TestSphinxUsesGuildCarvingAndTimberWithoutStoneOrCamps(t *testing.T) {
	w, id, _ := pyramidFixture(t)
	w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool { return b.ID == id || b.Kind == "workcamp" })
	w.Roads = slices.DeleteFunc(w.Roads, func(p Point) bool { return p.X > 10 && p.Y >= 24 })
	for x := 11; x <= 18; x++ {
		if e := w.Build("road", x, 30); e != nil {
			t.Fatal(e)
		}
	}
	id = addGoodsBuilding(t, &w, "sphinx", 11, 24)
	if dir := os.Getenv("NILE_MONUMENT_FAMILIES_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "sphinx-start.nile")
	}
	seen := map[string]bool{}
	for range 24000 {
		if e := w.Step(1); e != nil {
			t.Fatal(e)
		}
		for _, a := range w.Walkers {
			if a.Role == "stonehauler" {
				t.Fatal("sphinx requested stone delivery")
			}
			if a.Role == "builder" && a.Stage == "work" {
				seen[w.building(a.Origin).Kind] = true
			}
		}
		if w.Tick%32 == 0 {
			if e := w.Validate(); e != nil {
				t.Fatalf("tick%d phase%s: %v", w.Tick, w.building(id).Stage, e)
			}
		}
		if monumentComplete(*w.building(id)) {
			break
		}
	}
	b := w.building(id)
	if !monumentComplete(*b) || b.ConstructionDone != 22 || b.MonumentMaterials["timber"] != 2000 || !seen["carpenters"] || !seen["stonemasons"] {
		t.Fatalf("unfinishedSphinx %s progress%d materials%v", b.Stage, b.ConstructionDone, b.MonumentMaterials)
	}
	if e := w.Build("sphinx", 11, 20); e == nil {
		t.Fatal("secondSphinxallowed")
	}
	if dir := os.Getenv("NILE_MONUMENT_FAMILIES_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "sphinx-complete.nile")
	}
}
func saveMonumentFixture(t *testing.T, w World, dir, name string) {
	t.Helper()
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	s, e := Open(filepath.Join(dir, name))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Save(w); e != nil {
		t.Fatal(e)
	}
}

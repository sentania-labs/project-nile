package sandbox

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestRestoredMediumPyramidRetainsItsRoadEntrance(t *testing.T) {
	b := Building{Kind: "mediumpyramid", X: 17, Y: 10, Width: 12, Height: 12}
	original := Point{21, 22}
	if b.roadAccess(original) {
		t.Fatal("new medium pyramid used the former entrance")
	}
	b.MonumentStaging = original
	monumentRefresh(&b)
	if !b.roadAccess(original) || b.MonumentStaging != original {
		t.Fatal("older city's existing roads no longer reach its pyramid")
	}
}

func TestComplexWorshipCapacityCountedOnce(t *testing.T) {
	w := NewTown()
	w.Buildings = []Building{{Kind: "hut", Pop: 10000}, {Kind: "complex_bast", Workers: 1, Connected: true}}
	// Eighty percent worship coverage for one of five known gods.
	if got := w.cultureCoverage()[6]; got != 16 {
		t.Fatalf("religious coverage %d, want 16", got)
	}
}

func TestSQLiteResumesBuilderAtLegacyMediumPyramid(t *testing.T) {
	w, small, _ := pyramidFixture(t)
	w.MapExtent = 64
	w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool { return b.ID == small })
	for x := 27; x <= 34; x++ {
		if err := w.Build("road", x, 28); err != nil {
			t.Fatal(err)
		}
	}
	id := addGoodsBuilding(t, &w, "mediumpyramid", 30, 16)
	w.building(id).MonumentStaging = Point{34, 28}
	w.refresh()
	for range 400 {
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		for _, a := range w.Walkers {
			if a.Role != "builder" || a.Target != id || a.Route[len(a.Route)-1] != (Point{34, 28}) {
				continue
			}
			store, err := Open(filepath.Join(t.TempDir(), "medium.nile"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err = store.Save(w); err != nil {
				t.Fatal(err)
			}
			restored, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			if err = restored.Step(40); err != nil {
				t.Fatal(err)
			}
			if err = w.Step(40); err != nil || restored.Hash() != w.Hash() {
				t.Fatal("restored builder diverged", err)
			}
			return
		}
	}
	t.Fatal("no crew walked toward the older entrance")
}

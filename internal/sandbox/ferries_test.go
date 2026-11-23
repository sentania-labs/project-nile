package sandbox

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestWestBankFerryHaulsReservedStoneThroughRestore(t *testing.T) {
	w := NewValley()
	build := func(kind string, x, y int) {
		t.Helper()
		if err := w.Build(kind, x, y); err != nil {
			t.Fatal(kind, err)
		}
	}
	road := func(x1, x2, y int) {
		t.Helper()
		var path []Point
		for x := x1; x <= x2; x++ {
			path = append(path, Point{x, y})
		}
		if err := w.BuildRoads(path); err != nil {
			t.Fatal(err)
		}
	}
	zone := func(x1, y1, x2, y2 int) {
		t.Helper()
		if err := w.ZoneHousing(Point{x1, y1}, Point{x2, y2}); err != nil {
			t.Fatal(err)
		}
	}
	y := 31
	east, west := w.bank(y), w.westBank(y)
	build("ferry", east-1, y)
	build("ferry", west-1, y)
	road(east+1, 49, y+1)
	road(west-12, west-2, y+1)
	zone(west-11, y+2, west-4, y+3)
	zone(east+2, y+2, east+10, y+3)
	road(41, 60, 7)
	build("stonequarry", 55, 8)
	zone(43, 5, 54, 6)
	build("storageyard", west-11, y-2)
	for n := 0; n < 700; n++ {
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		ferry := w.Ferries[0]
		if !ferry.Sailing || ferry.Position < 1 {
			continue
		}
		loaded := slices.ContainsFunc(w.Walkers, func(a Walker) bool {
			return a.Resource == "stone" && a.Cargo == 100 && (slices.Contains(ferry.Riders[0], a.ID) || slices.Contains(ferry.Riders[1], a.ID))
		})
		if !loaded {
			continue
		}
		before := w.Hash()
		if err := w.Build("clear", west-1, y); err == nil || w.Hash() != before {
			t.Fatal("removed a landing during transit")
		}
		store, err := Open(filepath.Join(t.TempDir(), "ferry.nile"))
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		if err := store.Save(w); err != nil {
			t.Fatal(err)
		}
		restored, err := store.Load()
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 60; i++ {
			if err := w.Step(1); err != nil {
				t.Fatal(err)
			}
			if err := restored.Step(1); err != nil {
				t.Fatal(err)
			}
			if slices.ContainsFunc(w.Buildings, func(b Building) bool { return b.Kind == "storageyard" && b.Goods["stone"] >= 100 }) {
				if err := restored.Validate(); err != nil {
					t.Fatal(err)
				}
				if w.Hash() != restored.Hash() {
					t.Fatal("ferry cargo replay diverged")
				}
				return
			}
		}
		t.Fatal("stone did not reach the western storage yard")
	}
	t.Fatal("no stone loaded onto a ferry")
}

package sandbox

import (
	"path/filepath"
	"testing"
)

func TestSaveMusterCrossingNile(t *testing.T) {
	w, fort := armyFixture(t)
	w.WestBanks = make([]int, w.mapSize())
	for y := range w.WestBanks {
		w.WestBanks[y] = 2
	}
	if err := w.Build("shipbridge", 6, 13); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 700 && w.Companies[0].Stationed == 0; tick++ {
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
	}
	if w.Companies[0].Stationed == 0 {
		t.Fatal("company has not recruited")
	}
	if err := w.MilitaryOrder(fort, "muster", Point{1, 13}); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 80; tick++ {
		runValid(t, &w, 1)
		for _, troop := range w.Troops {
			if !w.navigableWater(Point{troop.X, troop.Y}) {
				continue
			}
			store, err := Open(filepath.Join(t.TempDir(), "crossing.nile"))
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
			runValid(t, &w, 20)
			runValid(t, &restored, 20)
			if restored.Hash() != w.Hash() {
				t.Fatal("mid-bridge replay diverged")
			}
			return
		}
	}
	t.Fatal("company never reached the bridge")
}

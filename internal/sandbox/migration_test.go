package sandbox

import (
	"path/filepath"
	"testing"
)

func TestMigrantFamiliesReserveSeparateHomesAndReload(t *testing.T) {
	w := NewTown()
	for _, p := range []Point{{9, 20}, {11, 25}} {
		if err := w.Build("hut", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Step(16); err != nil {
		t.Fatal(err)
	}
	parties := map[int64]bool{}
	for _, a := range w.Walkers {
		if a.Role == "immigrant" {
			if a.People != 4 || parties[a.Target] {
				t.Fatal("families must reserve distinct homes")
			}
			parties[a.Target] = true
		}
	}
	if len(parties) != 2 {
		t.Fatal("one long journey must not block other homes from receiving families")
	}
	store, err := Open(filepath.Join(t.TempDir(), "families.nile"))
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
	if err = w.Step(48); err != nil {
		t.Fatal(err)
	}
	for range 48 {
		if err = restored.Step(1); err != nil {
			t.Fatal(err)
		}
	}
	if restored.Hash() != w.Hash() || w.population() != 10 || w.Arrivals != 10 || w.Departures != 0 {
		t.Fatalf("families did not arrive conservatively: pop%d arrivals%d departures%d", w.population(), w.Arrivals, w.Departures)
	}
	if err = w.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrantLegacySingleAndCapacityLoss(t *testing.T) {
	w := NewTown()
	if err := w.Build("hut", 9, 2); err != nil {
		t.Fatal(err)
	}
	home := &w.Buildings[0]
	w.arriveMigrants(Walker{}, home)
	if home.Pop != 1 || w.Arrivals != 1 {
		t.Fatal("legacy in-flight walkers must still carry one person")
	}
	w.arriveMigrants(Walker{People: 4}, home)
	w.arriveMigrants(Walker{People: 4}, home)
	w.refresh()
	if home.Pop != 5 || w.Arrivals != 9 || w.Departures != 4 {
		t.Fatal("a full or devolved home must account for rejected arrivals")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrantFamiliesPrioritizeAmpleRoomBeforeSmallVacancies(t *testing.T) {
	w := NewTown()
	for _, p := range []Point{{9, 2}, {9, 6}, {9, 10}} {
		if err := w.Build("hut", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	// Isolate dispatch from housing evolution: an earlier home lacks one
	// person, a later established home can hold a family, and a vacant lot
	// must take precedence over both. This is the original three-pass order.
	w.Buildings[0].Pop, w.Buildings[0].Capacity = 4, 5
	w.Buildings[1].Pop, w.Buildings[1].Capacity = 4, 13
	w.Tick = MigrationInterval
	w.dispatchMigrants()
	if len(w.Walkers) != 1 || w.Walkers[0].Target != w.Buildings[2].ID || w.Walkers[0].People != 4 {
		t.Fatal("vacant lot did not receive the first family")
	}
	w.dispatchMigrants()
	if len(w.Walkers) != 2 || w.Walkers[1].Target != w.Buildings[1].ID || w.Walkers[1].People != 4 {
		t.Fatal("small early vacancy starved the later family-sized vacancy")
	}
	w.dispatchMigrants()
	if len(w.Walkers) != 3 || w.Walkers[2].Target != w.Buildings[0].ID || w.Walkers[2].People != 1 {
		t.Fatal("final pass did not fill the remaining small vacancy")
	}
}

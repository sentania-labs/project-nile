package sandbox

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/sentania-labs/project-nile/internal/probe"
)

func TestConnectedMigrationAndWaterCapacity(t *testing.T) {
	w := Initial()
	if err := w.Step(1000); err != nil {
		t.Fatal(err)
	}
	if w.View().Population != 10 {
		t.Fatalf("dry capacity: %+v", w.View())
	}
	if err := w.Build("well", 14, 23); err != nil {
		t.Fatal(err)
	}
	if w.Buildings[0].Tier != 1 || w.Buildings[0].Capacity != 7 || w.Buildings[1].Tier != 0 {
		t.Fatal("well should affect only nearby hut")
	}
	if err := w.Step(1000); err != nil {
		t.Fatal(err)
	}
	if w.View().Population != 12 {
		t.Fatalf("watered capacity: %d", w.View().Population)
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestVacantWateredPlotEvolvesOnlyAfterArrival(t *testing.T) {
	w := Initial()
	if err := w.Build("well", 14, 23); err != nil {
		t.Fatal(err)
	}
	if !w.Buildings[0].Water || w.Buildings[0].Tier != 0 {
		t.Fatal("vacant plot evolved")
	}
	if err := w.Step(100); err != nil {
		t.Fatal(err)
	}
	if w.Buildings[0].Pop == 0 || w.Buildings[0].Tier != 1 || w.Buildings[0].Capacity != 7 {
		t.Fatal("occupied watered house did not evolve")
	}
}

func TestFullMapResponseBound(t *testing.T) {
	w := World{Roads: []Point{Entry}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1}
	for x := 6; x < Size; x++ {
		for y := 0; y < Size; y++ {
			if (Point{x, y}) == Entry {
				continue
			}
			if err := w.Build("hut", x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal(w.View())
	if err != nil {
		t.Fatal(err)
	}
	if len(response) > 256*1024 {
		t.Fatalf("full map state exceeds expected bound: %d", len(response))
	}
	t.Logf("full map state: %d bytes for %d buildings", len(response), len(w.Buildings))
}

func TestDisconnectedHouseCannotReceiveMigrants(t *testing.T) {
	w := World{Roads: []Point{Entry}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1}
	if err := w.Build("hut", 9, 15); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(200); err != nil {
		t.Fatal(err)
	}
	if w.View().Population != 0 || len(w.Walkers) != 0 || w.Buildings[0].Connected {
		t.Fatal("disconnected migration")
	}
	for _, x := range []int{7, 8} {
		if err := w.Build("road", x, 15); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Step(100); err != nil {
		t.Fatal(err)
	}
	if w.View().Population != 5 {
		t.Fatal("connected house did not fill")
	}
}

func TestMigrantMovesOnRoadAndSurvivesReload(t *testing.T) {
	w := Initial()
	if err := w.Step(8); err != nil {
		t.Fatal(err)
	}
	if len(w.Walkers) != 1 {
		t.Fatal("no migrant dispatched")
	}
	path := filepath.Join(t.TempDir(), "sandbox.nile")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(w); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	resumed, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	for range 128 {
		before := append([]Walker{}, w.Walkers...)
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		if len(before) > 0 && len(w.Walkers) > 0 && before[0].ID == w.Walkers[0].ID {
			if !adjacent(Point{before[0].X, before[0].Y}, Point{w.Walkers[0].X, w.Walkers[0].Y}) {
				t.Fatal("migrant teleported")
			}
		}
		if err := w.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if err := resumed.Step(128); err != nil {
		t.Fatal(err)
	}
	if w.Hash() != resumed.Hash() {
		t.Fatal("replay diverged after reload")
	}
}

func TestPlacementErrorsPreserveWorld(t *testing.T) {
	w := Initial()
	for _, test := range []struct {
		kind string
		x, y int
	}{{"hut", 5, 15}, {"well", 32, 0}, {"road", 6, -1}, {"hut", 10, 10}, {"well", 14, 21}, {"bazaar", 15, 22}} {
		before := w.Hash()
		if w.Build(test.kind, test.x, test.y) == nil || w.Hash() != before {
			t.Fatalf("invalid placement mutated world: %+v", test)
		}
	}
}

func TestSaveModesCannotOverwriteEachOther(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.nile")
	s, err := probe.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(probe.Initial()); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if other, err := Open(path); err == nil {
		other.Close()
		t.Fatal("sandbox accepted diagnostic save")
	}
}

func TestInvalidSnapshotRouteRejected(t *testing.T) {
	w := Initial()
	w.Step(8)
	w.Walkers[0].Route[1] = Point{31, 31}
	if w.Validate() == nil {
		t.Fatal("corrupt route accepted")
	}
}

func TestClearingPreservesResidentsAndReservations(t *testing.T) {
	w := Initial()
	if err := w.Build("well", 14, 23); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(8); err != nil {
		t.Fatal(err)
	}
	path := w.Walkers[0].Route
	block := path[len(path)-1]
	before := w.Hash()
	if w.Build("clear", block.X, block.Y) == nil || w.Hash() != before {
		t.Fatal("removed reserved road")
	}
	if err := w.Step(1000); err != nil {
		t.Fatal(err)
	}
	before = w.Hash()
	if w.Build("clear", 14, 23) == nil || w.Hash() != before {
		t.Fatal("removed well needed for occupied housing")
	}
	if err := w.Build("clear", 7, 15); err != nil {
		t.Fatal(err)
	}
	if w.Buildings[0].Connected || w.View().Population != 12 {
		t.Fatal("clearing lost residents or failed to disconnect the road")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

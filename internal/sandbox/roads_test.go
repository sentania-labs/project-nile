package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
)

func TestNewKingdomRoadPassesThroughMapWithoutConstructionExpense(t *testing.T) {
	w := NewTown()
	v := w.View()
	if v.Entry != (Point{10, 0}) || v.Exit == nil || *v.Exit != (Point{10, 31}) || len(v.KingdomRoad) != Size || len(v.Roads) != Size || w.Expenses != 0 || w.Treasury != 2500 || v.HousingRoadReach != 2 {
		t.Fatal("new town lacks free initial pass-through road")
	}
	for _, p := range v.KingdomRoad {
		if !w.roadSet()[p] {
			t.Fatal("Kingdom Road is disconnected at creation")
		}
	}
	for _, p := range []Point{v.Entry, *v.Exit} {
		before := w.Hash()
		if w.Build("clear", p.X, p.Y) == nil || w.Hash() != before {
			t.Fatal("clearing an endpoint changed the map")
		}
	}
	if err := w.Build("clear", 10, 15); err != nil {
		t.Fatalf("interior Kingdom Road should remain editable: %v", err)
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRoadDragIsAtomicAndChargesOnlyNewTiles(t *testing.T) {
	w := NewTown()
	if err := w.BuildRoads([]Point{{10, 15}, {11, 15}, {12, 15}, {11, 15}, {11, 15}}); err != nil {
		t.Fatal(err)
	}
	if w.Expenses != 8 || len(w.Roads) != Size+2 {
		t.Fatal("existing or repeated tiles were charged")
	}
	before := w.Hash()
	if err := w.BuildRoads([]Point{{10, 15}, {11, 15}, {12, 15}}); err != nil || w.Hash() != before {
		t.Fatal("existing road drag was not a no-op")
	}
	if err := w.Build("hut", 14, 15); err != nil {
		t.Fatal(err)
	}
	for _, path := range [][]Point{nil, {{12, 15}, {13, 15}, {14, 15}}, {{11, 15}, {12, 16}}, {{6, 2}, {5, 2}}, {{31, 31}, {32, 31}}} {
		before := w.Hash()
		if w.BuildRoads(path) == nil || w.Hash() != before {
			t.Fatalf("invalid drag partly built: %v", path)
		}
	}
	w.Expenses, w.Treasury = 2496, 4
	before = w.Hash()
	if w.BuildRoads([]Point{{12, 15}, {12, 16}, {12, 17}}) == nil || w.Hash() != before {
		t.Fatal("insufficient funds produced a partial road")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestHousingRoadAccessTwoTilesGeometryAndObstructions(t *testing.T) {
	// Manual p40 establishes distance two. Diagonal and obstruction cases use
	// Akhenaten's square radius scan, pending original executable confirmation.
	for _, tc := range []struct {
		name         string
		house        Point
		blocker      bool
		isolatedRoad bool
		want         bool
	}{
		{"adjacent", Point{7, 15}, false, false, true},
		{"second_row", Point{8, 15}, false, false, true},
		{"square_corner_reference_inference", Point{8, 17}, false, false, true},
		{"third_row", Point{9, 15}, false, false, false},
		{"third_diagonal", Point{9, 18}, false, false, false},
		{"behind_building_reference_inference", Point{8, 15}, true, false, true},
		{"nearby_disconnected_road", Point{9, 15}, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := World{Roads: []Point{Entry}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1}
			if tc.blocker {
				if err := w.Build("well", 7, 15); err != nil {
					t.Fatal(err)
				}
			}
			if tc.isolatedRoad {
				if err := w.Build("road", 10, 15); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Build("hut", tc.house.X, tc.house.Y); err != nil {
				t.Fatal(err)
			}
			home := w.Buildings[len(w.Buildings)-1]
			if home.Connected != tc.want {
				t.Fatal("incorrect housing road access")
			}
			if err := w.Step(200); err != nil {
				t.Fatal(err)
			}
			if (w.population() > 0) != tc.want {
				t.Fatal("migration disagrees with access")
			}
			if err := w.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHousingReachDoesNotExpandIndustrialRoadAccess(t *testing.T) {
	w := NewTown()
	if err := w.Build("granary", 12, 10); err != nil {
		t.Fatal(err)
	}
	if w.Buildings[0].Connected {
		t.Fatal("industrial road access expanded to housing reach")
	}
	if err := w.Build("road", 11, 10); err != nil {
		t.Fatal(err)
	}
	if !w.Buildings[0].Connected {
		t.Fatal("industry failed to connect at its perimeter")
	}
}

func TestBuildersUseSecondRowHousingRoadAccess(t *testing.T) {
	w := builderFixture(t)
	w.Buildings[0].X, w.Buildings[0].Y = 8, 12
	w.refresh()
	seenWork, seenReturn := false, false
	for range 200 {
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		for _, walker := range w.Walkers {
			if walker.Role == "builder" {
				seenWork = seenWork || walker.Stage == "work"
				seenReturn = seenReturn || walker.Stage == "return"
			}
		}
		if err := w.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if !seenWork || !seenReturn {
		t.Fatal("second-row residents could not commute and return")
	}
}

func TestVersionFourSaveKeepsCityRoadsAndActiveMigrant(t *testing.T) {
	w := neighborhood()
	if err := w.Build("hut", 10, 0); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(8); err != nil {
		t.Fatal(err)
	}
	if len(w.Walkers) != 1 {
		t.Fatal("missing migration fixture")
	}
	w.Version = 4
	raw, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	s, err := Open(filepath.Join(t.TempDir(), "legacy-city.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.db.Exec("INSERT INTO snapshots(payload,checksum) VALUES(?,?)", raw, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.entry() != Entry || len(loaded.KingdomRoad) != 0 || loaded.ExitPoint != nil || !slices.Equal(loaded.Roads, w.Roads) || len(loaded.Buildings) != len(w.Buildings) || len(loaded.Walkers) != len(w.Walkers) {
		t.Fatal("migration rebuilt saved city or changed entry")
	}
	w.Version = SnapshotVersion
	w.addResourceTerrain()
	w.LegacyTaxes = true
	w.refresh()
	if loaded.Hash() != w.Hash() {
		t.Fatal("migration changed persisted jobs")
	}
	if err := loaded.Step(200); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(200); err != nil {
		t.Fatal(err)
	}
	if loaded.Hash() != w.Hash() {
		t.Fatal("legacy save replay diverged")
	}
	if err := s.Save(loaded); err != nil {
		t.Fatal(err)
	}
	loadedAgain, err := s.Load()
	if err != nil || loadedAgain.Hash() != loaded.Hash() {
		t.Fatal("migrated city failed to round trip")
	}
}

func TestFloodPresentationProfileAndSubmergedFarmWork(t *testing.T) {
	w := Initial()
	for phase := int64(0); phase < YearTicks; phase++ {
		w.Tick = (phase - SeasonOffset + YearTicks) % YearTicks
		level := w.View().FloodLevel
		if level < 0 || level > 1 || (phase >= 48 && level != 0) {
			t.Fatalf("invalid flood level at phase %d: %f", phase, level)
		}
		if phase >= 12 && phase <= 24 && level != 1 {
			t.Fatal("missing peak inundation")
		}
	}
	w = Initial()
	w.Tick = YearTicks - SeasonOffset - 1
	w.refresh()
	if err := w.Step(1); err != nil {
		t.Fatal(err)
	}
	for range 47 {
		for _, b := range w.Buildings {
			if b.Kind == "farm" && b.Work != 0 {
				t.Fatal("submerged farm grew crops")
			}
		}
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Step(1); err != nil {
		t.Fatal(err)
	}
	for _, b := range w.Buildings {
		if b.Kind == "farm" && b.Work != b.Workers {
			t.Fatal("farm did not resume work after inundation")
		}
	}
}

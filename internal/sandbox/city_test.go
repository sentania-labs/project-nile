package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"
)

func buildMonumentDistrict(t *testing.T, w *World) {
	t.Helper()
	for _, b := range []struct {
		kind string
		x, y int
	}{{"claypit", 6, 5}, {"brickworks", 24, 12}, {"storageyard", 25, 6}, {"mastaba", 24, 17}} {
		if err := w.Build(b.kind, b.x, b.y); err != nil {
			t.Fatalf("%s: %v", b.kind, err)
		}
	}
	roads := []Point{{8, 5}, {9, 5}, {10, 5}, {10, 6}, {10, 7}, {10, 8}, {10, 9}, {23, 10}, {24, 10}, {25, 10}, {25, 9}, {23, 12}, {23, 18}}
	for _, p := range roads {
		if err := w.BuildRoads([]Point{p}); err != nil {
			t.Fatal(err)
		}
	}
}
func TestCityBuildsMonumentWithPhysicalMaterials(t *testing.T) {
	w := Initial()
	buildMonumentDistrict(t, &w)
	materialSeen := map[string]bool{}
	completed := false
	for range 3000 {
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		for _, walker := range w.Walkers {
			if walker.Role == "porter" && walker.Cargo > 0 {
				materialSeen[walker.Resource] = true
			}
		}
		if err := w.Validate(); err != nil {
			t.Fatalf("tick%d: %v", w.Tick, err)
		}
		if w.View().MonumentProgress == 64 {
			completed = true
			break
		}
	}
	if !completed {
		t.Fatalf("monument stalled: %+v", w.View())
	}
	for _, resource := range []string{"clay", "straw", "bricks"} {
		if !materialSeen[resource] {
			t.Fatalf("no physical %s carrier", resource)
		}
	}
	if w.BricksConsumed != 64 || w.Income == 0 || w.Expenses == 0 {
		t.Fatal("missing construction or city accounts")
	}
	t.Logf("monument complete at tick%d, treasury%d, population%d", w.Tick, w.Treasury, w.population())
}
func TestNewTownAndConstructionBudget(t *testing.T) {
	w := NewTown()
	if len(w.Roads) != Size || len(w.Buildings) != 0 || w.Treasury != 2500 {
		t.Fatal("invalid new town")
	}
	if err := w.ZoneHousing(Point{7, 14}, Point{8, 14}); err != nil {
		t.Fatal(err)
	}
	if w.Treasury != 2480 || w.Expenses != 20 {
		t.Fatal("housing zoning did not charge cost")
	}
	before := w.Hash()
	if w.Build("granary", 7, 14) == nil || w.Hash() != before {
		t.Fatal("failed placement charged money")
	}
	w.Treasury = 0
	w.Expenses = 2500
	before = w.Hash()
	if w.Build("well", 10, 10) == nil || w.Hash() != before {
		t.Fatal("overspending allowed")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestServicePatrolsPreventHazardsAndRepairsCostMoney(t *testing.T) {
	protected := Initial()
	unprotected := Initial()
	for i := len(unprotected.Buildings) - 1; i >= 0; i-- {
		if unprotected.Buildings[i].Kind == "firehouse" || unprotected.Buildings[i].Kind == "architect" {
			p := unprotected.Buildings[i].point()
			if err := unprotected.Build("clear", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	for range 2 {
		protected.Step(1000)
		unprotected.Step(1000)
	}
	if protected.FireCount != 0 || protected.CollapseCount != 0 || unprotected.FireCount == 0 {
		t.Fatal("service prevention not demonstrated")
	}
	var damaged *Building
	for i := range unprotected.Buildings {
		if unprotected.Buildings[i].Damaged != "" {
			damaged = &unprotected.Buildings[i]
			break
		}
	}
	if damaged == nil {
		t.Fatal("no damaged building")
	}
	before := unprotected.Treasury
	if err := unprotected.Repair(damaged.X, damaged.Y); err != nil {
		t.Fatal(err)
	}
	if damaged.Damaged != "" || unprotected.Treasury >= before {
		t.Fatal("repair did not repair and charge")
	}
	if err := unprotected.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestMaterialDeliverySaveReplay(t *testing.T) {
	w := Initial()
	buildMonumentDistrict(t, &w)
	found := false
	for range 600 {
		w.Step(1)
		for _, walker := range w.Walkers {
			if walker.Role == "porter" && walker.Cargo > 0 {
				found = true
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no material delivery")
	}
	s, err := Open(filepath.Join(t.TempDir(), "city.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	w.Step(1000)
	restored.Step(1000)
	if w.Hash() != restored.Hash() {
		t.Fatal("material jobs diverged after save")
	}
}

func TestFreshTownCanReachCompletedMonument(t *testing.T) {
	w := NewTown()
	for x := 10; x <= 22; x++ {
		if err := w.BuildRoads([]Point{{x, 10}}); err != nil {
			t.Fatal(err)
		}
		if err := w.BuildRoads([]Point{{x, 20}}); err != nil {
			t.Fatal(err)
		}
	}
	for y := 11; y < 20; y++ {
		w.BuildRoads([]Point{{10, y}})
		w.BuildRoads([]Point{{22, y}})
	}
	for x := 7; x < 10; x++ {
		w.BuildRoads([]Point{{x, 15}})
	}
	if err := w.ZoneHousing(Point{14, 21}, Point{15, 21}); err != nil {
		t.Fatal(err)
	}
	if err := w.ZoneHousing(Point{19, 21}, Point{20, 21}); err != nil {
		t.Fatal(err)
	}
	for _, b := range []struct {
		kind string
		x, y int
	}{{"well", 14, 23}, {"well", 20, 23}, {"farm", 7, 10}, {"granary", 14, 6}, {"bazaar", 19, 8}, {"firehouse", 12, 9}, {"architect", 21, 9}, {"taxoffice", 11, 17}} {
		if err := w.Build(b.kind, b.x, b.y); err != nil {
			t.Fatal(err)
		}
	}
	buildMonumentDistrict(t, &w)
	cost := w.Expenses
	for range 3000 {
		w.Step(1)
		if err := w.Validate(); err != nil {
			t.Fatalf("tick%d: %v", w.Tick, err)
		}
		if w.View().MonumentProgress == 64 {
			t.Logf("fresh town: construction%d, completiontick%d, population%d, treasury%d", cost, w.Tick, w.population(), w.Treasury)
			return
		}
	}
	t.Fatalf("fresh town did not finish: %+v", w.View())
}

func TestMaterialBackpressureAndServiceClearingConserveState(t *testing.T) {
	w := Initial()
	buildMonumentDistrict(t, &w)
	for range 3 {
		w.Step(1000)
		if err := w.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	// Completed monuments stop accepting bricks; production must respect all
	// finite stores rather than dumping or inventing surplus material.
	if w.View().MonumentProgress != 64 {
		t.Fatal("construction incomplete")
	}
	before := w.View()
	for range 2 {
		w.Step(1000)
		if err := w.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	after := w.View()
	if after.BricksConsumed != before.BricksConsumed {
		t.Fatal("completed monument continued consuming bricks")
	}
}

func TestVersionTwoEconomySnapshotMigratesToCity(t *testing.T) {
	w := Initial()
	for i := len(w.Buildings) - 1; i >= 0; i-- {
		if w.Buildings[i].Kind == "firehouse" || w.Buildings[i].Kind == "architect" || w.Buildings[i].Kind == "taxoffice" {
			p := w.Buildings[i].point()
			if err := w.Build("clear", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	w.BudgetEnabled = false
	w.HazardsEnabled = false
	w.Treasury = 0
	w.Step(80)
	w.StrawProduced = 0
	for i := range w.Buildings {
		w.Buildings[i].Straw = 0
	}
	w.Version = 2
	raw, _ := json.Marshal(w)
	var legacy map[string]json.RawMessage
	json.Unmarshal(raw, &legacy)
	for _, key := range []string{"treasury", "income", "expenses", "last_income", "last_wages", "budget_enabled", "hazards_enabled", "fire_count", "collapse_count", "clay_produced", "clay_consumed", "straw_produced", "straw_consumed", "bricks_produced", "bricks_consumed"} {
		delete(legacy, key)
	}
	var buildings []map[string]json.RawMessage
	json.Unmarshal(legacy["buildings"], &buildings)
	for _, b := range buildings {
		for _, key := range []string{"clay", "straw", "bricks", "fire_risk", "collapse_risk", "damaged", "fire_until", "collapse_until", "tax_until", "fire_covered", "collapse_covered", "tax_covered", "construction_done", "construction_required", "stage", "repair_cost"} {
			delete(b, key)
		}
	}
	legacy["buildings"], _ = json.Marshal(buildings)
	var walkers []map[string]json.RawMessage
	json.Unmarshal(legacy["walkers"], &walkers)
	for _, walker := range walkers {
		delete(walker, "resource")
	}
	legacy["walkers"], _ = json.Marshal(walkers)
	raw, _ = json.Marshal(legacy)
	sum := sha256.Sum256(raw)
	store, err := Open(filepath.Join(t.TempDir(), "economy-v2.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err = store.db.Exec("INSERT INTO snapshots(payload,checksum) VALUES(?,?)", raw, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	w.enableCity()
	if loaded.Hash() != w.Hash() {
		t.Fatal("v2 migration changed economy state beyond explicit city defaults")
	}
}

func TestSaveOnEvictionTick(t *testing.T) {
	for _, cause := range []string{"fire", "collapse"} {
		t.Run(cause, func(t *testing.T) {
			w := Initial()
			home := &w.Buildings[0]
			if home.Tier == 0 || home.Pop == 0 {
				t.Fatal("expected occupied evolved home")
			}
			if cause == "fire" {
				home.FireRisk = 99
			} else {
				home.CollapseRisk = 99
			}
			if err := w.Step(1); err != nil {
				t.Fatal(err)
			}
			if home.Damaged != cause || home.Pop != 0 || home.Tier != 0 {
				t.Fatalf("unexpected home on damage tick: %+v", home)
			}
			store, err := Open(filepath.Join(t.TempDir(), "eviction.nile"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err := store.Save(w); err != nil {
				t.Fatal(err)
			}
			loaded, err := store.Load()
			if err != nil || loaded.Hash() != w.Hash() {
				t.Fatalf("damaged town failed to round trip: %v", err)
			}
		})
	}
}

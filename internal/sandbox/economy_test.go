package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestStarterPhysicalEconomyConservesGrain(t *testing.T) {
	w := Initial()
	roles := map[string]bool{}
	fed := map[int64]bool{}
	for tick := 1; tick <= 2000; tick++ {
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		for _, walker := range w.Walkers {
			if walker.Cargo > 0 {
				roles[walker.Role] = true
			}
		}
		for _, b := range w.Buildings {
			if b.Kind == "hut" && b.Tier == 2 {
				fed[b.ID] = true
			}
		}
		if err := w.Validate(); err != nil {
			t.Fatalf("tick%d: %v; %+v", tick, err, w.View())
		}
		if tick >= 600 {
			for _, b := range w.Buildings {
				if b.Kind == "hut" && b.Tier != 2 {
					t.Fatalf("starter ration failed at tick%d: %+v", tick, w.View())
				}
			}
		}
	}
	if !roles["cart"] || !roles["buyer"] || !roles["seller"] || len(fed) != 4 {
		t.Fatalf("missing physical chain: roles%v fed%v", roles, fed)
	}
	if w.GrainProduced == 0 || w.GrainConsumed == 0 || w.View().Population != 36 {
		t.Fatalf("economy did not mature: %+v", w.View())
	}
}
func TestFoodCutoffDevolvesAndAccountsForPopulation(t *testing.T) {
	w := Initial()
	if err := w.Step(1000); err != nil {
		t.Fatal(err)
	}
	cut := false
	for range 200 {
		if err := w.Build("clear", 7, 15); err == nil {
			cut = true
			break
		}
		w.Step(1)
	}
	if !cut {
		t.Fatal("could not clear an unreserved connection")
	}
	if err := w.Step(500); err != nil {
		t.Fatal(err)
	}
	for _, b := range w.Buildings {
		if b.Kind == "hut" && (b.Tier != 1 || b.Pop > 7) {
			t.Fatalf("disconnected house failed to devolve: %+v", b)
		}
	}
	if w.Departures == 0 {
		t.Fatal("capacity loss was not accounted")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestSaveRestoreWithGoodsInFlight(t *testing.T) {
	w := Initial()
	found := false
	for range 400 {
		w.Step(1)
		for _, walker := range w.Walkers {
			if walker.Role == "buyer" && walker.Cargo > 0 {
				found = true
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no in-flight cargo")
	}
	s, err := Open(filepath.Join(t.TempDir(), "goods.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	w.Step(1000)
	restored.Step(1000)
	if w.Hash() != restored.Hash() {
		t.Fatal("in-flight replay diverged")
	}
	if err = w.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestLoadsLegacyStreetSnapshot(t *testing.T) {
	original := neighborhood()
	original.Step(8)
	raw, _ := json.Marshal(original)
	var legacy map[string]json.RawMessage
	json.Unmarshal(raw, &legacy)
	for _, key := range []string{"version", "departures", "grain_produced", "grain_consumed"} {
		delete(legacy, key)
	}
	var buildings []map[string]json.RawMessage
	json.Unmarshal(legacy["buildings"], &buildings)
	for _, b := range buildings {
		for _, key := range []string{"width", "height", "stock_capacity", "workers", "workers_needed", "food", "market", "market_until", "work", "work_capacity"} {
			delete(b, key)
		}
	}
	legacy["buildings"], _ = json.Marshal(buildings)
	var walkers []map[string]json.RawMessage
	json.Unmarshal(legacy["walkers"], &walkers)
	for _, walker := range walkers {
		for _, key := range []string{"origin", "source", "stage", "reserved"} {
			delete(walker, key)
		}
	}
	legacy["walkers"], _ = json.Marshal(walkers)
	raw, _ = json.Marshal(legacy)
	sum := sha256.Sum256(raw)
	s, err := Open(filepath.Join(t.TempDir(), "legacy.nile"))
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
	if loaded.Hash() != original.Hash() {
		t.Fatal("legacy migration changed simulated state")
	}
	if err = s.Save(loaded); err != nil {
		t.Fatal(err)
	}
}
func TestFootprintCollisionAndPerimeterAccess(t *testing.T) {
	w := Initial()
	before := w.Hash()
	for _, p := range []Point{{15, 7}, {17, 9}, {9, 12}, {20, 9}} {
		if w.Build("road", p.X, p.Y) == nil || w.Hash() != before {
			t.Fatalf("allowed overlap at%v", p)
		}
	}
	if w.Build("farm", 8, 2) == nil {
		t.Fatal("allowed farm outside floodplain")
	}
	for _, b := range w.Buildings {
		if b.Kind == "granary" && !b.Connected {
			t.Fatal("granary perimeter road not recognized")
		}
	}
	if err := w.ZoneHousing(Point{14, 6}, Point{17, 9}); err == nil {
		t.Fatal("zoned housing over multi-tile building")
	}
}
func TestSellerUsesConnectedFrontage(t *testing.T) {
	w := Initial()
	if err := w.Build("road", 19, 7); err != nil {
		t.Fatal(err)
	}
	for range 200 {
		w.Step(1)
		for _, walker := range w.Walkers {
			if walker.Role == "seller" {
				if walker.Route[0] == (Point{19, 7}) {
					t.Fatal("seller used isolated frontage")
				}
				return
			}
		}
	}
	t.Fatal("no seller")
}
func TestUnstaffedFarmDoesNotGrowOrInventGrain(t *testing.T) {
	w := World{Version: SnapshotVersion, Roads: []Point{Entry}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1}
	if err := w.Build("farm", 6, 16); err != nil {
		t.Fatal(err)
	}
	w.Step(1000)
	if w.GrainProduced != 0 || w.View().Grain != 0 || w.Buildings[0].Workers != 0 {
		t.Fatal("unstaffed farm produced food")
	}
}

func TestCompetingBuyersKeepReservationsWithinStockAndCapacity(t *testing.T) {
	w := Initial()
	if err := w.Build("bazaar", 18, 12); err != nil {
		t.Fatal(err)
	}
	for _, p := range []Point{{20, 11}, {20, 12}} {
		if err := w.Build("road", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	buyers := map[int64]bool{}
	for range 1200 {
		w.Step(1)
		for _, walker := range w.Walkers {
			if walker.Role == "buyer" {
				buyers[walker.Origin] = true
			}
		}
		if err := w.Validate(); err != nil {
			t.Fatalf("tick%d: %v", w.Tick, err)
		}
	}
	if len(buyers) != 2 {
		t.Fatalf("not both buyers exercised: %v", buyers)
	}
}
func TestClearingProtectsPhysicalCargoAndBuildingFootprints(t *testing.T) {
	w := Initial()
	found := false
	for range 200 {
		w.Step(1)
		for _, walker := range w.Walkers {
			if walker.Role == "cart" && walker.Cargo > 0 {
				before := w.Hash()
				p := walker.Route[len(walker.Route)-1]
				if w.Build("clear", p.X, p.Y) == nil || w.Hash() != before {
					t.Fatal("cleared reserved cargo route")
				}
				if w.Build("clear", 15, 7) == nil || w.Hash() != before {
					t.Fatal("cleared granary with reserved delivery")
				}
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no cart cargo to protect")
	}
}

func TestMigrationPrioritizesConnectedEmptyPlots(t *testing.T) {
	w := Initial()
	if err := w.Build("hut", 30, 30); err != nil {
		t.Fatal(err)
	}
	disconnected := w.Buildings[len(w.Buildings)-1].ID
	if err := w.ZoneHousing(Point{14, 19}, Point{17, 19}); err != nil {
		t.Fatal(err)
	}
	newIDs := map[int64]bool{}
	for _, b := range w.Buildings {
		if b.Kind == "hut" && b.Y == 19 {
			newIDs[b.ID] = true
		}
	}
	w.Step(8)
	for _, walker := range w.Walkers {
		if walker.Role == "immigrant" && !newIDs[walker.Target] {
			t.Fatal("existing home took priority over vacant plots")
		}
	}
	w.Step(160)
	for _, b := range w.Buildings {
		if newIDs[b.ID] && b.Pop == 0 {
			t.Fatalf("new plot still vacant: %+v", b)
		}
		if b.ID == disconnected && b.Pop != 0 {
			t.Fatal("disconnected plot received residents")
		}
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

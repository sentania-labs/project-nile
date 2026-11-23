package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
)

func goldFixture(t *testing.T) World {
	t.Helper()
	w := Initial()
	if err := w.Step(1000); err != nil {
		t.Fatal(err)
	}
	if err := w.Build("goldmine", 28, 4); err != nil {
		t.Fatal(err)
	}
	path := []Point{}
	for x := 22; x <= 28; x++ {
		path = append(path, Point{x, 10})
	}
	for y := 9; y >= 6; y-- {
		path = append(path, Point{28, y})
	}
	if err := w.BuildRoads(path); err != nil {
		t.Fatal(err)
	}
	return w
}
func TestGoldMineNormalModelAndPlacementRequirements(t *testing.T) {
	mine, _ := catalog("goldmine")
	palace, _ := catalog("palace")
	if mine.Cost != 250 || mine.Workers != 12 || palace.Cost != 250 || palace.Workers != 20 {
		t.Fatal("original Normal model parameters diverged")
	}
	w := NewTown()
	for _, b := range []struct {
		kind string
		x, y int
	}{{"goldmine", 20, 4}, {"goldmine", 30, 4}, {"road", 30, 4}, {"hut", 31, 3}, {"palace", 20, 20}} {
		before := w.Hash()
		if w.Build(b.kind, b.x, b.y) == nil || w.Hash() != before {
			t.Fatalf("invalid resource placement changed city: %+v", b)
		}
	}
	if err := w.Build("goldmine", 28, 4); err != nil {
		t.Fatal(err)
	}
	if err := w.Build("palace", 11, 24); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	if w.Build("palace", 11, 0) == nil || w.Hash() != before {
		t.Fatal("multiple Palaces allowed")
	}
	before = w.Hash()
	if w.BuildRoads([]Point{{29, 8}, {30, 8}, {30, 7}}) == nil || w.Hash() != before {
		t.Fatal("road drag crossed ore rock")
	}
	if err := w.ZoneHousing(Point{29, 3}, Point{31, 3}); err != nil {
		t.Fatal(err)
	}
	for _, b := range w.Buildings {
		for _, p := range b.tiles() {
			if slices.Contains(w.OreRock, p) {
				t.Fatal("zoning covered rock")
			}
		}
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestGoldCreditsTreasuryOnlyAfterPhysicalPalaceDelivery(t *testing.T) {
	w := goldFixture(t)
	produced, carried, returned := false, false, false
	for range 600 {
		old := map[int64]Point{}
		for _, walker := range w.Walkers {
			if walker.Role == "goldcart" {
				old[walker.ID] = Point{walker.X, walker.Y}
			}
		}
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		if w.GoldProduced > 0 && w.GoldDelivered == 0 {
			produced = true
		}
		for _, walker := range w.Walkers {
			if walker.Role != "goldcart" {
				continue
			}
			carried = carried || walker.Cargo > 0
			returned = returned || walker.Stage == "return"
			if p, ok := old[walker.ID]; ok && abs(p.X-walker.X)+abs(p.Y-walker.Y) > 1 {
				t.Fatal("gold cart teleported")
			}
		}
		if err := w.Validate(); err != nil {
			t.Fatalf("tick %d: %v", w.Tick, err)
		}
	}
	v := w.View()
	if !produced || !carried || !returned || w.GoldDelivered == 0 || int64(v.Gold)+w.GoldDelivered != w.GoldProduced || v.TaxIncome+w.GoldDelivered != w.Income {
		t.Fatal("physical gold chain or income ledger missing")
	}
	t.Logf("tick %d: gold produced %d, delivered %d, uncredited %d, treasury %d", w.Tick, w.GoldProduced, w.GoldDelivered, v.Gold, w.Treasury)
}
func TestGoldInFlightSaveReplayAndRouteProtection(t *testing.T) {
	w := goldFixture(t)
	var cart Walker
	for range 300 {
		w.Step(1)
		for _, walker := range w.Walkers {
			if walker.Role == "goldcart" && walker.Cargo > 0 {
				cart = walker
				break
			}
		}
		if cart.ID != 0 {
			break
		}
	}
	if cart.ID == 0 {
		t.Fatal("no gold delivery")
	}
	before := w.Hash()
	p := cart.Route[len(cart.Route)/2]
	if w.Build("clear", p.X, p.Y) == nil || w.Hash() != before {
		t.Fatal("cleared reserved gold road")
	}
	if w.Build("clear", 11, 24) == nil || w.Hash() != before {
		t.Fatal("cleared Palace receiving gold")
	}
	s, err := Open(filepath.Join(t.TempDir(), "gold.nile"))
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
	w.Step(600)
	restored.Step(600)
	if w.Hash() != restored.Hash() {
		t.Fatal("gold delivery replay diverged")
	}
}
func TestGoldWithoutPalaceBackpressuresAndNeverUsesStorageYard(t *testing.T) {
	w := goldFixture(t)
	if err := w.Build("clear", 11, 24); err != nil {
		t.Fatal(err)
	}
	if err := w.Build("storageyard", 25, 6); err != nil {
		t.Fatal(err)
	}
	income := w.Income
	if err := w.Step(600); err != nil {
		t.Fatal(err)
	}
	if w.GoldDelivered != 0 || w.GoldProduced != GoldMineCapacity || w.Income != income {
		t.Fatal("missing Palace failed to stop credit or bound inventory")
	}
	for _, b := range w.Buildings {
		if b.Kind == "storageyard" && b.Gold != 0 {
			t.Fatal("gold sent to storage")
		}
	}
	for _, walker := range w.Walkers {
		if walker.Resource == "gold" {
			t.Fatal("gold dispatched without Palace")
		}
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := w.Build("palace", 11, 24); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(200); err != nil {
		t.Fatal(err)
	}
	if w.GoldDelivered == 0 {
		t.Fatal("gold did not resume after Palace rebuilt")
	}
}
func TestGoldWaitsWhenPalaceLosesStaffDuringTrip(t *testing.T) {
	w := goldFixture(t)
	for range 300 {
		w.Step(1)
		if w.View().GoldInTransit > 0 {
			break
		}
	}
	if w.View().GoldInTransit == 0 {
		t.Fatal("missing cart")
	}
	if err := w.SetPolicy(2, 0, "balanced"); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(150); err != nil {
		t.Fatal(err)
	}
	if w.GoldDelivered != 0 || w.View().GoldInTransit != GoldLoad {
		t.Fatal("unpaid Palace credited or lost gold")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := w.SetPolicy(2, 1, "balanced"); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(1); err != nil {
		t.Fatal(err)
	}
	if w.GoldDelivered != GoldLoad {
		t.Fatal("waiting cart did not settle once administration returned")
	}
}
func TestDisconnectedGoldMineCannotProduceOrDispatch(t *testing.T) {
	w := goldFixture(t)
	if err := w.Build("clear", 28, 8); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(400); err != nil {
		t.Fatal(err)
	}
	if w.GoldProduced != 0 || w.GoldDelivered != 0 {
		t.Fatal("disconnected mine produced gold")
	}
	if err := w.Build("road", 28, 8); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(300); err != nil {
		t.Fatal(err)
	}
	if w.GoldDelivered == 0 {
		t.Fatal("reconnected mine did not deliver")
	}
}
func TestFreshTaxesRequireOperatingPalaceAndLegacySaveKeepsPriorFlow(t *testing.T) {
	w := Initial()
	if err := w.Build("clear", 11, 24); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(400); err != nil {
		t.Fatal(err)
	}
	if w.Income != 0 || w.View().TaxesEnabled {
		t.Fatal("fresh city collected tax without Palace")
	}
	// Construct a genuine pre-resource-save shape, including an occupied ore
	// candidate. Migration must preserve all buildings, roads, jobs and money.
	if err := w.Build("hut", 30, 4); err == nil {
		t.Fatal("resource terrain fixture should block building")
	}
	w.OreRock = nil
	w.Grassland = nil
	if err := w.Build("hut", 30, 4); err != nil {
		t.Fatal(err)
	}
	w.Version = 5
	raw, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	s, err := Open(filepath.Join(t.TempDir(), "city-v5.nile"))
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
	if !loaded.LegacyTaxes || loaded.Treasury != w.Treasury || !slices.Equal(loaded.Roads, w.Roads) || len(loaded.Buildings) != len(w.Buildings) || slices.Contains(loaded.OreRock, Point{30, 4}) {
		t.Fatal("legacy city or tax semantics changed")
	}
	if err := loaded.Step(400); err != nil {
		t.Fatal(err)
	}
	if loaded.Income == 0 {
		t.Fatal("legacy taxes did not continue")
	}
	if err := loaded.Build("palace", 11, 24); err != nil {
		t.Fatal(err)
	}
	if loaded.LegacyTaxes {
		t.Fatal("building Palace did not adopt current taxation")
	}
	if err := s.Save(loaded); err != nil {
		t.Fatal(err)
	}
}

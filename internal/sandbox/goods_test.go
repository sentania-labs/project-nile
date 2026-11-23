package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
)

func goodsFixture(t *testing.T) World {
	t.Helper()
	w := NewTown()
	w.BudgetEnabled = false
	w.HazardsEnabled = false
	for x := 11; x <= 25; x++ {
		if err := w.BuildRoads([]Point{{x, 10}}); err != nil {
			t.Fatal(err)
		}
		if err := w.BuildRoads([]Point{{x, 20}}); err != nil {
			t.Fatal(err)
		}
	}
	for y := 11; y < 20; y++ {
		if err := w.BuildRoads([]Point{{25, y}}); err != nil {
			t.Fatal(err)
		}
	}
	for x := 11; x <= 22; x++ {
		if err := w.Build("hut", x, 21); err != nil {
			t.Fatal(err)
		}
		w.Buildings[len(w.Buildings)-1].Pop = 5
		w.Arrivals += 5
	}
	w.refresh()
	return w
}
func addGoodsBuilding(t *testing.T, w *World, kind string, x, y int) int64 {
	t.Helper()
	if err := w.Build(kind, x, y); err != nil {
		t.Fatal(err)
	}
	return w.Buildings[len(w.Buildings)-1].ID
}
func runValid(t *testing.T, w *World, ticks int) {
	t.Helper()
	for range ticks {
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		if err := w.Validate(); err != nil {
			t.Fatalf("tick %d: %v", w.Tick, err)
		}
	}
}
func TestProductionChainsUsePhysicalInputsAndDeliverOutput(t *testing.T) {
	for _, tc := range []struct {
		raw, factory, input, output string
		x, y                        int
	}{{"barleyfarm", "brewery", "barley", "beer", 7, 10}, {"flaxfarm", "weaver", "flax", "linen", 7, 10}, {"reedgatherer", "papyrusmaker", "reeds", "papyrus", 7, 24}, {"claypit", "potter", "clay", "pottery", 7, 5}} {
		t.Run(tc.output, func(t *testing.T) {
			w := goodsFixture(t)
			yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
			w.BuildRoads([]Point{{18, 9}})
			factory := addGoodsBuilding(t, &w, tc.factory, 14, 8)
			raw := addGoodsBuilding(t, &w, tc.raw, tc.x, tc.y)
			if tc.raw == "reedgatherer" {
				w.BuildRoads([]Point{{9, 24}})
			}
			if tc.raw == "claypit" {
				w.BuildRoads([]Point{{9, 5}})
			}
			inputSeen, outputSeen, gathered := false, false, false
			for range 900 {
				w.Step(1)
				for _, a := range w.Walkers {
					if a.Cargo > 0 && a.Resource == tc.input {
						inputSeen = true
					}
					if a.Cargo > 0 && a.Resource == tc.output {
						outputSeen = true
					}
					if a.Role == "reedharvester" && a.Cargo > 0 {
						gathered = true
					}
				}
				if err := w.Validate(); err != nil {
					t.Fatalf("tick%d: %v", w.Tick, err)
				}
			}
			if !inputSeen || !outputSeen || w.ResourceAmount(yard, tc.output) == 0 {
				t.Fatalf("chain stalled: raw%v factory%v yard%v", w.building(raw), w.building(factory), w.building(yard))
			}
			if tc.raw == "reedgatherer" && !gathered {
				t.Fatal("no physical reed harvest")
			}
			p, c := w.resourceCounters(tc.input)
			if p == 0 || c == 0 {
				t.Fatal("input production/consumption not accounted")
			}
		})
	}
}
func TestStorageRefuseGetEmptyAndInFlightOrdersPreserveGoods(t *testing.T) {
	w := goodsFixture(t)
	a := addGoodsBuilding(t, &w, "storageyard", 11, 6)
	b := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	w.BuildRoads([]Point{{11, 9}})
	w.BuildRoads([]Point{{18, 9}})
	if w.ProduceResource(a, "beer", 200) != 200 {
		t.Fatal("fixture production rejected")
	}
	w.refresh()
	if err := w.SetStorageOrder(18, 6, "beer", "refuse"); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 64)
	if w.ResourceAmount(b, "beer") != 0 {
		t.Fatal("refusing yard received goods")
	}
	if err := w.SetStorageOrder(18, 6, "beer", "get"); err != nil {
		t.Fatal(err)
	}
	found := false
	for range 64 {
		w.Step(1)
		for _, x := range w.Walkers {
			if x.Role == "goodsbuyer" && x.Origin == b {
				found = true
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("get order did not dispatch a physical buyer")
	}
	// Changing the order honours the cargo reservation already in flight.
	if err := w.SetStorageOrder(18, 6, "beer", "refuse"); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 100)
	if w.ResourceAmount(b, "beer") == 0 {
		t.Fatal("in-flight reservation was discarded")
	}
	if err := w.SetStorageOrder(18, 6, "beer", "empty"); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 100)
	if w.ResourceAmount(b, "beer") != 0 || w.ResourceAmount(a, "beer") != 200 {
		t.Fatal("empty order lost or failed to move goods")
	}
	before := w.Hash()
	if w.SetStorageOrder(11, 6, "beer", "teleport") == nil || w.Hash() != before {
		t.Fatal("invalid order mutated city")
	}
}
func TestGoodsBuyerReservationsSurviveSaveAndNeverOverbook(t *testing.T) {
	w := goodsFixture(t)
	source := addGoodsBuilding(t, &w, "storageyard", 11, 6)
	a := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	b := addGoodsBuilding(t, &w, "storageyard", 22, 6)
	for _, x := range []int{11, 18, 22} {
		w.BuildRoads([]Point{{x, 9}})
	}
	w.ProduceResource(source, "linen", 150)
	w.refresh()
	w.SetStorageOrder(18, 6, "linen", "get")
	w.SetStorageOrder(22, 6, "linen", "get")
	runValid(t, &w, 8)
	if w.reservedResource(source, "linen") != 150 {
		t.Fatal("competing reservations did not partition stock")
	}
	s, err := Open(filepath.Join(t.TempDir(), "goods.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Save(w); err != nil {
		t.Fatal(err)
	}
	resumed, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 200)
	runValid(t, &resumed, 200)
	if w.Hash() != resumed.Hash() {
		t.Fatal("goods replay diverged")
	}
	if w.ResourceAmount(a, "linen")+w.ResourceAmount(b, "linen") != 150 {
		t.Fatal("goods reservation conservation failed")
	}
}
func TestBazaarObtainsManufacturedGoodsFromStorage(t *testing.T) {
	w := goodsFixture(t)
	yard := addGoodsBuilding(t, &w, "storageyard", 11, 6)
	bazaar := addGoodsBuilding(t, &w, "bazaar", 18, 8)
	w.BuildRoads([]Point{{11, 9}})
	for _, r := range []string{"pottery", "beer", "linen"} {
		w.ProduceResource(yard, r, 100)
	}
	w.refresh()
	seen := map[string]bool{}
	for range 150 {
		w.Step(1)
		for _, a := range w.Walkers {
			if a.Role == "goodsbuyer" && a.Cargo > 0 {
				seen[a.Resource] = true
			}
		}
		if err := w.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []string{"pottery", "beer", "linen"} {
		if !seen[r] {
			t.Fatalf("bazaar never physically received %s", r)
		}
	}
	if w.building(bazaar).Kind != "bazaar" {
		t.Fatal("fixture missing bazaar")
	}
}
func TestResourceBuffersBackpressureAndCountersAreValidated(t *testing.T) {
	w := goodsFixture(t)
	yard := addGoodsBuilding(t, &w, "storageyard", 11, 6)
	if n := w.ProduceResource(yard, "papyrus", 1000); n != 800 {
		t.Fatalf("buffer admitted %d", n)
	}
	w.refresh()
	if w.ConsumeResource(yard, "papyrus", 20) != 20 {
		t.Fatal("resource consumption failed")
	}
	w.refresh()
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(w)
	var corrupt World
	json.Unmarshal(raw, &corrupt)
	corrupt.GoodsProduced["papyrus"]++
	if corrupt.Validate() == nil {
		t.Fatal("invalid resource ledger accepted")
	}
}

func TestVersionSixCityMigratesInventoriesWithoutLosingCargo(t *testing.T) {
	w := goldFixture(t)
	for range 300 {
		w.Step(1)
		if w.View().GoldInTransit > 0 {
			break
		}
	}
	if w.View().GoldInTransit == 0 {
		t.Fatal("missing legacy in-flight gold")
	}
	raw, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]json.RawMessage
	json.Unmarshal(raw, &legacy)
	legacy["version"] = json.RawMessage("6")
	for _, key := range []string{"goods_produced", "goods_consumed", "reed_beds", "hunting_grounds", "campaign"} {
		delete(legacy, key)
	}
	var buildings []map[string]json.RawMessage
	json.Unmarshal(legacy["buildings"], &buildings)
	for _, b := range buildings {
		for _, key := range []string{"goods", "storage_orders", "inventories", "resource_limits"} {
			delete(b, key)
		}
	}
	legacy["buildings"], _ = json.Marshal(buildings)
	raw, _ = json.Marshal(legacy)
	sum := sha256.Sum256(raw)
	store, err := Open(filepath.Join(t.TempDir(), "integrated-v6.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err = store.db.Exec("INSERT INTO snapshots(payload,checksum) VALUES(?,?)", raw, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if restored.GoldProduced != w.GoldProduced || restored.GoldDelivered != w.GoldDelivered || restored.Treasury != w.Treasury || !slices.Equal(restored.Roads, w.Roads) || len(restored.Walkers) != len(w.Walkers) {
		t.Fatal("migration changed existing city or cargo")
	}
	runValid(t, &restored, 300)
	if restored.GoldDelivered == 0 {
		t.Fatal("migrated delivery did not finish")
	}
	if err = store.Save(restored); err != nil {
		t.Fatal(err)
	}
	again, err := store.Load()
	if err != nil || again.Hash() != restored.Hash() {
		t.Fatal("migrated inventories failed to round trip")
	}
}
func TestReturningReedHarvesterSaveKeepsPhysicalCargo(t *testing.T) {
	w := goodsFixture(t)
	id := addGoodsBuilding(t, &w, "reedgatherer", 7, 24)
	w.BuildRoads([]Point{{9, 24}})
	found := false
	for range 100 {
		w.Step(1)
		for _, a := range w.Walkers {
			if a.Role == "reedharvester" && a.Cargo > 0 {
				found = true
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no returning reed load")
	}
	store, err := Open(filepath.Join(t.TempDir(), "reeds.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 100)
	runValid(t, &loaded, 100)
	if w.Hash() != loaded.Hash() || w.ResourceAmount(id, "reeds") == 0 {
		t.Fatal("reed cargo replay failed")
	}
}

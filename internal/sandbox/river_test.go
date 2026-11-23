package sandbox

import (
	"path/filepath"
	"testing"
)

func riverFixture(t *testing.T) World {
	t.Helper()
	w := goodsFixture(t)
	for x := 11; x <= 20; x++ {
		id := addGoodsBuilding(t, &w, "hut", x, 19)
		w.building(id).Pop = 5
		w.Arrivals += 5
	}
	w.refresh()
	return w
}
func riverRoad(t *testing.T, w *World, y, x int) {
	t.Helper()
	path := []Point{}
	for ; x <= 10; x++ {
		path = append(path, Point{x, y})
	}
	if err := w.BuildRoads(path); err != nil {
		t.Fatal(err)
	}
}
func TestFishingRequiresShipwrightAndPhysicalFishDistribution(t *testing.T) {
	w := riverFixture(t)
	wharf := addGoodsBuilding(t, &w, "fishingwharf", 5, 9)
	riverRoad(t, &w, 9, 7)
	addGoodsBuilding(t, &w, "granary", 14, 6)
	addGoodsBuilding(t, &w, "bazaar", 14, 11)
	runValid(t, &w, 50)
	if len(w.Boats) > 0 || w.GoodsProduced["fish"] > 0 {
		t.Fatal("fishing without a shipwright")
	}
	addGoodsBuilding(t, &w, "shipwright", 5, 3)
	riverRoad(t, &w, 4, 8)
	carried, cart, house := false, false, false
	var resumed *World
	store, err := Open(filepath.Join(t.TempDir(), "fish.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for range 650 {
		runValid(t, &w, 1)
		for _, boat := range w.Boats {
			if boat.Kind == "fishingboat" && boat.Cargo > 0 {
				carried = true
				if resumed == nil {
					if err := store.Save(w); err != nil {
						t.Fatal(err)
					}
					loaded, err := store.Load()
					if err != nil {
						t.Fatal(err)
					}
					resumed = &loaded
				}
			}
		}
		for _, a := range w.Walkers {
			if a.Resource == "fish" && a.Cargo > 0 {
				cart = true
			}
		}
		for _, b := range w.Buildings {
			if b.Kind == "hut" && inventory(b, "fish") > 0 {
				house = true
			}
		}
		if resumed != nil && resumed.Tick < w.Tick {
			runValid(t, resumed, 1)
		}
	}
	if !carried || !cart || !house || w.GoodsConsumed["fish"] == 0 || w.GoodsConsumed["timber"] != 0 || w.ShipsBuilt != 1 {
		t.Fatalf("incomplete fishing chain boats%v carts%v house%v produced%d consumed%d", carried, cart, house, w.GoodsProduced["fish"], w.GoodsConsumed["fish"])
	}
	if resumed == nil || resumed.Hash() != w.Hash() {
		t.Fatal("fish boat replay diverged")
	}
	if w.Build("clear", w.building(wharf).X+1, w.building(wharf).Y) == nil {
		t.Fatal("occupied wharf could be cleared")
	}
}
func TestSeaTradeUsesDockAndPhysicalWarehouseCarts(t *testing.T) {
	w := riverFixture(t)
	dock := addGoodsBuilding(t, &w, "dock", 5, 3)
	riverRoad(t, &w, 4, 8)
	yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	if err := w.BuildRoads([]Point{{18, 9}}); err != nil {
		t.Fatal(err)
	}
	if err := w.OpenTrade(seaRouteID); err != nil {
		t.Fatal(err)
	}
	w.SetTradePolicy("stone", "import", 300)
	foreign, land := false, false
	for range 160 {
		runValid(t, &w, 1)
		for _, a := range w.Boats {
			if a.Kind == "merchantship" && a.Stage == "import" {
				foreign = true
				if w.ResourceImported["stone"] != 0 {
					t.Fatal("sea import paid before arrival")
				}
			}
		}
		for _, a := range w.Walkers {
			if a.Origin == dock && a.Resource == "stone" && a.Cargo > 0 {
				land = true
			}
		}
	}
	if !foreign || !land || w.ResourceAmount(yard, "stone") != 300 || w.ImportSpent != 114 {
		t.Fatalf("sea import failed foreign%v land%v yard%d spent%d", foreign, land, w.ResourceAmount(yard, "stone"), w.ImportSpent)
	}
	w.SetTradePolicy("stone", "export", 100)
	fetch := false
	for range 180 {
		runValid(t, &w, 1)
		for _, a := range w.Walkers {
			if a.Role == "goodsbuyer" && a.Origin == dock && a.Resource == "stone" {
				fetch = true
			}
		}
	}
	if !fetch || w.ResourceExported["stone"] != 200 || w.cityResource("stone") != 100 || w.ResourceAmount(yard, "stone") != 100 || w.ExportEarned != 58 {
		t.Fatalf("sea export failed fetch%v sold%d remaining%d earnings%d", fetch, w.ResourceExported["stone"], w.cityResource("stone"), w.ExportEarned)
	}
}
func TestMilitaryShipsConsumeDeliveredTimberAndStayInWater(t *testing.T) {
	w := riverFixture(t)
	yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	w.BuildRoads([]Point{{18, 9}})
	shipwright := addGoodsBuilding(t, &w, "shipwright", 5, 3)
	riverRoad(t, &w, 4, 8)
	addGoodsBuilding(t, &w, "transportwharf", 5, 14)
	riverRoad(t, &w, 15, 7)
	runValid(t, &w, 100)
	if len(w.Boats) != 0 {
		t.Fatal("military ship built without timber")
	}
	w.ProduceResource(yard, "timber", 400)
	w.refresh()
	delivered := false
	for range 300 {
		runValid(t, &w, 1)
		for _, a := range w.Walkers {
			if a.Target == shipwright && a.Resource == "timber" && a.Cargo > 0 {
				delivered = true
			}
		}
		if len(w.Boats) > 0 && w.Boats[0].Stage == "moored" {
			break
		}
	}
	if !delivered || len(w.Boats) != 1 || w.GoodsConsumed["timber"] != 400 {
		t.Fatal("timber did not become a physical transport ship")
	}
	before := w.Hash()
	if w.NavalOrder(w.Boats[0].ID, "hold", Point{15, 15}) == nil || w.Hash() != before {
		t.Fatal("ship could sail onto land")
	}
	if err := w.NavalOrder(w.Boats[0].ID, "hold", Point{1, 28}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 50)
	if w.Boats[0].Stage != "hold" || w.Boats[0].X != 1 || w.Boats[0].Y != 28 {
		t.Fatal("ship failed water order")
	}
	w.NavalOrder(w.Boats[0].ID, "return", Point{})
	runValid(t, &w, 50)
	if w.Boats[0].Stage != "moored" {
		t.Fatal("ship failed return")
	}
}
func TestRiverPlacementRejectsInlandBendsAndCrossLandRoutes(t *testing.T) {
	w := NewValley()
	before := w.Hash()
	if w.Build("fishingwharf", 30, 30) == nil || w.Hash() != before {
		t.Fatal("inland wharf accepted")
	}
	// The valley shore bends inside these rows; pier placement must remain atomic.
	y := 0
	for ; y < w.mapSize()-2; y++ {
		if w.bank(y) != w.bank(y+1) {
			break
		}
	}
	if w.Build("fishingwharf", w.bank(y)-1, y) == nil || w.Hash() != before {
		t.Fatal("wharf crosses a shoreline bend")
	}
	route := w.waterPath(w.seaEntry(), Point{w.westBank(95), 95})
	if len(route) < 96 {
		t.Fatal("large-world river route failed")
	}
	if w.waterPath(w.seaEntry(), Point{95, 95}) != nil {
		t.Fatal("water route crossed land")
	}
}

func TestKingdomDeploymentBoardsSailsWaitsAndReturnsSurvivors(t *testing.T) {
	w, fort := armyFixture(t)
	for x := 11; x <= 20; x++ {
		id := addGoodsBuilding(t, &w, "hut", x, 19)
		w.building(id).Pop = 5
		w.Arrivals += 5
	}
	for x := 11; x <= 20; x++ {
		id := addGoodsBuilding(t, &w, "hut", x, 18)
		w.building(id).Pop = 5
		w.Arrivals += 5
	}
	yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	w.BuildRoads([]Point{{18, 9}})
	addGoodsBuilding(t, &w, "shipwright", 5, 3)
	riverRoad(t, &w, 4, 8)
	addGoodsBuilding(t, &w, "transportwharf", 5, 14)
	riverRoad(t, &w, 15, 7)
	w.ProduceResource(yard, "timber", 400)
	w.refresh()
	runValid(t, &w, 1000)
	if len(w.Boats) != 1 || w.Boats[0].Stage != "moored" || w.Companies[0].Stationed < 8 {
		t.Fatalf("deployment preparation failed %+v %+v", w.Boats, w.Companies)
	}
	if err := w.StartKingdomDeployment(42, 7, 8); err != nil {
		t.Fatal(err)
	}
	if w.Deployments[0].Status != "boarding" || w.Troops[0].Stage != "embark" {
		t.Fatal("soldiers teleported to ship")
	}
	before := w.Hash()
	if w.StartKingdomDeployment(42, 7, 8) == nil || w.MilitaryOrder(fort, "return", Point{}) == nil || w.NavalOrder(w.Boats[0].ID, "return", Point{}) == nil || w.Hash() != before {
		t.Fatal("active deployment could be overwritten")
	}
	store, err := Open(filepath.Join(t.TempDir(), "deployment.nile"))
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
	embarked := false
	for range 100 {
		runValid(t, &w, 1)
		runValid(t, &restored, 1)
		if w.Deployments[0].Status == "outbound" {
			embarked = true
		}
	}
	if !embarked || w.Deployments[0].Status != "awaits_outcome" || w.Hash() != restored.Hash() {
		t.Fatal("transport voyage or replay failed")
	}
	runValid(t, &w, 50)
	if w.Deployments[0].Status != "awaits_outcome" {
		t.Fatal("transport invented a campaign outcome")
	}
	lost := w.Military.Lost
	if err = w.CompleteKingdomDeployment(42, 6, false); err != nil {
		t.Fatal(err)
	}
	if w.Military.Lost != lost+2 || w.Deployments[0].Status != "returning" {
		t.Fatal("survivors not conserved")
	}
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 120)
	runValid(t, &restored, 120)
	if w.Hash() != restored.Hash() || w.Deployments[0].Status != "returned" || w.Boats[0].Stage != "moored" || w.Companies[0].Order != "return" || w.Companies[0].Stationed != w.Companies[0].Soldiers {
		t.Fatal("survivors failed to disembark, walk home and replay")
	}
}

func TestSeaInvadersLandWithoutNavyAndWarshipsStopLoadedTransports(t *testing.T) {
	for _, defend := range []bool{false, true} {
		w := riverFixture(t)
		if defend {
			yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
			w.BuildRoads([]Point{{18, 9}})
			addGoodsBuilding(t, &w, "shipwright", 5, 3)
			riverRoad(t, &w, 4, 8)
			addGoodsBuilding(t, &w, "warshipwharf", 5, 14)
			riverRoad(t, &w, 15, 8)
			w.ProduceResource(yard, "timber", 800)
			w.refresh()
			runValid(t, &w, 220)
			if len(w.Boats) != 1 || w.Boats[0].Kind != "warship" {
				t.Fatal("warship construction failed")
			}
			if err := w.NavalOrder(w.Boats[0].ID, "hold", Point{4, 10}); err != nil {
				t.Fatal(err)
			}
			runValid(t, &w, 30)
			if err := w.NavalOrder(w.Boats[0].ID, "seek", Point{}); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.SpawnSeaInvasion(73, "foreign", 8, w.seaEntry()); err != nil {
			t.Fatal(err)
		}
		if w.passengersAtSea(73) != 8 || len(w.Troops) != 0 {
			t.Fatal("naval invaders teleported onto land")
		}
		store, err := Open(filepath.Join(t.TempDir(), "sea-battle.nile"))
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Save(w); err != nil {
			t.Fatal(err)
		}
		copy, err := store.Load()
		if err != nil {
			t.Fatal(err)
		}
		store.Close()
		runValid(t, &w, 260)
		runValid(t, &copy, 260)
		if w.Hash() != copy.Hash() {
			t.Fatal("naval battle replay diverged")
		}
		if defend {
			if w.Military.Breaches != 0 || w.Military.Defeated != 8 || w.Naval.Destroyed < 1 {
				t.Fatalf("warship failed interception %+v %+v", w.Military, w.Naval)
			}
		} else {
			if w.Military.Breaches != 8 {
				t.Fatalf("unopposed sea invasion never landed %+v", w.Military)
			}
		}
	}
}

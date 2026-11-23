package sandbox

import (
	"path/filepath"
	"testing"
)

func TestLocalTransportBoardsCompanyCrossesRiverAndUnloadsPhysically(t *testing.T) {
	w := riverFixture(t)
	w.WestBanks = make([]int, w.mapSize())
	for y := range w.WestBanks {
		w.WestBanks[y] = 2
	}
	yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	if err := w.BuildRoads([]Point{{18, 9}}); err != nil {
		t.Fatal(err)
	}
	addGoodsBuilding(t, &w, "shipwright", 5, 3)
	riverRoad(t, &w, 4, 8)
	addGoodsBuilding(t, &w, "transportwharf", 5, 14)
	riverRoad(t, &w, 15, 7)
	addGoodsBuilding(t, &w, "recruiter", 11, 12)
	fort := addGoodsBuilding(t, &w, "archerfort", 17, 24)
	w.ProduceResource(yard, "timber", 800)
	w.refresh()
	runValid(t, &w, 450)
	if len(w.Boats) != 1 || w.Companies[0].Stationed != 16 {
		t.Fatalf("transport setup incomplete %+v %+v", w.Boats, w.Companies)
	}
	ship := w.Boats[0].ID
	before := w.Hash()
	if w.TransportCompany(ship, fort, Point{3, 20}) == nil || w.Hash() != before {
		t.Fatal("water landing mutated state")
	}
	if err := w.TransportCompany(ship, fort, Point{1, 20}); err != nil {
		t.Fatal(err)
	}
	if w.Deployments[0].Status != "boarding" {
		t.Fatal("company skipped boarding journey")
	}
	if w.Build("wall", 1, 20) == nil {
		t.Fatal("landing could be blocked after departure")
	}
	store, err := Open(filepath.Join(t.TempDir(), "crossing.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	copy, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	embarked := false
	for range 120 {
		runValid(t, &w, 1)
		runValid(t, &copy, 1)
		for _, troop := range w.Troops {
			if troop.Stage == "aboard" {
				embarked = true
			}
		}
	}
	if !embarked || w.Hash() != copy.Hash() || w.Deployments[0].Status != "landed" || w.Military.Lost != 0 {
		t.Fatalf("crossing failed %+v", w.Deployments)
	}
	for _, troop := range w.Troops {
		if troop.FortID == fort && (troop.X != 1 || troop.Y != 20 || troop.Stage != "field") {
			t.Fatal("soldier did not disembark on western shore")
		}
	}
	before = w.Hash()
	if w.MilitaryOrder(fort, "return", Point{}) == nil || w.Hash() != before {
		t.Fatal("company walked home through the river")
	}
	if err = w.TransportCompany(ship, fort, Point{6, 24}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 100)
	if err = w.MilitaryOrder(fort, "return", Point{}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 70)
	if w.Companies[0].Stationed != 16 {
		t.Fatal("transported company did not return to its fort")
	}
}

func TestTransportEvadesAndRepairConsumesPhysicalTimber(t *testing.T) {
	w := riverFixture(t)
	yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	w.BuildRoads([]Point{{18, 9}})
	shipwright := addGoodsBuilding(t, &w, "shipwright", 5, 3)
	riverRoad(t, &w, 4, 8)
	addGoodsBuilding(t, &w, "transportwharf", 5, 14)
	riverRoad(t, &w, 15, 7)
	w.ProduceResource(yard, "timber", 800)
	w.refresh()
	runValid(t, &w, 230)
	ship := w.Boats[0].ID
	// Exercise the repair transaction without inventing a production source:
	// the damaged hull is a focused combat-fixture condition; all repair timber
	// still came through the physical storage-yard shipment.
	a := w.boat(ship)
	if a == nil {
		t.Fatal("transport sank before repair test")
	}
	a.Hull = 25
	if w.ResourceAmount(shipwright, "timber") < 100 {
		t.Fatal("physical repair timber absent")
	}
	used := w.GoodsConsumed["timber"]
	if err := w.NavalOrder(ship, "repair", Point{}); err != nil {
		t.Fatal(err)
	}
	for range 40 {
		runValid(t, &w, 1)
		a = w.boat(ship)
		if a != nil && a.Hull == 40 {
			break
		}
	}
	if a == nil || a.Hull != 40 || w.GoodsConsumed["timber"] != used+100 {
		t.Fatalf("ship did not sail to physical shipwright timber for repair: ship%+v used%d previous%d ships%+v", a, w.GoodsConsumed["timber"], used, w.Boats)
	}
	if err := w.NavalOrder(ship, "hold", Point{3, 12}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 30)
	if err := w.SpawnSeaInvasion(231, "foreign", 8, w.seaEntry()); err != nil {
		t.Fatal(err)
	}
	if err := w.NavalOrder(ship, "evade", Point{}); err != nil {
		t.Fatal(err)
	}
	moved := false
	for range 20 {
		runValid(t, &w, 1)
		a := w.boat(ship)
		if a != nil && (a.X != 3 || a.Y != 12) {
			moved = true
		}
	}
	if !moved {
		t.Fatal("evasive order did not move transport away from pursuer")
	}

}

func TestRecordedNavalForcesUseRawPassengerAndEscortCounts(t *testing.T) {
	for _, test := range []struct{ troops, escorts int }{{300, 3}, {0, 5}, {17, 0}} {
		w := riverFixture(t)
		if err := w.SpawnSeaInvasion(301, "foreign", test.troops, w.seaEntry(), test.escorts); err != nil {
			t.Fatal(err)
		}
		if w.passengersAtSea(301) != test.troops || w.hostileWarships(301) != test.escorts || len(w.Boats) != (test.troops+15)/16+test.escorts {
			t.Fatal("recorded sea force was scaled like a land invasion")
		}
		runValid(t, &w, 3)
		if !w.Military.InvasionActive || w.Invasions[0].Status != "active" {
			t.Fatal("escort-only or loaded invasion was dismissed")
		}
		before := w.Hash()
		if w.SpawnSeaInvasion(302, "foreign", 0, w.seaEntry(), 0) == nil || w.Hash() != before {
			t.Fatal("empty invasion mutated state")
		}
	}
}

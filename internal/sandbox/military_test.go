package sandbox

import (
	"path/filepath"
	"testing"
)

func armyFixture(t *testing.T) (World, int64) {
	t.Helper()
	w := goodsFixture(t)
	addGoodsBuilding(t, &w, "coppermine", 28, 4)
	addGoodsBuilding(t, &w, "weaponsmith", 14, 8)
	addGoodsBuilding(t, &w, "recruiter", 18, 14)
	fort := addGoodsBuilding(t, &w, "fort", 17, 24)
	for _, path := range [][]Point{
		{{25, 10}, {26, 10}, {27, 10}, {28, 10}, {28, 9}, {28, 8}, {28, 7}, {28, 6}},
		{{25, 13}, {24, 13}, {23, 13}, {22, 13}, {21, 13}, {20, 13}, {19, 13}, {18, 13}},
	} {
		if err := w.BuildRoads(path); err != nil {
			t.Fatal(err)
		}
	}
	return w, fort
}
func TestMilitaryPhysicalCopperWeaponsRecruitmentAndBattle(t *testing.T) {
	w, fort := armyFixture(t)
	copper, weapons, recruit := false, false, false
	var saved *World
	store, err := Open(filepath.Join(t.TempDir(), "army.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for range 1400 {
		runValid(t, &w, 1)
		for _, a := range w.Walkers {
			if a.Cargo > 0 && a.Resource == "copper" {
				copper = true
			}
			if a.Cargo > 0 && a.Resource == "weapons" {
				weapons = true
			}
		}
		for _, a := range w.Troops {
			if a.Stage == "recruit" {
				recruit = true
				if saved == nil {
					if err = store.Save(w); err != nil {
						t.Fatal(err)
					}
					loaded, e := store.Load()
					if e != nil {
						t.Fatal(e)
					}
					saved = &loaded
				}
			}
		}
		if saved != nil && saved.Tick < w.Tick {
			runValid(t, saved, 1)
		}
	}
	if !copper || !weapons || !recruit || w.Military.Recruited != 16 || w.Companies[0].Stationed != 16 || w.GoodsConsumed["weapons"] != 1600 {
		t.Fatalf("chain not completed copper%v weapons%v recruit%v stats%+v companies%+v", copper, weapons, recruit, w.Military, w.Companies)
	}
	if saved == nil || saved.Hash() != w.Hash() {
		t.Fatal("recruits in flight did not replay")
	}
	before := w.Hash()
	if w.MilitaryOrder(fort, "muster", Point{1, 1}) == nil || w.Hash() != before {
		t.Fatal("invalid muster changed state")
	}
	if err := w.MilitaryOrder(fort, "muster", Point{26, 18}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 50)
	if err := w.TestInvasion(); err != nil {
		t.Fatal(err)
	}
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	battle, e := store.Load()
	if e != nil {
		t.Fatal(e)
	}
	runValid(t, &w, 100)
	runValid(t, &battle, 100)
	if w.Hash() != battle.Hash() || w.Military.Defeated != 4 || w.Military.Breaches != 0 || w.Military.InvasionActive {
		t.Fatalf("defense/replay failed: %+v", w.Military)
	}
	if err := w.MilitaryOrder(fort, "return", Point{}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 70)
	if w.Companies[0].Stationed != w.Companies[0].Soldiers {
		t.Fatal("company did not return to its fort")
	}
	if w.Build("clear", 17, 24) == nil {
		t.Fatal("occupied fort could be demolished")
	}
}
func TestUnopposedInvasionHasActualConsequences(t *testing.T) {
	w := goodsFixture(t)
	pop := w.population()
	if err := w.TestInvasion(); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	if w.TestInvasion() == nil || w.Hash() != before {
		t.Fatal("overlapping invasion mutated state")
	}
	runValid(t, &w, 140)
	if w.Military.Breaches != 4 || w.Departures != 4 || w.population() > pop || w.Military.InvasionActive {
		t.Fatalf("unopposed invaders had no consequence: %+v pop%d", w.Military, w.population())
	}
	w.Military.Defeated++
	if w.Validate() == nil {
		t.Fatal("corrupt military ledger accepted")
	}
}
func TestRecruiterRequiresPhysicalWeaponsAndPreservesTerrainRoutes(t *testing.T) {
	w, fort := armyFixture(t)
	for i := range w.Buildings {
		if w.Buildings[i].Kind == "coppermine" {
			w.Buildings[i].Damaged = "collapse"
		}
	}
	w.refresh()
	runValid(t, &w, 100)
	if w.Military.Recruited != 0 {
		t.Fatal("unarmed recruits spawned")
	}
	if err := w.Build("recruiter", 12, 25); err == nil {
		t.Fatal("duplicate recruiter accepted")
	}
	if err := w.Build("coppermine", 12, 12); err == nil {
		t.Fatal("copper mined without ore terrain")
	}
	if err := w.MilitaryOrder(fort, "muster", Point{18, 14}); err == nil {
		t.Fatal("troops could muster inside a building")
	}
}

func TestArchersTrainPhysicallyAndChariotsRequireTimberManufacturing(t *testing.T) {
	w := riverFixture(t)
	for x := 11; x <= 20; x++ {
		id := addGoodsBuilding(t, &w, "hut", x, 18)
		w.building(id).Pop = 5
		w.Arrivals += 5
	}
	w.refresh()
	recruiter := addGoodsBuilding(t, &w, "recruiter", 18, 14)
	w.BuildRoads([]Point{{18, 13}, {19, 13}, {20, 13}, {21, 13}, {22, 13}, {23, 13}, {24, 13}, {25, 13}})
	academy := addGoodsBuilding(t, &w, "academy", 11, 12)
	archers := addGoodsBuilding(t, &w, "archerfort", 17, 24)
	chariots := addGoodsBuilding(t, &w, "chariotfort", 17, 2)
	maker := addGoodsBuilding(t, &w, "chariotmaker", 14, 8)
	yard := addGoodsBuilding(t, &w, "storageyard", 26, 10)
	runValid(t, &w, 220)
	if w.Military.ArchersRecruited == 0 || w.Military.ChariotsRecruited != 0 || w.GoodsConsumed["weapons"] != 0 {
		t.Fatal("self-equipped archers or chariot input dependency failed")
	}
	if err := w.OpenTrade(tradeRouteID); err != nil {
		t.Fatal(err)
	}
	if err := w.SetTradePolicy("timber", "import", 1600); err != nil {
		t.Fatal(err)
	}
	sawDelivery, sawTraining, sawChariot := false, false, false
	for range 1500 {
		runValid(t, &w, 1)
		for _, a := range w.Walkers {
			if a.Target == maker && a.Resource == "timber" && a.Cargo > 0 {
				sawDelivery = true
			}
			if a.Target == recruiter && a.Resource == "chariots" && a.Cargo > 0 {
				sawChariot = true
			}
		}
		for _, a := range w.Troops {
			if a.School == academy && a.Stage == "training" {
				sawTraining = true
			}
		}
		if w.Military.ChariotsRecruited >= 4 {
			break
		}
	}
	if !sawDelivery || !sawTraining || !sawChariot || w.Military.ChariotsRecruited < 4 || w.GoodsConsumed["chariots"] != w.Military.ChariotsRecruited*100 {
		t.Fatalf("incomplete chariot chain timber%v training%v chariot%v stats%+v yard%d", sawDelivery, sawTraining, sawChariot, w.Military, w.ResourceAmount(yard, "timber"))
	}
	runValid(t, &w, 120)
	if err := w.MilitaryOrder(archers, "muster", Point{25, 22}); err != nil {
		t.Fatal(err)
	}
	if err := w.MilitaryOrder(chariots, "muster", Point{25, 17}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 45)
	if err := w.TestInvasion(); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(t.TempDir(), "combined-arms.nile"))
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
	runValid(t, &w, 100)
	runValid(t, &copy, 100)
	if w.Hash() != copy.Hash() || w.Military.Defeated != 4 || w.Military.Breaches != 0 {
		t.Fatalf("combined arms replay or defense failed %+v", w.Military)
	}
}

func wallFixture(t *testing.T, tower bool) World {
	w := goodsFixture(t)
	addGoodsBuilding(t, &w, "gatehouse", 10, 15)
	addGoodsBuilding(t, &w, "gatehouse", 25, 15)
	for y := 15; y <= 16; y++ {
		for x := 6; x < 32; x++ {
			if x == 10 || x == 11 || x == 25 || x == 26 {
				continue
			}
			addGoodsBuilding(t, &w, "wall", x, y)
		}
	}
	if tower {
		addGoodsBuilding(t, &w, "tower", 27, 15)
		w.BuildRoads([]Point{{25, 14}, {26, 14}, {27, 14}, {28, 14}})
		addGoodsBuilding(t, &w, "recruiter", 11, 12)
		runValid(t, &w, 130)
		if w.Military.SentriesRecruited != 2 {
			t.Fatal("tower did not receive physical sentries")
		}
	}
	return w
}
func TestWallsDelayRecordedInvadersAndStaffedTowersRepelThem(t *testing.T) {
	for _, defended := range []bool{false, true} {
		w := wallFixture(t, defended)
		if err := w.SpawnInvasion(91, "foreign", 4, Point{31, 13}); err != nil {
			t.Fatal(err)
		}
		before := w.Hash()
		if w.SpawnInvasion(91, "foreign", 4, Point{31, 13}) == nil || w.Hash() != before {
			t.Fatal("duplicate campaign invasion mutated state")
		}
		runValid(t, &w, 45)
		if w.Military.Breaches != 0 {
			t.Fatal("invaders walked through intact walls")
		}
		runValid(t, &w, 220)
		if defended {
			if w.Military.Defeated != 4 || w.Military.Breaches != 0 || w.Invasions[0].Status != "repelled" {
				t.Fatalf("tower defense failed %+v", w.Military)
			}
		} else {
			if w.Military.Breaches != 4 || w.Invasions[0].Status != "breached" {
				t.Fatalf("invaders failed to breach undefended walls %+v", w.Military)
			}
		}
	}
}

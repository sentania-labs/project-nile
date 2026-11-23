package sandbox

import (
	"path/filepath"
	"testing"
)

func TestSelectedArmyMarchesOffMapWaitsForCampaignAndReturnsSurvivors(t *testing.T) {
	w := riverFixture(t)
	addGoodsBuilding(t, &w, "recruiter", 11, 12)
	first := addGoodsBuilding(t, &w, "archerfort", 17, 24)
	second := addGoodsBuilding(t, &w, "archerfort", 17, 2)
	runValid(t, &w, 700)
	if w.company(first).Stationed != 16 || w.company(second).Stationed != 16 {
		t.Fatal("both companies did not recruit")
	}
	before := w.Hash()
	if w.StartArmyDeployment(52, 7, []int64{first, first}, false) == nil || w.Hash() != before {
		t.Fatal("duplicate selection partly dispatched company")
	}
	if err := w.StartArmyDeployment(52, 7, []int64{first, second}, false); err != nil {
		t.Fatal(err)
	}
	if w.ArmyDeploymentStatus(52) != "outbound" {
		t.Fatal("army skipped march to Kingdom Road")
	}
	if w.MilitaryOrder(first, "return", Point{}) == nil {
		t.Fatal("active army assignment overwritten")
	}
	runValid(t, &w, 70)
	if w.ArmyDeploymentStatus(52) != "awaits_outcome" {
		t.Fatal("army never left city")
	}
	defeat := w.armyCopy()
	if err := defeat.ResolveArmyDeployment(52, map[int64]int{first: 0, second: 0}, true); err != nil {
		t.Fatal(err)
	}
	if defeat.ArmyDeploymentStatus(52) != "defeated" || defeat.deployedFort(first) || defeat.deployedFort(second) || defeat.Military.Lost != 32 {
		t.Fatal("total defeat left dead companies waiting for a return journey")
	}
	if err := defeat.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, a := range w.View().Walkers {
		if a.Role == "archer" {
			t.Fatal("distant soldiers remained rendered in city")
		}
	}
	store, err := Open(filepath.Join(t.TempDir(), "distant-army.nile"))
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
	runValid(t, &w, 32)
	runValid(t, &copy, 32)
	if w.Hash() != copy.Hash() || w.ArmyDeploymentStatus(52) != "awaits_outcome" {
		t.Fatal("army invented an outcome or replay changed")
	}
	before = w.Hash()
	if w.ResolveArmyDeployment(52, map[int64]int{first: 17, second: 14}, false) == nil || w.Hash() != before {
		t.Fatal("invalid casualty count mutated state")
	}
	if err = w.ResolveArmyDeployment(52, map[int64]int{first: 12, second: 14}, false); err != nil {
		t.Fatal(err)
	}
	if w.Military.Lost != 6 || w.ArmyDeploymentStatus(52) != "awaits_return" {
		t.Fatal("campaign casualty accounting failed")
	}
	runValid(t, &w, 40)
	if w.ArmyDeploymentStatus(52) != "awaits_return" {
		t.Fatal("survivors returned before campaign travel deadline")
	}
	if err = w.LaunchArmyReturn(52); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 90)
	if w.ArmyDeploymentStatus(52) != "returned" || w.company(first).Stationed < 12 || w.company(second).Stationed < 14 {
		t.Fatal("survivors did not physically march home")
	}
}

func TestSelectedFleetCarriesTwoCompaniesAndHoldsSurvivorsUntilReturn(t *testing.T) {
	w := riverFixture(t)
	addGoodsBuilding(t, &w, "recruiter", 11, 12)
	first := addGoodsBuilding(t, &w, "archerfort", 17, 24)
	second := addGoodsBuilding(t, &w, "archerfort", 17, 2)
	yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	w.BuildRoads([]Point{{18, 9}})
	addGoodsBuilding(t, &w, "shipwright", 5, 3)
	riverRoad(t, &w, 4, 8)
	addGoodsBuilding(t, &w, "transportwharf", 5, 14)
	riverRoad(t, &w, 15, 7)
	addGoodsBuilding(t, &w, "transportwharf", 5, 24)
	riverRoad(t, &w, 25, 7)
	w.ProduceResource(yard, "timber", 800)
	w.refresh()
	runValid(t, &w, 700)
	if len(w.Boats) != 2 || w.Companies[0].Stationed != 16 || w.Companies[1].Stationed != 16 {
		t.Fatal("fleet did not equip two companies")
	}
	if err := w.StartArmyDeployment(61, 3, []int64{first, second}, true); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 110)
	if w.ArmyDeploymentStatus(61) != "awaits_outcome" {
		t.Fatal("two troop ships never reached distant route")
	}
	for _, a := range w.Troops {
		if a.Stage != "aboard" {
			t.Fatal("fleet company was not physically aboard")
		}
	}
	if err := w.ResolveArmyDeployment(61, map[int64]int{first: 0, second: 12}, false); err != nil {
		t.Fatal(err)
	}
	if w.ArmyDeploymentStatus(61) != "awaits_return" || w.deployedFort(first) || w.Military.Lost != 20 {
		t.Fatal("destroyed company was not released or counted")
	}
	runValid(t, &w, 40)
	if w.ArmyDeploymentStatus(61) != "awaits_return" {
		t.Fatal("surviving ship ignored campaign delay")
	}
	if err := w.LaunchArmyReturn(61); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 110)
	if w.ArmyDeploymentStatus(61) != "returned" || w.company(second).Stationed < 12 {
		t.Fatal("surviving company did not physically return")
	}
}

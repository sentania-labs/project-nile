package sandbox

import "testing"

func TestZoneLeavesActiveOverlandRouteOpen(t *testing.T) {
	w := NewTown()
	w.Walkers = append(w.Walkers, Walker{Role: "hunter", Route: []Point{{21, 20}, {22, 20}}})
	before := w.Treasury
	if err := w.ZoneHousing(Point{21, 20}, Point{23, 20}); err != nil {
		t.Fatal(err)
	}
	if len(w.Buildings) != 1 || w.Buildings[0].point() != (Point{23, 20}) || before-w.Treasury != constructionCost("hut") {
		t.Fatal("zoning obstructed a worker or charged for a reserved route")
	}
}
func TestIdleCropWorkExpiresAtInundation(t *testing.T) {
	for _, kind := range []string{"barleyfarm", "flaxfarm"} {
		w := NewTown()
		if err := w.Build(kind, 6, 10); err != nil {
			t.Fatal(err)
		}
		w.Buildings[0].Work = 250
		w.Tick = YearTicks - SeasonOffset
		w.produceGoods()
		if w.Buildings[0].Work != 0 {
			t.Errorf("idle %s retained pre-flood work", kind)
		}
	}
}

func TestHouseholdReligionRequiresAPriestVisit(t *testing.T) {
	w := NewTown()
	home := Building{Kind: "hut", Pop: 4, Water: true, Food: true, Market: true, Desirability: 6, ShrineAccess: true}
	w.refreshHome(&home, 2)
	if home.Tier != 2 {
		t.Fatalf("legacy shrine proximity promoted home to %d", home.Tier)
	}
	home.GodUntil[0] = w.Tick + MarketDuration
	w.refreshHome(&home, 2)
	if home.Tier != 3 {
		t.Fatalf("priest's visit did not advance household: %+v", home)
	}
}

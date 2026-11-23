package sandbox

import (
	"path/filepath"
	"testing"
)

func modernFarmFixture(t *testing.T) World {
	w := goodsFixture(t)
	w.AgricultureVersion = 1
	w.refresh()
	return w
}
func TestNormalizedGrainUsesCampCrewsHundredUnitCartsAndReplay(t *testing.T) {
	w := modernFarmFixture(t)
	farm := addGoodsBuilding(t, &w, "farm", 7, 6)
	addGoodsBuilding(t, &w, "granary", 14, 6)
	addGoodsBuilding(t, &w, "bazaar", 14, 11)
	runValid(t, &w, 240)
	if w.GrainProduced != 0 || w.building(farm).WorkersNeeded != 0 {
		t.Fatal("floodplain produced without a physical camp crew")
	}
	addGoodsBuilding(t, &w, "workcamp", 11, 12)
	store, err := Open(filepath.Join(t.TempDir(), "grain.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	crew, load, food := false, false, false
	var restored *World
	for range 500 {
		runValid(t, &w, 1)
		for _, a := range w.Walkers {
			if a.Role == "farmworker" && a.Source == farm && a.Stage == "work" {
				crew = true
			}
			if a.Resource == "grain" && a.Role == "freight" && a.Cargo == 100 {
				load = true
				if restored == nil {
					if err = store.Save(w); err != nil {
						t.Fatal(err)
					}
					r, e := store.Load()
					if e != nil {
						t.Fatal(e)
					}
					restored = &r
				}
			}
		}
		for _, b := range w.Buildings {
			if b.Kind == "hut" && b.Stock > 0 {
				food = true
			}
		}
		if restored != nil && restored.Tick < w.Tick {
			runValid(t, restored, 1)
		}
	}
	if !crew || !load || !food || w.GrainConsumed == 0 || w.GrainProduced < 400 || restored == nil || restored.Hash() != w.Hash() {
		t.Fatalf("grain chain crew%v cart%v food%v produced%d consumed%d", crew, load, food, w.GrainProduced, w.GrainConsumed)
	}
	if w.building(farm).StockCapacity != 400 {
		t.Fatal("grain buffer still uses legacy scale")
	}
}
func TestMeadowWaterLiftAndCanalsChangeRealYield(t *testing.T) {
	w := modernFarmFixture(t)
	farm := addGoodsBuilding(t, &w, "farm", 11, 3)
	if w.building(farm).FarmType != "meadow" || w.building(farm).WorkersNeeded != 10 {
		t.Fatal("meadow did not hire local staff")
	}
	dryFertility := w.building(farm).Fertility
	if err := w.BuildCanals([]Point{{6, 5}, {7, 5}, {10, 6}, {11, 6}, {12, 6}}); err != nil {
		t.Fatal(err)
	}
	if w.building(farm).Irrigated {
		t.Fatal("floodplain water rose onto meadow without a lift")
	}
	lift := addGoodsBuilding(t, &w, "waterlift", 8, 5)
	if !w.building(farm).Irrigated || w.building(farm).Fertility <= dryFertility {
		t.Fatal("lift and road-crossing canal did not irrigate meadow")
	}
	before := w.Hash()
	if w.BuildCanals([]Point{{6, 6}, {11, 3}}) == nil || w.Hash() != before {
		t.Fatal("invalid canal partly built or overwrote a farm")
	}
	dry := w
	dry.Buildings = append([]Building{}, w.Buildings...)
	dry.building(lift).Damaged = "collapse"
	dry.refresh()
	if dry.building(farm).Irrigated {
		t.Fatal("damaged lift still supplied irrigation")
	}
	runValid(t, &w, 240)
	runValid(t, &dry, 240)
	if w.GrainProduced <= dry.GrainProduced || dry.GrainProduced == 0 {
		t.Fatalf("irrigation has no yield effect irrigated%d dry%d", w.GrainProduced, dry.GrainProduced)
	}
	if w.GrainProduced < 200 {
		t.Fatal("meadow did not harvest twice")
	}
	before = w.Hash()
	if w.Build("farm", 16, 3) == nil || w.Hash() != before {
		t.Fatal("farm cultivated desert")
	}
	if err := w.Build("clear", 10, 6); err != nil {
		t.Fatal(err)
	}
	if !w.roadSet()[Point{10, 6}] || w.building(farm).Irrigated {
		t.Fatal("clearing ditch removed road or left water connected")
	}
}
func TestLegacyAgricultureScaleRemainsReadable(t *testing.T) {
	w := goodsFixture(t)
	farm := addGoodsBuilding(t, &w, "farm", 7, 6)
	addGoodsBuilding(t, &w, "granary", 14, 6)
	runValid(t, &w, 240)
	if w.GrainProduced <= 0 || w.GrainProduced > 48 || w.building(farm).StockCapacity != 48 {
		t.Fatalf("legacy farm quantities changed produced%d cap%d version%d workers%d", w.GrainProduced, w.building(farm).StockCapacity, w.AgricultureVersion, w.building(farm).Workers)
	}
	store, err := Open(filepath.Join(t.TempDir(), "legacy-farm.nile"))
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
	if restored.AgricultureVersion != 0 || restored.Hash() != w.Hash() {
		t.Fatal("legacy agriculture silently changed")
	}
}

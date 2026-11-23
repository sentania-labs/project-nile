package sandbox

import (
	"path/filepath"
	"testing"
)

func TestNubtCompletesThroughPhysicalHuntingAndWaterWalkers(t *testing.T) {
	w, err := NewCampaign(0)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.Build("farm", 6, 12); err == nil {
		t.Fatal("Nubt cannot substitute grain farming for its hunting tutorial")
	}
	for y := 1; y <= 15; y++ {
		for _, x := range []int{9, 11} {
			if err = w.Build("hut", x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	built := map[string]bool{}
	savedDuringHunt := false
	for w.Tick < 8000 && !w.Campaign.Won {
		for _, choice := range []struct {
			kind string
			x, y int
		}{{"firehouse", 9, 17}, {"architect", 9, 19}, {"granary", 11, 21}, {"huntinglodge", 11, 26}, {"bazaar", 11, 18}, {"watersupply", 11, 29}} {
			if w.Campaign.Allows(choice.kind) && !built[choice.kind] {
				if err = w.Build(choice.kind, choice.x, choice.y); err != nil {
					t.Fatalf("tick %d build %s: %v", w.Tick, choice.kind, err)
				}
				built[choice.kind] = true
			}
		}
		for _, b := range w.Buildings {
			if b.Damaged != "" {
				if err = w.Repair(b.X, b.Y); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err = w.Step(1); err != nil {
			t.Fatal(err)
		}
		if w.Tick%32 == 0 {
			if err = w.Validate(); err != nil {
				t.Fatalf("tick %d: %v", w.Tick, err)
			}
		}
		if !savedDuringHunt {
			for _, a := range w.Walkers {
				if a.Role == "hunter" && a.Cargo > 0 {
					store, openErr := Open(filepath.Join(t.TempDir(), "nubt.nile"))
					if openErr != nil {
						t.Fatal(openErr)
					}
					if err = store.Save(w); err != nil {
						t.Fatal(err)
					}
					restored, loadErr := store.Load()
					if loadErr != nil {
						t.Fatal(loadErr)
					}
					if restored.Hash() != w.Hash() {
						t.Fatal("hunter cargo or tutorial progress changed on reload")
					}
					if err = store.Close(); err != nil {
						t.Fatal(err)
					}
					w = restored
					savedDuringHunt = true
					break
				}
			}
		}
	}
	if err = w.Validate(); err != nil {
		t.Fatal(err)
	}
	if !w.Campaign.Won || !savedDuringHunt || w.GrainProduced != 0 || w.GoodsProduced["meat"] == 0 {
		t.Fatalf("Nubt incomplete: tick=%d population=%d treasury=%d campaign=%+v meat=%v", w.Tick, w.population(), w.Treasury, w.Campaign, w.GoodsProduced)
	}
	before := w.Hash()
	if next, nextErr := w.AdvanceCampaign(1); nextErr != nil || next.Campaign.MissionID != 1 || len(next.Campaign.Completed) != 1 || w.Hash() != before {
		t.Fatal("mission transition lost completion history or mutated the previous village")
	}
	store, err := Open(filepath.Join(t.TempDir(), "victory.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Load()
	if err != nil || !restored.Campaign.Won {
		t.Fatalf("victory reload failed: %v", err)
	}
	if err = w.Step(20); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if err = restored.Step(1); err != nil {
			t.Fatal(err)
		}
	}
	if w.Hash() != restored.Hash() {
		t.Fatal("campaign replay changed with tick batching after reload")
	}
	t.Logf("Nubt won at tick %d with %d residents, %d meat produced and %d treasury; hunter cargo survived SQLite reload", w.Tick, w.population(), w.GoodsProduced["meat"], w.Treasury)
}

func TestThinisCompletesWithDeliveredGoldBastAndEntertainment(t *testing.T) {
	w, err := NewCampaign(1)
	if err != nil {
		t.Fatal(err)
	}
	build := func(kind string, x, y int) {
		t.Helper()
		if err := w.Build(kind, x, y); err != nil {
			t.Fatalf("tick%d %s at%d,%d: %v", w.Tick, kind, x, y, err)
		}
	}
	for x := 11; x <= 28; x++ {
		build("road", x, 10)
	}
	for y := 6; y < 10; y++ {
		build("road", 28, y)
	}
	for y := 11; y <= 20; y++ {
		build("road", 22, y)
	}
	for x := 11; x < 22; x++ {
		build("road", x, 18)
		build("road", x, 20)
	}
	for y := 1; y <= 5; y++ {
		build("hut", 9, y)
		build("hut", 11, y)
	}
	for x := 12; x <= 18; x++ {
		build("hut", x, 11)
	}
	for x := 11; x <= 21; x++ {
		build("hut", x, 19)
	}
	for x := 13; x <= 21; x++ {
		build("hut", x, 21)
	}
	for _, b := range []struct {
		kind string
		x, y int
	}{
		{"palace", 11, 24}, {"goldmine", 28, 4}, {"granary", 14, 6},
		{"bazaar", 19, 8}, {"bazaar", 11, 6}, {"huntinglodge", 23, 18}, {"watersupply", 11, 22},
		{"firehouse", 12, 9}, {"architect", 21, 9},
	} {
		build(b.kind, b.x, b.y)
	}
	templesBuilt, entertainmentBuilt := false, false
	for w.Tick < 7000 && !w.Campaign.Won {
		if w.Campaign.ReligionOpened && !templesBuilt {
			build("temple_bast", 15, 15)
			templesBuilt = true
		}
		if w.Campaign.EntertainmentOpened && !entertainmentBuilt {
			build("jugglerschool", 20, 11)
			build("booth", 10, 10)
			entertainmentBuilt = true
		}
		for _, b := range w.Buildings {
			if b.Damaged != "" {
				if err := w.Repair(b.X, b.Y); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		if w.Tick%32 == 0 {
			if err := w.Validate(); err != nil {
				t.Fatalf("tick%d: %v", w.Tick, err)
			}
		}
	}
	if !w.Campaign.Won {
		for _, b := range w.Buildings {
			if b.Kind == "hut" {
				t.Logf("home%d,%d pop%d tier%d desire%d water%v food%v god%v entertain%v", b.X, b.Y, b.Pop, b.Tier, b.Desirability, b.Water, b.Food, b.GodUntil, b.EntertainmentUntil)
			}
		}
		t.Fatalf("Thinis incomplete at%d: pop%d gold%d favor%v campaign%+v", w.Tick, w.population(), w.GoldDelivered, w.Favor, w.Campaign)
	}
	if w.GoldDelivered < 500 || !w.Campaign.ReligionOpened || !w.Campaign.EntertainmentOpened || !entertainmentBuilt {
		t.Fatal("Thinis bypassed its physical tutorial progression")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Thinis won at%d with%d residents, gold delivered%d, favor%v, treasury%d", w.Tick, w.population(), w.GoldDelivered, w.Favor, w.Treasury)
}

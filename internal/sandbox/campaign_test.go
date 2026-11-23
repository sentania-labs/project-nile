package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	captureCampaign(t, w)
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
			build("booth", 22, 20)
			build("temple_bast", 15, 22)
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
	captureCampaign(t, w)
}

func TestPerwadjytCompletesWithSeasonalFigsAndPottery(t *testing.T) {
	w, err := NewCampaign(2)
	if err != nil {
		t.Fatal(err)
	}
	build := func(kind string, x, y int) {
		t.Helper()
		if kind == "road" && slices.Contains(w.Roads, Point{x, y}) {
			return
		}
		if err := w.Build(kind, x, y); err != nil {
			t.Fatalf("tick%d %s at%d,%d: %v", w.Tick, kind, x, y, err)
		}
	}
	for _, y := range []int{6, 14, 22, 30} {
		for x := 11; x <= 28; x++ {
			build("road", x, y)
		}
	}
	for y := 6; y <= 30; y++ {
		build("road", 28, y)
	}
	build("road", 23, 15)
	reserved := []Building{{X: 8, Y: 29, Width: 2, Height: 2}, {X: 23, Y: 16, Width: 2, Height: 2}, {X: 23, Y: 19, Width: 3, Height: 3}, {X: 26, Y: 16, Width: 2, Height: 2}}
	free := func(b Building) bool {
		occupied := w.occupied()
		for _, p := range b.tiles() {
			if !valid(p) || occupied[p] || slices.Contains(w.HuntingGrounds, p) {
				return false
			}
			for _, r := range reserved {
				if r.contains(p) {
					return false
				}
			}
		}
		return true
	}
	// Reserve workplaces first. Homes below use the remaining roadside plots.
	for _, b := range []struct {
		kind string
		x, y int
	}{
		{"workcamp", 11, 8}, {"workcamp", 11, 16},
		{"granary", 15, 2}, {"granary", 19, 15},
		{"bazaar", 11, 0}, {"bazaar", 19, 4}, {"bazaar", 26, 12}, {"bazaar", 26, 28},
		{"watersupply", 11, 3}, {"watersupply", 11, 19}, {"watersupply", 13, 28},
		{"temple_osiris", 19, 7}, {"temple_osiris", 19, 23},
		{"palace", 11, 24},
		{"firehouse", 26, 7}, {"architect", 26, 15},
		{"jugglerschool", 26, 19}, {"booth", 10, 6}, {"booth", 27, 21}, {"booth", 10, 22},
	} {
		build(b.kind, b.x, b.y)
	}
	for _, y := range []int{0, 4, 8, 12} {
		build("figfarm", 7, y)
	}
	for y := 0; y < Size; y++ {
		for x := 9; x < Size; x++ {
			b := Building{Kind: "hut", X: x, Y: y, Width: 1, Height: 1}
			adjRoad := false
			for _, p := range w.Roads {
				if b.adjacent(p) {
					adjRoad = true
					break
				}
			}
			if adjRoad && free(b) {
				build("hut", x, y)
			}
		}
	}
	industry, decorated, saved, health := false, false, false, false
	for w.Tick < 6000 && !w.Campaign.Won {
		if w.Campaign.HealthOpened && !health && w.Treasury >= constructionCost("physician") {
			build("physician", 26, 16)
			health = true
		}
		if w.Campaign.IndustryOpened && !industry {
			build("claypit", 8, 29)
			build("potter", 23, 16)
			build("storageyard", 23, 19)
			industry = true
		}
		if w.Campaign.BeautificationOpened && !decorated {
			for _, p := range []Point{{17, 24}, {17, 25}, {18, 24}, {18, 25}, {16, 24}, {16, 25}, {16, 8}, {16, 9}, {17, 8}, {17, 9}, {18, 8}, {18, 9}} {
				if free(Building{X: p.X, Y: p.Y, Width: 1, Height: 1}) {
					build("garden", p.X, p.Y)
				}
			}
			for _, p := range []Point{{10, 10}, {28, 10}, {10, 18}, {28, 18}} {
				build("roadblock", p.X, p.Y)
				for _, role := range []string{"firefighter", "architect", "doctor"} {
					if err := w.RoadPermission(p, role, true); err != nil {
						t.Fatal(err)
					}
				}
			}
			for x := 11; x <= 21; x++ {
				build("plaza", x, 22)
			}
			decorated = true
		}
		for _, b := range w.Buildings {
			if b.Damaged != "" && w.Treasury >= b.RepairCost {
				if err := w.Repair(b.X, b.Y); err != nil {
					t.Fatalf("tick%d pop%d funds%d repairing %s: %v", w.Tick, w.population(), w.Treasury, b.Kind, err)
				}
			}
		}
		if err := w.Step(1); err != nil {
			t.Fatal(err)
		}
		if w.Tick%64 == 0 {
			if err := w.Validate(); err != nil {
				t.Fatalf("tick%d: %v", w.Tick, err)
			}
		}
		if !saved && w.GoodsProduced["figs"] > 0 && slices.ContainsFunc(w.Walkers, func(a Walker) bool { return a.Role == "farmworker" && a.Stage == "work" }) {
			store, err := Open(filepath.Join(t.TempDir(), "perwadjyt.nile"))
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Save(w); err != nil {
				t.Fatal(err)
			}
			restored, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			if restored.Hash() != w.Hash() {
				t.Fatal("farm crew or campaign state changed on reload")
			}
			_ = store.Close()
			w = restored
			saved = true
		}
	}
	if !w.Campaign.Won {
		for _, b := range w.Buildings {
			if b.Kind != "hut" || b.Desirability >= 12 {
				t.Logf("%s%d,%d pop%d tier%d staffed%d/%d work%d goods%v desire%d water%v food%v market%v worship%v entertain%d", b.Kind, b.X, b.Y, b.Pop, b.Tier, b.Workers, b.WorkersNeeded, b.Work, b.Goods, b.Desirability, b.Water, b.Food, b.Market, b.GodUntil, entertainmentScore(b, w.Tick))
			}
		}
		t.Fatalf("Perwadjyt incomplete tick%d pop%d funds%d obs%+v campaign%+v goods%v", w.Tick, w.population(), w.Treasury, w.campaignObservation(), w.Campaign, w.GoodsProduced)
	}
	if !saved || w.GoodsProduced["figs"] == 0 || w.GoodsProduced["pottery"] == 0 {
		t.Fatal("campaign bypassed physical supply chains")
	}
	t.Logf("Perwadjyt won tick%d population%d treasury%d production%v", w.Tick, w.population(), w.Treasury, w.GoodsProduced)
	captureCampaign(t, w)
}

// Optional reproducible UI fixtures are produced only by winning through the
// same construction and simulation actions used above, never injected stocks.
func captureCampaign(t *testing.T, w World) {
	t.Helper()
	dir := os.Getenv("NILE_CAMPAIGN_CAPTURE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(dir, fmt.Sprintf("mission-%d.nile", w.Campaign.MissionID)))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
}

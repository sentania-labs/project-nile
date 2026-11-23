package sandbox

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestNekhenCompletesWithGrainPotteryAndBeer(t *testing.T) {
	w, err := NewCampaign(3)
	if err != nil {
		t.Fatal(err)
	}
	buildTeachingNeighborhood(t, &w)
	saved := false
	for w.Tick < 6000 && !w.Campaign.Won {
		for _, b := range w.Buildings {
			if b.Damaged != "" && w.Treasury >= b.RepairCost {
				if err = w.Repair(b.X, b.Y); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err = w.Step(1); err != nil {
			t.Fatal(err)
		}
		if w.Tick%128 == 0 {
			if err = w.Validate(); err != nil {
				t.Fatalf("tick%d: %v", w.Tick, err)
			}
		}
		if !saved && w.GoodsProduced["beer"] > 0 {
			s, err := Open(filepath.Join(t.TempDir(), "nekhen.nile"))
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Save(w); err != nil {
				t.Fatal(err)
			}
			restored, err := s.Load()
			_ = s.Close()
			if err != nil {
				t.Fatal(err)
			}
			if restored.Hash() != w.Hash() {
				t.Fatal("Nekhen economy changed on reload")
			}
			w = restored
			saved = true
		}
	}
	if !w.Campaign.Won {
		for _, b := range w.Buildings {
			if b.Kind != "hut" || b.Desirability >= 20 {
				t.Logf("%s %d,%d pop%d tier%d workers%d/%d work%d stock%d goods%v desire%d entertainment%d physician%d", b.Kind, b.X, b.Y, b.Pop, b.Tier, b.Workers, b.WorkersNeeded, b.Work, b.Stock, b.Goods, b.Desirability, entertainmentScore(b, w.Tick), b.ServiceUntil[1]-w.Tick)
			}
		}
		t.Fatalf("Nekhen incomplete tick%d pop%d funds%d observation%+v production%v grain%d", w.Tick, w.population(), w.Treasury, w.campaignObservation(), w.GoodsProduced, w.GrainProduced)
	}
	if !saved || w.GrainProduced == 0 || w.GoodsProduced["beer"] == 0 || w.GoodsProduced["pottery"] == 0 {
		t.Fatal("Nekhen skipped real production")
	}
	t.Logf("Nekhen won tick%d population%d funds%d grain%d goods%v", w.Tick, w.population(), w.Treasury, w.GrainProduced, w.GoodsProduced)
	captureCampaign(t, w)
}

func buildTeachingNeighborhood(t *testing.T, w *World) {
	t.Helper()
	foodFarm, primaryTemple := "farm", "temple_ra"
	if w.Campaign.MissionID == 4 {
		foodFarm, primaryTemple = "chickpeafarm", "temple_ptah"
	}
	build := func(kind string, x, y int) {
		t.Helper()
		if kind == "road" && slices.Contains(w.Roads, Point{x, y}) {
			return
		}
		if err := w.Build(kind, x, y); err != nil {
			t.Fatalf("tick%d build%s at%d,%d funds%d: %v", w.Tick, kind, x, y, w.Treasury, err)
		}
	}
	for _, y := range []int{6, 14, 22, 30, 38} {
		for x := 11; x <= 28; x++ {
			build("road", x, y)
		}
	}
	for y := 6; y <= 38; y++ {
		build("road", 28, y)
	}
	build("road", 23, 15)
	build("road", 23, 23)
	build("road", 15, 37)
	for _, b := range []struct {
		kind string
		x, y int
	}{
		{"workcamp", 11, 8}, {"workcamp", 11, 16}, {"workcamp", 15, 31},
		{"granary", 15, 2}, {"granary", 19, 15},
		{"bazaar", 11, 0}, {"bazaar", 19, 4}, {"bazaar", 26, 12}, {"bazaar", 26, 28}, {"bazaar", 26, 36},
		{"watersupply", 11, 3}, {"watersupply", 11, 19}, {"watersupply", 13, 28}, {"watersupply", 15, 35},
		{primaryTemple, 19, 7}, {"temple_osiris", 19, 23}, {"temple_bast", 19, 35},
		{"palace", 11, 24}, {"taxoffice", 23, 12}, {"taxoffice", 26, 32},
		{"firehouse", 26, 7}, {"architect", 26, 15}, {"firehouse", 26, 24},
		{"jugglerschool", 26, 19}, {"conservatory", 23, 7},
		{"bandstand", 10, 5}, {"bandstand", 26, 21}, {"bandstand", 10, 21},
		{"physician", 24, 4}, {"physician", 26, 16}, {"physician", 23, 28},
		{"claypit", 8, 29}, {"potter", 23, 16}, {"storageyard", 23, 19}, {"brewery", 23, 24},
	} {
		build(b.kind, b.x, b.y)
	}
	if w.Campaign.MissionID == 4 {
		for _, p := range []Point{{22, 4}, {15, 12}, {15, 20}, {15, 28}, {19, 31}} {
			build("taxoffice", p.X, p.Y)
		}
		for _, p := range []Point{{13, 12}, {13, 20}} {
			build("watersupply", p.X, p.Y)
		}
		build("bazaar", 21, 12)
	}
	for _, y := range []int{0, 4, 8, 12} {
		build(foodFarm, 7, y)
	}
	for _, y := range []int{16, 20} {
		build("barleyfarm", 7, y)
	}
	for _, p := range []Point{{17, 24}, {17, 25}, {18, 24}, {18, 25}, {16, 24}, {16, 25}, {16, 8}, {16, 9}, {17, 8}, {17, 9}, {18, 8}, {18, 9}, {22, 23}, {22, 24}, {22, 25}, {21, 26}, {20, 26}, {19, 26}} {
		if !w.occupied()[p] {
			build("garden", p.X, p.Y)
		}
	}
	for _, y := range []int{6, 14, 22, 30, 38} {
		for x := 12; x <= 25; x++ {
			build("plaza", x, y)
		}
	}
	for _, p := range []Point{{10, 10}, {28, 10}, {10, 18}, {28, 18}, {10, 27}, {28, 27}, {10, 35}, {28, 35}} {
		if !slices.Contains(w.Roads, p) {
			continue
		}
		build("roadblock", p.X, p.Y)
		for _, role := range []string{"firefighter", "architect", "doctor", "taxcollector"} {
			if err := w.RoadPermission(p, role, true); err != nil {
				t.Fatal(err)
			}
		}
	}
	for y := 0; y < 41; y++ {
		for x := 9; x < 32; x++ {
			p := Point{x, y}
			if w.Campaign.MissionID == 4 && (y == 14 && x > 28 || x >= 8 && x <= 9 && y >= 23 && y <= 24 || x >= 11 && x <= 12 && y >= 28 && y <= 29) {
				continue
			}
			b := Building{Kind: "hut", X: x, Y: y, Width: 1, Height: 1}
			if !w.buildable(p) || w.occupied()[p] || slices.Contains(w.HuntingGrounds, p) {
				continue
			}
			if slices.ContainsFunc(w.Roads, func(road Point) bool { return b.adjacent(road) }) {
				build("hut", x, y)
			}
		}
	}
}

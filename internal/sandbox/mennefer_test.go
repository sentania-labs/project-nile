package sandbox

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestMenneferWinsWithPapyrusExportsAndImportedMastaba(t *testing.T) {
	w, err := NewCampaign(4)
	if err != nil {
		t.Fatal(err)
	}
	buildTradingNeighborhood(t, &w)
	build := func(kind string, x, y int) {
		t.Helper()
		if kind == "road" && slices.Contains(w.Roads, Point{x, y}) {
			return
		}
		if err := w.Build(kind, x, y); err != nil {
			t.Fatalf("tick %d: build %s at %d,%d, funds %d: %v", w.Tick, kind, x, y, w.Treasury, err)
		}
	}
	education, commerce, saved, monument := false, false, false, false
	satellites := 0
	for w.Tick < 22000 && !w.Campaign.Won {
		for _, b := range w.Buildings {
			if b.Damaged != "" && w.Treasury >= b.RepairCost {
				if err = w.Repair(b.X, b.Y); err != nil {
					t.Fatal(err)
				}
			}
		}
		if w.Tick == 1200 {
			if err = w.SetPolicy(10, 36, "balanced"); err != nil {
				t.Fatal(err)
			}
		}
		for _, request := range w.Campaign.Requests() {
			if (request.Status == "requested" || request.Status == "overdue") && w.requestStock(request.Resource) >= request.Quantity {
				if err = w.DispatchCampaignRequest(request.ID); err != nil {
					t.Fatal(err)
				}
			}
		}
		if w.Campaign.EducationOpened && !education && w.Treasury >= 750 {
			for x := 23; x <= 37; x++ {
				build("road", x, 18)
			}
			for y := 18; y <= 32; y++ {
				build("road", 37, y)
			}
			build("reedgatherer", 8, 23)
			build("reedgatherer", w.bank(39)+6, 39)
			build("reedgatherer", w.bank(42)+6, 42)
			build("papyrusmaker", 35, 19)
			build("papyrusmaker", 35, 23)
			build("scribalschool", 25, 19)
			build("storageyard", 34, 29)
			build("firehouse", 38, 21)
			build("architect", 38, 25)
			education = true
		}
		if w.Campaign.TradeOpened && !commerce && w.Treasury >= 1200 {
			for _, route := range []string{"city_3", "city_5"} {
				if err = w.OpenTrade(route); err != nil {
					t.Fatal(err)
				}
			}
			if err = w.SetTradePolicy("papyrus", "export", 200); err != nil {
				t.Fatal(err)
			}
			if err = w.SetTradePolicy("bricks", "import", 800); err != nil {
				t.Fatal(err)
			}
			commerce = true
		}
		if w.Campaign.MonumentOpened && w.Treasury >= 4500 && satellites < 2 {
			places := []Point{{23, 48}, {28, 64}}
			buildTradingQuarter(t, &w, places[satellites])
			satellites++
		}
		if w.Campaign.MonumentOpened && satellites == 2 && !monument && w.Treasury >= 1500 {
			for y := 18; y <= 40; y++ {
				build("road", 39, y)
			}
			for x := 37; x <= 40; x++ {
				build("road", x, 28)
			}
			for x := 37; x <= 39; x++ {
				build("road", x, 18)
			}
			build("workcamp", 40, 30)
			build("bricklayers", 40, 34)
			build("storageyard", 40, 37)
			build("firehouse", 38, 33)
			build("architect", 38, 35)
			build("smallmastaba", 40, 18)
			if err = w.SetTradePolicy("bricks", "import", 1600); err != nil {
				t.Fatal(err)
			}
			monument = true
		}
		if err = w.Step(1); err != nil {
			t.Fatal(err)
		}
		if w.Tick%240 == 0 {
			t.Logf("year %d population %d funds %d income %d wages %d prosperity %d (%+d) cap %d exports %d imports %d", w.Tick/240, w.population(), w.Treasury, w.LastIncome, w.LastWages, w.Campaign.Ratings.Prosperity, w.Campaign.Ratings.LastChange, w.prosperityInputs().HousingCap, w.ExportEarned, w.ImportSpent)
		}
		if w.Tick%128 == 0 {
			if err = w.Validate(); err != nil {
				t.Fatalf("tick %d: %v", w.Tick, err)
			}
		}
		if !saved && w.ResourceImported["bricks"] > 0 {
			s, err := Open(filepath.Join(t.TempDir(), "mennefer.nile"))
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
				t.Fatal("campaign commerce or ratings changed on reload")
			}
			w, saved = restored, true
		}
	}
	if !w.Campaign.Won || !saved || w.GoodsProduced["papyrus"] == 0 || w.BricksConsumed != 10800 || w.ResourceImported["bricks"] < 10800 {
		var tierPopulation, tierCapacity [20]int
		for _, b := range w.Buildings {
			if b.Kind == "hut" {
				tierPopulation[b.Tier] += b.Pop
				tierCapacity[b.Tier] += b.Capacity
			}
			if b.Kind != "hut" || b.Tier >= 8 {
				t.Logf("%s %d,%d pop %d tier %d workers %d/%d stock %v desire %d entertainment %d", b.Kind, b.X, b.Y, b.Pop, b.Tier, b.Workers, b.WorkersNeeded, b.Inventories, b.Desirability, entertainmentScore(b, w.Tick))
			}
		}
		t.Logf("Population by tier %v capacity %v arrivals%d departures%d disease%d", tierPopulation, tierCapacity, w.Arrivals, w.Departures, w.DiseaseDeaths)
		t.Fatalf("Men-nefer tutorial incomplete: tick %d population %d funds %d campaign %+v production %v imports %v", w.Tick, w.population(), w.Treasury, w.Campaign, w.GoodsProduced, w.ResourceImported)
	}
	t.Logf("Men-nefer won at tick %d population %d treasury %d ratings %+v", w.Tick, w.population(), w.Treasury, w.campaignObservation())
	captureCampaign(t, w)
}

func buildTradingNeighborhood(t *testing.T, w *World) {
	t.Helper()
	build := func(kind string, x, y int) {
		t.Helper()
		if w.Campaign.MissionID == 5 {
			if strings.HasPrefix(kind, "temple_") {
				kind = "temple_seth"
			}
			if kind == "workcamp" || kind == "brewery" {
				return
			}
			if kind == "chickpeafarm" || kind == "barleyfarm" {
				kind = "huntinglodge"
				x++
			}
		}
		if kind == "road" && slices.Contains(w.Roads, Point{x, y}) {
			return
		}
		if err := w.Build(kind, x, y); err != nil {
			t.Fatalf("build %s at %d,%d: %v", kind, x, y, err)
		}
	}
	for _, y := range []int{4, 14, 22, 32} {
		for x := 10; x <= 22; x++ {
			build("road", x, y)
		}
	}
	for y := 4; y <= 32; y++ {
		build("road", 22, y)
	}
	for _, b := range []struct {
		kind string
		x, y int
	}{
		{"granary", 15, 0}, {"granary", 15, 33},
		{"watersupply", 11, 5}, {"watersupply", 11, 11}, {"watersupply", 11, 23}, {"watersupply", 11, 29},
		{"bazaar", 11, 8}, {"bazaar", 20, 9}, {"bazaar", 11, 26}, {"bazaar", 20, 27},
		{"temple_ptah", 14, 5}, {"temple_osiris", 14, 11}, {"temple_bast", 14, 23}, {"temple_ptah", 14, 29},
		{"physician", 17, 5}, {"physician", 17, 23}, {"physician", 23, 30},
		{"taxoffice", 19, 5}, {"taxoffice", 19, 12}, {"taxoffice", 19, 23}, {"taxoffice", 19, 30},
		{"firehouse", 19, 3}, {"architect", 20, 3}, {"firehouse", 18, 21}, {"architect", 20, 21},
		{"bandstand", 21, 3}, {"bandstand", 21, 21}, {"jugglerschool", 23, 6}, {"conservatory", 23, 9},
		{"workcamp", 11, 15}, {"workcamp", 15, 15}, {"workcamp", 19, 15},
		{"palace", 11, 18}, {"storageyard", 15, 19}, {"claypit", 8, 30}, {"potter", 23, 15}, {"brewery", 23, 19},
	} {
		build(b.kind, b.x, b.y)
	}
	for _, y := range []int{0, 4, 8, 12} {
		build("chickpeafarm", 7, y)
	}
	for _, y := range []int{16, 20} {
		build("barleyfarm", 7, y)
	}
	for _, y := range []int{4, 14, 22, 32} {
		for x := 11; x <= 21; x++ {
			build("plaza", x, y)
		}
	}
	for _, p := range []Point{{10, 1}, {10, 17}, {22, 17}, {10, 35}, {22, 35}} {
		if slices.Contains(w.Roads, p) {
			build("roadblock", p.X, p.Y)
			for _, role := range []string{"firefighter", "architect", "taxcollector"} {
				if err := w.RoadPermission(p, role, true); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, b := range []Building{{X: 17, Y: 8, Width: 3, Height: 3}, {X: 17, Y: 26, Width: 3, Height: 3}} {
		for _, p := range b.tiles() {
			if !w.occupied()[p] {
				build("garden", p.X, p.Y)
			}
		}
	}
	for y := 0; y < 37; y++ {
		for x := 9; x <= 24; x++ {
			if x == 9 && y >= 23 && y <= 24 || x >= 23 && y >= 23 && y <= 24 || x >= 23 && y == 18 {
				continue
			}
			p := Point{x, y}
			b := Building{Kind: "hut", X: x, Y: y, Width: 1, Height: 1}
			if w.buildable(p) && !w.occupied()[p] && !slices.Contains(w.HuntingGrounds, p) && slices.ContainsFunc(w.Roads, b.adjacent) {
				build("hut", x, y)
			}
		}
	}
}

func buildTradingQuarter(t *testing.T, w *World, corner Point) {
	t.Helper()
	left, top := corner.X, corner.Y
	build := func(kind string, x, y int) {
		t.Helper()
		if w.Campaign.MissionID == 5 {
			if strings.HasPrefix(kind, "temple_") {
				kind = "temple_seth"
			}
			if kind == "workcamp" {
				kind = "huntinglodge"
			}
		}
		if kind == "road" && slices.Contains(w.Roads, Point{x, y}) {
			return
		}
		if err := w.Build(kind, x, y); err != nil {
			t.Fatalf("quarter tick%d %s at%d,%d: %v", w.Tick, kind, x, y, err)
		}
	}
	for x := w.bank(top) + 5; x <= left+12; x++ {
		build("road", x, top)
	}
	for x := left; x <= left+12; x++ {
		build("road", x, top+10)
	}
	for y := top; y <= top+10; y++ {
		build("road", left, y)
		build("road", left+12, y)
	}
	for _, b := range []struct {
		kind string
		x, y int
	}{
		{"watersupply", -2, 2}, {"watersupply", -2, 7},
		{"bazaar", 1, 1}, {"bazaar", 1, 7}, {"temple_ptah", 4, 1}, {"temple_osiris", 4, 7},
		{"physician", 7, 1}, {"taxoffice", 9, 1}, {"taxoffice", 9, 8},
		{"jugglerschool", 13, 2}, {"conservatory", 13, 5}, {"bandstand", 11, -1},
		{"firehouse", 8, -1}, {"architect", 9, -1}, {"granary", 4, 11}, {"workcamp", 1, 11},
	} {
		build(b.kind, left+b.x, top+b.y)
	}
	for y := top + 4; y <= top+6; y++ {
		for x := left + 4; x <= left+8; x++ {
			build("garden", x, y)
		}
	}
	for _, y := range []int{top, top + 10} {
		for x := left + 1; x < left+12; x++ {
			build("plaza", x, y)
		}
	}
	build("roadblock", left-1, top)
	for _, role := range []string{"firefighter", "architect"} {
		if err := w.RoadPermission(Point{left - 1, top}, role, true); err != nil {
			t.Fatal(err)
		}
	}
	if w.Campaign.MissionID == 5 {
		build("huntinglodge", left+13, top+8)
		build("huntinglodge", left+1, top+14)
	}
	for y := top - 1; y <= top+14; y++ {
		for x := left - 1; x <= left+14; x++ {
			p := Point{x, y}
			b := Building{Kind: "hut", X: x, Y: y, Width: 1, Height: 1}
			if w.buildable(p) && !w.occupied()[p] && !slices.Contains(w.HuntingGrounds, p) && slices.ContainsFunc(w.Roads, b.adjacent) {
				build("hut", x, y)
			}
		}
	}
	if w.Campaign.MissionID == 5 {
		return
	}
	farms := 0
	for y := top - 7; y < top+10 && farms < 2; y++ {
		for x := w.bank(y); x < w.bank(y)+3 && farms < 2; x++ {
			b := Building{Kind: "chickpeafarm", X: x, Y: y, Width: 3, Height: 3}
			if w.farmType(b) != "floodplain" || slices.ContainsFunc(b.tiles(), func(p Point) bool { return w.occupied()[p] || slices.Contains(w.HuntingGrounds, p) }) {
				continue
			}
			if err := w.Build("chickpeafarm", x, y); err != nil {
				continue
			}
			for cx := x + 3; cx < w.bank(y)+6; cx++ {
				build("road", cx, y)
			}
			farms++
		}
	}
	if farms != 2 {
		t.Fatal("quarter could not establish two food farms")
	}
}

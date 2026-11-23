package sandbox

import "testing"

func TestManorSubdivisionConservesResidentsAndGoods(t *testing.T) {
	w := World{NextID: 2, Buildings: []Building{{ID: 1, Kind: "hut", X: 7, Y: 7, Width: 3, Height: 3, LotSide: 3, Tier: 13, Pop: 98, Stock: 107, Goods: map[string]int{"jewelry": 286}}}}
	w.subdivideHouse(1)
	if len(w.Buildings) != 6 || w.Buildings[0].lotSide() != 2 {
		t.Fatal("manor failed to split into residence and five apartments")
	}
	stock, jewels := 0, 0
	for _, b := range w.Buildings {
		stock += b.Stock
		jewels += inventory(b, "jewelry")
		if b.Stock > resourceLimit(b, "grain") || inventory(b, "jewelry") > resourceLimit(b, "jewelry") {
			t.Fatal("overfilled a new household")
		}
	}
	if stock != 107 || jewels != 286 || w.population() != 98 {
		t.Fatal("house subdivision lost property or residents")
	}
}

func TestMonthlyFinanceSeparatesNobleAndCommonAssessments(t *testing.T) {
	w := World{LegacyTaxes: true, Treasury: 500, Buildings: []Building{{ID: 1, Kind: "hut", Pop: 100, Tier: 14, TaxUntil: 2}, {ID: 2, Kind: "hut", Pop: 10, Tier: 0, TaxUntil: 2}, {ID: 3, Kind: "hut", Pop: 30, Tier: 4, TaxUntil: 2}, {ID: 4, Kind: "jeweler", Workers: 12}}}
	w.startMonthlyFinance()
	w.settleMonthlyAccounts()
	if w.LastIncome != 37 || w.LastWages != 3 || w.Treasury != 534 {
		t.Fatalf("incorrect monthly ledger %+v", w)
	}
}

func TestShipBridgeCarriesSettlersAndKeepsRiverNavigable(t *testing.T) {
	w := NewValley()
	const y = 28
	west, east := w.westBank(y), w.bank(y)
	if err := w.Build("shipbridge", east, y); err != nil {
		t.Fatal(err)
	}
	if len(w.waterPath(Point{west + 2, y - 1}, Point{west + 2, y + 1})) == 0 {
		t.Fatal("ship bridge cut shipping lane")
	}
	var road []Point
	for x := east; x <= east+8; x++ {
		road = append(road, Point{x, y})
	}
	if err := w.BuildRoads(road); err != nil {
		t.Fatal(err)
	}
	if err := w.Build("hut", west-1, y+1); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(160); err != nil {
		t.Fatal(err)
	}
	if w.population() == 0 {
		t.Fatal("settlers did not cross to housing on the opposite bank")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEstateCannotExpandAcrossIrrigationDitch(t *testing.T) {
	w := NewValley()
	if err := w.Build("hut", 60, 30); err != nil {
		t.Fatal(err)
	}
	home := w.Buildings[0].ID
	for _, p := range []Point{{59, 30}, {61, 30}} {
		if err := w.BuildCanals([]Point{p}); err != nil {
			t.Fatal(err)
		}
	}
	w.building(home).WantedSide = 2
	if w.growHouse(home) {
		t.Fatal("residence grew onto an irrigation ditch")
	}
	w.refresh()
	db, err := Open(t.TempDir() + "/city.nile")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Save(w); err != nil {
		t.Fatal(err)
	}
	reload, err := db.Load()
	if err != nil || reload.Hash() != w.Hash() {
		t.Fatal("failed to restore canal beside home", err)
	}
}

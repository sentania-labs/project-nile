package campaign

import "testing"

func TestCultureOriginalThresholds(t *testing.T) {
	full := [11]int{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100}
	for _, tt := range []struct{ index, coverage, want int }{
		{0, 0, 5}, {0, 1, 15}, {0, 19, 15}, {0, 20, 25}, {0, 99, 75}, {0, 100, 100},
		{7, 89, 65}, {7, 90, 100}, {9, 89, 70}, {9, 90, 100}, {6, 0, 0}, {8, 0, 100},
	} {
		c := full
		c[tt.index] = tt.coverage
		if got := Culture(c); got != tt.want {
			t.Fatalf("coverage[%d]=%d: culture %d, want %d", tt.index, tt.coverage, got, tt.want)
		}
	}
}

func TestAnnualProsperityUsesFinanceAndHousingCap(t *testing.T) {
	p := ProsperityInputs{Previous: 10, HousingCap: 60, Income: 1000, Expenses: 900, Exports: 300, Imports: 200, FoodTypes: 2, Unemployment: 4}
	if got := AnnualProsperity(p); got != 18 {
		t.Fatalf("profitable employed export economy = %d, want 18", got)
	}
	p.HousingCap = 15
	if got := AnnualProsperity(p); got != 15 {
		t.Fatalf("housing cap ignored: %d", got)
	}
	p.HousingCap, p.Exports, p.Imports, p.Unemployment = 60, 0, 0, 5
	p.Income, p.Expenses, p.Construction = 900, 1000, 300
	if got := AnnualProsperity(p); got != 13 {
		t.Fatalf("partial construction allowance = %d", got)
	}
	p.Construction, p.UnpaidTribute, p.LaborShortage, p.WageDifference = 0, true, true, -11
	if got := AnnualProsperity(p); got != 0 {
		t.Fatalf("unprofitable underpaid economy did not decline: %d", got)
	}
}

func TestOriginalTributeBoundaries(t *testing.T) {
	for _, tt := range []struct {
		population   int
		profit, want int64
	}{{500, 100, 50}, {501, 100, 150}, {1000, 100, 150}, {1001, 100, 225}, {2001, 100, 300}, {3001, 100, 400}, {5001, 100, 500}, {100, 4000, 1000}, {1000, -1, 0}, {1001, -1, 100}, {2001, -1, 200}} {
		amount, unpaid := Tribute(1, tt.population, 1, tt.profit)
		if amount != tt.want || unpaid {
			t.Fatalf("population %d profit %d: %d/%v", tt.population, tt.profit, amount, unpaid)
		}
	}
	if amount, unpaid := Tribute(1, 1500, 0, 1000); amount != 0 || !unpaid {
		t.Fatal("insolvent tribute should record failure")
	}
	if amount, unpaid := Tribute(10, 1500, 0, 1000); amount != 0 || unpaid {
		t.Fatal("Pharaoh must be exempt from tribute")
	}
}

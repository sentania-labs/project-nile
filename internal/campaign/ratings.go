package campaign

// RatingState retains annual financial boundaries and monthly wage samples.
// Values are simulation counters, not renderer-supplied rating overrides.
type RatingState struct {
	PriorIncome       int64 `json:"prior_income"`
	PriorExpenses     int64 `json:"prior_expenses"`
	PriorImports      int64 `json:"prior_imports"`
	PriorExports      int64 `json:"prior_exports"`
	PriorConstruction int64 `json:"prior_construction"`
	PriorWage         int   `json:"prior_wage"`
	Prosperity        int   `json:"prosperity"`
	LastChange        int   `json:"last_change"`
	Month             int64 `json:"month"`
	Income            int64 `json:"income"`
	Expenses          int64 `json:"expenses"`
	Imports           int64 `json:"imports"`
	Exports           int64 `json:"exports"`
	Construction      int64 `json:"construction"`
	WageSum           int   `json:"wage_sum"`
	WageMonths        int   `json:"wage_months"`
	LastTribute       int64 `json:"last_tribute"`
	TributePaid       int64 `json:"tribute_paid"`
	UnpaidYears       int   `json:"unpaid_years"`
}

// Culture coverage order follows original 0x0045a500: jugglers, musicians,
// dancers, senet, schools, libraries, religion, physicians, apothecaries,
// dentists and mortuaries. Apothecaries do not enter the culture calculation.
// These are factual table values at 0x0055d5e8, interpreted by 0x0041bb20.
var cultureRows = []struct {
	coverage int
	scores   [7]int
	lastBand int
}{
	{0, [7]int{5, 15, 25, 40, 55, 75, 100}, 100},
	{1, [7]int{10, 20, 30, 45, 60, 80, 100}, 100},
	{2, [7]int{15, 25, 35, 50, 65, 85, 100}, 100},
	{3, [7]int{50, 55, 65, 75, 85, 95, 100}, 100},
	{6, [7]int{0, 10, 25, 40, 60, 85, 100}, 100},
	{4, [7]int{40, 45, 50, 60, 75, 85, 100}, 100},
	{5, [7]int{60, 65, 70, 75, 80, 90, 100}, 100},
	{7, [7]int{15, 20, 25, 35, 50, 65, 100}, 90},
	{9, [7]int{20, 25, 30, 40, 55, 70, 100}, 90},
	{10, [7]int{35, 45, 55, 70, 85, 100, 100}, 100},
}

func cultureBand(coverage, lastBand int) int {
	if coverage <= 0 {
		return 0
	}
	if coverage >= lastBand {
		return 6
	}
	return min(5, coverage/20+1)
}

func Culture(coverage [11]int) int {
	rating := 100
	for _, row := range cultureRows {
		rating = min(rating, row.scores[cultureBand(coverage[row.coverage], row.lastBand)])
	}
	// The separate original bonus pass uses 100, including for medical rows.
	bonus := 0
	for _, row := range cultureRows {
		if row.scores[cultureBand(coverage[row.coverage], 100)]+30 < rating {
			bonus += 5
		}
	}
	return min(100, rating+bonus)
}

// AnnualProsperity evaluates annual finance inputs at 0x0045ffb0.
// The caller applies the result quarterly, as verified at 0x0045ff50.
// Housing cap is population-weighted, confirmed by 0x00460590.
type ProsperityInputs struct {
	Previous, HousingCap, Unemployment, FoodTypes, WageDifference int
	HutPercent, ManorPercent                                      int
	Income, Expenses, Construction, Imports, Exports              int64
	LaborShortage, UnpaidTribute, Senet, Monument                 bool
}

func AnnualProsperity(p ProsperityInputs) int {
	change := 0
	profit := p.Income - p.Expenses
	switch {
	case profit > 0:
		change += 5
	case profit+p.Construction/2 > 0:
		change += 3
	case profit < 0:
		change -= 8
	case profit+p.Construction < 0:
		change -= 5
	}
	if p.Exports > p.Imports {
		change++
	} else if p.Exports < p.Imports {
		change--
	}
	switch {
	case p.LaborShortage:
		change--
	case p.Unemployment < 5:
		change += 2
	case p.Unemployment >= 20:
		change -= 3
	case p.Unemployment >= 15:
		change -= 2
	case p.Unemployment >= 10:
		change--
	}
	if p.FoodTypes < 2 {
		change--
	}
	switch {
	case p.WageDifference >= 6:
		change += 2
	case p.WageDifference > 0:
		change++
	case p.WageDifference < -10:
		change -= 3
	case p.WageDifference < -5:
		change -= 2
	case p.WageDifference < 0:
		change--
	}
	if p.HutPercent > 95 {
		change--
	}
	if p.ManorPercent > 10 {
		change += 2
	}
	if p.UnpaidTribute {
		change--
	}
	if p.Senet {
		change++
	}
	if p.Monument {
		change++
	}
	return max(0, min(100, p.HousingCap, p.Previous+change))
}

// Tribute follows original 0x0044d340. A solvent treasury can be made
// negative by tribute; inability to pay is assessed before the annual debit.
func Tribute(rank, population int, treasury, profit int64) (amount int64, unpaid bool) {
	if rank == 10 {
		return 0, false
	}
	if treasury <= 0 {
		return 0, true
	}
	if profit <= 0 {
		if population > 2000 {
			return 200, false
		}
		if population > 1000 {
			return 100, false
		}
		return 0, false
	}
	minimum := int64(50)
	for i, threshold := range []int{500, 1000, 2000, 3000, 5000} {
		if population > threshold {
			minimum = []int64{150, 225, 300, 400, 500}[i]
		}
	}
	return max(minimum, profit/4), false
}

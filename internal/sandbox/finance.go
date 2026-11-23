package sandbox

import "slices"

// Column t of the retail Normal housing model, from Small Hut to Palatial Estate.
var houseTaxWeights = [...]int{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 10, 11, 12, 13, 15, 16}

func (w *World) startMonthlyFinance() {
	w.MonthlyFinance = true
	w.RoyalCredit = true
	w.KingdomWage = 30
	w.AgricultureVersion = 1
	w.TaxRate, w.WageRate = 7, 30
}

// Normal monthly assessment. Citizens and noble households each carry their
// own rounding remainder between houses within the same assessment.
func (w *World) settleMonthlyAccounts() {
	var remainder [2]int64
	w.LastIncome = 0
	workers := 0
	enabled := w.taxesEnabled()
	for i := range w.Buildings {
		b := &w.Buildings[i]
		workers += b.Workers
		b.LastTax = 0
		if !enabled || b.Kind != "hut" || b.TaxUntil <= w.Tick || b.Pop == 0 {
			continue
		}
		group := 0
		if b.Tier >= 14 {
			group = 1
		}
		value := remainder[group] + int64(b.Pop*houseTaxWeights[b.Tier]*w.TaxRate)
		b.LastTax = value / 200
		remainder[group] = value % 200
		w.LastIncome += b.LastTax
	}
	w.LastWages = int64(w.WageRate * workers / 120)
	w.Treasury += w.LastIncome - w.LastWages
	w.Income += w.LastIncome
	w.Expenses += w.LastWages
	if w.RoyalCredit && w.Treasury < 0 && w.Tick%YearTicks == 0 {
		// The retail manual calls for annual payment, even below the credit limit.
		// The 10 percent rate is provisional. Limit accrued interest to the
		// exactly representable integer range of the desktop protocol.
		charge := min(-(w.Treasury / 10), max(0, MaxTick-w.Expenses))
		if w.templeBenefit("complex_ra", 0) {
			charge /= 2
		}
		w.Household.Interest += charge
		w.Expenses += charge
		w.Treasury -= charge
	}
}

// Pharaoh manual pp.106–107. Modern cities borrow for construction up to
// 5,000 debens. Existing cities keep the financial contract of their save.
func (w World) constructionCredit() int64 {
	if w.RoyalCredit {
		return 5000
	}
	return 0
}
func (w World) cannotBuildFor(cost int64) bool {
	return w.BudgetEnabled && cost > w.Treasury+w.constructionCredit()
}
func (w World) sitePrice(kind string) int64 {
	if kind == "palace" && w.RoyalCredit {
		return 900
	}
	return constructionCost(kind)
}
func (w World) pricedCatalog() []CatalogEntry {
	if !w.RoyalCredit {
		return buildingCatalog
	}
	result := slices.Clone(buildingCatalog)
	for i := range result {
		if result[i].Kind == "palace" {
			result[i].Cost = w.sitePrice("palace")
			break
		}
	}
	return slices.DeleteFunc(result, func(e CatalogEntry) bool { return e.Kind == "mastaba" })
}

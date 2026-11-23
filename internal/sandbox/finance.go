package sandbox

// Column t of the retail Normal housing model, from Small Hut to Palatial Estate.
var houseTaxWeights = [...]int{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 10, 11, 12, 13, 15, 16}

func (w *World) startMonthlyFinance() {
	w.MonthlyFinance = true
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
}

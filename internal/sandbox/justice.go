package sandbox

import (
	"errors"
	"slices"
)

var salaryByRank = [...]int{0, 2, 5, 8, 12, 20, 30, 40, 60, 80, 100}
var taxSentiment = [...]int{3, 2, 2, 2, 1, 1, 1, 0, 0, -1, -2, -2, -3, -3, -3, -5, -5, -5, -5, -6, -6, -6, -6, -6, -6, -6}

func init() {
	buildingCatalog = append(buildingCatalog, CatalogEntry{"courthouse", "Courthouse", 3, 3, 150, "Government", 10}, CatalogEntry{"residence", "Personal residence", 3, 3, 150, "Government", 0})
}

func (w World) hasResidence() bool {
	return slices.ContainsFunc(w.Buildings, func(b Building) bool { return isResidence(b.Kind) && b.Damaged == "" })
}
func (w *World) SetSalary(rank int) error {
	if rank < 0 || rank >= len(salaryByRank) || !w.hasResidence() {
		return errors.New("a personal residence and a valid salary rank are required")
	}
	w.SalaryRank = rank
	return nil
}
func (w *World) DonateSavings(amount int64) error {
	if amount <= 0 || amount > w.Savings {
		return errors.New("choose an amount within personal savings")
	}
	w.Savings -= amount
	w.Treasury += amount
	w.Income += amount
	w.Household.Donated += amount
	return nil
}
func (w *World) paySalary() {
	if !w.hasResidence() || w.Treasury <= 0 {
		return
	}
	n := min(int64(salaryByRank[w.SalaryRank]), w.Treasury)
	w.Treasury -= n
	w.Expenses += n
	w.Savings += n
}
func crimeTarget(b Building) bool {
	return isPalace(b.Kind) || b.Kind == "taxoffice" || b.Kind == "courthouse" || isResidence(b.Kind)
}
func (w World) sentiment() int {
	n, pop := 0, 0
	for _, b := range w.Buildings {
		if b.Kind == "hut" && b.Pop > 0 {
			mood := b.Happiness
			if !b.MoodSet {
				mood = 60
			}
			n += mood * b.Pop
			pop += b.Pop
		}
	}
	if pop == 0 {
		return 60
	}
	return n / pop
}
func wageSentiment(rate int) int {
	switch diff := rate - 30; {
	case diff < 0:
		return min(-1, diff/2)
	case diff > 7:
		return 4
	case diff > 4:
		return 3
	case diff > 1:
		return 2
	default:
		return diff
	}
}
func (w *World) justiceTick() {
	if !w.MonthlyFinance || w.Tick%(YearTicks/12) != 0 {
		return
	}
	w.paySalary()
	pop := w.population()
	workforce, employed := w.availableWorkforce(), 0
	for _, b := range w.Buildings {
		employed += b.Workers
	}
	unemployment := 100 * max(0, workforce-employed) / max(1, workforce)
	var origin int64
	complex := w.templeComplex()
	wage := 30 + w.WageRate - w.prevailingWage()
	if wage < 30 && hasTempleBenefit(complex, "complex_ra", 2) {
		wage = min(30, wage+4)
	}
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind != "hut" || b.Pop == 0 {
			continue
		}
		if !b.MoodSet {
			b.Happiness = 60
			b.MoodSet = true
		}
		if pop < 300 {
			b.Happiness = 50
			if pop < 200 {
				b.Happiness = 60
			}
			continue
		}
		change := taxSentiment[w.TaxRate] + wageSentiment(wage)
		if hasTempleBenefit(complex, "complex_bast", 2) {
			change += 2
		}
		if unemployment > 25 {
			change -= 3
		} else if unemployment > 15 {
			change--
		}
		if b.Tier >= 2 {
			if b.Food {
				change++
			} else {
				change -= 2
			}
		}
		b.Happiness = min(100, max(0, b.Happiness+change))
		if b.Happiness < 30 && b.ServiceUntil[6] <= w.Tick && b.ServiceUntil[8] <= w.Tick {
			risk := 10
			if hasTempleBenefit(complex, "complex_ra", 1) {
				risk = 5
			}
			b.CrimeRisk = min(100, b.CrimeRisk+risk)
		} else {
			b.CrimeRisk = max(0, b.CrimeRisk-10)
		}
		if origin == 0 && b.CrimeRisk >= 60 && b.Connected {
			origin = b.ID
		}
	}
	if origin == 0 || !w.HazardsEnabled || slices.ContainsFunc(w.Walkers, func(a Walker) bool { return a.Role == "robber" }) {
		return
	}
	home := w.building(origin)
	var route []Point
	var target int64
	for _, b := range w.Buildings {
		if !crimeTarget(b) || !b.Connected || b.Damaged != "" || isResidence(b.Kind) && w.Savings == 0 || !isResidence(b.Kind) && w.Treasury <= 0 {
			continue
		}
		path := w.path(w.accessTiles(*home), b)
		if len(path) > 0 && (route == nil || len(path) < len(route)) {
			route, target = path, b.ID
		}
	}
	if w.spawn("robber", "robbery", origin, target, 0, route, 0, 0) {
		home.CrimeRisk = 0
	}
}

func (w *World) interceptRobber(a Walker) bool {
	for _, patrol := range w.Walkers {
		if patrol.Role == "policestation" && nearBuilding(Point{patrol.X, patrol.Y}, Building{X: a.X, Y: a.Y, Width: 1, Height: 1}, 1) {
			w.Arrests++
			return true
		}
	}
	return false
}
func (w *World) rob(target *Building) {
	if target.Damaged != "" {
		return
	}
	if isResidence(target.Kind) {
		n := w.Savings * 9 / 100
		w.Savings -= n
		w.Stolen += n
		return
	}
	// A 9% loss is the Normal-model theft percentage. Use proceeds from
	// the latest local assessment while per-building tax storage is provisional.
	n := min(max(int64(0), w.Treasury), w.LastIncome) * 9 / 100
	w.Stolen += n
	w.Expenses += n
	w.Treasury -= n
}

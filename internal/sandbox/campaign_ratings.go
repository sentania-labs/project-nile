package sandbox

import (
	"errors"
	"slices"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

var housingProsperity = [...]int{5, 10, 15, 20, 25, 30, 35, 45, 50, 55, 60, 70, 80, 90, 500, 600, 700, 800, 1600, 1900}

// Original citywide cultural capacity counts staffed buildings, including
// jugglers in Bandstands and Pavilions. Walked household access remains separate.
func (w World) cultureCoverage() [11]int {
	var coverage [11]int
	if w.population() == 0 {
		return coverage
	}
	godsKnown := [5]int{1, 1, 1, 1, 1}
	if w.Campaign != nil {
		m, _ := campaign.Definition(w.Campaign.MissionID)
		godsKnown = m.KnownGods
	}
	var worship [5]int
	pop := w.population()
	for _, b := range w.Buildings {
		if b.Damaged != "" {
			continue
		}
		if b.Kind == "hut" {
			for _, pair := range [][2]int{{7, 1}, {8, 0}, {9, 2}, {10, 3}} {
				if b.ServiceUntil[pair[1]] > w.Tick {
					coverage[pair[0]] += b.Pop
				}
			}
		}
		if god := slices.Index(shrines, b.Kind); god >= 0 {
			capacity := 300
			if godsKnown[god] == 2 {
				capacity = 150
			}
			worship[god] += capacity
		}
		if !b.Connected || b.Workers == 0 {
			continue
		}
		if god := templeGod(b.Kind); god >= 0 {
			capacity := 750
			if godsKnown[god] == 2 {
				capacity = 375
			}
			if slices.Contains(complexKinds, b.Kind) {
				capacity = 8000
			}
			worship[god] += capacity
		}
		switch b.Kind {
		case "pavilion":
			coverage[2] += 1200
			fallthrough
		case "bandstand":
			coverage[1] += 700
			fallthrough
		case "booth":
			coverage[0] += 400
		case "senethouse":
			coverage[3] += 5000
		case "scribalschool":
			coverage[4] += 300
		case "library":
			coverage[5] += 800
		}
	}
	for i := range coverage {
		// Original school denominator is the under-14 population. Until age
		// cohorts are implemented, all residents is a conservative denominator.
		coverage[i] = min(100, coverage[i]*100/pop)
	}
	known := 0
	for god, status := range godsKnown {
		if status > 0 {
			known++
			coverage[6] += min(100, worship[god]*100/pop)
		}
	}
	coverage[6] /= max(1, known)
	return coverage
}

func (w World) prosperityInputs() campaign.ProsperityInputs {
	p := campaign.ProsperityInputs{}
	weighted, huts, manors, employed, needed := 0, 0, 0, 0, 0
	for _, b := range w.Buildings {
		employed += b.Workers
		if b.Connected && b.Damaged == "" {
			needed += b.WorkersNeeded
		}
		if b.Kind == "hut" {
			weighted += housingProsperity[b.Tier] * b.Pop
			if b.Tier < 2 {
				huts += b.Pop
			}
			if b.Tier >= 14 {
				manors += b.Pop
			}
			foods := 0
			for _, r := range foodKinds {
				if inventory(b, r) > 0 {
					foods++
				}
			}
			// Original uses the maximum food types eaten by a household,
			// not citywide average. Stored foods approximate that monthly sample.
			p.FoodTypes = max(p.FoodTypes, foods)
		}
		p.Senet = p.Senet || b.Kind == "senethouse" && b.Workers > 0 && b.Connected && b.Damaged == ""
		p.Monument = p.Monument || isMonument(b.Kind) && monumentComplete(b)
	}
	pop, workforce := max(1, w.population()), w.availableWorkforce()
	p.HousingCap, p.HutPercent, p.ManorPercent = weighted/pop, huts*100/pop, manors*100/pop
	p.Unemployment = max(0, workforce-employed) * 100 / max(1, workforce)
	p.LaborShortage = needed > employed
	return p
}

func (w World) ratingState() *campaign.RatingState {
	if w.Campaign != nil {
		return w.Campaign.Ratings
	}
	return w.SandboxRatings
}
func (w World) cultureRating() int {
	score := campaign.Culture(w.cultureCoverage())
	// Cleopatra manual, p.9: culturally ambitious cities require a supplied zoo.
	if w.Campaign == nil || w.Campaign.MissionID == 31 || w.Campaign.MissionID == 37 || w.Campaign.MissionID >= 38 {
		if score >= 75 && w.zooCoverage() == 0 {
			score = 74
		}
	}
	return score
}
func (w *World) spendConstruction(amount int64) {
	w.spend(amount)
	if r := w.ratingState(); w.BudgetEnabled && r != nil {
		r.Construction += amount
	}
}

func (w *World) campaignRatingsTick() {
	if w.Campaign == nil && !w.RoyalCredit {
		return
	}
	month := w.Tick / (YearTicks / 12)
	r := w.ratingState()
	if r == nil {
		// Existing saves begin a fresh financial window; do not replay years.
		r = &campaign.RatingState{Month: month, Income: w.Income, Expenses: w.Expenses, Imports: w.ImportSpent, Exports: w.ExportEarned}
		if w.Campaign != nil {
			w.Campaign.Ratings = r
		} else {
			w.SandboxRatings = r
		}
		return
	}
	if month <= r.Month {
		return
	}
	r.Month = month
	r.WageSum += w.WageRate
	r.WageMonths++
	if month%3 != 0 {
		return
	}
	p := w.prosperityInputs()
	if month%12 == 0 {
		p.Income, p.Expenses = w.Income-r.Income, w.Expenses-r.Expenses
		m := campaign.Mission{}
		if w.Campaign != nil {
			m, _ = campaign.Definition(w.Campaign.MissionID)
		}
		r.LastTribute, p.UnpaidTribute = campaign.Tribute(m.Rank, w.population(), w.Treasury, p.Income-p.Expenses)
		if p.UnpaidTribute {
			r.UnpaidYears++
		} else {
			r.UnpaidYears = 0
		}
		w.Treasury -= r.LastTribute
		w.Expenses += r.LastTribute
		r.TributePaid += r.LastTribute
		if w.Campaign != nil {
			w.annualKingdomRating(m, p, r)
		} else {
			gain := 0
			if p.UnpaidTribute {
				gain = -3
			} else if p.Income > p.Expenses+r.LastTribute {
				gain = 1
			}
			if w.templeBenefit("complex_ra", 0) {
				gain++
			}
			if gain != 0 {
				rating := max(0, min(100, w.kingdomStanding()+gain))
				w.Household.Reputation = &rating
			}
		}
		r.PriorIncome, r.PriorExpenses = p.Income, p.Expenses
		r.PriorConstruction = r.Construction
		r.PriorImports, r.PriorExports = w.ImportSpent-r.Imports, w.ExportEarned-r.Exports
		r.PriorWage = r.WageSum / max(1, r.WageMonths)
		r.Income, r.Expenses = w.Income, w.Expenses
		r.Imports, r.Exports = w.ImportSpent, w.ExportEarned
		r.Construction, r.WageSum, r.WageMonths = 0, 0, 0
	}
	p.Previous = r.Prosperity
	p.Income, p.Expenses = r.PriorIncome, r.PriorExpenses
	p.Construction, p.Imports, p.Exports = r.PriorConstruction, r.PriorImports, r.PriorExports
	p.UnpaidTribute = r.UnpaidYears > 0
	p.WageDifference = r.PriorWage - w.prevailingWage()
	r.Prosperity = campaign.AnnualProsperity(p)
	r.LastChange = r.Prosperity - p.Previous
}

// Original 0x00460740: annual decay, tribute, salary rank and progress reviews.
func (w *World) annualKingdomRating(m campaign.Mission, p campaign.ProsperityInputs, r *campaign.RatingState) {
	change := 0
	if w.templeBenefit("complex_ra", 0) {
		change++
	}
	if m.ID > campaign.Thinis {
		change--
	}
	if r.UnpaidYears > 0 {
		change -= []int{0, 3, 5, 8}[min(3, r.UnpaidYears)]
	} else if p.Income > p.Expenses+r.LastTribute {
		change++
	}
	difference := w.SalaryRank - m.Rank
	if difference > 0 {
		change -= difference
	} else if difference < 0 && m.Rank > 0 {
		change++
	}
	progress := w.campaignObservation().Progress()
	for i, year := range m.MilestoneYears {
		if year <= 0 || int(r.Month/12) != year {
			continue
		}
		meets := true
		for name, current := range map[string]int{"population": progress.Population, "culture": progress.Culture, "prosperity": progress.Prosperity, "monuments": progress.Monuments, "kingdom": progress.Kingdom} {
			goal := m.Goals[name]
			if goal.Enabled != 0 && current < goal.Value*(i+1)/4 {
				meets = false
			}
		}
		if meets {
			change += 5
		} else {
			change -= 2
		}
		break
	}
	w.Campaign.Kingdom = min(100, max(0, w.Campaign.Kingdom+change))
}

func (w World) validateCampaignRatings() error {
	if w.SandboxRatings != nil && (w.Campaign != nil || !w.RoyalCredit) {
		return errors.New("incompatible city rating accounts")
	}
	r := w.ratingState()
	if r == nil {
		return nil
	}
	if r.PriorIncome < 0 || r.PriorIncome > w.Income || r.PriorExpenses < 0 || r.PriorExpenses > w.Expenses || r.PriorImports < 0 || r.PriorImports > w.ImportSpent || r.PriorExports < 0 || r.PriorExports > w.ExportEarned || r.PriorConstruction < 0 || r.PriorConstruction > r.PriorExpenses || r.PriorWage < 0 || r.PriorWage > 100 {
		return errors.New("invalid previous-year campaign finance")
	}
	if r.Prosperity < 0 || r.Prosperity > 100 || r.Month < 0 || r.Month > w.Tick/(YearTicks/12) || r.Income < 0 || r.Income > w.Income || r.Expenses < 0 || r.Expenses > w.Expenses || r.Imports < 0 || r.Imports > w.ImportSpent || r.Exports < 0 || r.Exports > w.ExportEarned || r.Construction < 0 || r.Construction > w.Expenses-r.Expenses || r.WageMonths < 0 || r.WageMonths > 11 || r.WageSum < 0 || r.WageSum > 100*r.WageMonths || r.LastTribute < 0 || r.TributePaid < r.LastTribute || r.TributePaid > w.Expenses || r.UnpaidYears < 0 || int64(r.UnpaidYears) > r.Month/12 {
		return errors.New("invalid persisted campaign ratings")
	}
	return nil
}

package sandbox

import (
	"errors"
	"slices"
)

const BuilderShiftTicks int64 = 96

// These fiscal feedback rules are prototype tuning, not original-game claims.
func (w World) availableWorkforce() int {
	if w.BudgetEnabled && w.WageRate == 0 {
		return 0
	}
	return w.population() / 2
}
func (w World) migrationInterval() int64 {
	if w.BudgetEnabled && w.TaxRate > 5 {
		return MigrationInterval * int64(w.TaxRate-4)
	}
	return MigrationInterval
}

func validPolicy(tax, wage int, priority string) bool {
	return tax >= 0 && tax <= 10 && wage >= 0 && wage <= 5 && (priority == "balanced" || priority == "food" || priority == "construction" || priority == "services")
}
func (w *World) policyDefaults() { w.TaxRate = 2; w.WageRate = 1; w.LaborPriority = "balanced" }
func (w *World) SetPolicy(tax, wage int, priority string) error {
	if !validPolicy(tax, wage, priority) {
		return errors.New("tax must be 0..10, wages 0..5, and priority balanced, food, construction, or services")
	}
	w.TaxRate, w.WageRate, w.LaborPriority = tax, wage, priority
	w.refresh()
	return nil
}
func staffingRank(priority, kind string) int {
	entry, _ := catalog(kind)
	switch priority {
	case "food":
		switch entry.Category {
		case "Food":
			return 0
		case "Services":
			return 1
		case "Government":
			return 2
		default:
			return 3
		}
	case "construction":
		switch entry.Category {
		case "Industry":
			return 0
		case "Monuments":
			return 1
		case "Food":
			return 2
		case "Services":
			return 3
		default:
			return 4
		}
	case "services":
		switch entry.Category {
		case "Services", "Government":
			return 0
		case "Food":
			return 1
		default:
			return 2
		}
	}
	return 0
}
func (w World) builderCount(site, home int64) int {
	n := 0
	for _, walker := range w.Walkers {
		if walker.Role == "builder" && (site == 0 || walker.Source == site) && (home == 0 || walker.Origin == home) {
			n++
		}
	}
	return n
}
func (w *World) dispatchBuilders() {
	for _, site := range w.Buildings {
		if site.Kind != "mastaba" || !site.Connected || site.Damaged != "" || site.ConstructionDone >= 64 || site.Workers == 0 {
			continue
		}
		for w.builderCount(site.ID, 0) < site.Workers {
			homeID := int64(0)
			var best []Point
			fewest := int(^uint(0) >> 1)
			for _, home := range w.Buildings {
				assigned := w.builderCount(0, home.ID)
				if home.Kind != "hut" || home.Damaged != "" || !home.Connected || home.Pop/2 <= assigned {
					continue
				}
				path := w.path(w.perimeter(home), site)
				if len(path) > 0 && (best == nil || assigned < fewest || (assigned == fewest && len(path) < len(best))) {
					homeID = home.ID
					best = path
					fewest = assigned
				}
			}
			if !w.spawn("builder", "commute", homeID, site.ID, site.ID, best, 0, 0) {
				break
			}
		}
	}
}
func (w World) builderAllowed(walker Walker, site, home Building) bool {
	if site.Damaged != "" || site.ConstructionDone >= 64 || site.Workers == 0 || home.Damaged != "" || home.Pop < 2 {
		return false
	}
	siteEarlier, homeEarlier := 0, 0
	for _, other := range w.Walkers {
		if other.Role != "builder" || other.ID >= walker.ID {
			continue
		}
		if other.Source == site.ID && other.Stage != "return" {
			siteEarlier++
		}
		if other.Origin == home.ID {
			homeEarlier++
		}
	}
	return siteEarlier < site.Workers && homeEarlier < home.Pop/2
}
func builderReturn(walker *Walker) {
	// Reverse only the route already walked if a commute is cancelled mid-way.
	route := slices.Clone(walker.Route[:walker.Next])
	slices.Reverse(route)
	walker.Route = route
	walker.Next = 1
	walker.Target = walker.Origin
	walker.Stage = "return"
	walker.ShiftUntil = 0
	walker.X, walker.Y = route[0].X, route[0].Y
}
func (w *World) advanceBuilder(walker *Walker) bool {
	site, home := w.building(walker.Source), w.building(walker.Origin)
	if walker.Stage != "return" && (!w.builderAllowed(*walker, *site, *home) || (walker.Stage == "work" && w.Tick >= walker.ShiftUntil)) {
		builderReturn(walker)
		return true
	}
	if walker.Stage == "work" {
		return true
	}
	if walker.Next < len(walker.Route) {
		p := walker.Route[walker.Next]
		walker.X, walker.Y = p.X, p.Y
		walker.Next++
	}
	if walker.Next < len(walker.Route) {
		return true
	}
	if walker.Stage == "return" {
		return false
	}
	walker.Stage = "work"
	walker.ShiftUntil = min(MaxTick, w.Tick+BuilderShiftTicks)
	return true
}
func (w World) validateBuilder(walker Walker, target, home Building) error {
	site := w.building(walker.Source)
	if home.Kind != "hut" || site == nil || site.Kind != "mastaba" || walker.Cargo != 0 || walker.Resource != "" || walker.ShiftUntil < 0 || walker.ShiftUntil > MaxTick {
		return errors.New("invalid construction worker")
	}
	switch walker.Stage {
	case "commute":
		if target.ID != site.ID || walker.ShiftUntil != 0 || !home.adjacent(walker.Route[0]) {
			return errors.New("invalid builder commute")
		}
	case "work":
		if target.ID != site.ID || walker.Next != len(walker.Route) || walker.ShiftUntil < w.Tick || !home.adjacent(walker.Route[0]) {
			return errors.New("invalid on-site builder")
		}
	case "return":
		if target.ID != home.ID || walker.ShiftUntil != 0 {
			return errors.New("invalid builder return")
		}
	default:
		return errors.New("unknown builder stage")
	}
	return nil
}

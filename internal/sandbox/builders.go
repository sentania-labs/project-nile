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
	workers := 0
	for _, b := range w.Buildings {
		if b.Kind == "hut" && b.Tier < 14 {
			workers += b.Pop
		}
	}
	return workers / 2
}
func (w World) migrationInterval() int64 {
	if w.MonthlyFinance {
		return MigrationInterval * int64(1+max(0, w.TaxRate-8)/3)
	}
	if w.BudgetEnabled && w.TaxRate > 5 {
		return MigrationInterval * int64(w.TaxRate-4)
	}
	return MigrationInterval
}

func (w World) validPolicy(tax, wage int, priority string) bool {
	maxTax, maxWage := 10, 5
	if w.MonthlyFinance {
		maxTax, maxWage = 25, 100
	}
	return tax >= 0 && tax <= maxTax && wage >= 0 && wage <= maxWage && (priority == "balanced" || priority == "food" || priority == "construction" || priority == "services")
}
func (w *World) policyDefaults() { w.TaxRate = 2; w.WageRate = 1; w.LaborPriority = "balanced" }
func (w *World) SetPolicy(tax, wage int, priority string) error {
	if !w.validPolicy(tax, wage, priority) {
		return errors.New("tax rate, wage rate or labor priority is outside the available range")
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
	w.dispatchPyramidBuilders()
	w.dispatchStoneHaulers()
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
	if isPyramid(site.Kind) {
		return w.pyramidBuilderAllowed(walker, site, home)
	}
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
		if !w.depositBuilderSupplies(walker, home.ID) {
			return true
		}
		if walker.Cargo > 0 {
			walker.Cargo -= w.DepositResource(home.ID, walker.Resource, walker.Cargo)
			return walker.Cargo > 0
		}
		return false
	}
	if isPyramid(site.Kind) {
		if !w.depositBuilderSupplies(walker, site.ID) {
			return true
		}
		walker.Cargo -= w.DepositResource(site.ID, walker.Resource, walker.Cargo)
		walker.Stage = "work"
		walker.ShiftUntil = MaxTick
		return true
	}
	walker.Stage = "work"
	walker.ShiftUntil = min(MaxTick, w.Tick+BuilderShiftTicks)
	return true
}
func (w World) validateBuilder(walker Walker, target, home Building) error {
	site := w.building(walker.Source)
	if site != nil && isPyramid(site.Kind) {
		return w.validatePyramidBuilder(walker, target, home, *site)
	}
	if home.Kind != "hut" || site == nil || site.Kind != "mastaba" || walker.Cargo != 0 || walker.Resource != "" || walker.ShiftUntil < 0 || walker.ShiftUntil > MaxTick {
		return errors.New("invalid construction worker")
	}
	switch walker.Stage {
	case "commute":
		if target.ID != site.ID || walker.ShiftUntil != 0 || !home.roadAccess(walker.Route[0]) {
			return errors.New("invalid builder commute")
		}
	case "work":
		if target.ID != site.ID || walker.Next != len(walker.Route) || walker.ShiftUntil < w.Tick || !home.roadAccess(walker.Route[0]) {
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

func (w World) guildAssignments(id int64) int {
	n := 0
	for _, a := range w.Walkers {
		if a.Origin == id {
			n++
		}
	}
	return n
}
func pyramidGuildLimit(kind string) int {
	switch kind {
	case "workcamp":
		return 4
	case "stonemasons", "bricklayers":
		return 2
	case "carpenters", "artisans":
		return 1
	}
	return 0
}
func (w *World) dispatchPyramidBuilders() {
	for _, site := range w.Buildings {
		phase, ok := currentMonumentPhase(site)
		if !ok || site.Damaged != "" || !site.StagingAccess {
			continue
		}
		guild := phase.Guild
		count := 0
		for _, a := range w.Walkers {
			if a.Role == "builder" && a.Source == site.ID {
				if b := w.building(a.Origin); b != nil && b.Kind == guild {
					count++
				}
			}
		}
		for _, origin := range w.Buildings {
			if origin.Kind != guild || origin.Damaged != "" || !origin.Connected {
				continue
			}
			limit := monumentGuildLimit(site, guild)
			for count < limit && w.guildAssignments(origin.ID) < origin.Workers {
				cargo := 0
				if !suppliesReady(*w.building(origin.ID), phase.Supplies) {
					break
				}
				if len(phase.Supplies) > 0 && (inventory(site, "paint")+w.incomingResource(site.ID, "paint") > 0 || inventory(site, "clay")+w.incomingResource(site.ID, "clay") > 0) {
					break
				}
				if guild == "carpenters" && phase.Timber > 0 && inventory(site, "timber")+w.incomingResource(site.ID, "timber") == 0 {
					if inventory(*w.building(origin.ID), "timber") < CarpenterSupply {
						break
					}
					cargo = CarpenterSupply
				}
				route := w.path(w.perimeter(origin), site)
				if !w.spawn("builder", "commute", origin.ID, site.ID, site.ID, route, cargo, 0) {
					break
				}
				if len(phase.Supplies) > 0 {
					a := &w.Walkers[len(w.Walkers)-1]
					a.Supplies = map[string]int{}
					b := w.building(origin.ID)
					for r, n := range phase.Supplies {
						a.Supplies[r] = n
						setInventory(b, r, inventory(*b, r)-n)
					}
				}
				if cargo > 0 {
					a := &w.Walkers[len(w.Walkers)-1]
					a.Resource = "timber"
					b := w.building(origin.ID)
					setInventory(b, "timber", inventory(*b, "timber")-cargo)
				}
				count++
			}
		}
	}
}
func (w World) pyramidBuilderAllowed(a Walker, site, origin Building) bool {
	if site.Stage == "complete" || site.Damaged != "" || origin.Damaged != "" || origin.Workers == 0 {
		return false
	}
	phase, ok := currentMonumentPhase(site)
	if !ok {
		return false
	}
	if origin.Kind == "artisans" && a.Stage == "work" && !suppliesReady(site, phase.Supplies) {
		return false
	}
	if (origin.Kind == "workcamp" && phase.Guild != "workcamp") || (discreteMonumentJobs(site.Kind) && (phase.Guild != origin.Kind || (origin.Kind == "carpenters" && a.Stage == "work" && phase.Timber > 0 && inventory(site, "timber") == 0))) {
		return false
	}
	earlier := 0
	for _, other := range w.Walkers {
		if other.Origin == origin.ID && other.ID < a.ID {
			earlier++
		}
	}
	return earlier < origin.Workers
}
func (w World) validatePyramidBuilder(a Walker, target, origin, site Building) error {
	if !validBuilderSupplies(a, origin) {
		return errors.New("invalid artisan supplies")
	}
	if pyramidGuildLimit(origin.Kind) == 0 || a.Reserved != 0 || (a.Cargo != 0 && a.Cargo != CarpenterSupply) || a.Cargo > 0 && origin.Kind != "carpenters" || (a.Resource != "" && a.Resource != "timber") || a.Cargo > 0 && a.Resource != "timber" {
		return errors.New("invalid pyramid guild worker")
	}
	switch a.Stage {
	case "commute", "work":
		if target.ID != site.ID || !origin.roadAccess(a.Route[0]) || a.Stage == "work" && (a.Cargo != 0 || a.Next != len(a.Route) || a.ShiftUntil != MaxTick) || a.Stage == "commute" && a.ShiftUntil != 0 {
			return errors.New("invalid pyramid worker journey")
		}
	case "return":
		if target.ID != origin.ID || a.ShiftUntil != 0 {
			return errors.New("invalid pyramid worker return")
		}
	default:
		return errors.New("unknown pyramid worker stage")
	}
	return nil
}

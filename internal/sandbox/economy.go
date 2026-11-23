package sandbox

import "slices"

func (w World) reservedStock(id int64) int {
	n := 0
	for _, walker := range w.Walkers {
		if walker.Role == "buyer" && walker.Stage == "fetch" && walker.Source == id {
			n += walker.Reserved
		}
	}
	return n
}
func (w World) reservedSpace(id int64) int {
	n := 0
	for _, walker := range w.Walkers {
		if walker.Role == "cart" && walker.Stage == "deliver" && walker.Target == id {
			n += walker.Cargo
		}
		if walker.Role == "buyer" && walker.Origin == id {
			if walker.Stage == "fetch" {
				n += walker.Reserved
			} else {
				n += walker.Cargo
			}
		}
		if walker.Role == "seller" && walker.Origin == id {
			n += walker.Cargo
		}
	}
	return n
}
func (w World) active(origin int64, role string) bool {
	for _, walker := range w.Walkers {
		if walker.Origin == origin && walker.Role == role {
			return true
		}
	}
	return false
}
func (w *World) spawn(role, stage string, origin, target, source int64, route []Point, cargo, reserved int) bool {
	if len(route) == 0 || w.NextID >= MaxTick {
		return false
	}
	w.Walkers = append(w.Walkers, Walker{ID: w.NextID, X: route[0].X, Y: route[0].Y, Role: role, Stage: stage, Origin: origin, Target: target, Source: source, Route: route, Next: 1, Cargo: cargo, Reserved: reserved})
	w.NextID++
	return true
}
func (w *World) produceAndConsume() {
	w.produceGoods()
	w.harvestFigs()
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind == "farm" && b.Damaged == "" {
			if w.phase() == 0 {
				b.Work = 0
			}
			if w.phase() >= 48 && w.phase() < 192 {
				b.Work = min(288, b.Work+b.Workers)
			}
			if w.phase() == 192 {
				amount := min(b.StockCapacity-b.Stock, b.Work/6)
				b.Stock += amount
				w.GrainProduced += int64(amount)
				straw := min(32-b.Straw, amount/2)
				b.Straw += straw
				w.StrawProduced += int64(straw)
				b.Work = 0
			}
		}
		if b.Kind == "hut" && b.Pop > 0 && w.Tick%ConsumptionInterval == 0 {
			amount := min(b.Stock, (b.Pop+4)/5)
			b.Stock -= amount
			w.GrainConsumed += int64(amount)
		}
	}
}

// Sellers traverse an entire road component and return, using deterministic DFS.
// This is a provisional roaming policy, not a recovered Pharaoh junction rule.
func (w World) patrol(start Point) []Point { return w.patrolFor(start, "seller") }

func (w World) patrolFor(start Point, role string) []Point {
	if w.blockedStreet(start, role) {
		return nil
	}
	roads := w.roadSet()
	visited := map[Point]bool{}
	route := []Point{start}
	var walk func(Point)
	walk = func(p Point) {
		visited[p] = true
		for _, d := range directions {
			next := Point{p.X + d.X, p.Y + d.Y}
			if roads[next] && !visited[next] && !w.blockedStreet(next, role) {
				route = append(route, next)
				walk(next)
				route = append(route, p)
			}
		}
	}
	walk(start)
	return route
}
func (w *World) dispatch() {
	// Migration has one party in flight. Other walkers do not block recruitment.
	hasMigrant := false
	for _, walker := range w.Walkers {
		if walker.Role == "immigrant" {
			hasMigrant = true
			break
		}
	}
	if !hasMigrant && !(w.BudgetEnabled && w.WageRate == 0) && w.Tick%w.migrationInterval() == 0 {
		// Fill new connected plots before adding residents to established homes.
		// This is an explicit prototype policy, not a recovered Pharaoh rule.
		for pass := 0; pass < 2 && !hasMigrant; pass++ {
			for _, b := range w.Buildings {
				if (pass == 0 && b.Pop != 0) || (pass == 1 && b.Pop == 0) {
					continue
				}
				if b.Kind == "hut" && b.Connected && b.Pop < b.Capacity {
					route := w.path([]Point{w.entry()}, b)
					if w.spawn("immigrant", "travel", 0, b.ID, 0, route, 0, 0) {
						hasMigrant = true
						break
					}
				}
			}
		}
	}
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if !b.Connected || b.Damaged != "" || b.Workers < b.WorkersNeeded {
			continue
		}
		switch b.Kind {
		case "farm":
			if b.Stock == 0 || w.active(b.ID, "cart") {
				continue
			}
			var targetID int64
			var best []Point
			amount := 0
			for _, target := range w.Buildings {
				if target.Kind != "granary" || target.Workers < target.WorkersNeeded {
					continue
				}
				free := w.ResourceFreeSpace(target.ID, "grain")
				if free <= 0 {
					continue
				}
				route := w.path(w.perimeter(*b), target)
				if len(route) > 0 && (best == nil || len(route) < len(best)) {
					best = route
					targetID = target.ID
					amount = min(16, min(b.Stock, free))
				}
			}
			if w.spawn("cart", "deliver", b.ID, targetID, targetID, best, amount, 0) {
				b.Stock -= amount
			}
		case "bazaar":
			if !w.active(b.ID, "buyer") {
				free := w.ResourceFreeSpace(b.ID, "grain")
				if free > 0 {
					var sourceID int64
					var best []Point
					amount := 0
					for _, source := range w.Buildings {
						if source.Kind != "granary" || source.Workers < source.WorkersNeeded {
							continue
						}
						available := source.Stock - w.reservedResource(source.ID, "grain")
						if available <= 0 {
							continue
						}
						route := w.path(w.perimeter(*b), source)
						if len(route) > 0 && (best == nil || len(route) < len(best)) {
							best = route
							sourceID = source.ID
							amount = min(16, min(available, free))
						}
					}
					w.spawn("buyer", "fetch", b.ID, sourceID, sourceID, best, 0, amount)
				}
			}
			if b.Stock > 0 && !w.active(b.ID, "seller") {
				entryPath := w.path([]Point{w.entry()}, *b)
				if len(entryPath) > 0 {
					route := w.patrol(entryPath[len(entryPath)-1])
					amount := min(32, b.Stock)
					if w.spawn("seller", "patrol", b.ID, b.ID, 0, route, amount, 0) {
						b.Stock -= amount
					}
				}
			}
		}
	}
	w.dispatchMaterials()
	w.dispatchPerformers()
	w.dispatchServices()
	w.dispatchFarmWorkers()
	w.dispatchBuilders()
	w.dispatchGold()
	w.dispatchTrader()
	w.dispatchGoods()
	w.dispatchHunters()
}
func reverseJourney(walker *Walker, target int64, stage string) {
	route := slices.Clone(walker.Route)
	slices.Reverse(route)
	walker.Route = route
	walker.Next = 1
	walker.X, walker.Y = route[0].X, route[0].Y
	walker.Target = target
	walker.Stage = stage
}
func (w *World) moveWalkers() {
	remaining := make([]Walker, 0, len(w.Walkers))
	for _, walker := range w.Walkers {
		w.respectRoadblock(&walker)
		if walker.Role == "trader" {
			if w.advanceTrader(&walker) {
				remaining = append(remaining, walker)
			}
			continue
		}
		if walker.Role == "farmworker" {
			if w.advanceFarmWorker(&walker) {
				remaining = append(remaining, walker)
			}
			continue
		}
		if walker.Role == "hunter" {
			if w.advanceHunter(&walker) {
				remaining = append(remaining, walker)
			}
			continue
		}
		if isGoodsRole(walker.Role) {
			if w.advanceGoods(&walker) {
				remaining = append(remaining, walker)
			}
			continue
		}
		if walker.Role == "builder" {
			if w.advanceBuilder(&walker) {
				remaining = append(remaining, walker)
			}
			continue
		}
		if walker.Next < len(walker.Route) {
			p := walker.Route[walker.Next]
			walker.X, walker.Y = p.X, p.Y
			walker.Next++
		}
		if (walker.Role == "firefighter" || walker.Role == "architect" || walker.Role == "taxcollector" || walker.Role == "watercarrier" || walker.Role == "priest" || walker.Role == "entertainer" || slices.Contains(amenities, walker.Role)) && walker.Stage == "patrol" {
			w.visitService(walker)
		}
		if walker.Role == "seller" && walker.Stage == "patrol" {
			position := Point{walker.X, walker.Y}
			for i := range w.Buildings {
				b := &w.Buildings[i]
				if b.Kind != "hut" || b.Pop == 0 || !b.roadAccess(position) {
					continue
				}
				b.MarketUntil = min(MaxTick, w.Tick+MarketDuration)
				// Maintain a reserve for a normal patrol round. Top up rather
				// than repeatedly favoring the first home on the route.
				amount := min(walker.Cargo, max(0, min(8, b.StockCapacity)-b.Stock))
				b.Stock += amount
				walker.Cargo -= amount
			}
		}
		if walker.Next < len(walker.Route) {
			remaining = append(remaining, walker)
			continue
		}
		target := w.building(walker.Target)
		switch walker.Role {
		case "performer":
			if school := w.building(walker.Origin); school != nil && target.Damaged == "" {
				if style := slices.Index(schools, school.Kind); style >= 0 {
					target.ShowUntil[style] = min(MaxTick, w.Tick+MarketDuration)
				}
			}
		case "immigrant":
			w.Arrivals++
			if target.Pop < target.Capacity {
				target.Pop++
			} else {
				w.Departures++
			}
		case "goldcart":
			if w.arriveGold(&walker, target) {
				remaining = append(remaining, walker)
			}
		case "cart":
			if walker.Stage == "deliver" {
				target.Stock += walker.Cargo
				walker.Cargo = 0
				reverseJourney(&walker, walker.Origin, "return")
				remaining = append(remaining, walker)
			}
		case "buyer":
			if walker.Stage == "fetch" {
				target.Stock -= walker.Reserved
				walker.Cargo = walker.Reserved
				walker.Reserved = 0
				reverseJourney(&walker, walker.Origin, "return")
				remaining = append(remaining, walker)
			} else {
				target.Stock += walker.Cargo
			}
		case "seller":
			target.Stock += walker.Cargo
		case "porter":
			if walker.Stage == "deliver" {
				*materialStock(target, walker.Resource) += walker.Cargo
				walker.Cargo = 0
				reverseJourney(&walker, walker.Origin, "return")
				remaining = append(remaining, walker)
			}
		case "firefighter", "architect", "taxcollector":
			// Patrol complete; the staffed source will dispatch the next walker.

		}
	}
	w.Walkers = remaining
}

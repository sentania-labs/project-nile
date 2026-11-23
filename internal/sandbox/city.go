package sandbox

import (
	"errors"
	"slices"
)

// Standard building prices follow the original Normal difficulty evidence.
// Mastaba pricing, wages, hazard intervals and recipes remain provisional.
type CatalogEntry struct {
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Cost     int64  `json:"cost"`
	Category string `json:"category"`
	Workers  int    `json:"workers"`
}

var buildingCatalog = []CatalogEntry{
	{"road", "Road", 1, 1, 4, "Infrastructure", 0},
	{"hut", "Housing plot", 1, 1, 10, "Housing", 0},
	{"well", "Well", 1, 1, 5, "Services", 0},
	{"farm", "Floodplain farm", 3, 3, 40, "Food", 2},
	{"granary", "Granary", 4, 4, 250, "Food", 1},
	{"bazaar", "Bazaar", 2, 2, 40, "Food", 2},
	{"claypit", "Clay pit", 2, 2, 40, "Industry", 2},
	{"brickworks", "Brickworks", 2, 2, 60, "Industry", 3},
	{"storageyard", "Storage yard", 3, 3, 70, "Industry", 1},
	{"mastaba", "Small mastaba", 3, 3, 400, "Monuments", 3},
	{"firehouse", "Firehouse", 1, 1, 30, "Services", 1},
	{"architect", "Architect's post", 1, 1, 30, "Services", 1},
	{"taxoffice", "Tax office", 2, 2, 75, "Government", 2},
}

func catalog(kind string) (CatalogEntry, bool) {
	for _, entry := range buildingCatalog {
		if entry.Kind == kind {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}
func constructionCost(kind string) int64 { entry, _ := catalog(kind); return entry.Cost }
func (w *World) spend(amount int64) {
	if w.BudgetEnabled {
		w.Treasury -= amount
		w.Expenses += amount
	}
}
func NewTown() World {
	return World{Version: SnapshotVersion, Roads: []Point{Entry}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1, Treasury: 2500, BudgetEnabled: true, HazardsEnabled: true, TaxRate: 2, WageRate: 1, LaborPriority: "balanced"}
}
func monumentStage(done int) string {
	switch {
	case done >= 64:
		return "complete"
	case done >= 48:
		return "finishing"
	case done >= 16:
		return "masonry"
	default:
		return "foundation"
	}
}
func materialStock(b *Building, resource string) *int {
	switch resource {
	case "clay":
		return &b.Clay
	case "straw":
		return &b.Straw
	case "bricks":
		return &b.Bricks
	}
	return nil
}
func materialCapacity(b Building, resource string) int {
	switch b.Kind {
	case "farm":
		if resource == "straw" {
			return 32
		}
	case "claypit":
		if resource == "clay" {
			return 32
		}
	case "brickworks":
		if resource == "clay" || resource == "straw" {
			return 24
		}
		if resource == "bricks" {
			return 32
		}
	case "storageyard":
		return 64
	case "mastaba":
		if resource == "bricks" {
			return max(0, 64-b.ConstructionDone)
		}
	}
	return 0
}
func (w World) materialReserved(id int64, resource string) int {
	n := 0
	for _, walker := range w.Walkers {
		if walker.Role == "porter" && walker.Stage == "deliver" && walker.Target == id && walker.Resource == resource {
			n += walker.Cargo
		}
	}
	return n
}
func (w World) materialTotal(resource string) int {
	n := 0
	for _, b := range w.Buildings {
		copy := b
		n += *materialStock(&copy, resource)
	}
	for _, walker := range w.Walkers {
		if walker.Resource == resource {
			n += walker.Cargo
		}
	}
	return n
}
func materialReceiver(source, target Building, resource string) bool {
	if source.ID == target.ID || target.Damaged != "" || target.Workers < target.WorkersNeeded {
		return false
	}
	if resource == "clay" || resource == "straw" {
		return target.Kind == "brickworks" || (source.Kind != "storageyard" && target.Kind == "storageyard")
	}
	return target.Kind == "mastaba" && target.ConstructionDone < 64 || (source.Kind != "storageyard" && target.Kind == "storageyard")
}
func (w *World) dispatchMaterials() {
	for i := range w.Buildings {
		source := &w.Buildings[i]
		if !source.Connected || source.Damaged != "" || source.Workers < source.WorkersNeeded || w.active(source.ID, "porter") {
			continue
		}
		if source.Kind != "farm" && source.Kind != "claypit" && source.Kind != "brickworks" && source.Kind != "storageyard" {
			continue
		}
		for _, resource := range []string{"bricks", "clay", "straw"} {
			available := *materialStock(source, resource)
			if available == 0 {
				continue
			}
			if source.Kind == "brickworks" && resource != "bricks" {
				continue
			}
			var best []Point
			targetID := int64(0)
			amount := 0
			bestPriority := 3
			for _, target := range w.Buildings {
				if !materialReceiver(*source, target, resource) {
					continue
				}
				copy := target
				free := materialCapacity(target, resource) - *materialStock(&copy, resource) - w.materialReserved(target.ID, resource)
				if free <= 0 {
					continue
				}
				path := w.path(w.perimeter(*source), target)
				priority := 0
				if target.Kind == "storageyard" {
					priority = 1
				}
				if len(path) > 0 && (best == nil || priority < bestPriority || (priority == bestPriority && len(path) < len(best))) {
					best = path
					bestPriority = priority
					targetID = target.ID
					amount = min(8, min(available, free))
				}
			}
			if w.spawn("porter", "deliver", source.ID, targetID, targetID, best, amount, 0) {
				w.Walkers[len(w.Walkers)-1].Resource = resource
				*materialStock(source, resource) -= amount
				break
			}
		}
	}
}
func (w *World) dispatchServices() {
	for _, b := range w.Buildings {
		role := ""
		switch b.Kind {
		case "firehouse":
			role = "firefighter"
		case "architect":
			role = "architect"
		case "taxoffice":
			role = "taxcollector"
		default:
			continue
		}
		if !b.Connected || b.Damaged != "" || b.Workers < b.WorkersNeeded || w.active(b.ID, role) {
			continue
		}
		entryPath := w.path([]Point{Entry}, b)
		if len(entryPath) > 0 {
			w.spawn(role, "patrol", b.ID, b.ID, 0, w.patrol(entryPath[len(entryPath)-1]), 0, 0)
		}
	}
}
func nearBuilding(p Point, b Building, reach int) bool {
	return p.X >= b.X-reach && p.X < b.X+b.Width+reach && p.Y >= b.Y-reach && p.Y < b.Y+b.Height+reach
}
func (w *World) visitService(walker Walker) {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if !nearBuilding(Point{walker.X, walker.Y}, *b, 2) {
			continue
		}
		switch walker.Role {
		case "firefighter":
			b.FireUntil = min(MaxTick, w.Tick+128)
			b.FireRisk = max(0, b.FireRisk-40)
		case "architect":
			b.CollapseUntil = min(MaxTick, w.Tick+128)
			b.CollapseRisk = max(0, b.CollapseRisk-40)
		case "taxcollector":
			if b.Kind == "hut" && b.Pop > 0 {
				b.TaxUntil = min(MaxTick, w.Tick+128)
			}
		}
	}
}
func (w *World) cityTick() {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Damaged != "" {
			continue
		}
		if b.Workers >= b.WorkersNeeded && b.Connected {
			if b.Kind == "claypit" && w.Tick%24 == 0 {
				amount := min(4, 32-b.Clay)
				b.Clay += amount
				w.ClayProduced += int64(amount)
			}
			if b.Kind == "brickworks" && w.Tick%24 == 0 && b.Clay >= 2 && b.Straw >= 1 && b.Bricks <= 30 {
				b.Clay -= 2
				b.Straw--
				b.Bricks += 2
				w.ClayConsumed += 2
				w.StrawConsumed++
				w.BricksProduced += 2
			}

		}
		crew := min(b.CrewOnSite, b.Workers)
		if b.Kind == "mastaba" && crew > 0 && b.Connected && w.Tick%int64(24/crew) == 0 && b.Bricks > 0 && b.ConstructionDone < 64 {
			b.Bricks--
			b.ConstructionDone++
			w.BricksConsumed++
		}
		if w.HazardsEnabled && b.Kind != "well" && !(b.Kind == "hut" && b.Pop == 0) {
			if b.Kind != "mastaba" && w.Tick%16 == b.ID%16 && b.FireUntil <= w.Tick {
				b.FireRisk = min(100, b.FireRisk+1)
			}
			if w.Tick%24 == b.ID%24 && b.CollapseUntil <= w.Tick {
				b.CollapseRisk = min(100, b.CollapseRisk+1)
			}
			if b.FireRisk >= 100 {
				b.Damaged = "fire"
				w.FireCount++
			} else if b.CollapseRisk >= 100 {
				b.Damaged = "collapse"
				w.CollapseCount++
			}
		}
	}
	if w.BudgetEnabled && w.Tick%48 == 0 {
		w.LastIncome, w.LastWages = 0, 0
		for i := range w.Buildings {
			b := &w.Buildings[i]
			b.LastTax = 0
			if b.Kind == "hut" && b.TaxUntil > w.Tick {
				b.LastTax = int64(b.Pop * w.TaxRate)
				w.LastIncome += b.LastTax
			}
			w.LastWages += int64(b.Workers * w.WageRate)
		}
		w.Treasury += w.LastIncome - w.LastWages
		w.Income += w.LastIncome
		w.Expenses += w.LastWages
	}
}
func (w *World) Repair(x, y int) error {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if !b.contains(Point{x, y}) {
			continue
		}
		if b.Damaged == "" {
			return errors.New("this building does not need repairs")
		}
		cost := max(int64(5), constructionCost(b.Kind)/2)
		if w.BudgetEnabled && w.Treasury < cost {
			return errors.New("not enough treasury for repairs")
		}
		w.spend(cost)
		b.Damaged = ""
		b.FireRisk = 0
		b.CollapseRisk = 0
		w.refresh()
		return nil
	}
	return errors.New("select a damaged building")
}
func (w World) cityView(v *View) {
	v.Treasury, v.Income, v.Expenses = w.Treasury, w.Income, w.Expenses
	v.LastIncome, v.LastWages = w.LastIncome, w.LastWages
	v.FireCount, v.CollapseCount = w.FireCount, w.CollapseCount
	v.Clay, v.Straw, v.Bricks = w.materialTotal("clay"), w.materialTotal("straw"), w.materialTotal("bricks")
	v.ClayProduced, v.ClayConsumed = w.ClayProduced, w.ClayConsumed
	v.StrawProduced, v.StrawConsumed = w.StrawProduced, w.StrawConsumed
	v.BricksProduced, v.BricksConsumed = w.BricksProduced, w.BricksConsumed
	for _, b := range w.Buildings {
		if b.Kind == "mastaba" {
			v.MonumentProgress += b.ConstructionDone
			v.MonumentGoal += 64
		}
	}
	v.BuildingCatalog = buildingCatalog
	v.TaxRate, v.WageRate, v.LaborPriority = w.TaxRate, w.WageRate, w.LaborPriority
}

func (w *World) enableCity() {
	w.Version = SnapshotVersion
	w.BudgetEnabled = true
	w.HazardsEnabled = true
	w.Treasury = 2500
	w.policyDefaults()
	w.refresh()
}
func (w World) validateCity() error {
	if !validPolicy(w.TaxRate, w.WageRate, w.LaborPriority) {
		return errors.New("invalid fiscal or labor policy")
	}
	if w.Income < 0 || w.Expenses < 0 || w.LastIncome < 0 || w.LastWages < 0 || w.FireCount < 0 || w.CollapseCount < 0 {
		return errors.New("invalid city accounts")
	}
	if w.BudgetEnabled && w.Treasury != 2500+w.Income-w.Expenses {
		return errors.New("treasury conservation failed")
	}
	for _, resource := range []string{"clay", "straw", "bricks"} {
		produced, consumed := w.ClayProduced, w.ClayConsumed
		if resource == "straw" {
			produced, consumed = w.StrawProduced, w.StrawConsumed
		} else if resource == "bricks" {
			produced, consumed = w.BricksProduced, w.BricksConsumed
		}
		if produced < 0 || consumed < 0 || int64(w.materialTotal(resource))+consumed != produced {
			return errors.New("material conservation failed: " + resource)
		}
		for _, b := range w.Buildings {
			copy := b
			stock := *materialStock(&copy, resource)
			if stock < 0 || stock+w.materialReserved(b.ID, resource) > materialCapacity(b, resource) {
				return errors.New("material capacity exceeded: " + resource)
			}
		}
	}
	for _, b := range w.Buildings {
		if b.LastTax < 0 || b.FireRisk < 0 || b.FireRisk > 100 || b.CollapseRisk < 0 || b.CollapseRisk > 100 || b.FireUntil < 0 || b.CollapseUntil < 0 || b.TaxUntil < 0 || b.ConstructionDone < 0 || b.ConstructionDone > 64 || (b.Kind != "mastaba" && b.ConstructionDone != 0) {
			return errors.New("invalid civic building state")
		}
		if b.Damaged != "" && b.Damaged != "fire" && b.Damaged != "collapse" {
			return errors.New("unknown damage state")
		}
	}
	return nil
}
func (w World) validateCityWalker(walker Walker, target Building) error {
	origin := w.building(walker.Origin)
	if origin == nil || walker.Reserved != 0 {
		return errors.New("invalid civic walker origin")
	}
	if walker.Role == "builder" {
		return w.validateBuilder(walker, target, *origin)
	}
	if walker.Role == "porter" {
		source := w.building(walker.Source)
		if source == nil || walker.Cargo > 8 || (walker.Resource != "clay" && walker.Resource != "straw" && walker.Resource != "bricks") {
			return errors.New("invalid material porter")
		}
		if walker.Stage == "deliver" {
			if target.ID != source.ID || walker.Cargo == 0 || !origin.adjacent(walker.Route[0]) {
				return errors.New("invalid material delivery")
			}
		} else if walker.Stage == "return" {
			if target.ID != origin.ID || walker.Cargo != 0 || !source.adjacent(walker.Route[0]) {
				return errors.New("invalid porter return")
			}
		} else {
			return errors.New("invalid porter stage")
		}
		return nil
	}
	wantKind := "firehouse"
	if walker.Role == "architect" {
		wantKind = "architect"
	} else if walker.Role == "taxcollector" {
		wantKind = "taxoffice"
	}
	if origin.Kind != wantKind || target.ID != origin.ID || walker.Cargo != 0 || walker.Resource != "" || walker.Source != 0 || (walker.Stage != "patrol" && walker.Stage != "return") || (walker.Stage == "patrol" && walker.Route[0] != walker.Route[len(walker.Route)-1]) {
		return errors.New("invalid service patrol")
	}
	return nil
}

func isPatrol(role string) bool {
	return role == "seller" || role == "firefighter" || role == "architect" || role == "taxcollector"
}
func (w *World) clearRoad(index int, p Point) error {
	candidate := *w
	candidate.Roads = slices.Delete(slices.Clone(w.Roads), index, index+1)
	candidate.Buildings = slices.Clone(w.Buildings)
	candidate.Walkers = slices.Clone(w.Walkers)
	for i := range candidate.Walkers {
		walker := &candidate.Walkers[i]
		if !slices.Contains(walker.Route, p) {
			continue
		}
		if !isPatrol(walker.Role) || p == (Point{walker.X, walker.Y}) {
			return errors.New("wait until the walker or delivery has passed this road")
		}
		origin := candidate.building(walker.Origin)
		route := candidate.path([]Point{{walker.X, walker.Y}}, *origin)
		if len(route) == 0 {
			return errors.New("this road would strand a service walker")
		}
		walker.Route = route
		walker.Next = 1
		walker.Stage = "return"
	}
	candidate.refresh()
	if err := candidate.Validate(); err != nil {
		return err
	}
	*w = candidate
	return nil
}

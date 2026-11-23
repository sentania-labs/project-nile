package sandbox

import (
	"errors"
	"slices"
)

type goodsRecipe struct {
	Input, Output             string
	InputAmount, OutputAmount int
}

// Dependencies and producer-to-manufacturer preference follow the manual.
// Batch sizes, employee-tick durations and finite buffers are provisional.
var goodsRecipes = map[string]goodsRecipe{
	"brewery": {"barley", "beer", 100, 100}, "papyrusmaker": {"reeds", "papyrus", 100, 100},
	"potter": {"clay", "pottery", 8, 100}, "weaver": {"flax", "linen", 100, 100},
}

func (w *World) addGoodsTerrain() {
	if w.ReedBeds != nil {
		return
	}
	w.ReedBeds = []Point{}
	occupied := w.occupied()
	for y := 25; y <= 29; y++ {
		for x := 6; x <= 8; x++ {
			p := Point{x, y}
			if !occupied[p] {
				w.ReedBeds = append(w.ReedBeds, p)
			}
		}
	}
}
func (w World) goodsPlacement(b Building) error {
	if b.Kind == "barleyfarm" || b.Kind == "flaxfarm" {
		for _, p := range b.tiles() {
			if p.X > 9 {
				return errors.New("this farm must fit on the floodplain")
			}
		}
	}
	if b.Kind == "reedgatherer" && len(w.ReedBeds) == 0 {
		return errors.New("no harvestable marsh reeds in this region")
	}
	return nil
}
func (w *World) produceGoods() {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind == "hut" && b.Pop > 0 && w.Tick%ConsumptionInterval == 0 {
			// Grain remains the preferred food for legacy households. Meat supplies
			// remaining demand, without relabelling hunted food as grain.
			need := max(0, (b.Pop+4)/5-b.Stock)
			need -= w.ConsumeResource(b.ID, "meat", need)
			w.ConsumeResource(b.ID, "figs", need)
			for _, r := range []string{"pottery", "beer", "linen", "jewelry"} {
				if houseWantsResource(*b, r) {
					w.ConsumeResource(b.ID, r, max(1, b.Pop/10))
				}
			}
		}
		if b.Kind == "barleyfarm" || b.Kind == "flaxfarm" {
			if w.phase() == 0 {
				b.Work = 0
			}
			if b.Damaged == "" && b.Connected && b.Workers > 0 && w.phase() >= 48 && w.phase() < 192 {
				b.Work = min(1440, b.Work+b.Workers)
			}
			if w.phase() == 192 {
				r := "barley"
				if b.Kind == "flaxfarm" {
					r = "flax"
				}
				if b.Damaged == "" {
					w.ProduceResource(b.ID, r, b.Work/12)
				}
				b.Work = 0
			}
			continue
		}
		if b.Damaged != "" || !b.Connected || b.Workers == 0 {
			continue
		}
		if recipe, ok := goodsRecipes[b.Kind]; ok {
			if inventory(*b, recipe.Input) < recipe.InputAmount || w.ResourceFreeSpace(b.ID, recipe.Output) < recipe.OutputAmount {
				continue
			}
			b.Work = min(576, b.Work+b.Workers)
			if b.Work >= 576 {
				w.ConsumeResource(b.ID, recipe.Input, recipe.InputAmount)
				w.ProduceResource(b.ID, recipe.Output, recipe.OutputAmount)
				b.Work = 0
			}
		}
	}
}
func (w World) activeResource(origin int64, role, r string) bool {
	for _, a := range w.Walkers {
		if a.Origin == origin && a.Role == role && a.Resource == r {
			return true
		}
	}
	return false
}
func outputResource(kind string) string {
	switch kind {
	case "barleyfarm":
		return "barley"
	case "flaxfarm":
		return "flax"
	case "reedgatherer":
		return "reeds"
	case "huntinglodge":
		return "meat"
	case "figfarm":
		return "figs"
	}
	if r, ok := goodsRecipes[kind]; ok {
		return r.Output
	}
	return ""
}
func goodsOperating(b Building) bool { return b.Connected && b.Damaged == "" && b.Workers > 0 }
func (w World) freightReceiver(source, target Building, r string) (bool, int) {
	if source.ID == target.ID || !goodsOperating(target) {
		return false, 0
	}
	if supplyForService(target.Kind) == r {
		return true, 0
	}
	if recipe, ok := goodsRecipes[target.Kind]; ok && recipe.Input == r {
		return true, 0
	}
	if target.Kind == "granary" && foodResource(r) {
		return true, 0
	}
	if target.Kind == "storageyard" && storageAccepts(target, r) {
		if source.Kind != "storageyard" || storageOrder(source, r) == "empty" {
			return true, 1
		}
	}
	return false, 0
}
func (w *World) dispatchGoods() {
	for i := range w.Buildings {
		source := &w.Buildings[i]
		if !goodsOperating(*source) {
			continue
		}
		output := outputResource(source.Kind)
		if (output != "" || source.Kind == "storageyard") && !w.active(source.ID, "freight") {
			for offset := 0; offset < len(resourceCatalog); offset++ {
				r := resourceCatalog[(int(w.Tick/8)+i+offset)%len(resourceCatalog)].ID
				if output != "" && r != output {
					continue
				}
				// Legacy material porters retain clay/straw/bricks routing. Empty orders
				// are handled here, with exact reservations shared by both mechanisms.
				if source.Kind == "storageyard" && (r == "clay" || r == "straw" || r == "bricks") && storageOrder(*source, r) != "empty" {
					continue
				}
				available := inventory(*source, r) - w.reservedResource(source.ID, r)
				if available <= 0 {
					continue
				}
				var route []Point
				targetID := int64(0)
				amount, priority := 0, 3
				for _, target := range w.Buildings {
					ok, p := w.freightReceiver(*source, target, r)
					if !ok {
						continue
					}
					free := w.ResourceFreeSpace(target.ID, r)
					if free == 0 {
						continue
					}
					path := w.path(w.perimeter(*source), target)
					if len(path) > 0 && (route == nil || p < priority || (p == priority && len(path) < len(route))) {
						route = path
						targetID = target.ID
						priority = p
						amount = min(100, available, free)
					}
				}
				if w.spawn("freight", "deliver", source.ID, targetID, targetID, route, amount, 0) {
					w.Walkers[len(w.Walkers)-1].Resource = r
					setInventory(source, r, inventory(*source, r)-amount)
					break
				}
			}
		}
		if source.Kind == "storageyard" || source.Kind == "bazaar" {
			w.dispatchGoodsBuyers(*source)
		}
		if source.Kind == "bazaar" {
			for _, r := range []string{"meat", "figs", "pottery", "beer", "linen", "jewelry"} {
				if inventory(*source, r) == 0 || w.activeResource(source.ID, "goodsseller", r) {
					continue
				}
				entry := w.path([]Point{w.entry()}, *source)
				if len(entry) == 0 {
					continue
				}
				amount := min(100, inventory(*source, r))
				if w.spawn("goodsseller", "patrol", source.ID, source.ID, 0, w.patrol(entry[len(entry)-1]), amount, 0) {
					w.Walkers[len(w.Walkers)-1].Resource = r
					setInventory(source, r, inventory(*source, r)-amount)
				}
			}
		}
		if source.Kind == "reedgatherer" && !w.active(source.ID, "reedharvester") && w.ResourceFreeSpace(source.ID, "reeds") >= 100 {
			route := w.landPath(w.perimeter(*source), w.ReedBeds)
			if w.spawn("reedharvester", "gather", source.ID, source.ID, source.ID, route, 0, 100) {
				w.Walkers[len(w.Walkers)-1].Resource = "reeds"
			}
		}
	}
}
func (w *World) dispatchGoodsBuyers(receiver Building) {
	for _, info := range resourceCatalog {
		r := info.ID
		if receiver.Kind == "storageyard" && storageOrder(receiver, r) != "get" {
			continue
		}
		if receiver.Kind == "bazaar" && !foodResource(r) && !householdGood(r) {
			continue
		}
		if w.activeResource(receiver.ID, "goodsbuyer", r) {
			continue
		}
		free := w.ResourceFreeSpace(receiver.ID, r)
		if free <= 0 {
			continue
		}
		var best []Point
		sourceID := int64(0)
		amount := 0
		for _, source := range w.Buildings {
			if source.ID == receiver.ID || !goodsOperating(source) {
				continue
			}
			if receiver.Kind == "storageyard" {
				if source.Kind != "storageyard" || storageOrder(source, r) == "get" {
					continue
				}
			} else {
				if r != "grain" && foodResource(r) {
					if source.Kind != "granary" && source.Kind != "storageyard" {
						continue
					}
				} else if source.Kind != "storageyard" {
					continue
				}
			}
			available := inventory(source, r) - w.reservedResource(source.ID, r)
			if available <= 0 {
				continue
			}
			path := w.path(w.perimeter(receiver), source)
			if len(path) > 0 && (best == nil || len(path) < len(best)) {
				best = path
				sourceID = source.ID
				amount = min(100, free, available)
			}
		}
		if w.spawn("goodsbuyer", "fetch", receiver.ID, sourceID, sourceID, best, 0, amount) {
			w.Walkers[len(w.Walkers)-1].Resource = r
		}
	}
}
func isGoodsRole(role string) bool {
	return role == "freight" || role == "goodsbuyer" || role == "goodsseller" || role == "reedharvester"
}
func (w *World) advanceGoods(a *Walker) bool {
	if a.Next < len(a.Route) {
		p := a.Route[a.Next]
		a.X, a.Y = p.X, p.Y
		a.Next++
	}
	if a.Role == "goodsseller" && a.Stage == "patrol" {
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if b.Kind != "hut" || b.Pop == 0 || !houseWantsResource(*b, a.Resource) || !b.roadAccess(Point{a.X, a.Y}) {
				continue
			}
			n := min(a.Cargo, max(0, 16-inventory(*b, a.Resource)))
			setInventory(b, a.Resource, inventory(*b, a.Resource)+n)
			a.Cargo -= n
			b.MarketUntil = min(MaxTick, w.Tick+MarketDuration)
		}
	}
	if a.Next < len(a.Route) {
		return true
	}
	target := w.building(a.Target)
	switch a.Role {
	case "freight":
		if a.Stage == "return" {
			return false
		}
		n := w.DepositResource(target.ID, a.Resource, a.Cargo)
		a.Cargo -= n
		if a.Cargo > 0 {
			return true
		}
		reverseJourney(a, a.Origin, "return")
		return true
	case "goodsbuyer":
		if a.Stage == "fetch" {
			n := min(a.Reserved, inventory(*target, a.Resource))
			setInventory(target, a.Resource, inventory(*target, a.Resource)-n)
			a.Cargo = n
			a.Reserved = 0
			reverseJourney(a, a.Origin, "return")
			return true
		}
		n := w.DepositResource(target.ID, a.Resource, a.Cargo)
		a.Cargo -= n
		return a.Cargo > 0
	case "goodsseller":
		n := w.DepositResource(target.ID, a.Resource, a.Cargo)
		a.Cargo -= n
		return a.Cargo > 0
	case "reedharvester":
		if a.Stage == "gather" {
			a.Cargo = w.RecordResourceProduction("reeds", a.Reserved)
			a.Reserved = 0
			reverseJourney(a, a.Origin, "return")
			return true
		}
		n := w.DepositResource(target.ID, a.Resource, a.Cargo)
		a.Cargo -= n
		return a.Cargo > 0
	}
	return false
}
func (w World) validateGoodsWalker(a Walker, target Building) error {
	origin := w.building(a.Origin)
	source := w.building(a.Source)
	if origin == nil || !validResource(a.Resource) || a.Cargo > 100 || a.Reserved > 100 {
		return errors.New("invalid goods walker")
	}
	switch a.Role {
	case "freight":
		if source == nil || a.Reserved != 0 || (origin.Kind != "storageyard" && outputResource(origin.Kind) != a.Resource) {
			return errors.New("invalid freight source")
		}
		if a.Stage == "deliver" {
			if target.ID != source.ID || a.Cargo == 0 || !origin.roadAccess(a.Route[0]) {
				return errors.New("invalid freight delivery")
			}
		} else if a.Stage == "return" {
			if target.ID != origin.ID || a.Cargo != 0 || !source.roadAccess(a.Route[0]) {
				return errors.New("invalid freight return")
			}
		} else {
			return errors.New("invalid freight stage")
		}
	case "goodsbuyer":
		if source == nil || (origin.Kind != "bazaar" && origin.Kind != "storageyard") {
			return errors.New("invalid resource buyer")
		}
		if origin.Kind == "storageyard" && source.Kind != "storageyard" {
			return errors.New("storage buyer must fetch from another storage yard")
		}
		if origin.Kind == "bazaar" && (source.Kind != "storageyard" && !(foodResource(a.Resource) && source.Kind == "granary")) {
			return errors.New("bazaar buyer must fetch from storage")
		}
		if a.Stage == "fetch" {
			if target.ID != source.ID || a.Cargo != 0 || a.Reserved == 0 || !origin.roadAccess(a.Route[0]) {
				return errors.New("invalid resource reservation")
			}
		} else if a.Stage == "return" {
			if target.ID != origin.ID || a.Reserved != 0 || a.Cargo == 0 || !source.roadAccess(a.Route[0]) {
				return errors.New("invalid resource buyer return")
			}
		} else {
			return errors.New("invalid resource buyer stage")
		}
	case "goodsseller":
		if origin.Kind != "bazaar" || target.ID != origin.ID || a.Reserved != 0 || a.Source != 0 || (a.Stage != "patrol" && a.Stage != "return") {
			return errors.New("invalid goods seller")
		}
	case "reedharvester":
		if origin.Kind != "reedgatherer" || target.ID != origin.ID || a.Source != origin.ID || a.Resource != "reeds" {
			return errors.New("invalid reed gatherer")
		}
		if a.Stage == "gather" {
			if a.Cargo != 0 || a.Reserved != 100 || !origin.roadAccess(a.Route[0]) || !slices.Contains(w.ReedBeds, a.Route[len(a.Route)-1]) {
				return errors.New("invalid reed harvest trip")
			}
		} else if a.Stage == "return" {
			if a.Cargo != 100 || a.Reserved != 0 || !slices.Contains(w.ReedBeds, a.Route[0]) {
				return errors.New("invalid reed harvest return")
			}
		} else {
			return errors.New("invalid reed harvest stage")
		}
	}
	return nil
}

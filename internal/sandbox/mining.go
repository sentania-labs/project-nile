package sandbox

import (
	"errors"
)

// Normal model rows 161/84 establish construction costs and 12/20 employees.
// Manual pp.70-75 and 101-104 establish adjacent metallic rock, Palace-only
// delivery, groundwater and tax dependency. Production work, load size,
// buffering and the minimum operating crew are provisional prototype tuning.
const GoldMineCapacity = 200
const GoldLoad = 100
const GoldWork = 576 // 48 ticks with all 12 employees.

func workCapacity(kind string) int {
	if extracted[kind] != "" || kind == "jeweler" {
		return 576
	}
	if campCrops[kind] != "" {
		return FigSeasonWork
	}
	switch kind {
	case "shipwright":
		return 800
	case "farm":
		return 288
	case "coppermine":
		return 480
	case "recruiter":
		return 160
	case "goldmine":
		return GoldWork
	case "barleyfarm", "flaxfarm", "figfarm":
		return 1440
	case "chariotmaker", "cattleranch", "weaponsmith", "brewery", "potter", "weaver", "papyrusmaker":
		return 576
	}
	return 0
}
func walkerCapacity(role string) int {
	if role == "stonehauler" || role == "builder" {
		return 400
	}
	if role == "trader" || role == "goldcart" || isGoodsRole(role) || role == "hunter" {
		return GoldLoad
	}
	return 32
}

func (w *World) addResourceTerrain() {
	w.addGoodsTerrain()
	w.addHuntingGrounds()
	if w.Grassland == nil {
		w.Grassland = []Point{}
		for y := 0; y < Size; y++ {
			for x := 10; x <= 14; x++ {
				w.Grassland = append(w.Grassland, Point{x, y})
			}
		}
	}
	if w.OreRock == nil {
		w.OreRock = []Point{}
		occupied := w.occupied()
		for y := 3; y <= 7; y++ {
			for x := 30; x <= 31; x++ {
				p := Point{x, y}
				if !occupied[p] {
					w.OreRock = append(w.OreRock, p)
				}
			}
		}
	}
}
func (w World) hasPalace() bool {
	for _, b := range w.Buildings {
		if b.Kind == "palace" {
			return true
		}
	}
	return false
}
func (w World) palaceActive() bool {
	for _, b := range w.Buildings {
		if b.Kind == "palace" && b.Connected && b.Damaged == "" && b.Workers > 0 {
			return true
		}
	}
	return false
}
func (w World) taxesEnabled() bool { return w.LegacyTaxes || w.palaceActive() }
func (w World) mineTouchesOre(b Building) bool {
	for _, p := range w.OreRock {
		if b.adjacent(p) {
			return true
		}
	}
	return false
}
func (w World) palaceGroundwater(b Building) bool {
	for _, p := range w.Grassland {
		if b.contains(p) {
			return true
		}
	}
	return false
}
func (w World) resourcePlacement(b Building) error {
	if b.Kind == "goldmine" && !w.mineTouchesOre(b) {
		return errors.New("place a gold mine beside exposed metallic rock")
	}
	if b.Kind == "palace" {
		if w.hasPalace() {
			return errors.New("the city can have only one Palace")
		}
		if !w.palaceGroundwater(b) {
			return errors.New("at least one Palace tile must be on grassland for groundwater")
		}
	}
	return nil
}
func (w *World) mineGold() {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind != "goldmine" || b.Damaged != "" || !b.Connected || b.Workers == 0 || b.Gold >= GoldMineCapacity {
			continue
		}
		b.Work = min(GoldWork, b.Work+b.Workers)
		if b.Work >= GoldWork && b.Gold <= GoldMineCapacity-GoldLoad {
			b.Gold += GoldLoad
			w.GoldProduced += GoldLoad
			b.Work -= GoldWork
		}
	}
}
func (w *World) dispatchGold() {
	for i := range w.Buildings {
		mine := &w.Buildings[i]
		if mine.Kind != "goldmine" || mine.Gold < GoldLoad || mine.Workers == 0 || !mine.Connected || mine.Damaged != "" || w.active(mine.ID, "goldcart") {
			continue
		}
		for _, palace := range w.Buildings {
			if palace.Kind != "palace" || palace.Workers == 0 || !palace.Connected || palace.Damaged != "" {
				continue
			}
			path := w.path(w.perimeter(*mine), palace)
			if w.spawn("goldcart", "deliver", mine.ID, palace.ID, palace.ID, path, GoldLoad, 0) {
				w.Walkers[len(w.Walkers)-1].Resource = "gold"
				mine.Gold -= GoldLoad
			}
			break
		}
	}
}
func (w *World) arriveGold(walker *Walker, target *Building) bool {
	if walker.Stage == "return" {
		return false
	}
	// A destination may lose labor or suffer damage while a cart is travelling.
	// Its cargo waits at the destination and is credited exactly once on receipt.
	if !target.Connected || target.Damaged != "" || target.Workers == 0 {
		return true
	}
	amount := int64(walker.Cargo)
	w.GoldDelivered += amount
	w.Income += amount
	w.Treasury += amount
	walker.Cargo = 0
	reverseJourney(walker, walker.Origin, "return")
	return true
}
func (w World) miningView(v *View) {
	v.Grassland = append([]Point{}, w.Grassland...)
	v.WestBanks, v.Woodland, v.QuarryRock = w.WestBanks, w.Woodland, w.QuarryRock
	v.OreRock = append([]Point{}, w.OreRock...)
	v.GoldProduced, v.GoldDelivered = w.GoldProduced, w.GoldDelivered
	v.TaxIncome = w.Income - w.GoldDelivered - w.ExportEarned
	v.PalaceActive = w.palaceActive()
	v.TaxesEnabled = w.taxesEnabled()
	for _, b := range w.Buildings {
		v.Gold += b.Gold
	}
	for _, walker := range w.Walkers {
		if walker.Resource == "gold" {
			v.Gold += walker.Cargo
			v.GoldInTransit += walker.Cargo
		}
	}
}
func (w World) validateMining() error {
	ore := map[Point]bool{}
	for _, p := range w.OreRock {
		if !w.buildable(p) || ore[p] {
			return errors.New("invalid ore terrain")
		}
		ore[p] = true
	}
	grass := map[Point]bool{}
	for _, p := range w.Grassland {
		if !w.buildable(p) || grass[p] || ore[p] {
			return errors.New("invalid grassland terrain")
		}
		grass[p] = true
	}
	palaces, remaining := 0, int64(0)
	for _, b := range w.Buildings {
		remaining += int64(b.Gold)
		if b.Kind == "goldmine" {
			if !w.mineTouchesOre(b) || b.Gold < 0 || b.Gold > GoldMineCapacity {
				return errors.New("invalid gold mine")
			}
		} else if b.Gold != 0 {
			return errors.New("gold stored outside mine")
		}
		if b.Kind == "palace" {
			palaces++
			if !w.palaceGroundwater(b) {
				return errors.New("Palace lacks groundwater")
			}
		}
	}
	for _, walker := range w.Walkers {
		if walker.Resource == "gold" {
			remaining += int64(walker.Cargo)
		}
	}
	if palaces > 1 || w.GoldProduced < 0 || w.GoldDelivered < 0 || w.GoldDelivered > w.Income || w.GoldProduced != remaining+w.GoldDelivered {
		return errors.New("gold conservation or Palace uniqueness failed")
	}
	return nil
}
func (w World) validateGoldWalker(walker Walker, target Building) error {
	mine, palace := w.building(walker.Origin), w.building(walker.Source)
	if mine == nil || mine.Kind != "goldmine" || palace == nil || palace.Kind != "palace" || walker.Resource != "gold" || walker.Reserved != 0 {
		return errors.New("invalid gold cart")
	}
	if walker.Stage == "deliver" {
		if target.ID != palace.ID || walker.Cargo != GoldLoad || !mine.adjacent(walker.Route[0]) {
			return errors.New("invalid gold delivery")
		}
	} else if walker.Stage == "return" {
		if target.ID != mine.ID || walker.Cargo != 0 || !palace.adjacent(walker.Route[0]) {
			return errors.New("invalid gold cart return")
		}
	} else {
		return errors.New("invalid gold cart stage")
	}
	return nil
}

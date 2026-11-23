package sandbox

import "slices"

// Normal model for Common Residence through Palatial Estate. Tax-paying
// households expand at levels 11, 15 and 19 (levels here use zero-based tiers).
type estateRule struct {
	name                                                                                 string
	up, down, capacity, side, entertainment, gods, schools, food, health, linen, jewelry int
	dentist                                                                              bool
}

var estateRules = []estateRule{
	{"Common Residence", 32, 29, 80, 2, 30, 1, 1, 1, 1, 0, 0, false},
	{"Spacious Residence", 40, 36, 84, 2, 35, 1, 0, 2, 1, 0, 0, true},
	{"Elegant Residence", 48, 43, 88, 2, 40, 1, 1, 2, 1, 1, 0, true},
	{"Fancy Residence", 53, 48, 92, 2, 45, 2, 1, 2, 1, 1, 0, true},
	{"Common Manor", 58, 50, 100, 3, 50, 2, 1, 2, 1, 1, 1, true},
	{"Spacious Manor", 63, 53, 108, 3, 55, 2, 1, 2, 2, 1, 1, true},
	{"Elegant Manor", 68, 60, 116, 3, 60, 3, 2, 2, 2, 1, 1, true},
	{"Stately Manor", 74, 68, 124, 3, 70, 3, 2, 2, 2, 1, 1, true},
	{"Modest Estate", 80, 70, 184, 4, 80, 3, 2, 2, 2, 1, 2, true},
	{"Palatial Estate", 90, 85, 200, 4, 90, 3, 2, 3, 2, 1, 2, true},
}

func (b Building) lotSide() int { return max(1, b.LotSide) }
func (b Building) footprintDimensions() (int, int) {
	if b.Kind == "hut" {
		return b.lotSide(), b.lotSide()
	}
	return dimensions(b.Kind)
}

func (w World) refreshEstate(b *Building, previous, score, health int) {
	gods, education, food := 0, 0, 0
	for _, end := range b.GodUntil {
		if end > w.Tick {
			gods++
		}
	}
	if gods == 0 && b.ShrineAccess {
		gods = 1
	}
	for _, i := range []int{4, 5} {
		if b.ServiceUntil[i] > w.Tick {
			education++
		}
	}
	for _, r := range foodKinds {
		if inventory(*b, r) > 0 {
			food++
		}
	}
	for i, r := range estateRules {
		tier := i + 10
		threshold := r.up
		if previous >= tier {
			threshold = r.down
		}
		if b.Desirability < threshold || score < r.entertainment || gods < r.gods || education < r.schools || food < r.food || health < r.health || inventory(*b, "linen") < r.linen || inventory(*b, "jewelry") < r.jewelry || r.dentist && b.ServiceUntil[2] <= w.Tick {
			if previous > tier {
				continue
			}
			break
		}
		if r.side > b.lotSide() {
			b.WantedSide = r.side
			break
		}
		b.Tier, b.TierName, b.Capacity = tier, r.name, r.capacity
	}
}

func (w World) houseInUse(id int64) bool {
	return slices.ContainsFunc(w.Walkers, func(a Walker) bool { return a.Target == id || a.Origin == id || a.Source == id })
}

func (w *World) growHouse(id int64) bool {
	house := w.building(id)
	if house == nil {
		return false
	}
	old := *house
	side := old.WantedSide
	if side <= old.lotSide() || side > 4 {
		return false
	}
	margin := side - old.lotSide()
	roads := w.roadSet()
	for _, offset := range []Point{{0, 0}, {-margin, 0}, {0, -margin}, {-margin, -margin}} {
		lot := Building{X: old.X + offset.X, Y: old.Y + offset.Y, Width: side, Height: side}
		tiles := lot.tiles()
		obstructed := slices.ContainsFunc(tiles, func(p Point) bool {
			return !w.buildable(p) || roads[p] || slices.Contains(w.Canals, p) || slices.Contains(w.OreRock, p) || slices.Contains(w.QuarryRock, p) || slices.Contains(w.Woodland, p) || slices.Contains(w.HuntingGrounds, p) || w.offRoadReserved(p)
		})
		if obstructed {
			continue
		}
		absorbed := map[int64]bool{}
		residents := old.Pop
		totals := map[string]int{}
		for _, other := range w.Buildings {
			if other.ID == id || !slices.ContainsFunc(tiles, other.contains) {
				continue
			}
			if ((other.Kind != "hut" && other.Kind != "garden") || !lot.contains(other.point()) || !lot.contains(Point{other.X + other.Width - 1, other.Y + other.Height - 1}) || w.houseInUse(other.ID) || other.Damaged != "") {
				obstructed = true
				break
			}
			absorbed[other.ID] = true
			residents += other.Pop
			for _, r := range resourceCatalog {
				totals[r.ID] += inventory(other, r.ID)
			}
		}
		maxPopulation := map[int]int{2: 80, 3: 100, 4: 184}
		if obstructed || residents > maxPopulation[side] || w.houseInUse(id) && lot.point() != old.point() {
			continue
		}
		house.X, house.Y, house.Width, house.Height, house.LotSide = lot.X, lot.Y, side, side, side
		house.Pop = residents
		for r, n := range totals {
			if n > 0 {
				setInventory(house, r, inventory(*house, r)+n)
			}
		}
		w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool { return absorbed[b.ID] })
		return true
	}
	return false
}

func houseSideForTier(tier int) int {
	if tier < 10 {
		return 1
	}
	return estateRules[tier-10].side
}

func (w *World) subdivideHouse(id int64) {
	b := w.building(id)
	if b == nil || b.Damaged != "" || w.houseInUse(id) {
		return
	}
	old := *b
	side := old.lotSide()
	if side <= houseSideForTier(old.Tier) || w.NextID+int64(2*side) > MaxTick {
		return
	}
	center := side - 1
	fringe := 2*side - 1
	tiles := side * side
	b.Width, b.Height, b.LotSide = center, center, center
	b.Goods = nil
	remainder := map[string]int{}
	b.Pop = old.Pop - fringe*(old.Pop/(fringe+1))
	for _, r := range resourceCatalog {
		n := inventory(old, r.ID)
		if n > 0 {
			centerStock := min(resourceLimit(*b, r.ID), n-fringe*(n/tiles))
			setInventory(b, r.ID, centerStock)
			remainder[r.ID] = n - centerStock - fringe*(n/tiles)
		}
	}
	for _, p := range old.tiles() {
		if p.X < old.X+center && p.Y < old.Y+center {
			continue
		}
		home := old
		home.ID, home.X, home.Y = w.NextID, p.X, p.Y
		w.NextID++
		home.Width, home.Height, home.LotSide = 1, 1, 0
		home.Tier = min(old.Tier, 9)
		home.Pop, home.Goods = old.Pop/(fringe+1), nil
		for _, r := range resourceCatalog {
			n := inventory(old, r.ID) / tiles
			if remainder[r.ID] > 0 {
				n++
				remainder[r.ID]--
			}
			if n > 0 || inventory(home, r.ID) > 0 {
				setInventory(&home, r.ID, n)
			}
		}
		w.Buildings = append(w.Buildings, home)
	}
}

func (w *World) resizeHousing() {
	var ready, shrink []int64
	for _, b := range w.Buildings {
		if b.Kind == "hut" && b.WantedSide > b.lotSide() {
			ready = append(ready, b.ID)
		} else if b.Kind == "hut" && b.lotSide() > houseSideForTier(b.Tier) {
			shrink = append(shrink, b.ID)
		}
	}
	for _, id := range shrink {
		w.subdivideHouse(id)
	}
	for _, id := range ready {
		w.growHouse(id)
	}
}

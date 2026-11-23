package sandbox

import "errors"

const maxCitySize = 228

// The legacy neighborhood has a straight bank and 32 tiles per side. Larger
// sites persist each row's dry-season shore so simulation and art agree.
func (w World) mapSize() int {
	if w.MapExtent == 0 {
		return Size
	}
	return w.MapExtent
}
func (w World) bank(y int) int {
	if y >= 0 && y < len(w.Banks) {
		return w.Banks[y]
	}
	return 6
}
func (w World) westBank(y int) int {
	if y >= 0 && y < len(w.WestBanks) {
		return w.WestBanks[y]
	}
	return 0
}

func (w World) navigableWater(p Point) bool {
	return p.Y >= 0 && p.Y < w.mapSize() && p.X >= w.westBank(p.Y) && p.X < w.bank(p.Y)
}

func (w World) buildable(p Point) bool {
	n := w.mapSize()
	return p.Y >= 0 && p.Y < n && p.X >= 0 && p.X < n && !w.navigableWater(p)
}
func (w World) floodplain(p Point) bool {
	return w.buildable(p) && p.X < w.bank(p.Y)+4 && p.X >= w.westBank(p.Y)-4
}

func (w World) validateGeography() error {
	if w.MapExtent != 0 && (w.MapExtent < Size || w.MapExtent > maxCitySize) {
		return errors.New("invalid city map size")
	}
	if len(w.Banks) != 0 && len(w.Banks) != w.mapSize() {
		return errors.New("incomplete riverbank")
	}
	if len(w.WestBanks) != 0 && len(w.WestBanks) != w.mapSize() {
		return errors.New("incomplete western shoreline")
	}
	for y, west := range w.WestBanks {
		if west < 0 || west+3 >= w.bank(y) {
			return errors.New("invalid western shoreline")
		}
	}
	for _, bank := range w.Banks {
		if bank < 1 || bank+5 >= w.mapSize() {
			return errors.New("invalid riverbank")
		}
	}
	return nil
}

// A hand-authored single-city sandbox. Bends in the river define fertile
// ground and the edges of the meadow. The winding trail has two land exits.
func NewValley() World {
	w := NewTown()
	w.startMonthlyFinance()
	w.MapExtent = 96
	n := w.mapSize()
	w.Banks = make([]int, n)
	w.Roads = nil
	w.KingdomRoad = nil
	w.Grassland = nil
	w.OreRock = nil
	w.HuntingGrounds = nil
	w.ReedBeds = nil
	w.WestBanks = make([]int, n)
	shoreline := []int{12, 10, 13, 18, 16, 11, 9, 12, 16, 14, 10, 8, 11}
	for y := 0; y < n; y++ {
		part := y / 8
		bank := (shoreline[part]*(8-y%8) + shoreline[part+1]*(y%8)) / 8
		bank += 24
		west := bank - 9
		// Reserve a long straight western reach for both pyramid complexes.
		// The banks approach it over twelve rows to avoid an abrupt bend.
		blend := min(12, max(0, min(y-12, 78-y)))
		west = (west*(12-blend) + 34*blend) / 12
		bank = max(bank, west+8)
		w.WestBanks[y] = west
		for x := west - 7; x < west-4; x++ {
			w.Grassland = append(w.Grassland, Point{x, y})
		}
		w.Banks[y] = bank
		for x := bank + 4; x < bank+9; x++ {
			w.Grassland = append(w.Grassland, Point{x, y})
		}
		if y%30 >= 19 && y%30 < 24 {
			for x := bank; x < bank+2; x++ {
				w.ReedBeds = append(w.ReedBeds, Point{x, y})
			}
		}
	}
	for _, base := range []Point{{62, 9}, {78, 45}, {64, 81}} {
		for dy := 0; dy < 4; dy++ {
			for dx := 0; dx < 2+dy%2; dx++ {
				w.OreRock = append(w.OreRock, Point{base.X + dx, base.Y + dy})
			}
		}
	}
	for _, base := range []Point{{58, 19}, {60, 42}, {66, 70}} {
		w.HuntingGrounds = append(w.HuntingGrounds, base, Point{base.X + 1, base.Y + 2})
	}
	for _, c := range []Point{{52, 12}, {66, 47}, {12, 57}, {55, 73}} {
		for _, o := range []Point{{0, 0}, {0, 1}, {1, 1}, {1, 2}, {2, 1}, {3, 2}} {
			w.Woodland = append(w.Woodland, Point{c.X + o.X, c.Y + o.Y})
		}
	}
	for _, c := range []Point{{57, 9}, {83, 48}, {8, 59}, {69, 80}} {
		for _, o := range []Point{{0, 0}, {0, 1}, {1, 1}, {2, 1}, {2, 2}} {
			w.QuarryRock = append(w.QuarryRock, Point{c.X + o.X, c.Y + o.Y})
		}
	}
	x := w.Banks[0] + 5
	for y := 0; y < n; y++ {
		target := w.Banks[y] + 5
		w.KingdomRoad = append(w.KingdomRoad, Point{x, y})
		for x != target {
			if x < target {
				x++
			} else {
				x--
			}
			w.KingdomRoad = append(w.KingdomRoad, Point{x, y})
		}
	}
	w.Roads = append([]Point{}, w.KingdomRoad...)
	start, end := w.Roads[0], w.Roads[len(w.Roads)-1]
	w.EntryPoint, w.ExitPoint = &start, &end
	w.refresh()
	return w
}

// Scenario dimensions are historical facts. Coastlines and starting terrain
// are independently authored. The first three teaching towns keep their
// starting bend, hunting range and ore field within the original study area.
func (w *World) extendCampaignSite(id, extent int) {
	if extent <= Size || extent > maxCitySize {
		return
	}
	w.startMonthlyFinance()
	w.MapExtent = extent
	w.Banks = make([]int, extent)
	for y := range w.Banks {
		w.Banks[y] = 6
	}
	x := w.ExitPoint.X
	delta := 1
	for y := Size; y < extent; y++ {
		bank := w.Banks[y-1]
		if (y+id)%3 == 0 {
			bank += delta
		}
		if bank >= 16+id%5 {
			delta = -1
		}
		if bank <= 5 {
			delta = 1
		}
		w.Banks[y] = bank
		for cx := bank + 4; cx <= bank+8; cx++ {
			w.Grassland = append(w.Grassland, Point{cx, y})
		}
		if y%31 == 13 {
			w.HuntingGrounds = append(w.HuntingGrounds, Point{bank + 18, y})
		}
		if y%27 < 3 {
			w.ReedBeds = append(w.ReedBeds, Point{bank, y})
		}
		target := max(x, bank+5)
		if y%16 == 0 && x > bank+5 {
			target = x - 1
		}
		point := Point{x, y}
		w.Roads = append(w.Roads, point)
		w.KingdomRoad = append(w.KingdomRoad, point)
		for x != target {
			if x < target {
				x++
			} else {
				x--
			}
			point = Point{x, y}
			w.Roads = append(w.Roads, point)
			w.KingdomRoad = append(w.KingdomRoad, point)
		}
	}
	end := w.KingdomRoad[len(w.KingdomRoad)-1]
	w.ExitPoint = &end
}

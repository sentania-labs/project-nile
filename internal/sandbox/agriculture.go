package sandbox

import (
	"errors"
	"slices"
)

// Manual pp62–65: floodplain farms borrow Work Camp crews; meadow farms hire
// their own ten workers. Grain, barley and pomegranates have two meadow harvests.
// Normal rows7/8 supply30/8Db and five Water Lift workers. Seasonal tick mapping,
// yields, initial fertility and orientation-free lift intakes remain provisional.
func init() {
	buildingCatalog = append(buildingCatalog, CatalogEntry{"waterlift", "Water Lift", 2, 2, 30, "Food", 5}, CatalogEntry{"irrigationditch", "Irrigation Ditch", 1, 1, 8, "Food", 0})
}
func cropResource(kind string) string {
	if r := campCrops[kind]; r != "" {
		return r
	}
	switch kind {
	case "farm":
		return "grain"
	case "barleyfarm":
		return "barley"
	case "flaxfarm":
		return "flax"
	}
	return ""
}
func (w World) farmType(b Building) string {
	if cropResource(b.Kind) == "" {
		return ""
	}
	flood, meadow := true, true
	for _, p := range b.tiles() {
		flood = flood && w.floodplain(p)
		meadow = meadow && slices.Contains(w.Grassland, p) && !w.floodplain(p)
	}
	if flood {
		return "floodplain"
	}
	if meadow {
		return "meadow"
	}
	return ""
}
func buildingWorkCapacity(b Building) int {
	if b.AgricultureVersion > 0 && cropResource(b.Kind) != "" {
		if b.FarmType == "meadow" {
			return meadowPeriod(b.Kind) * FarmCrew
		}
		return FigSeasonWork
	}
	return workCapacity(b.Kind)
}
func meadowPeriod(kind string) int {
	if kind == "farm" || kind == "barleyfarm" || kind == "pomegranatefarm" {
		return 120
	}
	return 240
}
func (w World) managedFloodFarm(b Building) bool {
	return (campCrops[b.Kind] != "" || w.AgricultureVersion > 0 && cropResource(b.Kind) != "") && (b.FarmType == "" || b.FarmType == "floodplain")
}
func (w World) agriculturePlacement(b Building) error {
	if cropResource(b.Kind) != "" && w.AgricultureVersion > 0 && w.farmType(b) == "" {
		return errors.New("the farm must fit entirely on floodplain or meadow")
	}
	if b.Kind == "waterlift" {
		edge := false
		for _, p := range w.accessTiles(b) {
			if w.water(p) || w.floodplain(p) {
				edge = true
			}
		}
		if !edge {
			return errors.New("a Water Lift must border water or the floodplain")
		}
	}
	return nil
}
func (w *World) BuildCanals(tiles []Point) error {
	if !w.Campaign.Allows("irrigationditch") {
		return errors.New("irrigation has not opened in this mission")
	}
	if len(tiles) == 0 || len(tiles) > w.mapSize()*w.mapSize() {
		return errors.New("choose irrigation ditch tiles")
	}
	occupied := w.occupied()
	for _, p := range w.Roads {
		delete(occupied, p)
	}
	for _, b := range w.Buildings {
		for _, p := range b.tiles() {
			occupied[p] = true
		}
	}
	old := map[Point]bool{}
	for _, p := range w.Canals {
		old[p] = true
	}
	added := []Point{}
	for _, p := range tiles {
		if !w.buildable(p) || occupied[p] || slices.Contains(w.HuntingGrounds, p) {
			return errors.New("irrigation ditches need open land or a road")
		}
		if !old[p] {
			added = append(added, p)
			old[p] = true
		}
	}
	cost := int64(len(added)) * constructionCost("irrigationditch")
	if w.BudgetEnabled && w.Treasury < cost {
		return errors.New("not enough treasury for these irrigation ditches")
	}
	w.spend(cost)
	w.Canals = append(w.Canals, added...)
	w.refresh()
	return nil
}
func (w World) wateredCanals() map[Point]bool {
	canal := map[Point]bool{}
	for _, p := range w.Canals {
		canal[p] = true
	}
	wet := map[Point]bool{}
	spread := func(seeds []Point, low bool) {
		queue := []Point{}
		for _, p := range seeds {
			if canal[p] && w.floodplain(p) == low && !wet[p] {
				wet[p] = true
				queue = append(queue, p)
			}
		}
		for i := 0; i < len(queue); i++ {
			for _, d := range directions {
				p := Point{queue[i].X + d.X, queue[i].Y + d.Y}
				if canal[p] && w.floodplain(p) == low && !wet[p] {
					wet[p] = true
					queue = append(queue, p)
				}
			}
		}
	}
	seeds := []Point{}
	for _, p := range w.Canals {
		if w.floodplain(p) {
			for _, d := range directions {
				if w.water(Point{p.X + d.X, p.Y + d.Y}) {
					seeds = append(seeds, p)
					break
				}
			}
		}
	}
	spread(seeds, true)
	seeds = nil
	for _, b := range w.Buildings {
		if b.Kind != "waterlift" || !goodsOperating(b) {
			continue
		}
		supplied := false
		edges := w.accessTiles(b)
		for _, p := range edges {
			if w.water(p) || wet[p] && w.floodplain(p) {
				supplied = true
			}
		}
		if supplied {
			seeds = append(seeds, edges...)
		}
	}
	spread(seeds, false)
	return wet
}
func nearFarm(b Building, p Point) bool {
	dx := max(b.X-p.X, 0, p.X-(b.X+b.Width-1))
	dy := max(b.Y-p.Y, 0, p.Y-(b.Y+b.Height-1))
	return dx+dy <= 2
}
func (w *World) refreshAgriculture() {
	wet := w.wateredCanals()
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.AgricultureVersion == 0 || cropResource(b.Kind) == "" {
			continue
		}
		b.Irrigated = false
		for p := range wet {
			if w.floodplain(p) == (b.FarmType == "floodplain") && nearFarm(*b, p) {
				b.Irrigated = true
				break
			}
		}
		if b.SoilFertility == 0 {
			b.SoilFertility = baseSoil(*b)
		}
		b.Fertility = b.SoilFertility
		if b.Irrigated {
			b.Fertility = min(100, b.Fertility+25)
		}
	}
}
func baseSoil(b Building) int {
	n := 80 + (b.X+b.Y)%3*10
	if b.FarmType == "meadow" {
		n -= 20
	}
	return n
}
func (w *World) produceMeadowCrops() {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.AgricultureVersion > 0 && b.FarmType == "floodplain" && w.phase() == 24 {
			b.SoilFertility = baseSoil(*b)
		}
		if b.AgricultureVersion == 0 || b.FarmType != "meadow" {
			continue
		}
		if goodsOperating(*b) {
			b.Work = min(b.WorkCapacity, b.Work+b.Workers)
		}
		if int(w.phase())%meadowPeriod(b.Kind) == 0 {
			if b.Damaged == "" {
				w.harvestCrop(b, 200)
			}
			b.Work = 0
		}
	}
}
func (w *World) harvestCrop(b *Building, maximum int) {
	amount := maximum * b.Work / max(1, b.WorkCapacity) * b.Fertility / 100
	actual := w.ProduceResource(b.ID, cropResource(b.Kind), amount)
	if actual > 0 && b.FarmType == "floodplain" {
		b.SoilFertility = max(20, b.SoilFertility-5)
	}
	if b.Kind == "farm" {
		w.ProduceResource(b.ID, "straw", actual/2)
	}
}
func (w World) agricultureView(v *View) {
	v.AgricultureVersion = w.AgricultureVersion
	v.Canals = append([]Point{}, w.Canals...)
	v.WateredCanals = []Point{}
	wet := w.wateredCanals()
	for _, p := range w.Canals {
		if wet[p] {
			v.WateredCanals = append(v.WateredCanals, p)
		}
	}
}
func (w World) validateAgriculture() error {
	if w.AgricultureVersion < 0 || w.AgricultureVersion > 1 {
		return errors.New("unsupported agriculture rules")
	}
	seen := map[Point]bool{}
	for _, p := range w.Canals {
		if !w.buildable(p) || seen[p] {
			return errors.New("invalid irrigation ditch")
		}
		seen[p] = true
		for _, b := range w.Buildings {
			if b.contains(p) {
				return errors.New("irrigation ditch overlaps a building")
			}
		}
	}
	for _, b := range w.Buildings {
		if b.SoilFertility < 0 || b.SoilFertility > 100 {
			return errors.New("invalid farm fertility")
		}
		if b.AgricultureVersion != w.AgricultureVersion {
			return errors.New("inconsistent farm rules")
		}
		if err := w.agriculturePlacement(b); err != nil {
			return err
		}
	}
	return nil
}

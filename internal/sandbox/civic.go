package sandbox

import "slices"

// Normal-model rows 60–64 give all five neighborhood temples the same cost
// and staff. Distinct priests keep a home’s access to each god independent.
var gods = []string{"Osiris", "Ra", "Ptah", "Seth", "Bast"}
var temples = []string{"temple_osiris", "temple_ra", "temple_ptah", "temple_seth", "temple_bast"}
var shrines = []string{"shrine_osiris", "shrine_ra", "shrine_ptah", "shrine_seth", "shrine_bast"}

func init() {
	for i, god := range gods {
		buildingCatalog = append(buildingCatalog, CatalogEntry{temples[i], "Temple to " + god, 3, 3, 150, "Religion", 8},
			CatalogEntry{shrines[i], "Shrine to " + god, 1, 1, 100, "Religion", 0})
	}
	buildingCatalog = append(buildingCatalog, CatalogEntry{"garden", "Garden", 1, 1, 12, "Beautification", 0},
		CatalogEntry{"roadblock", "Roadblock", 1, 1, 4, "Infrastructure", 0}, CatalogEntry{"plaza", "Plaza", 1, 1, 15, "Beautification", 0})
}

type influence struct{ base, step, change, radius int }

// Initial desirability and falloff from original Normal building model.
// Distance is measured from the closest footprint edge in square rings.
var influences = map[string]influence{
	"booth": {2, 1, -1, 2}, "bandstand": {4, 1, -1, 4}, "pavilion": {6, 1, -1, 6}, "jugglerschool": {2, 1, -1, 2},
	"well": {1, 1, -1, 1}, "watersupply": {4, 1, -1, 4}, "garden": {3, 1, -1, 3},
	"palace": {8, 2, -2, 6}, "taxoffice": {3, 1, -1, 3}, "firehouse": {-2, 1, 1, 2},
	"goldmine": {-16, 2, 3, 6}, "claypit": {-3, 1, 1, 2}, "farm": {-2, 1, 1, 2},
	"barleyfarm": {-2, 1, 1, 2}, "flaxfarm": {-2, 1, 1, 2}, "brewery": {-5, 1, 1, 5},
	"potter": {-4, 1, 1, 4}, "weaver": {-3, 1, 1, 3}, "brickworks": {-4, 1, 1, 4},
	"huntinglodge": {-4, 1, 2, 4}, "plaza": {4, 1, -2, 2},
}

func (w World) desirabilityMap() [Size][Size]int {
	var scores [Size][Size]int
	apply := func(x, y, width, height int, f influence) {
		for cy := max(0, y-f.radius); cy < min(Size, y+height+f.radius); cy++ {
			for cx := max(0, x-f.radius); cx < min(Size, x+width+f.radius); cx++ {
				d := max(x-cx, y-cy, cx-(x+width-1), cy-(y+height-1))
				if d <= 0 {
					continue
				}
				value := f.base + (d-1)/f.step*f.change
				if value*f.base > 0 {
					scores[cy][cx] += value
				}
			}
		}
	}
	for _, b := range w.Buildings {
		f, ok := influences[b.Kind]
		switch {
		case slices.Contains(temples, b.Kind):
			f, ok = influence{6, 2, -2, 6}, true
		case slices.Contains(shrines, b.Kind):
			f, ok = influence{4, 1, -1, 4}, true
		}
		if ok {
			apply(b.X, b.Y, b.Width, b.Height, f)
		}
	}
	for _, feature := range w.RoadFeatures {
		if feature.Kind == "plaza" {
			apply(feature.X, feature.Y, 1, 1, influences["plaza"])
		}
	}
	return scores
}

// Before entertainment and manufactured goods, religion unlocks a fourth
// housing level and delivered clean water a fifth. Negative and positive
// thresholds are distinct to avoid oscillation at the desirability boundary.
func (w World) refreshHome(b *Building, previous int) {
	b.Tier, b.Capacity, b.TierName = 0, 5, "Crude Hut"
	if b.Pop == 0 || b.Damaged != "" {
		return
	}
	threshold := func(tier, up, down int) bool {
		if previous >= tier {
			return b.Desirability >= down
		}
		return b.Desirability >= up
	}
	if !b.Water || !threshold(1, -10, -12) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 1, 7, "Sturdy Hut"
	if !b.Food || !b.Market || !threshold(2, -5, -7) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 2, 9, "Meager Shanty"
	hasTemple := false
	for _, until := range b.GodUntil {
		if until > w.Tick {
			hasTemple = true
		}
	}
	if !hasTemple || !threshold(3, 0, -2) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 3, 11, "Common Shanty"
	if b.WaterUntil <= w.Tick || !threshold(4, 4, 2) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 4, 13, "Rough Cottage"
	score := entertainmentScore(*b, w.Tick)
	if score < 10 || !threshold(5, 8, 6) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 5, 15, "Ordinary Cottage"
	if score < 13 || !threshold(6, 12, 10) || b.Goods["pottery"] <= 0 {
		return
	}
	b.Tier, b.Capacity, b.TierName = 6, 16, "Modest Homestead"
	health := 0
	for _, i := range []int{1, 3} {
		if b.ServiceUntil[i] > w.Tick {
			health++
		}
	}
	if health == 0 || score < 16 || !threshold(7, 16, 14) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 7, 17, "Spacious Homestead"
	if b.Goods["beer"] == 0 || score < 20 || !threshold(8, 20, 18) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 8, 18, "Modest Apartment"
	if b.ServiceUntil[1] <= w.Tick || score < 25 || !threshold(9, 25, 22) {
		return
	}
	b.Tier, b.Capacity, b.TierName = 9, 19, "Spacious Apartment"
}

// The next tier must already be able to receive the good it needs to evolve.
func houseWantsResource(b Building, resource string) bool {
	if b.Kind != "hut" || b.Pop == 0 {
		return false
	}
	switch resource {
	case "grain", "meat", "figs":
		return true
	case "pottery":
		return b.Tier >= 5
	case "beer":
		return b.Tier >= 7
	case "linen":
		return b.Tier >= 11
	case "jewelry":
		return b.Tier >= 13
	}
	return false
}

func neutralFavor() [5]int { return [5]int{50, 50, 50, 50, 50} }

// Religious coverage is citywide. A family's access to priests is updated
// separately when a priest actually walks past the home.
func (w *World) religionTick() {
	if w.Campaign != nil && w.Campaign.MissionID == 1 && !w.Campaign.Allows("temple_bast") {
		for i := range w.Favor {
			w.Favor[i] = 30
		}
		return
	}
	if w.Tick%(YearTicks/12) != 0 {
		return
	}
	pop := max(1, w.population())
	var coverage [5]int
	for _, b := range w.Buildings {
		if b.Damaged != "" {
			continue
		}
		if i := slices.Index(shrines, b.Kind); i >= 0 {
			coverage[i] += 300
		}
		if i := slices.Index(temples, b.Kind); i >= 0 && b.Connected && b.Workers >= b.WorkersNeeded {
			coverage[i] += 750
		}
	}
	swing := min(50, max(0, (pop-350)/50)*10)
	for i := range coverage {
		target := min(50+swing, max(50-swing, coverage[i]*100/pop))
		if w.Favor[i] < target {
			w.Favor[i]++
		} else if w.Favor[i] > target {
			w.Favor[i]--
		}
	}
}

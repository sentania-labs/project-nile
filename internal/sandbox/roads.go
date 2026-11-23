package sandbox

import (
	"errors"
	"slices"
)

func (w World) entry() Point {
	if w.EntryPoint != nil {
		return *w.EntryPoint
	}
	return Entry
}

// Scenario geography, not a claim about a particular original campaign map.
// Existing saves retain their own roads and arrival point during migration.
func (w *World) addKingdomRoad() {
	entry, exit := Point{10, 0}, Point{10, Size - 1}
	w.EntryPoint, w.ExitPoint = &entry, &exit
	w.KingdomRoad = make([]Point, 0, Size)
	roads := w.roadSet()
	for y := 0; y < Size; y++ {
		p := Point{10, y}
		w.KingdomRoad = append(w.KingdomRoad, p)
		if !roads[p] {
			w.Roads = append(w.Roads, p)
		}
	}
}

// The manual's two-tile housing access rule is verified. Square geometry and
// ignoring intervening buildings are corroborated by Akhenaten's
// city_maintenance.cpp:check_kingdome_access and grid/road_access.cpp radius
// scan, but remain an inference pending the original executable experiment.
// This affects road access, not the independent reach of service walkers.
func (b Building) roadAccess(p Point) bool {
	if slices.Contains(venues, b.Kind) && b.contains(p) {
		return true
	}
	if b.Kind == "hut" {
		return !b.contains(p) && nearBuilding(p, b, HousingRoadReach)
	}
	return b.adjacent(p)
}

func (w World) accessTiles(b Building) []Point {
	reach := 1
	if b.Kind == "hut" {
		reach = HousingRoadReach
	}
	out := make([]Point, 0, 24)
	for y := b.Y - reach; y < b.Y+b.Height+reach; y++ {
		for x := b.X - reach; x < b.X+b.Width+reach; x++ {
			p := Point{x, y}
			if b.roadAccess(p) {
				out = append(out, p)
			}
		}
	}
	return out
}

// BuildRoads commits a complete mouse-drag path once, after validation. Existing
// road and repeated path tiles cost nothing; invalid paths never partly build.
func (w *World) BuildRoads(path []Point) error {
	if len(path) == 0 || len(path) > Size*Size {
		return errors.New("a road path must contain 1..1024 tiles")
	}
	occupied, roads := w.occupied(), w.roadSet()
	added := make([]Point, 0, len(path))
	for i, p := range path {
		if !valid(p) {
			return errors.New("roads must stay on dry land inside the map")
		}
		if i > 0 && p != path[i-1] && !adjacent(p, path[i-1]) {
			return errors.New("road path tiles must connect along their edges")
		}
		if roads[p] {
			continue
		}
		if occupied[p] {
			return errors.New("the road path crosses a building")
		}
		if slices.Contains(w.HuntingGrounds, p) {
			return errors.New("leave the hunting grounds open for wildlife")
		}
		roads[p] = true
		added = append(added, p)
	}
	cost := int64(len(added)) * constructionCost("road")
	if w.BudgetEnabled && w.Treasury < cost {
		return errors.New("not enough treasury for the complete road")
	}
	if len(added) == 0 {
		return nil
	}
	if w.Version == 0 {
		w.Version = SnapshotVersion
	}
	if w.LaborPriority == "" {
		w.policyDefaults()
	}
	w.spend(cost)
	w.Roads = append(w.Roads, added...)
	w.refresh()
	return nil
}

func (w World) validateMapAccess() error {
	roads := w.roadSet()
	if !valid(w.entry()) || !roads[w.entry()] {
		return errors.New("invalid city arrival road")
	}
	if w.ExitPoint != nil && (!valid(*w.ExitPoint) || !roads[*w.ExitPoint]) {
		return errors.New("invalid city exit road")
	}
	if len(w.KingdomRoad) == 0 {
		return nil
	}
	if len(w.KingdomRoad) > Size*Size || w.ExitPoint == nil || w.KingdomRoad[0] != w.entry() || w.KingdomRoad[len(w.KingdomRoad)-1] != *w.ExitPoint {
		return errors.New("invalid Kingdom Road endpoints")
	}
	seen := map[Point]bool{}
	for i, p := range w.KingdomRoad {
		if !valid(p) || seen[p] || (i > 0 && !adjacent(p, w.KingdomRoad[i-1])) {
			return errors.New("invalid Kingdom Road route")
		}
		seen[p] = true
	}
	return nil
}

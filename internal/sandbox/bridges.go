package sandbox

import (
	"errors"
	"slices"
)

type Bridge struct {
	Y    int    `json:"y"`
	West int    `json:"west"`
	East int    `json:"east"`
	Kind string `json:"kind"`
}

func init() {
	buildingCatalog = append(buildingCatalog, CatalogEntry{"bridge", "Low bridge", 1, 1, 40, "Infrastructure", 0}, CatalogEntry{"shipbridge", "Ship bridge", 1, 1, 100, "Infrastructure", 0})
}
func (w World) crossing(p Point) (Bridge, bool) {
	for _, b := range w.Bridges {
		if p.Y == b.Y && p.X >= b.West && p.X <= b.East {
			return b, true
		}
	}
	return Bridge{}, false
}
func (w World) walkable(p Point) bool {
	if w.buildable(p) {
		return true
	}
	_, ok := w.crossing(p)
	return ok
}
func (w World) waterObstacles() map[Point]bool {
	blocked := w.occupied()
	for _, b := range w.Bridges {
		if b.Kind == "shipbridge" {
			for x := b.West; x <= b.East; x++ {
				delete(blocked, Point{x, b.Y})
			}
		}
	}
	return blocked
}
func (w *World) buildBridge(kind string, p Point) error {
	west, east := w.westBank(p.Y), w.bank(p.Y)
	if west == 0 || p.Y < 0 || p.Y >= w.mapSize() || (p.X != west-1 && p.X != east && p.X != west && p.X != east-1) {
		return errors.New("start the crossing at either riverbank")
	}
	bridge := Bridge{Y: p.Y, West: west, East: east - 1, Kind: kind}
	occupied := w.occupied()
	roads := w.roadSet()
	cost := int64(east-west) * constructionCost(kind)
	if w.BudgetEnabled && w.Treasury < cost {
		return errors.New("not enough treasury for the full span")
	}
	for x := west - 1; x <= east; x++ {
		site := Point{x, p.Y}
		if occupied[site] && !(roads[site] && w.buildable(site)) {
			return errors.New("crossing is obstructed")
		}
		if slices.Contains(w.HuntingGrounds, site) {
			return errors.New("wildlife occupies the landing")
		}
		if kind == "bridge" {
			for _, boat := range w.Boats {
				if slices.Contains(boat.Route, site) {
					return errors.New("a vessel is using the crossing")
				}
			}
		}
	}
	for x := west - 1; x <= east; x++ {
		site := Point{x, p.Y}
		if !roads[site] {
			w.Roads = append(w.Roads, site)
		}
	}
	w.Bridges = append(w.Bridges, bridge)
	w.spend(cost)
	w.refresh()
	return nil
}
func (w *World) clearBridge(b Bridge) error {
	for _, a := range w.Walkers {
		for _, p := range a.Route {
			if p.Y == b.Y && p.X >= b.West && p.X <= b.East {
				return errors.New("a worker is using this bridge")
			}
		}
	}
	candidate := *w
	candidate.Roads = slices.DeleteFunc(slices.Clone(w.Roads), func(p Point) bool { return p.Y == b.Y && p.X >= b.West && p.X <= b.East })
	candidate.Bridges = slices.DeleteFunc(slices.Clone(w.Bridges), func(other Bridge) bool { return other == b })
	candidate.Buildings = slices.Clone(w.Buildings)
	candidate.refresh()
	if err := candidate.Validate(); err != nil {
		return err
	}
	*w = candidate
	return nil
}
func (w World) validateBridges() error {
	seen := map[int]bool{}
	roads := w.roadSet()
	for _, b := range w.Bridges {
		if b.Y < 0 || b.Y >= w.mapSize() || b.West < 1 || b.West != w.westBank(b.Y) || b.East != w.bank(b.Y)-1 || seen[b.Y] || (b.Kind != "bridge" && b.Kind != "shipbridge") {
			return errors.New("invalid river crossing")
		}
		seen[b.Y] = true
		for x := b.West; x <= b.East; x++ {
			if !roads[Point{x, b.Y}] {
				return errors.New("missing bridge segment")
			}
		}
	}
	return nil
}

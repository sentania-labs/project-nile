package sandbox

import (
	"errors"
	"slices"
)

// Each paired landing supplies a boat. When either boat fills, both cross.
// The timeout for a partial load is provisional (manual pp.110–112).
const ferryWait = 8

type FerryLine struct {
	Landings  [2]int64   `json:"landings"`
	Route     []Point    `json:"route"`
	Position  int        `json:"position"`
	Reverse   bool       `json:"reverse,omitempty"`
	Sailing   bool       `json:"sailing,omitempty"`
	WaitSince int64      `json:"wait_since,omitempty"`
	Riders    [2][]int64 `json:"riders"`
}

func init() {
	buildingCatalog = append(buildingCatalog, CatalogEntry{"ferry", "Ferry landing", 2, 2, 100, "Infrastructure", 5})
}
func (w World) ferryRoad(b Building) Point {
	p := w.berth(b)
	if w.westPier(b) {
		p.X = b.X - 1
	} else {
		p.X = b.X + b.Width
	}
	return p
}
func (w World) pairedFerry(id int64) bool {
	return slices.ContainsFunc(w.Ferries, func(f FerryLine) bool { return f.Landings[0] == id || f.Landings[1] == id })
}
func (w *World) pairFerry(b Building) {
	for _, other := range w.Buildings {
		if other.ID == b.ID || other.Kind != "ferry" || w.westPier(other) == w.westPier(b) || w.pairedFerry(other.ID) || abs(other.Y-b.Y) > 6 {
			continue
		}
		path := w.waterPath(w.berth(other), w.berth(b))
		if len(path) < 2 {
			continue
		}
		w.Ferries = append(w.Ferries, FerryLine{Landings: [2]int64{other.ID, b.ID}, Route: path})
		return
	}
}
func ferryRoadsConnect(edges map[Point]Point, p, q Point) bool {
	dest, ok := edges[p]
	return ok && dest == q
}
func (w World) ferryEdges(unstaffed bool) map[Point]Point {
	if len(w.Ferries) == 0 {
		return nil
	}
	edges := map[Point]Point{}
	for _, f := range w.Ferries {
		a, b := w.building(f.Landings[0]), w.building(f.Landings[1])
		if a == nil || b == nil || !unstaffed && (a.Damaged != "" || b.Damaged != "" || a.Workers == 0 || b.Workers == 0) {
			continue
		}
		p, q := w.ferryRoad(*a), w.ferryRoad(*b)
		edges[p], edges[q] = q, p
	}
	return edges
}
func (w World) removeFerryAllowed(b Building) bool {
	if b.Kind != "ferry" {
		return true
	}
	for _, f := range w.Ferries {
		if !slices.Contains(f.Landings[:], b.ID) {
			continue
		}
		if f.Sailing {
			return false
		}
		for _, a := range w.Walkers {
			for i := 1; i < len(a.Route); i++ {
				if !adjacent(a.Route[i-1], a.Route[i]) && (a.Route[i-1] == w.ferryRoad(b) || a.Route[i] == w.ferryRoad(b)) {
					return false
				}
			}
		}
	}
	return true
}
func (w *World) advanceFerries() {
	for i := range w.Ferries {
		f := &w.Ferries[i]
		a, b := w.building(f.Landings[0]), w.building(f.Landings[1])
		piers := [2]Point{w.ferryRoad(*a), w.ferryRoad(*b)}
		if f.Sailing {
			f.Position++
			if f.Position < len(f.Route)-1 {
				continue
			}
			for side, ids := range f.Riders {
				for _, id := range ids {
					for j := range w.Walkers {
						passenger := &w.Walkers[j]
						if passenger.ID == id {
							dest := piers[1-side]
							passenger.X, passenger.Y = dest.X, dest.Y
							passenger.Next++
							break
						}
					}
				}
			}
			f.Sailing = false
			f.Position = 0
			f.Reverse = !f.Reverse
			f.Riders = [2][]int64{}
			continue
		}
		var waiting [2][]int64
		for _, p := range w.Walkers {
			if p.Next >= len(p.Route) {
				continue
			}
			if p.Role != "immigrant" && (a.Workers == 0 || b.Workers == 0 || a.Damaged != "" || b.Damaged != "") {
				continue
			}
			here := Point{p.X, p.Y}
			for side, bank := range piers {
				if here == bank && p.Route[p.Next] == piers[1-side] && len(waiting[side]) < 4 {
					waiting[side] = append(waiting[side], p.ID)
				}
			}
		}
		if len(waiting[0])+len(waiting[1]) == 0 {
			f.WaitSince = 0
			continue
		}
		if f.WaitSince == 0 {
			f.WaitSince = w.Tick
		}
		if len(waiting[0]) == 4 || len(waiting[1]) == 4 || w.Tick-f.WaitSince >= ferryWait {
			f.Sailing = true
			f.Riders = waiting
			f.WaitSince = 0
		}
	}
}
func (w World) validateFerries() error {
	seen := map[int64]bool{}
	passengers := map[int64]bool{}
	for _, f := range w.Ferries {
		a, b := w.building(f.Landings[0]), w.building(f.Landings[1])
		if a == nil || b == nil || a.Kind != "ferry" || b.Kind != "ferry" || w.westPier(*a) == w.westPier(*b) || seen[a.ID] || seen[b.ID] || len(f.Route) < 2 || f.Route[0] != w.berth(*a) || f.Route[len(f.Route)-1] != w.berth(*b) || f.Position < 0 || f.Position >= len(f.Route)-1 || f.WaitSince < 0 || f.WaitSince > w.Tick || (!f.Sailing && f.Position != 0) {
			return errors.New("invalid ferry line")
		}
		seen[a.ID], seen[b.ID] = true, true
		for i, p := range f.Route {
			if !w.water(p) || i > 0 && !adjacent(f.Route[i-1], p) {
				return errors.New("invalid ferry water route")
			}
		}
		banks := [2]Point{w.ferryRoad(*a), w.ferryRoad(*b)}
		for side, ids := range f.Riders {
			if len(ids) > 4 || !f.Sailing && len(ids) > 0 {
				return errors.New("invalid ferry load")
			}
			for _, id := range ids {
				if passengers[id] || !slices.ContainsFunc(w.Walkers, func(p Walker) bool {
					return p.ID == id && p.Next >= 1 && p.Next < len(p.Route) && p.Route[p.Next-1] == banks[side] && p.Route[p.Next] == banks[1-side]
				}) {
					return errors.New("invalid ferry passenger")
				}
				passengers[id] = true
			}
		}
	}
	return nil
}

func (w World) ferryView(v *View) {
	for _, f := range w.Ferries {
		for side, id := range f.Landings {
			idx := f.Position
			if (side == 1) != f.Reverse {
				idx = len(f.Route) - 1 - idx
			}
			pos := f.Route[idx]
			source := side
			if f.Reverse {
				source = 1 - side
			}
			state := "moored"
			if f.Sailing {
				state = "crossing"
			}
			v.Boats = append(v.Boats, BoatView{FerryLane: 2*side - 1, ID: id, Home: id, Kind: "ferryboat", X: pos.X, Y: pos.Y, Stage: state, Cargo: len(f.Riders[source])})
			for _, id := range f.Riders[source] {
				for i := range v.Walkers {
					if v.Walkers[i].ID == id {
						v.Walkers[i].AboardFerry = true
						break
					}
				}
			}
		}
	}
}

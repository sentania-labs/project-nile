package sandbox

import (
	"errors"
	"slices"
)

// Manual pp184–185 establishes impassable walls, road-carrying gatehouses and
// staffed towers built into two-thick walls, supplied with Recruiter sentries.
// Normal model rows169/171/173 give35/400/350Db and six Tower employees.
// Wall health, two sentries/tower and siege damage cadence are provisional.
func init() {
	buildingCatalog = append(buildingCatalog, CatalogEntry{"wall", "Wall", 1, 1, 35, "Military", 0}, CatalogEntry{"gatehouse", "Gatehouse", 2, 2, 400, "Military", 0}, CatalogEntry{"tower", "Tower", 2, 2, 350, "Military", 6})
}
func fortified(kind string) bool { return kind == "wall" || kind == "gatehouse" || kind == "tower" }
func defenseHealth(kind string) int {
	if kind == "tower" {
		return 80
	}
	if kind == "gatehouse" {
		return 60
	}
	if kind == "wall" {
		return 40
	}
	return 0
}
func (w World) defenseAt(p Point) *Building {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if fortified(b.Kind) && b.contains(p) {
			return b
		}
	}
	return nil
}
func (w World) defensePlacement(b Building, occupied map[Point]bool) error {
	if b.Kind == "gatehouse" {
		roads := w.roadSet()
		crossing := false
		for _, p := range b.tiles() {
			if roads[p] {
				for _, old := range w.Buildings {
					if old.contains(p) {
						return errors.New("another building occupies this gate crossing")
					}
				}
				delete(occupied, p)
				crossing = true
			}
		}
		if !crossing {
			return errors.New("place the Gatehouse over a road")
		}
	}
	if b.Kind == "tower" {
		for _, p := range b.tiles() {
			wall := w.defenseAt(p)
			if wall == nil || wall.Kind != "wall" || wall.Damaged != "" {
				return errors.New("a Tower needs an intact two-by-two wall base")
			}
			delete(occupied, p)
		}
	}
	return nil
}
func (w *World) replaceTowerWalls(b Building) {
	if b.Kind == "tower" {
		w.Buildings = slices.DeleteFunc(w.Buildings, func(old Building) bool { return old.Kind == "wall" && b.contains(old.point()) })
	}
}
func (w World) siegePath(entry Point, goals []Point) []Point {
	blocked := w.occupied()
	roads := w.roadSet()
	passable := func(p Point) bool { return w.buildable(p) && (!blocked[p] || roads[p] || w.defenseAt(p) != nil) }
	targets := map[Point]bool{}
	for _, p := range goals {
		targets[p] = true
	}
	queue := []Point{entry}
	previous := map[Point]Point{entry: entry}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		if targets[p] {
			route := []Point{p}
			for previous[p] != p {
				p = previous[p]
				route = append(route, p)
			}
			slices.Reverse(route)
			return route
		}
		for _, d := range directions {
			q := Point{p.X + d.X, p.Y + d.Y}
			if _, seen := previous[q]; !seen && passable(q) {
				previous[q] = p
				queue = append(queue, q)
			}
		}
	}
	return nil
}
func (w *World) attackDefense(a *Troop) bool {
	if !a.Enemy || a.Next >= len(a.Route) {
		return false
	}
	b := w.defenseAt(a.Route[a.Next])
	if b == nil || b.Damaged != "" {
		return false
	}
	if w.Tick%4 == 0 {
		b.DefenseHP = max(0, b.DefenseHP-1)
		if b.DefenseHP == 0 {
			b.Damaged = "collapse"
			w.CollapseCount++
			for i := range w.Troops {
				if w.Troops[i].FortID == b.ID {
					w.Troops[i].HP = 0
				}
			}
		}
	}
	return true
}
func (w World) towerSlots(id int64) int {
	n := 0
	for _, a := range w.Troops {
		if a.FortID == id {
			n++
		}
	}
	return n
}
func (w *World) recruitSentry(recruiter *Building) bool {
	for _, tower := range w.Buildings {
		if tower.Kind != "tower" || !goodsOperating(tower) || w.towerSlots(tower.ID) >= 2 {
			continue
		}
		route := w.landPath(w.perimeter(*recruiter), w.perimeter(tower))
		if len(route) == 0 || w.NextID >= MaxTick {
			continue
		}
		w.Troops = append(w.Troops, Troop{ID: w.NextID, Kind: "sentry", Origin: recruiter.ID, FortID: tower.ID, X: route[0].X, Y: route[0].Y, HP: 4, Stage: "sentry_recruit", Route: route, Next: 1})
		w.NextID++
		w.Military.Recruited++
		w.Military.SentriesRecruited++
		recruiter.Work = 0
		return true
	}
	return false
}
func (w *World) arriveSentry(a *Troop) {
	if a.Stage == "sentry_recruit" && a.Next >= len(a.Route) {
		tower := w.building(a.FortID)
		a.X, a.Y = tower.X, tower.Y
		a.Stage = "guard"
		a.Origin = 0
		a.Route = []Point{{a.X, a.Y}}
		a.Next = 1
	}
}
func (w World) validateDefenses() error {
	for _, b := range w.Buildings {
		if b.DefenseHP < 0 || b.DefenseHP > defenseHealth(b.Kind) {
			return errors.New("invalid defensive structure health")
		}
		if b.Kind == "tower" && w.towerSlots(b.ID) > 2 {
			return errors.New("tower sentry capacity exceeded")
		}
	}
	return nil
}

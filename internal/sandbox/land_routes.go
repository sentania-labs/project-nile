package sandbox

import "slices"

func offRoadRole(role string) bool {
	return role == "hunter" || role == "reedharvester" || role == "lumberjack" || role == "farmworker"
}
func offRoadOutbound(a Walker) bool {
	return a.Role == "hunter" && a.Stage == "hunt" || (a.Role == "reedharvester" || a.Role == "lumberjack") && a.Stage == "gather"
}
func reservationCapacity(role string) int {
	if role == "stonehauler" {
		return 400
	}
	if isGoodsRole(role) || role == "hunter" || role == "trader" {
		return 100
	}
	return 16
}
func (w World) landPath(starts, destinations []Point) []Point {
	blocked := w.occupied()
	for _, p := range w.Roads {
		delete(blocked, p)
	}
	targets := map[Point]bool{}
	for _, p := range destinations {
		if w.walkable(p) && !blocked[p] {
			targets[p] = true
		}
	}
	queue := []Point{}
	previous := map[Point]Point{}
	for _, p := range starts {
		if w.walkable(p) && !blocked[p] {
			if _, ok := previous[p]; !ok {
				previous[p] = p
				queue = append(queue, p)
			}
		}
	}
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
			if _, seen := previous[q]; !seen && w.walkable(q) && !blocked[q] {
				previous[q] = p
				queue = append(queue, q)
			}
		}
	}
	return nil
}
func (w World) offRoadReserved(p Point) bool {
	if w.militaryReserved(p) {
		return true
	}
	for _, a := range w.Walkers {
		if offRoadRole(a.Role) && slices.Contains(a.Route, p) {
			return true
		}
	}
	return false
}

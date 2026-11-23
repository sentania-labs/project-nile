package sandbox

import "slices"

// Each step represents a whole road tile. The secondary figure models specify
// 384/512/640 motion updates; the update-to-tile conversion is provisional.
func patrolDistance(role string) int {
	switch role {
	case "architect", "firefighter", "watercarrier", "policestation", "entertainer", "senethouse":
		return 40
	case "taxcollector":
		return 32
	default:
		return 24
	}
}

func (w World) neighborhoodPatrol(start Point, role string) []Point {
	roads := w.roadSet()
	path := []Point{start}
	previous := Point{-1, -1}
	at := start
	seed := uint32(w.NextID)*2654435761 + uint32(start.X*31+start.Y+1)
	for range patrolDistance(role) {
		var choices [4]Point
		count := 0
		for _, d := range directions {
			q := Point{at.X + d.X, at.Y + d.Y}
			if q == previous || !roads[q] || w.blockedStreet(q, role) {
				continue
			}
			choices[count] = q
			count++
		}
		if count == 0 {
			break
		}
		seed = seed*1664525 + 1013904223
		next := choices[(seed>>16)%uint32(count)]
		path = append(path, next)
		previous, at = at, next
		if at == start {
			break
		}
	}
	// A patrol at the end of a blind alley retraces its walk. Erase closed
	// loops from its return leg instead of circulating through them twice.
	back := []Point{start}
	for _, p := range path[1:] {
		if i := slices.Index(back, p); i >= 0 {
			back = back[:i+1]
		} else {
			back = append(back, p)
		}
	}
	for i := len(back) - 2; i >= 0; i-- {
		path = append(path, back[i])
	}
	return path
}

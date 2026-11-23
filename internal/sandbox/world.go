// Package sandbox is an original, deliberately narrow street-building prototype.
// Capacities 5/7 come from the extracted Pharaoh housing model. Migration speed,
// well reach, and water-only evolution are explicit prototype approximations.
// Desirability, food, employment, service walkers, and later tiers are absent.
package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const Size = 32
const MaxTick int64 = 1<<53 - 1
const WellReach = 2         // Prototype distance, not a verified original-game constant.
const MigrationInterval = 8 // Prototype cadence, not original game time.

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

var Entry = Point{6, 15}
var directions = []Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

type Building struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Stock     int    `json:"stock"`
	Pop       int    `json:"pop"`
	Tier      int    `json:"tier"`
	Water     bool   `json:"water"`
	Capacity  int    `json:"capacity"`
	TierName  string `json:"tier_name"`
	Connected bool   `json:"connected"`
}

type Walker struct {
	ID     int64   `json:"id"`
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Cargo  int     `json:"cargo"`
	Role   string  `json:"role"`
	Target int64   `json:"target"`
	Route  []Point `json:"route"`
	Next   int     `json:"next"`
}

type World struct {
	Tick      int64      `json:"tick"`
	Roads     []Point    `json:"roads"`
	Buildings []Building `json:"buildings"`
	Walkers   []Walker   `json:"walkers"`
	NextID    int64      `json:"next_id"`
	Arrivals  int64      `json:"arrivals"`
}

type View struct {
	Tick       int64      `json:"tick"`
	Roads      []Point    `json:"roads"`
	Buildings  []Building `json:"buildings"`
	Walkers    []Walker   `json:"walkers"`
	Population int        `json:"population"`
	Grain      int        `json:"grain"`
	Entry      Point      `json:"entry"`
}

func Initial() World {
	w := World{Roads: []Point{}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1}
	for x := 10; x <= 22; x++ {
		w.Roads = append(w.Roads, Point{x, 10}, Point{x, 20})
	}
	for y := 11; y < 20; y++ {
		w.Roads = append(w.Roads, Point{10, y}, Point{22, y})
	}
	for x := 6; x < 10; x++ {
		w.Roads = append(w.Roads, Point{x, 15})
	}
	_ = w.Build("hut", 14, 21)
	_ = w.Build("hut", 20, 21)
	return w
}

func valid(p Point) bool { return p.X >= 6 && p.X < Size && p.Y >= 0 && p.Y < Size }
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func adjacent(a, b Point) bool  { return abs(a.X-b.X)+abs(a.Y-b.Y) == 1 }
func (b Building) point() Point { return Point{b.X, b.Y} }
func (w World) roadSet() map[Point]bool {
	roads := make(map[Point]bool, len(w.Roads))
	for _, p := range w.Roads {
		roads[p] = true
	}
	return roads
}

// route uses a fixed neighbor order, so ties cannot depend on map iteration.
func (w World) route(destination Point) []Point {
	roads := w.roadSet()
	if !roads[Entry] {
		return nil
	}
	queue := []Point{Entry}
	previous := map[Point]Point{Entry: Entry}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		if adjacent(p, destination) {
			path := []Point{p}
			for p != Entry {
				p = previous[p]
				path = append(path, p)
			}
			for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
				path[left], path[right] = path[right], path[left]
			}
			return path
		}
		for _, d := range directions {
			next := Point{p.X + d.X, p.Y + d.Y}
			if _, seen := previous[next]; roads[next] && !seen {
				previous[next] = p
				queue = append(queue, next)
			}
		}
	}
	return nil
}

func (w *World) refresh() {
	roads := w.roadSet()
	reached := map[Point]bool{}
	queue := []Point{}
	if roads[Entry] {
		reached[Entry] = true
		queue = append(queue, Entry)
	}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		for _, d := range directions {
			next := Point{p.X + d.X, p.Y + d.Y}
			if roads[next] && !reached[next] {
				reached[next] = true
				queue = append(queue, next)
			}
		}
	}
	for i := range w.Buildings {
		b := &w.Buildings[i]
		b.Connected = false
		for _, d := range directions {
			if reached[Point{b.X + d.X, b.Y + d.Y}] {
				b.Connected = true
				break
			}
		}
		b.Water = false
		b.Tier, b.Capacity, b.TierName = 0, 0, ""
		if b.Kind != "hut" {
			continue
		}
		for _, source := range w.Buildings {
			if source.Kind == "well" && abs(source.X-b.X) <= WellReach && abs(source.Y-b.Y) <= WellReach {
				b.Water = true
				break
			}
		}
		b.Capacity, b.TierName = 5, "Small Hut"
		if b.Water && b.Pop > 0 {
			b.Tier, b.Capacity, b.TierName = 1, 7, "Large Hut"
		}
	}
}

// Clearing is atomic. A departing migrant keeps its reserved route and place
// in the destination home until arrival.
func (w *World) clear(p Point) error {
	if p == Entry {
		return errors.New("the arrival road must stay open")
	}
	for _, walker := range w.Walkers {
		if slices.Contains(walker.Route, p) {
			return errors.New("wait until the migrant reaches home")
		}
	}
	if i := slices.Index(w.Roads, p); i >= 0 {
		w.Roads = slices.Delete(w.Roads, i, i+1)
		w.refresh()
		return nil
	}
	for i, b := range w.Buildings {
		if b.point() != p {
			continue
		}
		if b.Pop > 0 {
			return errors.New("this home has residents")
		}
		candidate := *w
		candidate.Buildings = slices.Delete(slices.Clone(w.Buildings), i, i+1)
		candidate.refresh()
		if candidate.Validate() != nil {
			return errors.New("this building serves current residents or an incoming migrant")
		}
		*w = candidate
		return nil
	}
	return errors.New("the tile is already empty")
}

func (w *World) Build(kind string, x, y int) error {
	p := Point{x, y}
	if !valid(p) {
		return errors.New("choose dry land inside the 32 by 32 map")
	}
	if kind == "clear" {
		return w.clear(p)
	}
	if kind != "road" && kind != "hut" && kind != "well" {
		return errors.New("available buildings: road, hut, well")
	}
	for _, road := range w.Roads {
		if road == p {
			return errors.New("tile already contains a road")
		}
	}
	for _, b := range w.Buildings {
		if b.point() == p {
			return errors.New("tile already contains a building")
		}
	}
	if w.NextID >= MaxTick {
		return errors.New("entity identifier limit reached")
	}
	if kind == "road" {
		w.Roads = append(w.Roads, p)
	} else {
		w.Buildings = append(w.Buildings, Building{ID: w.NextID, Kind: kind, X: x, Y: y})
		w.NextID++
	}
	w.refresh()
	return nil
}

func (w *World) Step(ticks int) error {
	if ticks < 1 || ticks > 1000 || w.Tick > MaxTick-int64(ticks) || w.NextID > MaxTick-int64(ticks) {
		return errors.New("ticks must be 1..1000 within the exact counter range")
	}
	for range ticks {
		w.Tick++
		if len(w.Walkers) > 0 {
			walker := &w.Walkers[0]
			if walker.Next < len(walker.Route) {
				p := walker.Route[walker.Next]
				walker.X, walker.Y = p.X, p.Y
				walker.Next++
			}
			if walker.Next == len(walker.Route) {
				for i := range w.Buildings {
					if w.Buildings[i].ID == walker.Target {
						w.Buildings[i].Pop++
						w.Arrivals++
						break
					}
				}
				w.Walkers = []Walker{}
				w.refresh()
			}
		}
		if w.Tick%MigrationInterval == 0 && len(w.Walkers) == 0 {
			for _, b := range w.Buildings {
				if b.Kind != "hut" || !b.Connected || b.Pop >= b.Capacity {
					continue
				}
				path := w.route(b.point())
				if len(path) == 0 {
					continue
				}
				w.Walkers = append(w.Walkers, Walker{ID: w.NextID, X: Entry.X, Y: Entry.Y, Role: "immigrant", Target: b.ID, Route: path, Next: 1})
				w.NextID++
				break
			}
		}
	}
	return nil
}

func (w World) View() View {
	v := View{Tick: w.Tick, Roads: w.Roads, Buildings: w.Buildings, Walkers: w.Walkers, Entry: Entry}
	for _, b := range w.Buildings {
		v.Population += b.Pop
	}
	return v
}

func (w World) Hash() string {
	b, _ := json.Marshal(w)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func (w World) Validate() error {
	if w.Tick < 0 || w.Tick > MaxTick || w.NextID < 1 || w.NextID > MaxTick || w.Arrivals < 0 || len(w.Walkers) > 1 || w.Roads == nil || w.Buildings == nil || w.Walkers == nil {
		return errors.New("invalid world header")
	}
	occupied := map[Point]bool{}
	ids := map[int64]bool{}
	for _, p := range w.Roads {
		if !valid(p) || occupied[p] {
			return errors.New("invalid road")
		}
		occupied[p] = true
	}
	population := 0
	for _, b := range w.Buildings {
		if !valid(b.point()) || occupied[b.point()] || b.ID < 1 || b.ID >= w.NextID || ids[b.ID] || (b.Kind != "hut" && b.Kind != "well") || b.Stock != 0 || b.Pop < 0 || b.Pop > b.Capacity || (b.Kind != "hut" && b.Pop != 0) {
			return errors.New("invalid building")
		}
		occupied[b.point()], ids[b.ID] = true, true
		population += b.Pop
	}
	if int64(population) != w.Arrivals {
		return errors.New("population does not match arrivals")
	}
	derived := w
	derived.Buildings = append([]Building{}, w.Buildings...)
	derived.refresh()
	for i := range w.Buildings {
		if w.Buildings[i] != derived.Buildings[i] {
			return errors.New("inconsistent housing state")
		}
	}
	roads := w.roadSet()
	for _, walker := range w.Walkers {
		if walker.ID < 1 || walker.ID >= w.NextID || ids[walker.ID] || walker.Cargo != 0 || walker.Role != "immigrant" || len(walker.Route) == 0 || walker.Route[0] != Entry || walker.Next < 1 || walker.Next > len(walker.Route) || walker.Route[walker.Next-1] != (Point{walker.X, walker.Y}) {
			return errors.New("invalid migrant")
		}
		var target *Building
		for i := range w.Buildings {
			if w.Buildings[i].ID == walker.Target {
				target = &w.Buildings[i]
				break
			}
		}
		if target == nil || target.Kind != "hut" || target.Pop >= target.Capacity || !adjacent(walker.Route[len(walker.Route)-1], target.point()) {
			return errors.New("invalid migrant destination")
		}
		for i, p := range walker.Route {
			if !roads[p] || (i > 0 && !adjacent(p, walker.Route[i-1])) {
				return fmt.Errorf("invalid route at step %d", i)
			}
		}
	}
	return nil
}

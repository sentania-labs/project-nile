package sandbox

import (
	"errors"
	"slices"
)

type NavalStats struct {
	EnemyShips int64 `json:"enemy_ships"`
	Destroyed  int64 `json:"destroyed"`
	Escaped    int64 `json:"escaped"`
	Lost       int64 `json:"lost"`
}

func enemyBoat(a Boat) bool { return a.Kind == "enemytransport" || a.Kind == "enemywarship" }
func militaryBoat(a Boat) bool {
	return a.Kind == "transportship" || a.Kind == "warship" || enemyBoat(a)
}
func hullMaximum(a Boat) int {
	if a.Kind == "warship" {
		return 80
	}
	if a.Kind == "enemywarship" {
		return 50
	}
	return 40
}
func hull(a Boat) int {
	if !militaryBoat(a) {
		return 0
	}
	if a.Hull == 0 {
		return hullMaximum(a)
	}
	return a.Hull
}

// Transports avoid nearby hostile warships by choosing a navigable neighboring
// tile farther from every pursuer, then resume the existing destination. The
// manual establishes evasion; this five-tile lookahead is provisional.
func (w *World) evadeTransport(a *Boat) {
	if a.Kind != "transportship" || a.Order != "" && a.Order != "evade" || a.Stage == "away" || a.Stage == "boarding" || a.Stage == "commission" || a.Stage == "repair" {
		return
	}
	danger := func(p Point) int {
		distance := w.mapSize() * 2
		for _, enemy := range w.Boats {
			if enemy.Kind == "enemywarship" {
				distance = min(distance, abs(p.X-enemy.X)+abs(p.Y-enemy.Y))
			}
		}
		return distance
	}
	start := Point{a.X, a.Y}
	best, score := start, danger(start)
	if score > 5 {
		return
	}
	blocked := w.waterObstacles()
	for _, p := range []Point{{a.X, a.Y - 1}, {a.X - 1, a.Y}, {a.X + 1, a.Y}, {a.X, a.Y + 1}} {
		if w.water(p) && !blocked[p] && danger(p) > score {
			best, score = p, danger(p)
		}
	}
	if best == start {
		return
	}
	end := a.Route[len(a.Route)-1]
	if a.DeploymentID == 0 && (a.Stage == "moored" || a.Stage == "hold") {
		end = best
	}
	onward := w.waterPath(best, end)
	if len(onward) == 0 {
		return
	}
	a.Route = append([]Point{start}, onward...)
	a.Next = 1
	if a.DeploymentID == 0 {
		a.Stage = "sail"
	}
}
func (w World) landingRoute() ([]Point, Point, int64) {
	var best []Point
	var land Point
	var home int64
	for _, b := range w.Buildings {
		if b.Kind != "hut" || b.Pop == 0 {
			continue
		}
		for y := 0; y < w.mapSize(); y++ {
			for _, p := range []Point{{w.bank(y), y}, {w.westBank(y) - 1, y}} {
				if !w.freeShore(p) {
					continue
				}
				path := w.invasionPath(p, w.accessTiles(b))
				if len(path) == 0 {
					continue
				}
				if best == nil || len(path) < len(best) {
					best = path
					land = p
					home = b.ID
				}
			}
		}
	}
	return best, land, home
}

// Campaign code supplies its verified force count. Boats carry at most one
// sixteen-person company. Sea battle damage and travel cadence remain provisional.
func (w *World) SpawnSeaInvasion(eventID int, class string, amount int, entry Point, escortCount ...int) error {
	escorts := 1
	if len(escortCount) > 0 {
		escorts = escortCount[0]
	}
	if eventID < 1 || w.invasion(eventID) != nil || amount < 0 || amount > 65535 || amount == 0 && escorts == 0 || len(escortCount) > 1 || escorts < 0 || escorts > 255 || !w.water(entry) || w.NextID > MaxTick-int64((amount+15)/16+escorts) {
		return errors.New("invalid or duplicate sea invasion")
	}
	if !slices.Contains([]string{"foreign", "egyptian", "kingdom", "bedouin", "test"}, class) {
		return errors.New("unknown invasion class")
	}
	_, land, home := w.landingRoute()
	if home == 0 {
		return errors.New("no shore landing reaches a settlement")
	}
	water, _ := w.shoreWater(land)
	route := w.waterPath(entry, water)
	if len(route) == 0 {
		return errors.New("no navigable invasion approach")
	}
	w.Invasions = append(w.Invasions, Invasion{Sea: true, EventID: eventID, Class: class, Entry: entry, Amount: amount, Status: "active", Started: w.Tick})
	for left := amount; left > 0; {
		n := min(16, left)
		left -= n
		w.Boats = append(w.Boats, Boat{ID: w.NextID, Kind: "enemytransport", Home: home, X: entry.X, Y: entry.Y, Stage: "invasion", Route: slices.Clone(route), Next: 1, Hull: 40, Soldiers: n, InvasionID: eventID, Landing: land})
		w.NextID++
		w.Naval.EnemyShips++
	}
	for range escorts {
		w.Boats = append(w.Boats, Boat{ID: w.NextID, Kind: "enemywarship", Home: home, X: entry.X, Y: entry.Y, Stage: "invasion", Route: slices.Clone(route), Next: 1, Hull: 50, InvasionID: eventID, Landing: land})
		w.NextID++
		w.Naval.EnemyShips++
	}
	w.Military.Invaders += int64(amount)
	w.Military.InvasionActive = true
	return nil
}
func (w *World) TestSeaInvasion() error {
	if w.Campaign != nil {
		return errors.New("test invasions are sandbox only")
	}
	if w.Military.InvasionActive {
		return errors.New("an invasion is already active")
	}
	return w.SpawnSeaInvasion(int(w.NextID), "test", 8, w.seaEntry())
}
func (w World) passengersAtSea(eventID int) int {
	n := 0
	for _, a := range w.Boats {
		if enemyBoat(a) && (eventID == 0 || a.InvasionID == eventID) {
			n += a.Soldiers
		}
	}
	return n
}

func (w World) hostileWarships(eventID int) int {
	n := 0
	for _, a := range w.Boats {
		if a.Kind == "enemywarship" && (eventID == 0 || a.InvasionID == eventID) {
			n++
		}
	}
	return n
}
func (w *World) landEnemy(a *Boat) bool {
	if a.Kind == "enemywarship" {
		return true
	}
	home := w.building(a.Home)
	if home == nil {
		return true
	}
	path := w.invasionPath(a.Landing, w.accessTiles(*home))
	if len(path) == 0 || w.NextID > MaxTick-int64(a.Soldiers) {
		return true
	}
	for range a.Soldiers {
		w.Troops = append(w.Troops, Troop{ID: w.NextID, InvasionID: a.InvasionID, Kind: "infantry", Origin: home.ID, X: a.Landing.X, Y: a.Landing.Y, Enemy: true, HP: 3, Stage: "raid", Route: slices.Clone(path), Next: 1})
		w.NextID++
	}
	a.Soldiers = 0
	entry := w.invasion(a.InvasionID).Entry
	w.boatJourney(a, entry, "invader_leave")
	return true
}
func (w *World) sinkBoat(a Boat) {
	if enemyBoat(a) {
		w.Naval.Destroyed++
		if a.Soldiers > 0 {
			w.Military.Defeated += int64(a.Soldiers)
			if d := w.invasion(a.InvasionID); d != nil {
				d.Defeated += a.Soldiers
			}
		}
		return
	}
	w.Naval.Lost++
	if a.DeploymentID != 0 {
		if d := w.deployment(a.DeploymentID); d != nil {
			ids := map[int64]bool{}
			for _, id := range d.Troops {
				ids[id] = true
			}
			w.Troops = slices.DeleteFunc(w.Troops, func(t Troop) bool {
				if ids[t.ID] && t.Stage == "aboard" {
					w.Military.Lost++
					return true
				}
				return false
			})
			d.Troops = nil
			d.Status = "sunk"
			d.Defeated = true
			for i := range w.Troops {
				t := &w.Troops[i]
				if ids[t.ID] {
					t.Stage = "field"
					t.Route = pointSlice(t.X, t.Y)
					t.Next = 1
				}
			}
			if c := w.company(d.FortID); c != nil {
				c.Order = "return"
			}
			_ = w.MilitaryOrder(d.FortID, "return", Point{})
		}
	}
}
func (w *World) repairJourney(a *Boat) bool {
	for _, yard := range w.Buildings {
		if yard.Kind == "shipwright" && goodsOperating(yard) && w.ResourceAmount(yard.ID, "timber") >= 100 && w.boatJourney(a, w.berth(yard), "repair") {
			a.Source = yard.ID
			return true
		}
	}
	return false
}
func (w *World) arriveRepair(a *Boat) bool {
	yard := w.building(a.Source)
	if yard == nil || !goodsOperating(*yard) || w.ResourceAmount(yard.ID, "timber") < 100 {
		return true
	}
	if w.ConsumeResource(yard.ID, "timber", 100) != 100 {
		return true
	}
	a.Hull = hullMaximum(*a)
	a.CrewFatigue = 0
	a.Source = 0
	w.boatJourney(a, w.berth(*w.building(a.Home)), "return")
	return true
}
func (w *World) navalFight() map[int64]bool {
	stopped := map[int64]bool{}
	damage := map[int64]int{}
	landHits := map[int64]int{}
	for i := range w.Boats {
		a := &w.Boats[i]
		w.evadeTransport(a)
		if !militaryBoat(*a) || a.Stage == "away" || a.Stage == "commission" || a.Stage == "repair" || a.DeploymentID != 0 && a.Stage == "boarding" {
			continue
		}
		if a.Hull == 0 {
			a.Hull = hullMaximum(*a)
		}
		a.CrewFatigue = max(0, a.CrewFatigue-1)
		if !enemyBoat(*a) && a.Hull < 15 && a.DeploymentID == 0 && w.repairJourney(a) {
			continue
		}
		if a.Kind != "warship" && a.Kind != "enemywarship" {
			continue
		}
		targetIndex := -1
		bestPriority, bestDistance := 9, 100000
		for j, b := range w.Boats {
			if enemyBoat(*a) == enemyBoat(b) || !militaryBoat(b) || b.Stage == "away" {
				continue
			}
			distance := abs(a.X-b.X) + abs(a.Y-b.Y)
			reach := 8
			if a.Order == "seek" || enemyBoat(*a) {
				reach = w.mapSize() * 2
			}
			if !enemyBoat(*a) && (a.Order == "" || a.Order == "hold") {
				reach = 3
			}
			if distance > reach {
				continue
			}
			priority := 2
			if (b.Kind == "enemytransport" && b.Soldiers > 0) || b.DeploymentID != 0 {
				priority = 0
			} else if b.Kind == "enemytransport" {
				priority = 3
			}
			if priority < bestPriority || priority == bestPriority && distance < bestDistance {
				targetIndex = j
				bestPriority = priority
				bestDistance = distance
			}
		}
		landTarget := -1
		if !enemyBoat(*a) && bestPriority > 0 {
			for j, t := range w.Troops {
				distance := abs(a.X-t.X) + abs(a.Y-t.Y)
				if t.Enemy && distance <= 3 && t.HP > 0 {
					landTarget = j
					break
				}
			}
		}
		if landTarget >= 0 {
			stopped[a.ID] = true
			if w.Tick%4 == 0 {
				landHits[w.Troops[landTarget].ID]++
			}
			continue
		}
		if targetIndex < 0 {
			continue
		}
		b := w.Boats[targetIndex]
		pursue := a.Order == "engage" || a.Order == "seek" || enemyBoat(*a)
		if bestDistance <= 3 {
			stopped[a.ID] = !pursue || bestDistance <= 1
			if bestDistance <= 1 {
				stopped[b.ID] = true
			}
			if w.Tick%4 == 0 {
				hit := 1
				if bestDistance <= 1 {
					hit = 4
					a.CrewFatigue = min(100, a.CrewFatigue+12)
				}
				damage[b.ID] += hit
			}
		}
		if pursue && bestDistance > 1 {
			w.boatJourney(a, Point{b.X, b.Y}, "combat")
		}

	}
	remaining := make([]Boat, 0, len(w.Boats))
	for _, a := range w.Boats {
		if militaryBoat(a) {
			a.Hull = hull(a) - damage[a.ID]
			if a.Hull <= 0 {
				w.sinkBoat(a)
				continue
			}
		}
		remaining = append(remaining, a)
	}
	w.Boats = remaining
	for i := range w.Troops {
		w.Troops[i].HP -= landHits[w.Troops[i].ID]
	}
	return stopped
}
func (w World) validateNaval() error {
	if w.Naval.EnemyShips < 0 || w.Naval.Destroyed < 0 || w.Naval.Escaped < 0 || w.Naval.Lost < 0 {
		return errors.New("invalid naval ledger")
	}
	alive := int64(0)
	for _, a := range w.Boats {
		if a.Hull < 0 || a.Hull > hullMaximum(a) || a.CrewFatigue < 0 || a.CrewFatigue > 100 || a.Soldiers < 0 || a.Soldiers > 16 {
			return errors.New("invalid vessel hull or crew")
		}
		if enemyBoat(a) {
			alive++
			if a.InvasionID <= 0 || w.invasion(a.InvasionID) == nil || a.DeploymentID != 0 || a.Cargo != 0 || a.Reserved != 0 || a.Resource != "" || a.Stage != "invasion" && a.Stage != "combat" && a.Stage != "invader_leave" {
				return errors.New("invalid hostile vessel")
			}
		}
	}
	if alive+w.Naval.Destroyed+w.Naval.Escaped != w.Naval.EnemyShips {
		return errors.New("naval conservation failed")
	}
	return nil
}

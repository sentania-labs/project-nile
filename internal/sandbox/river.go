package sandbox

import (
	"errors"
	"fmt"
	"slices"
)

// Manual pp66-67,77,94-95,177-181 establish shoreline facilities, shipwright
// construction, timber-free fishing boats and physical dock cart transfers.
// Normal model rows74/75/76/181/182 provide costs and staffing. The reference
// corroborates400 timber per military ship; work, fishing and visit cadence
// below remain compressed prototype rules.
type Boat struct {
	Hull         int     `json:"hull,omitempty"`
	CrewFatigue  int     `json:"crew_fatigue,omitempty"`
	Order        string  `json:"order,omitempty"`
	Soldiers     int     `json:"soldiers,omitempty"`
	InvasionID   int     `json:"invasion_id,omitempty"`
	Landing      Point   `json:"landing,omitempty"`
	Partner      string  `json:"partner,omitempty"`
	DeploymentID int     `json:"deployment_id,omitempty"`
	ID           int64   `json:"id"`
	Kind         string  `json:"kind"`
	Home         int64   `json:"home"`
	Source       int64   `json:"source"`
	X            int     `json:"x"`
	Y            int     `json:"y"`
	Stage        string  `json:"stage"`
	Resource     string  `json:"resource,omitempty"`
	Cargo        int     `json:"cargo"`
	Reserved     int     `json:"reserved,omitempty"`
	Route        []Point `json:"route"`
	Next         int     `json:"next"`
	WaitUntil    int64   `json:"wait_until,omitempty"`
}
type BoatView struct {
	FerryLane   int    `json:"ferry_lane,omitempty"`
	HullMax     int    `json:"hull_max"`
	Hull        int    `json:"hull"`
	CrewFatigue int    `json:"crew_fatigue"`
	Soldiers    int    `json:"soldiers"`
	Order       string `json:"order"`
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Home        int64  `json:"home"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Stage       string `json:"stage"`
	Resource    string `json:"resource"`
	Cargo       int    `json:"cargo"`
}

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"shipwright", "Shipwright", 3, 3, 350, "Industry", 20},
		CatalogEntry{"fishingwharf", "Fishing Wharf", 2, 2, 60, "Food", 6},
		CatalogEntry{"dock", "Dock", 3, 3, 100, "Industry", 12},
		CatalogEntry{"transportwharf", "Transport Wharf", 2, 2, 200, "Military", 5},
		CatalogEntry{"warshipwharf", "Warship Wharf", 3, 3, 600, "Military", 15})
}
func coastal(kind string) bool {
	return slices.Contains([]string{"shipwright", "fishingwharf", "dock", "transportwharf", "warshipwharf", "ferry"}, kind)
}
func (w World) water(p Point) bool {
	return w.navigableWater(p)
}
func (w World) coastalPlacement(b Building) error {
	if !coastal(b.Kind) {
		return nil
	}
	west := w.westPier(b)
	for y := b.Y; y < b.Y+b.Height; y++ {
		if y < 0 || y >= w.mapSize() || (!west && b.X != w.bank(y)-1) || (west && b.X+b.Width-1 != w.westBank(y)) {
			return errors.New("place the pier across a straight riverbank with its outer column over water")
		}
	}
	if !w.water(w.berth(b)) {
		return errors.New("the pier needs navigable water")
	}
	return nil
}
func (w World) buildingTileValid(b Building, p Point) bool {
	return w.buildable(p) || (isRoyalTomb(b.Kind) && w.cliff(p)) || (monumentWaterTile(b, p) && w.navigableWater(p)) || (coastal(b.Kind) && (p.X == b.X || p.X == b.X+b.Width-1) && w.water(p))
}
func (w World) westPier(b Building) bool {
	return w.westBank(b.Y) > 0 && b.X+b.Width-1 == w.westBank(b.Y)
}
func (w World) berth(b Building) Point {
	if w.westPier(b) {
		return Point{b.X + b.Width, b.Y + b.Height/2}
	}
	return Point{b.X - 1, b.Y + b.Height/2}
}
func (w World) waterPath(start, end Point) []Point {
	blocked := w.waterObstacles()
	if !w.water(start) || !w.water(end) || blocked[start] || blocked[end] {
		return nil
	}
	queue := []Point{start}
	previous := map[Point]Point{start: start}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		if p == end {
			out := []Point{p}
			for previous[p] != p {
				p = previous[p]
				out = append(out, p)
			}
			slices.Reverse(out)
			return out
		}
		for _, d := range directions {
			q := Point{p.X + d.X, p.Y + d.Y}
			if _, seen := previous[q]; !seen && w.water(q) && !blocked[q] {
				previous[q] = p
				queue = append(queue, q)
			}
		}
	}
	return nil
}
func (w World) fishingGrounds() []Point {
	out := []Point{}
	for y := 12; y < w.mapSize(); y += 16 {
		p := Point{min(w.westBank(y)+2, w.bank(y)-3), y}
		if w.water(p) {
			out = append(out, p)
		}
	}
	return out
}
func fishKey(p Point) string { return fmt.Sprintf("%d,%d", p.X, p.Y) }
func (w World) riverBuildingInUse(id int64) bool {
	for _, a := range w.Boats {
		if a.Home == id || a.Source == id {
			return true
		}
	}
	for _, b := range w.Buildings {
		if b.ShipTarget == id || (b.ID == id && b.ShipTarget != 0) {
			return true
		}
	}
	return false
}
func (w World) riverReserved(p Point) bool {
	for _, f := range w.Ferries {
		if slices.Contains(f.Route, p) {
			return true
		}
	}
	for _, a := range w.Boats {
		if slices.Contains(a.Route, p) {
			return true
		}
	}
	return false
}
func (w World) hasBoat(home int64) bool {
	for _, a := range w.Boats {
		if a.Home == home && a.Kind != "merchantship" {
			return true
		}
	}
	for _, b := range w.Buildings {
		if b.ShipTarget == home {
			return true
		}
	}
	return false
}
func boatKind(kind string) string {
	switch kind {
	case "fishingwharf":
		return "fishingboat"
	case "transportwharf":
		return "transportship"
	case "warshipwharf":
		return "warship"
	}
	return ""
}
func (w *World) buildShips() {
	for i := range w.Buildings {
		yard := &w.Buildings[i]
		if yard.Kind != "shipwright" || !goodsOperating(*yard) {
			continue
		}
		if yard.ShipTarget == 0 {
			for _, home := range w.Buildings {
				kind := boatKind(home.Kind)
				if kind == "" || !goodsOperating(home) || w.hasBoat(home.ID) || len(w.waterPath(w.berth(*yard), w.berth(home))) == 0 {
					continue
				}
				if kind != "fishingboat" && w.ResourceAmount(yard.ID, "timber") < 400 {
					continue
				}
				if kind != "fishingboat" {
					w.ConsumeResource(yard.ID, "timber", 400)
				}
				yard.ShipTarget = home.ID
				yard.Work = 0
				break
			}
		}
		home := w.building(yard.ShipTarget)
		if home == nil {
			continue
		}
		needed := 800
		if home.Kind == "fishingwharf" {
			needed = 400
		}
		yard.Work = min(needed, yard.Work+yard.Workers)
		if yard.Work < needed || w.NextID >= MaxTick {
			continue
		}
		route := w.waterPath(w.berth(*yard), w.berth(*home))
		if len(route) == 0 {
			continue
		}
		w.Boats = append(w.Boats, Boat{ID: w.NextID, Kind: boatKind(home.Kind), Home: home.ID, Source: yard.ID, X: route[0].X, Y: route[0].Y, Stage: "commission", Route: route, Next: 1})
		w.NextID++
		w.ShipsBuilt++
		yard.Work = 0
		yard.ShipTarget = 0
	}
}
func (w *World) boatJourney(a *Boat, target Point, stage string) bool {
	route := w.waterPath(Point{a.X, a.Y}, target)
	if len(route) == 0 {
		return false
	}
	a.Route = route
	a.Next = 1
	a.Stage = stage
	return true
}
func (w *World) riverTick() {
	w.buildShips()
	w.dispatchMerchantShip()
	stopped := w.navalFight()
	remaining := make([]Boat, 0, len(w.Boats))
	for _, a := range w.Boats {
		if stopped[a.ID] || a.CrewFatigue >= 95 || a.CrewFatigue >= 70 && w.Tick%2 == 0 {
			remaining = append(remaining, a)
			continue
		}
		if a.DeploymentID != 0 {
			w.advanceDeploymentBoat(&a)
			remaining = append(remaining, a)
			continue
		}
		if a.Next < len(a.Route) {
			p := a.Route[a.Next]
			a.X, a.Y = p.X, p.Y
			a.Next++
		}
		if a.Next < len(a.Route) {
			remaining = append(remaining, a)
			continue
		}
		if a.Kind == "merchantship" {
			if w.arriveMerchantShip(&a) {
				remaining = append(remaining, a)
			}
			continue
		}
		if enemyBoat(a) {
			if a.Stage == "invader_leave" {
				w.Naval.Escaped++
				continue
			}
			w.landEnemy(&a)
			remaining = append(remaining, a)
			continue
		}
		home := w.building(a.Home)
		switch a.Stage {
		case "repair":
			w.arriveRepair(&a)
		case "combat":
			a.Stage = "hold"
			a.Route = []Point{{a.X, a.Y}}
			a.Next = 1
		case "commission", "return":
			if a.Cargo > 0 {
				n := w.DepositResource(a.Home, a.Resource, a.Cargo)
				a.Cargo -= n
				if a.Cargo > 0 {
					remaining = append(remaining, a)
					continue
				}
			}
			a.Source = 0
			a.Reserved = 0
			a.Stage = "moored"
			a.Route = []Point{{a.X, a.Y}}
			a.Next = 1
		case "fish":
			a.Stage = "fishing"
			a.WaitUntil = min(MaxTick, w.Tick+32)
		case "fishing":
			key := fishKey(Point{a.X, a.Y})
			if w.Tick >= a.WaitUntil && w.FishingReady[key] <= w.Tick {
				a.Resource = "fish"
				if w.Tick < w.CityEvents.BloodUntil {
					a.WaitUntil = w.CityEvents.BloodUntil
					break
				}
				a.Cargo = w.RecordResourceProduction("fish", 100)
				a.Reserved = 0
				if w.FishingReady == nil {
					w.FishingReady = map[string]int64{}
				}
				w.FishingReady[key] = min(MaxTick, w.Tick+96)
				w.boatJourney(&a, w.berth(*home), "return")
			}
		case "sail":
			a.Stage = "hold"
			a.Route = []Point{{a.X, a.Y}}
			a.Next = 1
		}
		if a.Stage == "moored" && a.Kind == "fishingboat" && goodsOperating(*home) && w.ResourceFreeSpace(home.ID, "fish") >= 100 {
			for _, p := range w.fishingGrounds() {
				if w.FishingReady[fishKey(p)] <= w.Tick && w.boatJourney(&a, p, "fish") {
					a.Reserved = 100
					break
				}
			}
		}
		remaining = append(remaining, a)
	}
	w.Boats = remaining
}
func (w *World) NavalOrder(id int64, order string, p Point) error {
	for i := range w.Boats {
		a := &w.Boats[i]
		if a.ID != id {
			continue
		}
		if a.DeploymentID != 0 {
			return errors.New("this transport is serving a kingdom deployment")
		}
		if a.Kind != "transportship" && a.Kind != "warship" {
			return errors.New("select a military ship")
		}
		if !slices.Contains([]string{"hold", "return", "engage", "seek", "repair", "evade"}, order) {
			return errors.New("unknown naval order")
		}
		if order == "repair" {
			if w.repairJourney(a) {
				return nil
			}
			return errors.New("a staffed Shipwright needs100timber and an open water route for repairs")
		}
		if order == "engage" || order == "seek" {
			if a.Kind != "warship" {
				return errors.New("only warships can pursue enemies")
			}
			a.Order = order
			return nil
		}
		if order == "evade" {
			if a.Kind != "transportship" {
				return errors.New("only transports use evasion orders")
			}
			a.Order = order
			return nil
		}
		if order == "return" {
			p = w.berth(*w.building(a.Home))
		}
		stage := "sail"
		if order == "return" {
			stage = "return"
		}
		if !w.boatJourney(a, p, stage) {
			return errors.New("no navigable water route to that position")
		}
		a.Order = order
		return nil
	}
	return errors.New("ship not found")
}
func (w World) boatInventory(r string) int {
	n := 0
	for _, a := range w.Boats {
		if a.Kind != "merchantship" && a.Resource == r {
			n += a.Cargo
		}
	}
	return n
}
func (w World) boatIncoming(id int64, r string) int {
	n := 0
	for _, a := range w.Boats {
		if a.Home != id {
			continue
		}
		if a.Kind == "fishingboat" && r == "fish" {
			n += a.Cargo + a.Reserved
		}
		if a.Kind == "merchantship" && a.Stage == "import" && a.Resource == r {
			n += a.Cargo
		}
	}
	return n
}
func (w World) boatReserved(id int64, r string) int {
	n := 0
	for _, a := range w.Boats {
		if a.Kind == "merchantship" && a.Stage == "export" && a.Home == id && a.Resource == r {
			n += a.Reserved
		}
	}
	return n
}
func (w World) riverView(v *View) {
	v.Deployments = append([]KingdomDeployment{}, w.Deployments...)
	v.ArmyDispatches = append([]ArmyDispatch{}, w.ArmyDispatches...)
	v.FishingGrounds = w.fishingGrounds()
	v.Boats = []BoatView{}
	v.ShipsBuilt = w.ShipsBuilt
	v.Naval = w.Naval
	for _, a := range w.Boats {
		passengers, maximum := a.Soldiers, 0
		if militaryBoat(a) {
			maximum = hullMaximum(a)
		}
		for _, d := range w.Deployments {
			if d.BoatID == a.ID && d.Status != "boarding" && !closedDeployment(d) {
				passengers = len(d.Troops)
			}
		}
		v.Boats = append(v.Boats, BoatView{HullMax: maximum, ID: a.ID, Kind: a.Kind, Home: a.Home, X: a.X, Y: a.Y, Stage: a.Stage, Resource: a.Resource, Cargo: a.Cargo, Hull: hull(a), CrewFatigue: a.CrewFatigue, Soldiers: passengers, Order: a.Order})
	}
}
func (w World) validateRiver() error {
	if err := w.validateNaval(); err != nil {
		return err
	}
	if err := w.validateDeployments(); err != nil {
		return err
	}
	if w.ShipsBuilt < 0 {
		return errors.New("invalid ship count")
	}
	ids := map[int64]bool{}
	for _, b := range w.Buildings {
		ids[b.ID] = true
		if coastal(b.Kind) {
			if err := w.coastalPlacement(b); err != nil {
				return err
			}
		}
		if b.ShipTarget != 0 {
			target := w.building(b.ShipTarget)
			if b.Kind != "shipwright" || target == nil || boatKind(target.Kind) == "" {
				return errors.New("invalid shipwright order")
			}
		}
	}
	for _, a := range w.Walkers {
		ids[a.ID] = true
	}
	for _, a := range w.Troops {
		ids[a.ID] = true
	}
	homes := map[int64]bool{}
	ships := int64(0)
	blocked := w.waterObstacles()
	for _, a := range w.Boats {
		if a.ID < 1 || a.ID >= w.NextID || ids[a.ID] || a.Cargo < 0 || a.Cargo > 1200 || a.Reserved < 0 || a.Reserved > 1200 || a.Next < 1 || a.Next > len(a.Route) || a.Route[a.Next-1] != (Point{a.X, a.Y}) || a.WaitUntil < 0 || a.WaitUntil > MaxTick {
			return errors.New("invalid river vessel")
		}
		ids[a.ID] = true
		for i, p := range a.Route {
			if !w.water(p) || blocked[p] || (i > 0 && !adjacent(p, a.Route[i-1])) {
				return errors.New("vessel route leaves navigable water")
			}
		}
		if a.DeploymentID != 0 {
			found := false
			for _, d := range w.Deployments {
				if d.RequestID == a.DeploymentID && d.BoatID == a.ID {
					found = true
				}
			}
			if !found {
				return errors.New("vessel has unknown deployment")
			}
		}
		if enemyBoat(a) {
			continue
		}
		home := w.building(a.Home)
		if home == nil {
			return errors.New("vessel has no berth")
		}
		if a.Kind == "merchantship" {
			if err := w.validateMerchantShip(a, *home); err != nil {
				return err
			}
			continue
		}
		ships++
		if homes[a.Home] || boatKind(home.Kind) != a.Kind {
			return errors.New("invalid or duplicate home ship")
		}
		homes[a.Home] = true
		if !slices.Contains([]string{"commission", "moored", "fish", "fishing", "return", "sail", "hold", "boarding", "kingdom", "away", "kingdom_return", "combat", "repair"}, a.Stage) {
			return errors.New("invalid vessel stage")
		}
		if a.Stage == "commission" {
			source := w.building(a.Source)
			if source == nil || source.Kind != "shipwright" {
				return errors.New("vessel has no constructing shipwright")
			}
		}
		if a.Kind == "fishingboat" {
			if a.Cargo > 100 || a.Reserved > 100 || (a.Resource != "" && a.Resource != "fish") {
				return errors.New("invalid catch")
			}
			if (a.Stage == "fish" || a.Stage == "fishing") && (!slices.Contains(w.fishingGrounds(), a.Route[len(a.Route)-1]) || a.Reserved != 100 || a.Cargo != 0) {
				return errors.New("invalid fishing trip")
			}
		} else if a.Cargo != 0 || a.Reserved != 0 || a.Resource != "" {
			return errors.New("military vessel carries unexpected goods")
		}
		if (a.Stage == "return" || a.Stage == "commission" || a.Stage == "moored") && a.Route[len(a.Route)-1] != w.berth(*home) {
			return errors.New("vessel missed its berth")
		}
	}
	if ships+w.Naval.Lost != w.ShipsBuilt {
		return errors.New("ship conservation failed")
	}
	for key, ready := range w.FishingReady {
		valid := false
		for _, p := range w.fishingGrounds() {
			if key == fishKey(p) {
				valid = true
			}
		}
		if !valid || ready < 0 || ready > MaxTick {
			return errors.New("invalid fish replenishment state")
		}
	}
	return nil
}

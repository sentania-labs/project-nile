package sandbox

import "errors"

func closedDeployment(d KingdomDeployment) bool {
	return d.Status == "returned" || d.Status == "cancelled" || d.Status == "sunk" || d.Status == "landed" || d.Status == "defeated"
}

func (w World) shoreWater(p Point) (Point, bool) {
	blocked := w.waterObstacles()
	for _, water := range []Point{{p.X - 1, p.Y}, {p.X + 1, p.Y}, {p.X, p.Y - 1}, {p.X, p.Y + 1}} {
		if w.water(water) && !blocked[water] {
			return water, true
		}
	}
	return Point{}, false
}
func (w World) freeShore(p Point) bool {
	if !w.buildable(p) || w.occupied()[p] && !w.roadSet()[p] {
		return false
	}
	_, ok := w.shoreWater(p)
	return ok
}

// A transport carries one physical company between freely accessible shore
// tiles, as described in the original naval orders. No shore facility is needed
// for embarkation. The existing deployment ledger owns the passengers in flight.
func (w *World) TransportCompany(boatID, fortID int64, destination Point) error {
	a := w.boat(boatID)
	company := w.company(fortID)
	if a == nil || a.Kind != "transportship" || a.DeploymentID != 0 || a.Stage == "commission" || a.Stage == "repair" || company == nil || w.deployedFort(fortID) || w.NextID >= MaxTick {
		return errors.New("select an available transport and company")
	}
	if !w.freeShore(destination) {
		return errors.New("disembark on an open dry shore tile")
	}
	landings := []Point{}
	for y := 0; y < w.mapSize(); y++ {
		for _, p := range []Point{{w.bank(y), y}, {w.westBank(y) - 1, y}} {
			if w.freeShore(p) {
				landings = append(landings, p)
			}
		}
	}
	ids := []int64{}
	for _, soldier := range w.Troops {
		if soldier.FortID == fortID && (soldier.Stage == "garrison" || soldier.Stage == "field" || soldier.Stage == "march" || soldier.Stage == "return") {
			ids = append(ids, soldier.ID)
		}
	}
	if len(ids) == 0 {
		return errors.New("company has no soldiers ready to embark")
	}
	first := w.troop(ids[0])
	path := w.landPath(pointSlice(first.X, first.Y), landings)
	if len(path) == 0 {
		return errors.New("company cannot reach an open shore")
	}
	boarding := path[len(path)-1]
	boardingWater, _ := w.shoreWater(boarding)
	destinationWater, _ := w.shoreWater(destination)
	approach := w.waterPath(Point{a.X, a.Y}, boardingWater)
	if len(approach) == 0 || len(w.waterPath(boardingWater, destinationWater)) == 0 {
		return errors.New("transport cannot reach both shores")
	}
	routes := map[int64][]Point{}
	for _, id := range ids {
		troop := w.troop(id)
		route := w.landPath(pointSlice(troop.X, troop.Y), []Point{boarding})
		if len(route) == 0 {
			return errors.New("some soldiers cannot reach the boarding shore")
		}
		routes[id] = route
	}
	requestID := int(w.NextID) + 1000000000
	for w.deployment(requestID) != nil {
		requestID++
	}
	w.NextID++
	for _, id := range ids {
		troop := w.troop(id)
		troop.Route = routes[id]
		troop.Next = 1
		troop.Stage = "embark"
	}
	company.Order = "deployment"
	w.Deployments = append(w.Deployments, KingdomDeployment{RequestID: requestID, FortID: fortID, BoatID: boatID, Strength: len(ids), Troops: ids, Status: "boarding", Boarding: &boarding, Landing: &destination})
	a.DeploymentID = requestID
	a.Stage = "boarding"
	a.Route = approach
	a.Next = 1
	a.Order = "evade"
	w.syncCompanies()
	return nil
}

func (w *World) advanceLocalTransport(a *Boat, d *KingdomDeployment) bool {
	if a.Next < len(a.Route) {
		p := a.Route[a.Next]
		a.X, a.Y = p.X, p.Y
		a.Next++
	}
	if d.Status == "boarding" {
		if len(d.Troops) == 0 {
			d.Status = "cancelled"
			a.DeploymentID = 0
			a.Stage = "hold"
			if c := w.company(d.FortID); c != nil {
				c.Order = "return"
			}
			return true
		}
		if a.Next < len(a.Route) {
			return true
		}
		for _, id := range d.Troops {
			troop := w.troop(id)
			if troop == nil || troop.Stage != "boarding" {
				return true
			}
		}
		water, _ := w.shoreWater(*d.Landing)
		if !w.boatJourney(a, water, "sail") {
			return true
		}
		for _, id := range d.Troops {
			w.troop(id).Stage = "aboard"
		}
		d.Status = "outbound"
		w.syncPassengers(*a, d)
		return true
	}
	w.syncPassengers(*a, d)
	if a.Next < len(a.Route) {
		return true
	}
	for _, id := range d.Troops {
		troop := w.troop(id)
		troop.X, troop.Y = d.Landing.X, d.Landing.Y
		troop.Route = []Point{*d.Landing}
		troop.Next = 1
		troop.Stage = "field"
	}
	d.Status = "landed"
	d.Survivors = len(d.Troops)
	a.DeploymentID = 0
	a.Stage = "hold"
	a.Route = pointSlice(a.X, a.Y)
	a.Next = 1
	if company := w.company(d.FortID); company != nil {
		company.Order = "tight"
		company.Target = *d.Landing
	}
	return true
}

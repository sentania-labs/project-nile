package sandbox

import "errors"

type KingdomDeployment struct {
	RequestID  int     `json:"request_id"`
	TargetCity int     `json:"target_city"`
	Strength   int     `json:"strength"`
	FortID     int64   `json:"fort_id"`
	BoatID     int64   `json:"boat_id"`
	Troops     []int64 `json:"troops"`
	Status     string  `json:"status"`
	Survivors  int     `json:"survivors"`
	Defeated   bool    `json:"defeated"`
}

func (w *World) deployment(id int) *KingdomDeployment {
	for i := range w.Deployments {
		if w.Deployments[i].RequestID == id {
			return &w.Deployments[i]
		}
	}
	return nil
}
func (w *World) troop(id int64) *Troop {
	for i := range w.Troops {
		if w.Troops[i].ID == id {
			return &w.Troops[i]
		}
	}
	return nil
}
func (w *World) boat(id int64) *Boat {
	for i := range w.Boats {
		if w.Boats[i].ID == id {
			return &w.Boats[i]
		}
	}
	return nil
}
func (w World) deployedFort(id int64) bool {
	for _, d := range w.Deployments {
		if d.FortID == id && d.Status != "returned" && d.Status != "cancelled" {
			return true
		}
	}
	return false
}
func (w World) aboard(id int64) bool {
	for _, d := range w.Deployments {
		if d.Status == "outbound" || d.Status == "awaits_outcome" || d.Status == "returning" {
			for _, n := range d.Troops {
				if n == id {
					return true
				}
			}
		}
	}
	return false
}

// Strength is an actual troop count, 1..16. Campaign code owns mission eligibility
// and the historical outcome. This transport layer never invents a victory.
func (w *World) StartKingdomDeployment(requestID, targetCity, strength int) error {
	if requestID < 1 || targetCity < 0 || strength < 1 || strength > CompanyLimit || w.deployment(requestID) != nil {
		return errors.New("invalid or duplicate kingdom deployment")
	}
	var vessel *Boat
	for i := range w.Boats {
		a := &w.Boats[i]
		if a.Kind == "transportship" && a.Stage == "moored" && a.DeploymentID == 0 && goodsOperating(*w.building(a.Home)) && len(w.waterPath(Point{a.X, a.Y}, w.seaEntry())) > 0 {
			vessel = a
			break
		}
	}
	if vessel == nil {
		return errors.New("a staffed Transport Wharf and moored transport ship are required")
	}
	home := w.building(vessel.Home)
	landing := w.perimeter(*home)
	if len(landing) == 0 {
		return errors.New("transport wharf has no boarding road")
	}
	for _, company := range w.Companies {
		if company.Order != "return" || company.Stationed < strength || w.deployedFort(company.FortID) {
			continue
		}
		ids := []int64{}
		routes := map[int64][]Point{}
		for _, a := range w.Troops {
			if a.FortID != company.FortID || a.Stage != "garrison" || len(ids) >= strength {
				continue
			}
			route := w.landPath(pointSlice(a.X, a.Y), landing)
			if len(route) == 0 {
				continue
			}
			ids = append(ids, a.ID)
			routes[a.ID] = route
		}
		if len(ids) != strength {
			continue
		}
		for _, id := range ids {
			a := w.troop(id)
			a.Route = routes[id]
			a.Next = 1
			a.Stage = "embark"
		}
		for i := range w.Companies {
			if w.Companies[i].FortID == company.FortID {
				w.Companies[i].Order = "deployment"
			}
		}
		w.Deployments = append(w.Deployments, KingdomDeployment{RequestID: requestID, TargetCity: targetCity, Strength: strength, FortID: company.FortID, BoatID: vessel.ID, Troops: ids, Status: "boarding"})
		vessel.DeploymentID = requestID
		vessel.Stage = "boarding"
		w.syncCompanies()
		return nil
	}
	return errors.New("a garrisoned company cannot reach the transport wharf with enough soldiers")
}
func pointSlice(x, y int) []Point { return []Point{{x, y}} }
func (w *World) syncPassengers(a Boat, d *KingdomDeployment) {
	for _, id := range d.Troops {
		if troop := w.troop(id); troop != nil && troop.Stage == "aboard" {
			troop.X, troop.Y = a.X, a.Y
			troop.Route = pointSlice(a.X, a.Y)
			troop.Next = 1
		}
	}
}
func (w *World) advanceDeploymentBoat(a *Boat) bool {
	d := w.deployment(a.DeploymentID)
	if d.Status == "boarding" {
		arrived := true
		live := []int64{}
		for _, id := range d.Troops {
			troop := w.troop(id)
			if troop == nil {
				continue
			}
			live = append(live, id)
			if troop.Stage != "boarding" {
				arrived = false
			}
		}
		d.Troops = live
		if len(live) == 0 {
			d.Status = "cancelled"
			a.DeploymentID = 0
			a.Stage = "moored"
			for i := range w.Companies {
				if w.Companies[i].FortID == d.FortID {
					w.Companies[i].Order = "return"
				}
			}
			return true
		}
		if !arrived {
			return true
		}
		if !w.boatJourney(a, w.seaEntry(), "kingdom") {
			return true
		}
		for _, id := range d.Troops {
			w.troop(id).Stage = "aboard"
		}
		d.Status = "outbound"
		w.syncPassengers(*a, d)
	}
	if a.Next < len(a.Route) {
		p := a.Route[a.Next]
		a.X, a.Y = p.X, p.Y
		a.Next++
	}
	w.syncPassengers(*a, d)
	if a.Next < len(a.Route) {
		return true
	}
	if d.Status == "outbound" {
		d.Status = "awaits_outcome"
		a.Stage = "away"
		a.Route = pointSlice(a.X, a.Y)
		a.Next = 1
	}
	if d.Status == "returning" {
		home, fort := w.building(a.Home), w.building(d.FortID)
		landing := w.perimeter(*home)
		var route []Point
		for _, p := range landing {
			route = w.landPath([]Point{p}, w.garrisonTiles(*fort))
			if len(route) > 0 {
				break
			}
		}
		if len(route) == 0 {
			return true
		}
		for _, id := range d.Troops {
			troop := w.troop(id)
			troop.X, troop.Y = route[0].X, route[0].Y
			troop.Route = append([]Point{}, route...)
			troop.Next = 1
			troop.Stage = "return"
		}
		d.Status = "returned"
		a.Stage = "moored"
		a.DeploymentID = 0
		a.Route = pointSlice(a.X, a.Y)
		a.Next = 1
		for i := range w.Companies {
			if w.Companies[i].FortID == d.FortID {
				w.Companies[i].Order = "return"
			}
		}
	}
	return true
}

// The campaign's recovered resolution code supplies survivors and defeat. The
// ship then physically sails home; troops disembark and walk back to their fort.
func (w *World) CompleteKingdomDeployment(requestID, survivors int, defeated bool) error {
	d := w.deployment(requestID)
	if d == nil || d.Status != "awaits_outcome" || survivors < 0 || survivors > len(d.Troops) {
		return errors.New("invalid kingdom deployment result")
	}
	a := w.boat(d.BoatID)
	route := w.waterPath(Point{a.X, a.Y}, w.berth(*w.building(a.Home)))
	if len(route) == 0 {
		return errors.New("the returning transport has no water route")
	}
	lost := map[int64]bool{}
	for _, id := range d.Troops[survivors:] {
		lost[id] = true
	}
	living := make([]Troop, 0, len(w.Troops))
	for _, troop := range w.Troops {
		if lost[troop.ID] {
			w.Military.Lost++
		} else {
			living = append(living, troop)
		}
	}
	w.Troops = living
	d.Troops = d.Troops[:survivors]
	d.Survivors = survivors
	d.Defeated = defeated
	d.Status = "returning"
	a.Route = route
	a.Next = 1
	a.Stage = "kingdom_return"
	w.syncCompanies()
	return nil
}
func (w World) validateDeployments() error {
	requests := map[int]bool{}
	passengers := map[int64]bool{}
	for _, d := range w.Deployments {
		if d.RequestID < 1 || requests[d.RequestID] || d.TargetCity < 0 || d.Strength < 1 || d.Strength > 16 || d.Survivors < 0 || d.Survivors > d.Strength || len(d.Troops) > d.Strength {
			return errors.New("invalid deployment ledger")
		}
		requests[d.RequestID] = true
		if d.Status == "returned" || d.Status == "cancelled" {
			continue
		}
		if d.Status != "boarding" && d.Status != "outbound" && d.Status != "awaits_outcome" && d.Status != "returning" {
			return errors.New("invalid deployment status")
		}
		var vessel *Boat
		for i := range w.Boats {
			if w.Boats[i].ID == d.BoatID {
				vessel = &w.Boats[i]
			}
		}
		if vessel == nil || vessel.Kind != "transportship" || vessel.DeploymentID != d.RequestID || w.building(d.FortID) == nil {
			return errors.New("deployment lost its company or vessel")
		}
		for _, id := range d.Troops {
			if passengers[id] {
				return errors.New("soldier assigned to two transports")
			}
			passengers[id] = true
			var troop *Troop
			for i := range w.Troops {
				if w.Troops[i].ID == id {
					troop = &w.Troops[i]
				}
			}
			if troop == nil || troop.FortID != d.FortID {
				return errors.New("deployment soldier missing")
			}
			if d.Status != "boarding" && (troop.Stage != "aboard" || troop.X != vessel.X || troop.Y != vessel.Y) {
				return errors.New("passenger is not aboard its vessel")
			}
		}
	}
	return nil
}

// Combat may remove boarding soldiers after the river phase of this tick.
func (w *World) reconcileBoarding() {
	for i := range w.Deployments {
		d := &w.Deployments[i]
		if d.Status != "boarding" {
			continue
		}
		living := d.Troops[:0]
		for _, id := range d.Troops {
			if w.troop(id) != nil {
				living = append(living, id)
			}
		}
		d.Troops = living
	}
}

package sandbox

import (
	"encoding/json"
	"errors"
	"slices"
)

// ArmyDispatch groups independently travelling companies under one original
// campaign request. Calendar durations and battle outcomes belong to campaign.
type ArmyDispatch struct {
	RequestID int   `json:"request_id"`
	RouteID   int   `json:"route_id"`
	Naval     bool  `json:"naval"`
	Legs      []int `json:"legs"`
}

func (w World) armyDispatch(id int) *ArmyDispatch {
	for i := range w.ArmyDispatches {
		if w.ArmyDispatches[i].RequestID == id {
			return &w.ArmyDispatches[i]
		}
	}
	return nil
}
func (w World) armyCopy() World {
	b, _ := json.Marshal(w)
	var copy World
	_ = json.Unmarshal(b, &copy)
	return copy
}

func (w *World) StartArmyDeployment(requestID, routeID int, fortIDs []int64, naval bool) error {
	if requestID < 1 || routeID < 0 || len(fortIDs) < 1 || len(fortIDs) > 6 || w.armyDispatch(requestID) != nil {
		return errors.New("invalid or duplicate army dispatch")
	}
	copy := w.armyCopy()
	group := ArmyDispatch{RequestID: requestID, RouteID: routeID, Naval: naval}
	seen := map[int64]bool{}
	for _, fort := range fortIDs {
		company := copy.company(fort)
		if seen[fort] || company == nil || company.Stationed == 0 || copy.deployedFort(fort) {
			return errors.New("select distinct companies with soldiers at their forts")
		}
		seen[fort] = true
		leg := int(copy.NextID) + 1000000000
		for copy.deployment(leg) != nil {
			leg++
		}
		if copy.NextID >= MaxTick {
			return errors.New("entity identity space exhausted")
		}
		copy.NextID++
		var err error
		if naval {
			err = copy.StartKingdomDeployment(leg, routeID, company.Stationed, fort)
		} else {
			err = copy.startLandDeployment(leg, routeID, fort)
		}
		if err != nil {
			return err
		}
		group.Legs = append(group.Legs, leg)
	}
	copy.ArmyDispatches = append(copy.ArmyDispatches, group)
	*w = copy
	return nil
}

func (w *World) startLandDeployment(id, route int, fort int64) error {
	exit := w.entry()
	if w.ExitPoint != nil {
		exit = *w.ExitPoint
	}
	ids := []int64{}
	routes := map[int64][]Point{}
	for _, a := range w.Troops {
		if a.FortID != fort || a.Stage != "garrison" {
			continue
		}
		path := w.landPath(pointSlice(a.X, a.Y), []Point{exit})
		if len(path) == 0 {
			return errors.New("company cannot reach the Kingdom Road exit")
		}
		ids = append(ids, a.ID)
		routes[a.ID] = path
	}
	if len(ids) == 0 {
		return errors.New("company has no garrisoned soldiers")
	}
	for _, id := range ids {
		a := w.troop(id)
		a.Route = routes[id]
		a.Next = 1
		a.Stage = "kingdom_march"
	}
	w.company(fort).Order = "deployment"
	w.Deployments = append(w.Deployments, KingdomDeployment{Mode: "land", RequestID: id, TargetCity: route, FortID: fort, Strength: len(ids), Troops: ids, Status: "outbound"})
	w.syncCompanies()
	return nil
}

func (w World) ArmyDeploymentStatus(requestID int) string {
	g := w.armyDispatch(requestID)
	if g == nil {
		return ""
	}
	ready, held, returned, failed := 0, 0, 0, 0
	for _, id := range g.Legs {
		d := w.deployment(id)
		if d == nil {
			return "invalid"
		}
		switch d.Status {
		case "awaits_outcome":
			ready++
		case "awaits_return":
			held++
		case "returned":
			returned++
		case "sunk", "cancelled", "defeated":
			failed++
		case "returning":
			return "returning"
		default:
			return "outbound"
		}
	}
	if failed == len(g.Legs) {
		return "defeated"
	}
	if ready+failed == len(g.Legs) {
		return "awaits_outcome"
	}
	if held+failed == len(g.Legs) {
		return "awaits_return"
	}
	if returned+failed == len(g.Legs) {
		return "returned"
	}
	return "outbound"
}
func (w World) ArmyDeploymentForces(requestID int) map[int64]int {
	forces := map[int64]int{}
	if g := w.armyDispatch(requestID); g != nil {
		for _, id := range g.Legs {
			d := w.deployment(id)
			if d != nil {
				forces[d.FortID] = len(d.Troops)
			}
		}
	}
	return forces
}

func (w *World) ResolveArmyDeployment(requestID int, survivors map[int64]int, defeated bool) error {
	if w.ArmyDeploymentStatus(requestID) != "awaits_outcome" {
		return errors.New("army has not reached its distant destination")
	}
	g := w.armyDispatch(requestID)
	if len(survivors) != len(g.Legs) {
		return errors.New("supply a result for every dispatched company")
	}
	for _, id := range g.Legs {
		d := w.deployment(id)
		n, ok := survivors[d.FortID]
		if !ok || n < 0 || n > len(d.Troops) {
			return errors.New("invalid army survivor count")
		}
	}
	for _, id := range g.Legs {
		d := w.deployment(id)
		if closedDeployment(*d) {
			continue
		}
		n := survivors[d.FortID]
		lost := slices.Clone(d.Troops[n:])
		w.Troops = slices.DeleteFunc(w.Troops, func(a Troop) bool { return slices.Contains(lost, a.ID) })
		w.Military.Lost += int64(len(lost))
		d.Troops = d.Troops[:n]
		d.Survivors = n
		d.Defeated = defeated
		d.Status = "awaits_return"
		if n == 0 {
			d.Status = "defeated"
			if c := w.company(d.FortID); c != nil {
				c.Order = "return"
			}
			if a := w.boat(d.BoatID); a != nil {
				a.DeploymentID = 0
				if !w.boatJourney(a, w.berth(*w.building(a.Home)), "return") {
					a.Stage = "hold"
					a.Route = pointSlice(a.X, a.Y)
					a.Next = 1
				}
			}
		}
	}
	w.syncCompanies()
	return nil
}

func (w *World) LaunchArmyReturn(requestID int) error {
	if w.ArmyDeploymentStatus(requestID) != "awaits_return" {
		return errors.New("army is not awaiting return")
	}
	copy := w.armyCopy()
	for _, id := range copy.armyDispatch(requestID).Legs {
		d := copy.deployment(id)
		if closedDeployment(*d) {
			continue
		}
		if d.Mode != "land" {
			if err := copy.CompleteKingdomDeployment(id, len(d.Troops), d.Defeated); err != nil {
				return err
			}
			continue
		}
		fort := copy.building(d.FortID)
		for _, troopID := range d.Troops {
			a := copy.troop(troopID)
			route := copy.landPath(pointSlice(a.X, a.Y), copy.garrisonTiles(*fort))
			if len(route) == 0 {
				return errors.New("returning army cannot reach its fort")
			}
			a.Route = route
			a.Next = 1
			a.Stage = "return"
		}
		d.Status = "returning"
		copy.company(d.FortID).Order = "return"
	}
	copy.syncCompanies()
	*w = copy
	return nil
}

func (w *World) advanceLandDeployments() {
	for i := range w.Deployments {
		d := &w.Deployments[i]
		if d.Mode != "land" || closedDeployment(*d) {
			continue
		}
		d.Troops = slices.DeleteFunc(d.Troops, func(id int64) bool { return w.troop(id) == nil })
		if len(d.Troops) == 0 && d.Status == "outbound" {
			d.Status = "defeated"
			if c := w.company(d.FortID); c != nil {
				c.Order = "return"
			}
			continue
		}
		stage := "kingdom_away"
		if d.Status == "returning" {
			stage = "garrison"
		}
		arrived := true
		for _, id := range d.Troops {
			if w.troop(id).Stage != stage {
				arrived = false
			}
		}
		if arrived && d.Status == "outbound" {
			d.Status = "awaits_outcome"
		}
		if arrived && d.Status == "returning" {
			d.Status = "returned"
		}
	}
}
func (w World) validateLandDeployment(d KingdomDeployment) error {
	if d.BoatID != 0 || d.Landing != nil || d.Boarding != nil || w.building(d.FortID) == nil || !slices.Contains([]string{"outbound", "awaits_outcome", "awaits_return", "returning"}, d.Status) {
		return errors.New("invalid land army journey")
	}
	for _, id := range d.Troops {
		a := w.troop(id)
		if a == nil || a.FortID != d.FortID || a.Enemy {
			return errors.New("land army lost a company member")
		}
		if (d.Status == "awaits_outcome" || d.Status == "awaits_return") && a.Stage != "kingdom_away" {
			return errors.New("distant soldier remains in city")
		}
	}
	return nil
}

func (w World) landDeploymentOwns(a Troop) bool {
	for _, d := range w.Deployments {
		if d.Mode == "land" && !closedDeployment(d) && d.FortID == a.FortID && slices.Contains(d.Troops, a.ID) {
			return true
		}
	}
	return false
}
func (w World) validateArmyDispatches() error {
	seen := map[int]bool{}
	legs := map[int]bool{}
	for _, g := range w.ArmyDispatches {
		if g.RequestID < 1 || g.RouteID < 0 || seen[g.RequestID] || len(g.Legs) < 1 || len(g.Legs) > 6 {
			return errors.New("invalid army dispatch record")
		}
		seen[g.RequestID] = true
		forts := map[int64]bool{}
		for _, id := range g.Legs {
			d := w.deployment(id)
			if d == nil || legs[id] || forts[d.FortID] || d.TargetCity != g.RouteID || g.Naval == (d.Mode == "land") {
				return errors.New("inconsistent army company assignment")
			}
			legs[id] = true
			forts[d.FortID] = true
		}
	}
	return nil
}

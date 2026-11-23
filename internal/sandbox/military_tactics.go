package sandbox

import "slices"

// Orders and qualitative effects follow manual pp172–177. The tile-scale
// formation spacing, stamina windows, morale deltas and damage modifiers are
// provisional until the original combat update constants have been recovered.
var companyOrders = []string{"return", "muster", "tight", "loose", "engage", "mopup", "charge", "deployment"}

func (w World) company(id int64) *Company {
	for i := range w.Companies {
		if w.Companies[i].FortID == id {
			return &w.Companies[i]
		}
	}
	return nil
}
func morale(c Company) int {
	if c.Morale == 0 {
		return 60
	}
	return c.Morale
}
func maximumMorale(c Company) int {
	n := 60
	if c.Kind == "charioteer" {
		n = 80
	}
	if c.Experience > 0 {
		n += 20
	}
	return n
}
func (w World) pursuitRadius(a Troop) int {
	c := w.company(a.FortID)
	if c == nil {
		return 0
	}
	switch c.Order {
	case "muster", "engage", "charge":
		return 8
	case "mopup":
		return w.mapSize() * 2
	}
	return 0
}
func (w World) battleDamage(a, target Troop, distance int) int {
	damage := 1
	if a.Experience > 0 && w.Tick%8 == 0 {
		damage++
	}
	if troopUnit(a) == "charioteer" {
		damage++
		if c := w.company(a.FortID); c != nil && c.ChargeUntil > w.Tick {
			damage++
		}
	}
	if c := w.company(target.FortID); c != nil {
		if c.Order == "tight" && troopUnit(target) == "infantry" && distance <= 1 && w.Tick%8 == 0 {
			damage = max(0, damage-1)
		}
		if c.Order == "loose" && distance > 1 && w.Tick%8 == 0 {
			damage = max(0, damage-1)
		}
		if c.Order == "mopup" {
			damage++
		}
	}
	return damage
}
func (w World) tacticalMovement(a Troop) int {
	n := unitMovement(a, w.Tick)
	if c := w.company(a.FortID); c != nil && c.ChargeUntil > w.Tick && troopUnit(a) == "charioteer" {
		n = 3
	}
	return n
}
func (w *World) changeMorale(id int64, delta int) {
	if c := w.company(id); c != nil {
		c.Morale = max(1, min(maximumMorale(*c), morale(*c)+delta))
	}
}
func (w *World) tacticsTick(wasBattle bool) {
	if wasBattle && !w.Military.InvasionActive {
		for i := range w.Troops {
			a := &w.Troops[i]
			if !a.Enemy && a.Fought {
				a.Experience = min(4, a.Experience+1)
				a.Fought = false
				w.changeMorale(a.FortID, 8)
			}
		}
	}
	if w.Tick%20 != 0 {
		return
	}
	recall := []int64{}
	for i := range w.Companies {
		c := &w.Companies[i]
		if c.Order == "deployment" {
			continue
		}
		if c.Order == "return" || c.Stationed == c.Soldiers {
			c.Morale = min(maximumMorale(*c), morale(*c)+2)
		} else if !w.Military.InvasionActive {
			c.Morale = max(1, morale(*c)-2)
		}
		if morale(*c) < 15 && c.Order != "return" {
			recall = append(recall, c.FortID)
		}
	}
	for _, id := range recall {
		_ = w.MilitaryOrder(id, "return", Point{})
	}
}
func validCompanyOrder(order string) bool { return slices.Contains(companyOrders, order) }

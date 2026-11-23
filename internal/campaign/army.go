package campaign

import "errors"

// CompanyStrength and BattleLossPercent reproduce numeric rules from the
// original 0x004a2150/0x00439e80. Training is the original formation's numeric
// training contribution, not Nile's independently modeled experience level.
func CompanyStrength(people, training int, chariots bool) int {
	base := 100
	if chariots {
		base = 200
	}
	return max(0, people) * (base + max(0, training)) / 100
}

func BattleLossPercent(strength, required int, divineAid bool) int {
	if divineAid {
		return 0
	}
	if strength < required || strength <= 0 {
		return 100
	}
	margin := (strength - required) * 100 / strength
	for _, band := range [][2]int{{10, 70}, {25, 50}, {50, 25}, {75, 15}, {100, 10}} {
		if margin < band[0] {
			return band[1]
		}
	}
	return 5
}

func (s *State) ArmyTravelMonths(id int) int {
	for _, e := range s.Events {
		if e.ID == id && e.Item == 32 {
			for _, city := range Cities(s.MissionID) {
				if city.Route == e.Location && city.ArmyTravelMonths > 0 {
					return city.ArmyTravelMonths
				}
			}
		}
	}
	// Original 0x0043fdf0 uses twelve months when no route resolves.
	return 12
}

// DispatchArmy records timing only. The world must first validate and start
// the selected physical companies; ordinary goods dispatch cannot substitute.
func (s *State) DispatchArmy(id int) error {
	r, err := s.Request(id)
	if err != nil {
		return err
	}
	if r.Resource != "troops" || r.Status != "requested" {
		return errors.New("this is not an active military request")
	}
	for i := range s.Events {
		if s.Events[i].ID == id {
			s.Events[i].Status = "army_outbound"
			s.Events[i].Remaining = s.ArmyTravelMonths(id)
			return nil
		}
	}
	return errors.New("military request was not found")
}

// ResolveArmy selects the original completion/defeat/too-late chain. The
// simulation owns troop accounting and must apply casualties before calling.
func (s *State) ResolveArmy(id int, defeated bool) error {
	for i, e := range s.Events {
		if e.ID != id || (e.Status != "army_battle" && e.Status != "army_late") {
			continue
		}
		d, _ := s.template(e.SourceID)
		chain := d.Complete
		travel := s.ArmyTravelMonths(id)
		if e.Status == "army_late" {
			chain = d.Late
			travel = max(1, travel-e.Remaining)
		} else if defeated {
			chain = d.Defeat
			e.Status, e.Remaining = "completed", 0
		}
		if e.Status != "completed" {
			e.Status, e.Remaining = "army_return_wait", travel
		}
		s.Events[i] = e
		s.chain(chain, s.EventMonth)
		return nil
	}
	return errors.New("military request is not awaiting a battle outcome")
}

func (s *State) ArmyReturning(id int) error {
	for i := range s.Events {
		if s.Events[i].ID == id && s.Events[i].Status == "army_return" {
			s.Events[i].Status = "army_homebound"
			return nil
		}
	}
	return errors.New("military request is not ready to return")
}

func (s *State) ArmyReturned(id int) error {
	for i := range s.Events {
		if s.Events[i].ID == id && s.Events[i].Status == "army_homebound" {
			s.Events[i].Status = "completed"
			return nil
		}
	}
	return errors.New("military request is not returning")
}

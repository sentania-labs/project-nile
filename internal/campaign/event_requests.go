package campaign

import "slices"

// WithRequests applies the same request and distant-battle rules to a custom
// map. The temporary state has no chapter progression, and failed operations
// leave the saved event clock unchanged.
func (c *EventClock) WithRequests(definitions []EventDefinition, cities []City, kingdom int, apply func(*State) error) (int, error) {
	s := c.state(definitions, kingdom)
	s.Events = slices.Clone(s.Events)
	s.citySource = append([]City{}, cities...)
	if err := apply(&s); err != nil {
		return kingdom, err
	}
	c.accept(s)
	return s.Kingdom, nil
}

func (s *State) sourceCities() []City {
	if s.citySource != nil {
		return s.citySource
	}
	return Cities(s.MissionID)
}

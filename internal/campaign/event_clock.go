package campaign

// EventClock reuses original event semantics without giving a custom map a
// campaign identity, chapter objectives, or building-unlock restrictions.
type EventClock struct {
	SourceVersion int          `json:"source_version,omitempty"`
	Month         int64        `json:"month"`
	RNG           uint32       `json:"rng"`
	Events        []EventState `json:"events"`
}

func NewEventClock(definitions []EventDefinition, seed uint32, sourceVersion int) *EventClock {
	if seed == 0 {
		seed = 0x9e3779b9
	}
	clock := EventClock{SourceVersion: sourceVersion, Month: -1, RNG: seed}
	s := clock.state(definitions, 50)
	for _, d := range definitions {
		if d.Trigger&1 != 0 {
			continue
		}
		s.addEvent(d, 0, false)
		if d.Trigger&16 != 0 {
			s.Events[len(s.Events)-1].Status = "dormant"
		}
	}
	clock.accept(s)
	return &clock
}

func (c *EventClock) state(definitions []EventDefinition, kingdom int) State {
	if definitions == nil {
		definitions = []EventDefinition{}
	}
	return State{customSourceVersion: c.SourceVersion, EventsInitialized: true, EventMonth: c.Month, EventRNG: c.RNG, Events: c.Events, Kingdom: kingdom, eventSource: definitions}
}
func (c *EventClock) accept(s State) {
	c.Month, c.RNG, c.Events = s.EventMonth, s.EventRNG, s.Events
}
func (c *EventClock) Advance(definitions []EventDefinition, month int64, kingdom int) ([]EventEffect, int) {
	s := c.state(definitions, kingdom)
	effects := s.AdvanceEvents(month)
	c.accept(s)
	return effects, s.Kingdom
}
func (c *EventClock) Finish(definitions []EventDefinition, id int, follow bool) error {
	s := c.state(definitions, 50)
	err := s.FinishEffect(id, follow)
	c.accept(s)
	return err
}
func (c *EventClock) Validate(definitions []EventDefinition) error {
	s := c.state(definitions, 50)
	return s.validateEvents()
}
func (c *EventClock) InvasionWarnings(definitions []EventDefinition) []InvasionWarning {
	s := c.state(definitions, 50)
	return s.InvasionWarnings()
}

func (s *State) sourceEvents() []EventDefinition {
	if s.eventSource != nil {
		return s.eventSource
	}
	return Events(s.MissionID)
}

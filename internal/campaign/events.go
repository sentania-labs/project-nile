package campaign

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

//go:embed data/base-campaign-events.jsonl
var eventData []byte

// Selector is the original scenario's fixed value, range, or choice list.
// A negative Fixed selects the half-open interval [Min, Max). Otherwise
// nonnegative Min/Max are alternatives to Fixed. See original 0x0043c6b0.
type Selector struct{ Fixed, Min, Max int }

type EventDefinition struct {
	ID, Type, Month, Trigger, Deadline          int
	Item, Amount, Time, Location                Selector
	Complete, Refuse, Late, Defeat              int
	Sender, Subtype, WarningLeadMonths, Escorts int
}

var eventDefinitions = sync.OnceValue(func() [][]EventDefinition {
	var metadata struct {
		Layout struct {
			Columns []struct {
				Name string `json:"name"`
			} `json:"columns"`
		} `json:"event_layout"`
	}
	if err := json.Unmarshal(sourceData, &metadata); err != nil {
		panic(err)
	}
	columns := map[string]int{}
	for i, column := range metadata.Layout.Columns {
		columns[column.Name] = i
	}
	result := make([][]EventDefinition, len(definitions()))
	scanner := bufio.NewScanner(bytes.NewReader(eventData))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		var mission struct {
			ID      int     `json:"mission_id"`
			Records [][]int `json:"records"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &mission); err != nil {
			panic(err)
		}
		for _, row := range mission.Records {
			get := func(name string) int {
				i, ok := columns[name]
				if !ok || i >= len(row) {
					panic("missing campaign event field: " + name)
				}
				return row[i]
			}
			selectField := func(prefix string) Selector {
				return Selector{get(prefix + "_fixed_raw"), get(prefix + "_min_raw"), get(prefix + "_max_raw")}
			}
			result[mission.ID] = append(result[mission.ID], EventDefinition{
				ID: get("event_id"), Type: get("type_code"), Month: get("month_raw"), Trigger: get("trigger_code"), Deadline: get("months_initial_raw"),
				Item: selectField("item"), Amount: selectField("amount"), Time: Selector{get("time_fixed_raw"), get("time_aux_1_raw"), get("time_aux_2_raw")}, Location: selectField("location"),
				Complete: get("on_completed_action_raw"), Refuse: get("on_refusal_action_raw"), Late: get("on_too_late_action_raw"), Defeat: get("on_defeat_action_raw"), Sender: get("sender_faction_raw"), Subtype: get("subtype_raw"), WarningLeadMonths: get("warning_lead_months_raw"), Escorts: get("festival_deity_raw"),
			})
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return result
})

func Events(id int) []EventDefinition {
	if id < 0 || id >= len(eventDefinitions()) {
		return nil
	}
	return slices.Clone(eventDefinitions()[id])
}

var resourceIDs = [...]string{"", "grain", "beef", "lettuce", "chickpeas", "pomegranates", "figs", "fish", "meat", "straw", "weapons", "clay", "bricks", "pottery", "barley", "beer", "flax", "linen", "gems", "jewelry", "timber", "gold", "reeds", "papyrus", "stone", "limestone", "granite", "", "chariots", "copper", "sandstone", "debens", "troops"}

func ResourceID(id int) string {
	if id < 0 || id >= len(resourceIDs) {
		return ""
	}
	return resourceIDs[id]
}

// EventState contains chosen values and dates, so save/reload never rerolls an
// event. IDs identify instances; SourceID identifies the original template.
type EventState struct {
	Chained   bool   `json:"chained,omitempty"`
	ID        int    `json:"id"`
	SourceID  int    `json:"source_id"`
	DueMonth  int64  `json:"due_month"`
	Item      int    `json:"item"`
	Amount    int    `json:"amount"`
	Location  int    `json:"location"`
	Remaining int    `json:"remaining"`
	Status    string `json:"status"`
	Repeated  bool   `json:"repeated"`
}

type EventEffect struct{ ID, Type, Item, Amount, Location, Subtype, Escorts int }

func (s *State) eventRandom(n int) int {
	// Nile's deterministic generator is deliberately independent of Pharaoh's.
	// Source selector distributions are preserved; original random sequences are not.
	s.EventRNG ^= s.EventRNG << 13
	s.EventRNG ^= s.EventRNG >> 17
	s.EventRNG ^= s.EventRNG << 5
	if n <= 1 {
		return 0
	}
	return int(s.EventRNG % uint32(n))
}

func (s *State) selectValue(v Selector) int {
	if v.Fixed < 0 {
		if v.Min >= 0 && v.Max > v.Min {
			return v.Min + s.eventRandom(v.Max-v.Min)
		}
		return max(0, v.Min)
	}
	choices := []int{v.Fixed}
	if v.Min >= 0 {
		choices = append(choices, v.Min)
	}
	if v.Max >= 0 && v.Min >= 0 {
		choices = append(choices, v.Max)
	}
	return choices[s.eventRandom(len(choices))]
}

func (s *State) InitEvents() {
	if s == nil || s.EventsInitialized {
		return
	}
	s.EventsInitialized, s.Kingdom, s.EventMonth = true, 50, -1
	s.EventRNG = uint32(s.MissionID+1) * 0x9e3779b9
	for _, d := range Events(s.MissionID) {
		if d.Trigger&1 != 0 {
			continue
		}
		s.addEvent(d, 0, false)
		if d.Trigger&16 != 0 {
			s.Events[len(s.Events)-1].Status = "dormant"
		}
	}
}

func (s *State) addEvent(d EventDefinition, month int64, chained bool) {
	if len(s.Events) >= 10000 {
		return
	}
	delay := int64(s.selectValue(d.Time))
	due := delay*12 + int64(d.Month)
	if chained {
		due = month + delay
	}
	s.Events = append(s.Events, EventState{ID: len(s.Events) + 1, SourceID: d.ID, Chained: chained, DueMonth: due, Item: s.selectValue(d.Item), Amount: s.selectValue(d.Amount), Location: s.selectValue(d.Location), Status: "scheduled"})
}

func (s *State) template(id int) (EventDefinition, bool) {
	if s.MissionID < 0 || s.MissionID >= len(eventDefinitions()) {
		return EventDefinition{}, false
	}
	for _, d := range eventDefinitions()[s.MissionID] {
		if d.ID == id {
			return d, true
		}
	}
	return EventDefinition{}, false
}

func (s *State) chain(id int, month int64) {
	if d, ok := s.template(id); ok {
		s.addEvent(d, month, true)
	}
}

func (s *State) ChangeKingdom(delta int) { s.Kingdom = max(0, min(100, s.Kingdom+delta)) }

// AdvanceEvents consumes elapsed simulation months. It may produce effects for
// the owning simulation, which must apply them before acknowledging the event.
// Requests, deadlines and kingdom outcomes are owned here; resource deductions
// belong to the world. See original 0x00440970, 0x00440080 and 0x004ce180.
func (s *State) AdvanceEvents(month int64) []EventEffect {
	if s == nil {
		return nil
	}
	s.InitEvents()
	effects := []EventEffect{}
	for s.EventMonth < month {
		s.EventMonth++
		for i := 0; i < len(s.Events); i++ {
			e := s.Events[i]
			d, ok := s.template(e.SourceID)
			if !ok {
				continue
			}
			chain := -1
			switch e.Status {
			case "scheduled":
				if e.DueMonth > s.EventMonth {
					continue
				}
				if d.Type == 1 {
					e.Status, e.Remaining = "requested", d.Deadline
				} else {
					switch d.Type {
					case 17:
						s.ChangeKingdom(e.Amount)
						e.Status = "completed"
						chain = d.Complete
					case 18:
						s.ChangeKingdom(-e.Amount)
						e.Status = "completed"
						chain = d.Complete
					case 20:
						e.Status = "completed"
						chain = d.Complete
					default:
						e.Status = "pending_effect"
					}
				}
			case "requested", "overdue":
				e.Remaining--
				if e.Remaining <= 0 {
					if e.Item == 32 {
						e.Status = "refused"
						chain = d.Refuse
					} else if e.Status == "requested" {
						e.Status, e.Remaining = "overdue", 24
						s.ChangeKingdom(-3)
					} else {
						e.Status = "refused"
						chain = d.Refuse
					}
				}
			case "army_outbound":
				if s.EventMonth >= e.DueMonth+int64(d.Deadline) {
					e.Status = "army_battle"
					if e.Remaining >= 3 {
						e.Status = "army_late"
					}
				} else {
					e.Remaining = max(1, e.Remaining-1)
				}
			case "army_return_wait":
				e.Remaining--
				if e.Remaining == 0 {
					e.Status = "army_return"
				}
			case "dispatched", "dispatched_late":
				e.Remaining--
				if e.Remaining <= 0 {
					chain = d.Complete
					if e.Status == "dispatched_late" {
						chain = d.Late
					}
					e.Status = "completed"
				}
			}
			s.Events[i] = e
			if chain >= 0 {
				s.chain(chain, s.EventMonth)
			}
			if (e.Status == "completed" || e.Status == "refused") && d.Trigger&2 != 0 && !e.Repeated && !e.Chained {
				s.Events[i].Repeated = true
				before := len(s.Events)
				s.addEvent(d, 0, false)
				if len(s.Events) > before {
					s.Events[before].DueMonth += s.EventMonth / 12 * 12
					if s.Events[before].DueMonth <= s.EventMonth {
						s.Events[before].DueMonth += 12
					}
				}
			}
		}
	}
	for _, e := range s.Events {
		if e.Status == "pending_effect" {
			d, _ := s.template(e.SourceID)
			effects = append(effects, EventEffect{e.ID, d.Type, e.Item, e.Amount, e.Location, d.Subtype, max(0, d.Escorts)})
		}
	}
	return effects
}

func (s *State) CompleteEffect(id int) error {
	if s == nil {
		return errors.New("no campaign")
	}
	for i, e := range s.Events {
		if e.ID == id && e.Status == "pending_effect" {
			s.Events[i].Status = "completed"
			d, _ := s.template(e.SourceID)
			s.chain(d.Complete, s.EventMonth)
			return nil
		}
	}
	return errors.New("event is not awaiting an effect")
}

type Request struct {
	Location     int    `json:"location"`
	TravelMonths int    `json:"travel_months,omitempty"`
	ID           int    `json:"id"`
	Resource     string `json:"resource"`
	Quantity     int    `json:"quantity"`
	Months       int    `json:"months"`
	Status       string `json:"status"`
	Available    int    `json:"available"`
	CanDispatch  bool   `json:"can_dispatch"`
}

func (s *State) Requests() []Request {
	result := []Request{}
	if s == nil {
		return result
	}
	for _, e := range s.Events {
		d, _ := s.template(e.SourceID)
		if d.Type != 1 || (e.Status != "requested" && e.Status != "overdue" && e.Status != "dispatched" && e.Status != "dispatched_late" && !strings.HasPrefix(e.Status, "army_")) {
			continue
		}
		quantity := e.Amount
		if e.Item < 31 {
			quantity *= 100
		}
		travel := 0
		if e.Item == 32 {
			travel = s.ArmyTravelMonths(e.ID)
		}
		result = append(result, Request{Location: e.Location, TravelMonths: travel, ID: e.ID, Resource: ResourceID(e.Item), Quantity: quantity, Months: e.Remaining, Status: e.Status})
	}
	return result
}

func (s *State) Request(id int) (Request, error) {
	for _, r := range s.Requests() {
		if r.ID == id && (r.Status == "requested" || r.Status == "overdue") {
			return r, nil
		}
	}
	return Request{}, errors.New("request is not awaiting fulfillment")
}

func (s *State) DispatchRequest(id int) error {
	r, err := s.Request(id)
	if err != nil {
		return err
	}
	if r.Resource == "troops" {
		return errors.New("military requests require physical company dispatch")
	}
	for i, e := range s.Events {
		if e.ID == id {
			s.Events[i].Status = "dispatched"
			if e.Status == "overdue" {
				s.Events[i].Status = "dispatched_late"
			}
			// Original0x0043fdf0 sets a uniformly selected1..4month delivery
			// timer for both money and goods; army dispatch has another path.
			s.Events[i].Remaining = 1 + s.eventRandom(4)
			return nil
		}
	}
	return errors.New("request was not found")
}

func (s *State) RefuseRequest(id int) error {
	if _, err := s.Request(id); err != nil {
		return err
	}
	for i, e := range s.Events {
		if e.ID == id {
			s.Events[i].Status = "refused"
			s.Events[i].Remaining = 0
			d, _ := s.template(e.SourceID)
			s.chain(d.Refuse, s.EventMonth)
			return nil
		}
	}
	return errors.New("request was not found")
}

func (s *State) validateEvents() error {
	if !s.EventsInitialized {
		if len(s.Events) > 0 || s.EventRNG != 0 {
			return errors.New("uninitialized campaign events")
		}
		return nil
	}
	if s.Kingdom < 0 || s.Kingdom > 100 || s.EventMonth < -1 || s.EventRNG == 0 || len(s.Events) > 10000 {
		return errors.New("invalid campaign event clock or kingdom rating")
	}
	for i, e := range s.Events {
		if e.ID != i+1 || e.DueMonth < 0 || e.Amount < 0 || e.Remaining < 0 {
			return fmt.Errorf("invalid event %d", e.ID)
		}
		if _, ok := s.template(e.SourceID); !ok {
			return errors.New("unknown campaign event template")
		}
		switch e.Status {
		case "scheduled", "requested", "overdue", "dispatched", "dispatched_late", "completed", "refused", "pending_effect", "dormant", "army_outbound", "army_battle", "army_late", "army_return_wait", "army_return", "army_homebound":
		default:
			return errors.New("invalid event status")
		}
	}
	return nil
}

package campaign

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCustomRequestMutationUsesItsOwnCitiesAndIsAtomic(t *testing.T) {
	defs := []EventDefinition{{ID: 0, Type: 1, Deadline: 9, Complete: -1, Refuse: -1, Late: -1, Defeat: -1}}
	clock := &EventClock{Month: 4, RNG: 7, Events: []EventState{{ID: 1, SourceID: 0, DueMonth: 4, Item: 32, Amount: 7, Location: 6, Remaining: 9, Status: "requested"}}}
	cities := []City{{Route: 6, ArmyTravelMonths: 2}}
	before, _ := json.Marshal(clock)
	_, err := clock.WithRequests(defs, cities, 50, func(s *State) error {
		if err := s.DispatchArmy(1); err != nil {
			return err
		}
		return errors.New("physical dispatch rejected")
	})
	after, _ := json.Marshal(clock)
	if err == nil || string(before) != string(after) {
		t.Fatal("a failed custom request mutation changed the clock")
	}
	_, err = clock.WithRequests(defs, cities, 50, func(s *State) error { return s.DispatchArmy(1) })
	if err != nil || clock.Events[0].Status != "army_outbound" || clock.Events[0].Remaining != 2 {
		t.Fatalf("custom route was ignored: %+v %v", clock.Events, err)
	}
}

func TestCustomClockPersistsGoldRequestResourceIDs(t *testing.T) {
	defs := []EventDefinition{{ID: 0, Type: 1, Deadline: 9, Complete: -1, Refuse: -1, Late: -1, Defeat: -1}}
	for _, test := range []struct {
		version, item int
		resource      string
	}{{0, 32, "troops"}, {147, 31, "debens"}, {160, 32, "henna"}, {160, 36, "debens"}, {160, 37, "troops"}} {
		clock := NewEventClock(defs, 7, test.version)
		clock.Events = []EventState{{ID: 1, SourceID: 0, DueMonth: 4, Item: test.item, Amount: 3, Remaining: 9, Status: "requested"}}
		data, err := json.Marshal(clock)
		if err != nil {
			t.Fatal(err)
		}
		var restored EventClock
		if err = json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		_, err = restored.WithRequests(defs, nil, 50, func(state *State) error {
			requests := state.Requests()
			if len(requests) != 1 || requests[0].Resource != test.resource {
				t.Fatalf("source %d item %d: %+v", test.version, test.item, requests)
			}
			if test.resource == "henna" && (requests[0].Quantity != 300 || state.DispatchArmy(1) == nil) {
				t.Fatal("henna request accepted army dispatch or lost its load unit")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

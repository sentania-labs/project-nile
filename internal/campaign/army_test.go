package campaign

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOriginalBattleStrengthAndCasualtyBands(t *testing.T) {
	if CompanyStrength(16, 0, false) != 16 || CompanyStrength(16, 0, true) != 32 || CompanyStrength(9, 50, false) != 13 {
		t.Fatal("original per-company strength or truncation differs")
	}
	for _, row := range [][3]int{{5, 6, 100}, {10, 10, 70}, {10, 9, 50}, {20, 15, 25}, {20, 10, 15}, {20, 5, 10}, {20, 0, 5}} {
		if got := BattleLossPercent(row[0], row[1], false); got != row[2] {
			t.Fatalf("strength%d requirement%d casualties%d, want%d", row[0], row[1], got, row[2])
		}
	}
	if BattleLossPercent(1, 100, true) != 0 {
		t.Fatal("source Seth aid did not bypass losses")
	}
}

func armyScenario(t *testing.T, missionID, sourceID int) (*State, EventState) {
	t.Helper()
	s := &State{MissionID: missionID, EventsInitialized: true, EventRNG: 1, EventMonth: 0, Kingdom: 50}
	d, ok := s.template(sourceID)
	if !ok || d.Type != 1 || d.Item.Fixed != 32 {
		t.Fatal("fixture is not an original army request")
	}
	s.addEvent(d, 0, true)
	e := s.Events[0]
	s.AdvanceEvents(e.DueMonth)
	return s, s.Events[0]
}

func TestArmyDeadlineHasNoGoodsGraceAndCannotDispatchAsCargo(t *testing.T) {
	s, e := armyScenario(t, 8, 8)
	if err := s.DispatchRequest(e.ID); err == nil {
		t.Fatal("army request accepted ordinary cargo dispatch")
	}
	s.AdvanceEvents(e.DueMonth + 12)
	if s.Events[0].Status != "refused" || s.Events[0].Remaining != 0 {
		t.Fatal("military battle deadline was changed into goods grace")
	}
	if err := s.validateEvents(); err != nil {
		t.Fatal(err)
	}
}

func TestArmyBattleAndReturnClockSurviveReload(t *testing.T) {
	s, e := armyScenario(t, 8, 8)
	if s.ArmyTravelMonths(e.ID) != 1 {
		t.Fatal("Selima Oasis to Abu route duration differs from primary records")
	}
	if err := s.DispatchArmy(e.ID); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(s)
	var restored State
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	for month := e.DueMonth + 1; month <= e.DueMonth+12; month++ {
		s.AdvanceEvents(month)
		restored.AdvanceEvents(month)
		if !reflect.DeepEqual(s, &restored) {
			t.Fatal("battle scheduling changed on reload")
		}
	}
	if s.Events[0].Status != "army_battle" {
		t.Fatal("army did not await physical battle resolution")
	}
	if err := s.ResolveArmy(e.ID, false); err != nil {
		t.Fatal(err)
	}
	if s.Events[0].Status != "army_return_wait" || s.Events[0].Remaining != 1 {
		t.Fatal("victory skipped original return travel time")
	}
	s.AdvanceEvents(s.EventMonth + 1)
	if err := s.ArmyReturning(e.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.ArmyReturned(e.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.validateEvents(); err != nil {
		t.Fatal(err)
	}
}

func TestArmyLateArrivalUsesTravelRemainingAtBattleDeadline(t *testing.T) {
	s, e := armyScenario(t, 34, 14)
	if s.ArmyTravelMonths(e.ID) != 4 {
		t.Fatal("Byblos to Iken route duration differs from primary records")
	}
	s.AdvanceEvents(e.DueMonth + 17)
	if err := s.DispatchArmy(e.ID); err != nil {
		t.Fatal(err)
	}
	s.AdvanceEvents(e.DueMonth + 18)
	if s.Events[0].Status != "army_late" || s.Events[0].Remaining != 4 {
		t.Fatal("late army reached the original battle")
	}
	if err := s.ResolveArmy(e.ID, false); err != nil {
		t.Fatal(err)
	}
	if s.Events[0].Remaining != 1 || len(s.Events) != 2 || s.Events[1].SourceID != 16 {
		t.Fatal("late army did not receive its source outcome and shorter return")
	}
}

func TestDestroyedLateArmyDoesNotAwaitAnImpossibleReturn(t *testing.T) {
	s, e := armyScenario(t, 34, 14)
	s.Events[0].Status = "army_late"
	if err := s.ResolveArmy(e.ID, true); err != nil {
		t.Fatal(err)
	}
	if s.Events[0].Status != "completed" || s.Events[0].Remaining != 0 {
		t.Fatal("destroyed army was left waiting for survivors to return")
	}
}

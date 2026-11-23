package campaign

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOriginalEventCatalogAndEconomicCapabilities(t *testing.T) {
	total := 0
	for id := 0; id < 38; id++ {
		total += len(Events(id))
		s := &State{MissionID: id}
		s.InitEvents()
		if err := s.validateEvents(); err != nil {
			t.Fatalf("mission%d: %v", id, err)
		}
		for _, e := range s.Events {
			d, _ := s.template(e.SourceID)
			if d.Trigger&1 != 0 {
				t.Fatal("child template scheduled as an independent event")
			}
		}
	}
	if total != 1351 {
		t.Fatalf("declared event count=%d", total)
	}
	m, _ := Definition(Nekhen)
	for _, kind := range []string{"farm", "barleyfarm", "claypit", "potter", "brewery", "temple_ra", "temple_osiris", "temple_bast", "physician", "bandstand", "conservatory", "taxoffice"} {
		if !m.BaseAllows(kind) {
			t.Fatalf("Nekhen cannot build source-enabled %s", kind)
		}
	}
	for _, kind := range []string{"huntinglodge", "figfarm", "flaxfarm", "weaver", "goldmine", "temple_seth", "scribalschool", "library", "dock", "weaponsmith"} {
		if m.BaseAllows(kind) {
			t.Fatalf("Nekhen invented availability for %s", kind)
		}
	}
	mennefer, _ := Definition(4)
	if !mennefer.CanImportResource(6) || mennefer.CanImportResource(29) {
		t.Fatal("Men-nefer trading partners differ from source")
	}
	var dormant City
	for _, city := range Cities(5) {
		if city.Route == 5 {
			dormant = city
		}
	}
	if dormant.Name != "Men-nefer" || dormant.Type != 2 || len(dormant.Sells) != 0 || len(dormant.PotentialSells) != 3 || dormant.AnnualLimits[23] != 1500 {
		t.Fatal("Timna's dormant royal partner lost its original latent commerce")
	}
	e, _ := (&State{MissionID: 5}).template(6)
	if e.Location.Fixed != dormant.Route || e.Subtype != 2 || e.WarningLeadMonths != 8 {
		t.Fatal("political event target was confused with warning timing")
	}
}

func TestOriginalGovernmentRankAndComplexFlags(t *testing.T) {
	for _, row := range []struct {
		mission           int
		palace, residence string
	}{{4, "palace", "residence"}, {9, "townpalace", "familymansion"}, {21, "citypalace", "dynastymansion"}} {
		m, _ := Definition(row.mission)
		for _, kind := range []string{"palace", "townpalace", "citypalace", "residence", "familymansion", "dynastymansion"} {
			want := kind == row.palace || kind == row.residence
			if m.BaseAllows(kind) != want {
				t.Fatalf("mission%d rank%d has incorrect government availability for%s", m.ID, m.Rank, kind)
			}
		}
	}
	mennefer, _ := Definition(4)
	if mennefer.BaseAllows("complex_ptah") {
		t.Fatal("known deity was mistaken for permission to build its complex")
	}
	selima, _ := Definition(8)
	if !selima.BaseAllows("complex_seth") || selima.BaseAllows("complex_osiris") {
		t.Fatal("Selima's separate complex flags were lost")
	}
	baki, _ := Definition(35)
	for _, god := range []string{"osiris", "ra", "ptah", "seth"} {
		if !baki.BaseAllows("complex_" + god) {
			t.Fatal("Baki must permit the original four complex choices")
		}
	}
	if baki.BaseAllows("complex_bast") {
		t.Fatal("Baki invented a fifth complex choice")
	}
}

func eventBySource(t *testing.T, s *State, id int) EventState {
	t.Helper()
	for _, e := range s.Events {
		if e.SourceID == id {
			return e
		}
	}
	t.Fatalf("missing event template%d", id)
	return EventState{}
}

func TestRequestDeadlineGraceAndConsequencesSurviveReload(t *testing.T) {
	s := &State{MissionID: 4}
	s.InitEvents()
	e := eventBySource(t, s, 2)
	s.AdvanceEvents(e.DueMonth)
	r, err := s.Request(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Resource != "beer" || r.Quantity < 700 || r.Quantity >= 1000 || r.Months != 12 {
		t.Fatalf("wrong original request: %+v", r)
	}
	data, _ := json.Marshal(s)
	var restored State
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	for month := e.DueMonth + 1; month <= e.DueMonth+12; month++ {
		s.AdvanceEvents(month)
		restored.AdvanceEvents(month)
		if !reflect.DeepEqual(s, &restored) {
			t.Fatal("event choices or outcomes changed across reload")
		}
	}
	r, err = s.Request(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "overdue" || r.Months != 24 {
		t.Fatalf("request did not enter original24month grace: %+v", r)
	}
	if err = s.DispatchRequest(e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Request(e.ID); err == nil {
		t.Fatal("in-flight request can dispatch twice")
	}
	s.AdvanceEvents(s.EventMonth + int64(eventBySource(t, s, 2).Remaining))
	if e = eventBySource(t, s, 2); e.Status != "completed" {
		t.Fatalf("request did not complete: %+v", e)
	}
	if err = s.validateEvents(); err != nil {
		t.Fatal(err)
	}
}

func TestOriginalRequestCompletionSchedulesRelativeChild(t *testing.T) {
	s := &State{MissionID: 4}
	s.InitEvents()
	e := eventBySource(t, s, 2)
	s.AdvanceEvents(e.DueMonth)
	if err := s.DispatchRequest(e.ID); err != nil {
		t.Fatal(err)
	}
	delivered := e.DueMonth + int64(eventBySource(t, s, 2).Remaining)
	s.AdvanceEvents(delivered)
	child := eventBySource(t, s, 11)
	if child.DueMonth != delivered+1 {
		t.Fatalf("child delay interpreted as absolute years: %+v", child)
	}
	before := s.Kingdom
	s.AdvanceEvents(child.DueMonth)
	if s.Kingdom < before+5 {
		t.Fatal("source reputation reward did not run")
	}
	if err := s.RefuseRequest(e.ID); err == nil {
		t.Fatal("completed request can be refused")
	}
}

func TestChainedCopyDoesNotInheritRecurringTrigger(t *testing.T) {
	// Original 0x00440080 resets a copied event's flags to 8. Its source
	// template may recur, but the dynamically chained instance must not.
	s := &State{MissionID: Mennefer, EventsInitialized: true, EventRNG: 1, EventMonth: 0}
	d, _ := s.template(0)
	if d.Trigger&2 == 0 {
		t.Fatal("fixture is not the original recurring request")
	}
	s.addEvent(d, 0, true)
	s.Events[0].Status = "completed"
	s.AdvanceEvents(1)
	if len(s.Events) != 1 || s.Events[0].Repeated {
		t.Fatal("chained event multiplied its source recurrence")
	}
	s.addEvent(d, 0, false)
	s.Events[1].Status = "completed"
	s.AdvanceEvents(2)
	if len(s.Events) != 3 || !s.Events[1].Repeated {
		t.Fatal("independent recurring template did not renew")
	}
}

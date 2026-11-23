package campaign

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
)

func TestLicensedCleopatraCatalogAndOriginalNamespaces(t *testing.T) {
	if len(definitions()) != 53 {
		t.Fatal("missing original Gold mission identities")
	}
	total, cities := 0, 0
	for id := 38; id <= 52; id++ {
		m, ok := Definition(id)
		if !ok || m.SourceVersion() != 160 || m.Map.Width == 0 || m.InitialFunds == 0 {
			t.Fatalf("mission%d lost exact scenario metadata", id)
		}
		total += len(Events(id))
		cities += len(Cities(id))
		s, _ := NewScenario(id)
		s.InitEvents()
		if err := s.validateEvents(); err != nil {
			t.Fatalf("mission%d: %v", id, err)
		}
		if Playable(id) {
			t.Fatalf("unplayed mission%d must remain staged", id)
		}
	}
	if total != 414 || cities != 241 {
		t.Fatalf("lost source records: %d events, %d cities", total, cities)
	}
	first, _ := Definition(38)
	if first.StartYear != -1490 || first.InitialFunds != 8000 || first.Goals["population"].Enabled != 0 || first.Goals["population"].Value != 2000 || !reflect.DeepEqual(first.RequiredStructures, []string{"smallroyaltomb"}) {
		t.Fatal("Thutmose stored flags or exact monument changed")
	}
	if !first.HasLocalResource(32) || !first.CanImportResource(31) || !first.BaseAllows("workcamp") || !first.BaseAllows("hennafarm") || !first.BaseAllows("paintmaker") || !first.BaseAllows("lampmaker") || !first.BaseAllows("artisans") || !first.BaseAllows("smallroyaltomb") || first.BaseAllows("mediumroyaltomb") || first.BaseAllows("zoo") {
		t.Fatal("Thutmose availability does not match the source")
	}
	for _, row := range []struct {
		id        int
		resources []string
	}{
		{4, []string{"debens", "troops", "", "", "", "", ""}},
		{38, []string{"oil", "henna", "paint", "lamps", "marble", "debens", "troops"}},
	} {
		m, _ := Definition(row.id)
		for n, want := range row.resources {
			if got := m.ResourceID(31 + n); got != want {
				t.Fatalf("mission%d resource%d=%q want%q", row.id, 31+n, got, want)
			}
		}
	}
	for _, id := range []int{49, 51} {
		m, _ := Definition(id)
		if !m.BaseAllows("zoo") {
			t.Fatalf("mission%d lost original Zoo flag45", id)
		}
	}
	base, _ := Definition(3)
	if base.HasLocalResource(32) || base.BaseAllows("hennafarm") || base.BaseAllows("paintmaker") {
		t.Fatal("expansion commodity namespace leaked into the base campaign")
	}
	alex, _ := Definition(48)
	final, _ := Definition(51)
	if !slices.Contains(alex.RequiredStructures, "smallmausoleum") || !slices.Contains(final.RequiredStructures, "largemausoleum") {
		t.Fatal("scenario Mausoleum variant was replaced by a generic structure")
	}
}

func TestPiYerHousingAndManualCarryMetadata(t *testing.T) {
	m, _ := Definition(45)
	if m.Goals["housing_count"] != (Criterion{Value: 5, Enabled: 1}) || m.Goals["housing_level_index"].Value != 8 {
		t.Fatal("Pi-Yer lost five Modest Apartments")
	}
	p := Progress{Population: 2500, Culture: 25, Prosperity: 30, Kingdom: 45, Housing: 4}
	if m.GoalsMet(p) {
		t.Fatal("Pi-Yer accepted only four qualifying homes")
	}
	p.Housing = 5
	if !m.GoalsMet(p) {
		t.Fatal("Pi-Yer rejected original thresholds")
	}
	for _, row := range []struct {
		id              int
		savings, troops bool
		region          string
	}{
		{38, false, false, "Deir el-Medina"}, {39, false, false, "Deir el-Medina"},
		{42, true, true, ""}, {44, true, true, "Deir el-Medina"},
		{48, false, false, "Alexandria"}, {49, true, true, "Alexandria"}, {52, true, true, ""},
	} {
		e, ok := Expansion(row.id)
		if !ok || e.Savings != row.savings || e.Troops != row.troops || e.Region != row.region {
			t.Fatalf("manual carry policy changed at%d", row.id)
		}
	}
	e, _ := Expansion(38)
	if e.Population != 0 {
		t.Fatal("disabled source population appeared as a requirement")
	}
}

func TestActiumAndTutDeadlinesPersistAndCannotBeReversed(t *testing.T) {
	s, _ := NewScenario(52)
	o := Observation{Tick: 71 * 20, ElapsedMonths: 71, Population: 1000, Kingdom: 50}
	s.Observe(o)
	if s.Won || s.DefeatTick != 0 {
		t.Fatal("Actium must survive six complete years")
	}
	o.Tick += 20
	o.ElapsedMonths++
	s.Observe(o)
	if !s.Won || s.VictoryTick != 1440 {
		t.Fatal("Actium ignored original survival deadline")
	}
	v := s.View(o)
	if v.SurvivalMonths != 72 || v.ElapsedMonths != 72 {
		t.Fatal("missing player-visible original mission clock")
	}
	tut, _ := NewScenario(39)
	o = Observation{Tick: 95 * 20, ElapsedMonths: 95, Population: 1500, Kingdom: 45, Monuments: 22, BurialComplete: true}
	tut.Observe(o)
	if tut.Won {
		t.Fatal("monument rating cannot substitute for the specified Royal Tomb")
	}
	o.Tick += 20
	o.ElapsedMonths++
	tut.Observe(o)
	if tut.DefeatTick != 1920 || tut.Won {
		t.Fatal("Tut did not lose when eight years expired")
	}
	payload, _ := json.Marshal(tut)
	var restored State
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatal(err)
	}
	o.CompletedStructures = map[string]int{"mediumroyaltomb": 1}
	o.Tick++
	restored.Observe(o)
	if restored.Won || restored.DefeatTick != 1920 || restored.Validate(o.Tick) != nil || !restored.View(o).Failed || restored.View(o).Stage != "defeat" {
		t.Fatal("saved defeat could be reversed by later construction")
	}
	early, _ := NewScenario(39)
	o.Tick = 1900
	o.ElapsedMonths = 95
	early.Observe(o)
	if !early.Won {
		t.Fatal("a complete Tut city should win before its deadline")
	}
}

func originalGoldEvent(t *testing.T, mission, source int) (*State, EventState) {
	t.Helper()
	s, _ := NewScenario(mission)
	s.EventsInitialized = true
	s.EventRNG = 1
	s.EventMonth = 0
	s.Kingdom = 50
	d, ok := s.template(source)
	if !ok {
		t.Fatal("missing source template")
	}
	s.addEvent(d, 0, true)
	e := s.Events[0]
	s.AdvanceEvents(e.DueMonth)
	return s, s.Events[0]
}

func TestGoldRequestsKeepCommoditiesDistinctFromMoneyAndArmy(t *testing.T) {
	for _, row := range []struct {
		source   int
		resource string
	}{{10, "paint"}, {13, "henna"}, {15, "oil"}} {
		s, e := originalGoldEvent(t, 38, row.source)
		r, err := s.Request(e.ID)
		if err != nil || r.Resource != row.resource || r.Quantity != e.Amount*100 {
			t.Fatalf("source%d: request%+v error%v", row.source, r, err)
		}
		if err = s.DispatchRequest(r.ID); err != nil {
			t.Fatal(err)
		}
		payload, _ := json.Marshal(s)
		var restored State
		_ = json.Unmarshal(payload, &restored)
		for month := s.EventMonth + 1; month <= e.DueMonth+5; month++ {
			s.AdvanceEvents(month)
			restored.AdvanceEvents(month)
		}
		if !reflect.DeepEqual(s, &restored) || s.Events[0].Status != "completed" {
			t.Fatal("Gold shipment changed after replay")
		}
	}
	army, e := originalGoldEvent(t, 42, 10)
	r, err := army.Request(e.ID)
	if err != nil || r.Resource != "troops" || r.Quantity != e.Amount {
		t.Fatalf("Gold army became inventory: %+v %v", r, err)
	}
	if army.DispatchRequest(r.ID) == nil {
		t.Fatal("troops accepted as ordinary cargo")
	}
	if err = army.DispatchArmy(r.ID); err != nil {
		t.Fatal(err)
	}
	if army.Events[0].Status != "army_outbound" {
		t.Fatal("Gold army did not enter physical deployment lifecycle")
	}
	gift, _ := originalGoldEvent(t, 43, 11)
	effects := gift.AdvanceEvents(gift.EventMonth)
	if len(effects) == 0 || effects[0].SourceVersion != 160 || ResourceIDForVersion(effects[0].Item, effects[0].SourceVersion) != "debens" {
		t.Fatal("Gold money gift lost its resource namespace")
	}
}

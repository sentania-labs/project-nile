package campaign

import "testing"

func TestAllBaseMissionGoalsAreIndependentAndDisabledRatingsAreIgnored(t *testing.T) {
	for id := 0; id < 38; id++ {
		m, _ := Definition(id)
		p := Progress{Population: m.Goals["population"].Value, Housing: m.Goals["housing_count"].Value, Culture: m.Goals["culture"].Value, Prosperity: m.Goals["prosperity"].Value, Monuments: m.Goals["monuments"].Value, Kingdom: m.Goals["kingdom"].Value, BurialComplete: true}
		if !m.GoalsMet(p) {
			t.Fatalf("mission%d rejects all of its exact thresholds", id)
		}
		for _, goal := range m.Objectives(p) {
			q := p
			switch goal.ID {
			case "population":
				q.Population--
			case "housing":
				q.Housing--
			case "culture":
				q.Culture--
			case "prosperity":
				q.Prosperity--
			case "monuments":
				q.Monuments--
			case "kingdom":
				q.Kingdom--
			case "burial":
				q.BurialComplete = false
			}
			if m.GoalsMet(q) {
				t.Fatalf("mission%d ignored unmet %s", id, goal.ID)
			}
		}
		if m.Goals["culture"].Enabled == 0 {
			p.Culture = 0
		}
		if m.Goals["prosperity"].Enabled == 0 {
			p.Prosperity = 0
		}
		if !m.GoalsMet(p) {
			t.Fatalf("mission%d requires a disabled criterion", id)
		}
	}
}

func TestCompletedHistoryCannotSkipAnOriginalMission(t *testing.T) {
	s, _ := New(Perwadjyt)
	s.Completed = []int{Nubt}
	if s.Validate(0) == nil {
		t.Fatal("Nubt cannot lead directly to Perwadjyt")
	}
	s.Completed = []int{Nubt, Thinis}
	if err := s.Validate(0); err != nil {
		t.Fatal(err)
	}
}

package campaign

import "testing"

func TestSurvivalAndTimeLimitDeadlines(t *testing.T) {
	m := Mission{Goals: map[string]Criterion{"population": {Enabled: 1, Value: 6000}, "survival_years": {Enabled: 1, Value: 50}}}
	p := Progress{Population: 6000}
	if m.OutcomeAtMonth(p, 599) != ScenarioRunning || m.OutcomeAtMonth(p, 600) != ScenarioWon {
		t.Fatal("survival must wait for the final month")
	}
	p.Population--
	if m.OutcomeAtMonth(p, 600) != ScenarioLost {
		t.Fatal("population must still meet the requirement at the deadline")
	}
	delete(m.Goals, "survival_years")
	m.Goals["time_limit_years"] = Criterion{Enabled: 1, Value: 50}
	if m.OutcomeAtMonth(p, 599) != ScenarioRunning || m.OutcomeAtMonth(p, 600) != ScenarioLost {
		t.Fatal("deadline ignored")
	}
	p.Population++
	if m.OutcomeAtMonth(p, 1) != ScenarioWon || m.OutcomeAtMonth(p, 600) != ScenarioWon || m.OutcomeAtMonth(p, 601) != ScenarioLost {
		t.Fatal("unexpected time limit outcome")
	}
}

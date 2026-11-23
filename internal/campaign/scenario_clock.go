package campaign

// Cleopatra manual, Survival & Time Limit Missions. A survival city is judged
// at the deadline; a time-limit city can finish early by meeting all objectives.
type ScenarioOutcome string

const (
	ScenarioRunning ScenarioOutcome = ""
	ScenarioWon     ScenarioOutcome = "won"
	ScenarioLost    ScenarioOutcome = "lost"
)

func (m Mission) OutcomeAtMonth(p Progress, elapsed int64) ScenarioOutcome {
	survival := m.Goals["survival_years"]
	if survival.Enabled != 0 {
		if elapsed < int64(survival.Value)*12 {
			return ScenarioRunning
		}
		if p.Population > 0 && (len(m.Objectives(p)) == 0 || m.GoalsMet(p)) {
			return ScenarioWon
		}
		return ScenarioLost
	}
	limit := m.Goals["time_limit_years"]
	if limit.Enabled != 0 && elapsed > int64(limit.Value)*12 {
		return ScenarioLost
	}
	if m.GoalsMet(p) {
		return ScenarioWon
	}
	if limit.Enabled != 0 && elapsed == int64(limit.Value)*12 {
		return ScenarioLost
	}
	return ScenarioRunning
}

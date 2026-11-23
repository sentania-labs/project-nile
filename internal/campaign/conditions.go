package campaign

// Ratings are observations of authoritative city state. A scenario cannot win
// by having one rating compensate for another. Original0x004ced40 tests each
// enabled threshold independently and also checks burial goods for monuments.
type Progress struct {
	Population     int
	Housing        int
	Culture        int
	Prosperity     int
	Monuments      int
	Kingdom        int
	BurialComplete bool
}

func (m Mission) Objectives(p Progress) []Objective {
	result := []Objective{}
	for _, criterion := range []struct {
		id, label string
		value     int
	}{
		{"population", "Residents", p.Population},
		{"culture", "Culture rating", p.Culture},
		{"prosperity", "Prosperity rating", p.Prosperity},
		{"monuments", "Monument rating", p.Monuments},
		{"kingdom", "Kingdom rating", p.Kingdom},
	} {
		goal := m.Goals[criterion.id]
		if goal.Enabled != 0 {
			result = append(result, Objective{criterion.id, criterion.label, criterion.value, goal.Value, criterion.value >= goal.Value})
		}
	}
	if goal := m.Goals["housing_count"]; goal.Value > 0 {
		label := "Occupied homes at the required level"
		switch m.Goals["housing_level_index"].Value {
		case 2:
			label = "Occupied Meager Shanties or better"
		case 5:
			label = "Occupied Ordinary Cottages or better"
		case 6:
			label = "Occupied Modest Homesteads or better"
		case 8:
			label = "Occupied Modest Apartments or better"
		}
		result = append(result, Objective{"housing", label, p.Housing, goal.Value, p.Housing >= goal.Value})
	}
	if m.Goals["monuments"].Enabled != 0 {
		current := 0
		if p.BurialComplete {
			current = 1
		}
		result = append(result, Objective{"burial", "Required burial provisions", current, 1, p.BurialComplete})
	}
	return result
}

func (m Mission) GoalsMet(p Progress) bool {
	objectives := m.Objectives(p)
	if len(objectives) == 0 {
		return false
	}
	for _, objective := range objectives {
		if !objective.Complete {
			return false
		}
	}
	return true
}

func (o Observation) Progress() Progress {
	return Progress{Population: o.Population, Housing: o.QualifyingHomes, Culture: o.Culture, Prosperity: o.Prosperity, Monuments: o.Monuments, Kingdom: o.Kingdom, BurialComplete: o.BurialComplete}
}

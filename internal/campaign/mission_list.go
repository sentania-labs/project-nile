package campaign

type MissionChoice struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Period string `json:"period"`
	Year   int    `json:"year"`
}

func Playable(id int) bool { return id >= Nubt && id <= Mennefer }
func MissionChoices() []MissionChoice {
	var choices []MissionChoice
	for _, m := range definitions() {
		if Playable(m.ID) {
			choices = append(choices, MissionChoice{ID: m.ID, Name: m.Name, Period: m.Period, Year: m.StartYear})
		}
	}
	return choices
}

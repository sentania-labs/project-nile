package campaign

type InvasionWarning struct {
	LeadMonths  int    `json:"lead_months"`
	ID          int    `json:"id"`
	Class       string `json:"class"`
	Months      int    `json:"months"`
	Location    int    `json:"location"`
	Retaliation bool   `json:"retaliation"`
}

func (s *State) InvasionWarnings() []InvasionWarning {
	result := []InvasionWarning{}
	for _, e := range s.Events {
		d, ok := s.template(e.SourceID)
		if !ok || d.Type != 2 || e.Status != "scheduled" {
			continue
		}
		months := int(e.DueMonth - s.EventMonth)
		if months < 0 || months > d.Deadline {
			continue
		}
		class := map[int]string{1: "foreign", 2: "egyptian", 3: "kingdom", 4: "bedouin"}[e.Item]
		result = append(result, InvasionWarning{LeadMonths: d.Deadline, ID: e.ID, Class: class, Months: months, Location: e.Location, Retaliation: e.Retaliation})
	}
	return result
}

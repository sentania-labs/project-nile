package campaign

import "slices"

// Original Gold IDs38..52 and all numerical criteria come from the licensed
// campaign records. Only carry-over policy comes from the expansion manual.
const FirstCleopatra = 38

type ExpansionMission struct {
	Name                                     string
	Population, Culture, Prosperity, Kingdom int
	Homes, HousingTier                       int
	TimeLimit, Survival                      int
	Structures                               []string
	Savings, Troops                          bool
	Region                                   string
}

type expansionCarry struct {
	Savings, Troops bool
	Region          string
}

var cleopatraCarry = [...]expansionCarry{
	{Region: "Deir el-Medina"}, {Region: "Deir el-Medina"}, {Region: "Deir el-Medina"},
	{}, {Savings: true, Troops: true}, {Savings: true, Troops: true},
	{Savings: true, Troops: true, Region: "Deir el-Medina"}, {}, {}, {},
	{Region: "Alexandria"}, {Savings: true, Troops: true, Region: "Alexandria"}, {},
	{Savings: true, Troops: true, Region: "Alexandria"}, {Savings: true, Troops: true},
}

func Expansion(id int) (ExpansionMission, bool) {
	i := id - FirstCleopatra
	if i < 0 || i >= len(cleopatraCarry) {
		return ExpansionMission{}, false
	}
	m, ok := Definition(id)
	if !ok {
		return ExpansionMission{}, false
	}
	value := func(key string) int {
		g := m.Goals[key]
		if g.Enabled == 0 {
			return 0
		}
		return g.Value
	}
	c := cleopatraCarry[i]
	return ExpansionMission{
		Name: m.Name, Population: value("population"), Culture: value("culture"), Prosperity: value("prosperity"), Kingdom: value("kingdom"),
		Homes: value("housing_count"), HousingTier: value("housing_level_index"),
		TimeLimit: value("time_limit_years"), Survival: value("survival_years"),
		Structures: slices.Clone(m.RequiredStructures), Savings: c.Savings, Troops: c.Troops, Region: c.Region,
	}, true
}

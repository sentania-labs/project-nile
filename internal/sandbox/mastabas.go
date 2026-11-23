package sandbox

import "fmt"

// Original mode5 excludes the first special part from material-bearing courses.
// The live foreman confirms10,800 bricks (27 sledges) for the small mastaba.
// Other totals follow the same table0x0057982c / original foreman0x004d5a00.
func mastabaPlan(size int) monumentPlan {
	p := monumentPlan{Family: "mastaba", Size: size}
	effort := [3]int{50, 126, 180}[size]
	p.Phases = []monumentPhase{{Name: "clearing the site", Guild: "workcamp", Effort: effort}}
	sections := 9
	if size == 1 {
		sections = 20
	}
	if size == 2 {
		sections = 35
	}
	for course := 1; course <= 6; course++ {
		p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("brick course %d", course), Guild: "bricklayers", Resource: "bricks", Stone: sections * 200, Effort: 32})
	}
	if size == 2 {
		//27 of the35 material-bearing sections have a second six-course level.
		for course := 7; course <= 12; course++ {
			p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("brick course %d", course), Guild: "bricklayers", Resource: "bricks", Stone: 27 * 200, Effort: 32})
		}
	}
	return p
}
func init() {
	for i, kind := range []string{"smallmastaba", "mediummastaba", "largemastaba"} {
		monumentPlans[kind] = mastabaPlan(i)
		width, height := 4+i*2, 10+i*4
		label := []string{"Small mastaba", "Medium mastaba", "Large mastaba"}[i]
		buildingCatalog = append(buildingCatalog, CatalogEntry{kind, label, width, height, 0, "Monuments", 0})
	}
}

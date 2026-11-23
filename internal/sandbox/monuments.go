package sandbox

import (
	"errors"
	"fmt"
)

// Independent rules recovered from the original foreman and constants.
// 8x8 footprint; six 32-block base courses, six 8-block upper courses.
// A block is 100 inventory units, a paired-mason delivery is four blocks.
// Preparation effort, work cadence and aggregated guild scheduling are tuning.
const StoneSledge = 400
const CarpenterSupply = 400

type monumentPhase struct {
	Name, Guild, Resource string
	Stone, Timber, Effort int
	WorkOnly              bool
}
type monumentPlan struct {
	Phases  []monumentPhase
	Family  string
	Size    int
	Complex bool
}

// Constants are independently transcribed numerical facts. Original foreman
// 0x004d5a00 and tables 0x00588ef0/0x005335c0 distinguish core and casing.
// Work cadence and aggregate guild staffing remain prototype scheduling.
func pyramidPlan(family string, size int, complex bool) monumentPlan {
	p := monumentPlan{Family: family, Size: size, Complex: complex}
	side := 4 + 2*size
	p.Phases = []monumentPhase{
		{Name: "leveling", Guild: "workcamp", Effort: side * side * 8},
		{Name: "surveying", Guild: "workcamp", Effort: side * side * 4},
		{Name: "excavation", Guild: "workcamp", Effort: side * side * 8},
	}
	course := 0
	for tier := 0; tier < size+2; tier++ {
		cells := (side - tier*2) * (side - tier*2)
		layers := 6
		if family == "bent" && ((size == 0 && tier == 1) || (size == 1 && tier >= 1) || (size == 2 && tier >= 2)) {
			layers = 3
		}
		resource, guild := "stone", "stonemasons"
		if family == "brick" {
			resource, guild = "bricks", "bricklayers"
		}
		for layer := 0; layer < layers; layer++ {
			course++
			p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("core course %d", course), Guild: guild, Resource: resource, Stone: cells * 200, Effort: 32})
			if tier < size+1 {
				timber := 0
				if course == 1 {
					timber = CarpenterSupply
				}
				p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("ramp %d", course), Guild: "carpenters", Timber: timber, Effort: 32})
			}
		}
	}
	if family != "stepped" {
		// The core stops one tier before the last six/three courses of each
		// individual 2x2 section; those final courses are limestone facing.
		// The loops above included that last tier. Recover its material from the
		// verified section-height distribution rather than guessing a ratio.
		p = facedPyramidPlan(p, family, size)
	}
	return p
}
func facedPyramidPlan(p monumentPlan, family string, size int) monumentPlan {
	side := 4 + 2*size
	// Each square ring has a different height. Core uses h+1 six-course
	// levels; facing is a final level over every section. Brick levels use
	// the original compressed (three-course) heights.
	coreSections := [5]int{120, 336, 720, 1320, 2184}[size]
	facingSections := side * side * 6
	if family == "bent" {
		total := [3]int{192, 444, 996}[size]
		facingSections = [3]int{84, 168, 336}[size]
		coreSections = total - facingSections
	}
	// Our sequential layer scheduler preserves the verified totals. Courses
	// are aggregated into four-block sections for the physical sledge crews.
	wanted := coreSections * 200
	used := 0
	phases := p.Phases[:0]
	for _, phase := range p.Phases {
		if phase.Stone > 0 {
			phase.Stone = min(phase.Stone, wanted-used)
			if phase.Stone == 0 {
				continue
			}
			used += phase.Stone
		}
		phases = append(phases, phase)
	}
	p.Phases = phases
	p.Phases = append(p.Phases, monumentPhase{Name: "limestone casing", Guild: "stonemasons", Resource: "limestone", Stone: facingSections * 200, Effort: 32})
	p.Phases = append(p.Phases, monumentPhase{Name: "polishing", Guild: "stonemasons", Effort: side * side * 16})
	return p
}
func smallSteppedPlan() monumentPlan {
	p := pyramidPlan("stepped", 0, false)
	// Preserve the original released phase names and save layout.
	for i := range p.Phases {
		if p.Phases[i].Resource != "" {
			p.Phases[i].Name = fmt.Sprintf("course %d", (i-3)/2+1)
		}
	}
	// Exact original small plan: six32-block then six8-block courses.
	p.Phases = p.Phases[:3]
	for course := 1; course <= 12; course++ {
		stone := 3200
		if course > 6 {
			stone = 800
		}
		p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("course %d", course), Guild: "stonemasons", Resource: "stone", Stone: stone, Effort: 32})
		if course <= 6 {
			timber := 0
			if course == 1 {
				timber = 400
			}
			p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("ramp %d", course), Guild: "carpenters", Timber: timber, Effort: 32})
		}
	}
	return p
}
func sphinxPlan() monumentPlan {
	// Original task table0x00556c00:22 assignments grouped in15 phases.
	// Five carpentry jobs and17 stone-carving jobs. No stone is delivered.
	stages := []int{1, 2, 3, 4, 4, 4, 5, 5, 5, 6, 7, 7, 7, 8, 8, 9, 10, 11, 12, 13, 14, 15}
	efforts := []int{500, 700, 400, 600, 800, 400, 600, 800, 1200, 800, 1200, 650, 600, 650, 1200, 650, 650, 900, 500, 400, 540, 450}
	p := monumentPlan{Family: "sphinx"}
	for i, stage := range stages {
		guild, timber := "stonemasons", 0
		if i < 3 || i == 12 || i == 20 {
			guild, timber = "carpenters", 400
		}
		p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("carving phase %d", stage), Guild: guild, Timber: timber, Effort: efforts[i], WorkOnly: true})
	}
	return p
}

var monumentPlans = map[string]monumentPlan{"smallstepped": smallSteppedPlan(), "sphinx": sphinxPlan()}

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"smallstepped", "Small stepped pyramid", 8, 8, 0, "Monuments", 0},
		CatalogEntry{"sphinx", "Sphinx", 18, 6, 0, "Monuments", 0},
		CatalogEntry{"stonemasons", "Stonemasons' Guild", 2, 2, 150, "Monuments", 12},
		CatalogEntry{"carpenters", "Carpenters' Guild", 2, 2, 50, "Monuments", 8},
		CatalogEntry{"bricklayers", "Bricklayers' Guild", 2, 2, 100, "Monuments", 10})
	for _, family := range []string{"stepped", "pyramid", "bent", "brick"} {
		maxSize := 5
		if family == "bent" {
			maxSize = 2
		}

		for size := 0; size < maxSize; size++ {
			key, label := monumentIdentity(family, size)
			if key == "smallstepped" {
				continue
			}
			planFamily := family
			if family == "pyramid" {
				planFamily = "true"
			}
			plan := pyramidPlan(planFamily, size, size >= 3)
			plan.Family = family
			monumentPlans[key] = plan
			side := 8 + size*4
			buildingCatalog = append(buildingCatalog, CatalogEntry{key, label, side, side, 0, "Monuments", 0})
		}
	}
}
func monumentIdentity(family string, size int) (string, string) {
	names := []string{"Small", "Medium", "Large", "Pyramid complex", "Grand pyramid complex"}
	prefixes := []string{"small", "medium", "large", "complex", "grand"}
	label := names[size] + " "
	switch family {
	case "stepped":
		label += "stepped pyramid"
	case "pyramid":
		label += "pyramid"
	case "bent":
		label += "bent pyramid"
	case "brick":
		label += "brick-core pyramid"
	}
	if size >= 3 {
		label = names[size]
		if family == "stepped" {
			label = "Stepped " + label
		}
	}
	return prefixes[size] + family, label
}
func isMonument(kind string) bool { _, ok := monumentPlans[kind]; return kind == "mastaba" || ok }
func isPyramid(kind string) bool  { _, ok := monumentPlans[kind]; return ok }
func monumentRequirement(kind string) int {
	if kind == "mastaba" {
		return 64
	}
	total := 0
	for _, p := range monumentPlans[kind].Phases {
		total += p.Stone
		if p.WorkOnly {
			total++
		}
	}
	return total
}
func monumentStaging(b Building) Point { return Point{b.X + max(0, b.Width/2-2), b.Y + b.Height} }
func currentMonumentPhase(b Building) (monumentPhase, bool) {
	p, ok := monumentPlans[b.Kind]
	if !ok || b.MonumentPhase < 0 || b.MonumentPhase >= len(p.Phases) {
		return monumentPhase{}, false
	}
	return p.Phases[b.MonumentPhase], true
}
func phaseStoneStart(b Building) int {
	n := 0
	for i, p := range monumentPlans[b.Kind].Phases {
		if i >= b.MonumentPhase {
			break
		}
		n += p.Stone
		if p.WorkOnly {
			n++
		}
	}
	return n
}
func monumentResourceCapacity(b Building, r string) int {
	if b.Kind == "carpenters" && r == "timber" {
		return 800
	}
	if b.Kind == "mastaba" && r == "bricks" {
		return max(0, 64-b.ConstructionDone)
	}
	if isPyramid(b.Kind) {
		required := 0
		for _, p := range monumentPlans[b.Kind].Phases {
			if p.Resource == r {
				required += p.Stone
			}
			if r == "timber" {
				required += p.Timber
			}
		}
		// Keep space for cargo already reserved while a phase advances. Incoming
		// reservations cap dispatch, so a later material cannot fill this staging.
		return min(StoneSledge, max(0, required-b.MonumentMaterials[r]))
	}
	return 0
}
func monumentReceives(b Building, r string) bool { return monumentResourceCapacity(b, r) > 0 }
func monumentRefresh(b *Building) {
	b.ConstructionRequired = monumentRequirement(b.Kind)
	if plan, ok := monumentPlans[b.Kind]; ok {
		b.MonumentFamily = plan.Family
		b.MonumentRequiredMaterials = map[string]int{}
		for _, p := range plan.Phases {
			if p.Resource != "" {
				b.MonumentRequiredMaterials[p.Resource] += p.Stone
			}
			b.MonumentRequiredMaterials["timber"] += p.Timber
		}
		b.MonumentStaging = monumentStaging(*b)
		b.MonumentNeed = ""
		b.MonumentGuild = ""
		if p, ok := currentMonumentPhase(*b); ok {
			b.MonumentNeed = p.Resource
			b.MonumentGuild = p.Guild
		}
		b.MonumentPercent = monumentPercent(*b)
	}

	if b.Kind == "mastaba" {
		b.Stage = monumentStage(b.ConstructionDone)
	}
	if p, ok := currentMonumentPhase(*b); ok {
		b.Stage = p.Name
	}
	if monumentComplete(*b) {
		b.Stage = "complete"
		b.WorkersNeeded = 0
	}
}
func monumentComplete(b Building) bool {
	if b.Kind == "mastaba" {
		return b.ConstructionDone >= 64
	}
	plan, ok := monumentPlans[b.Kind]
	return ok && b.MonumentPhase >= len(plan.Phases)
}
func (w World) monumentCrew(b Building, guild string) int {
	n := 0
	for _, a := range w.Walkers {
		if a.Role == "builder" && a.Source == b.ID && a.Stage == "work" {
			if origin := w.building(a.Origin); origin != nil && origin.Kind == guild && origin.Workers > 0 && origin.Damaged == "" {
				n++
			}
		}
	}
	return n
}
func (w *World) constructMonument(b *Building) {
	if b.Kind == "mastaba" {
		crew := min(b.CrewOnSite, b.Workers)
		if crew > 0 && b.Connected && w.Tick%int64(24/crew) == 0 && b.Bricks > 0 && b.ConstructionDone < 64 {
			b.Bricks--
			b.ConstructionDone++
			w.BricksConsumed++
		}
		return
	}
	phase, ok := currentMonumentPhase(*b)
	if !ok || !b.StagingAccess || b.Damaged != "" {
		return
	}
	crew := w.monumentCrew(*b, phase.Guild)
	if crew == 0 || (phase.Guild == "stonemasons" || phase.Guild == "bricklayers") && crew < 2 {
		return
	}
	if phase.Stone > 0 && inventory(*b, phase.Resource) < StoneSledge {
		return
	}
	if phase.Timber > 0 && inventory(*b, "timber") < phase.Timber {
		return
	}
	b.MonumentWork += crew
	if b.MonumentWork < phase.Effort {
		return
	}
	b.MonumentWork = 0
	if b.MonumentMaterials == nil {
		b.MonumentMaterials = map[string]int{}
	}
	if phase.Stone > 0 {
		used := w.ConsumeResource(b.ID, phase.Resource, min(StoneSledge, phaseStoneStart(*b)+phase.Stone-b.ConstructionDone))
		b.ConstructionDone += used
		b.MonumentMaterials[phase.Resource] += used
		if b.ConstructionDone < phaseStoneStart(*b)+phase.Stone {
			return
		}
	}
	if phase.Timber > 0 {
		b.MonumentMaterials["timber"] += w.ConsumeResource(b.ID, "timber", phase.Timber)
	}
	if phase.WorkOnly {
		b.ConstructionDone++
	}
	b.MonumentPhase++
}
func validateMonument(b Building) error {
	if !isPyramid(b.Kind) {
		if b.MonumentPhase != 0 || b.MonumentWork != 0 || len(b.MonumentMaterials) > 0 {
			return errors.New("monument state on unrelated building")
		}
		return nil
	}
	plan := monumentPlans[b.Kind]
	if b.MonumentPhase < 0 || b.MonumentPhase > len(plan.Phases) || b.MonumentWork < 0 {
		return errors.New("invalid monument phase")
	}
	start := phaseStoneStart(b)
	limit := start
	p, active := currentMonumentPhase(b)
	if active {
		limit += p.Stone
		if p.WorkOnly {
			limit++
		}
		if b.MonumentWork >= p.Effort {
			return errors.New("invalid monument work")
		}
	} else if b.MonumentWork != 0 {
		return errors.New("completed monument has pending work")
	}
	if b.ConstructionDone < start || b.ConstructionDone > limit {
		return errors.New("invalid monument progress")
	}
	expected := map[string]int{}
	for i, p := range plan.Phases {
		if i >= b.MonumentPhase {
			break
		}
		if p.Resource != "" {
			expected[p.Resource] += p.Stone
		}
		expected["timber"] += p.Timber
	}
	if active && p.Stone > 0 {
		expected[p.Resource] += b.ConstructionDone - start
	}
	for _, r := range []string{"stone", "bricks", "limestone", "timber"} {
		if b.MonumentMaterials[r] != expected[r] {
			return fmt.Errorf("invalid monument %s accounting", r)
		}
	}
	for r, n := range b.MonumentMaterials {
		if n < 0 || (r != "stone" && r != "bricks" && r != "limestone" && r != "timber") {
			return errors.New("unknown monument material")
		}
	}
	return nil
}

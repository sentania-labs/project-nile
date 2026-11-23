package sandbox

import "fmt"

// Material gates and later quantities come from original placement0x00463d10
// and foremen0x00520aa0 /0x004c6b80. Aggregate work order/cadence remains tuning.
func init() {
	sun := monumentPlan{Family: "suntemple", UpfrontResource: "sandstone", UpfrontAmount: 22000}
	efforts := []int{260, 260, 450, 450, 360, 520, 360, 520, 300, 290, 270, 310, 380}
	carpenters := map[int]bool{0: true, 1: true, 2: true, 3: true, 5: true, 7: true, 9: true, 11: true}
	for i, n := range efforts {
		p := monumentPhase{Name: fmt.Sprintf("obelisk carving %d", i+1), Guild: "stonemasons", Effort: n, WorkOnly: true}
		if carpenters[i] {
			p.Name = fmt.Sprintf("obelisk scaffolding %d", i+1)
			p.Guild = "carpenters"
			p.Timber = CarpenterSupply
		}
		sun.Phases = append(sun.Phases, p)
	}
	sun.Phases = append(sun.Phases,
		monumentPhase{Name: "clearing the courtyard", Guild: "workcamp", Effort: 400},
		monumentPhase{Name: "vestibule", Guild: "stonemasons", Resource: "sandstone", Stone: 400, Effort: 32},
		monumentPhase{Name: "courtyard walls", Guild: "stonemasons", Resource: "sandstone", Stone: 14400, Effort: 32},
		monumentPhase{Name: "fore temple", Guild: "stonemasons", Resource: "sandstone", Stone: 1200, Effort: 32},
		monumentPhase{Name: "laying courtyard tiles", Guild: "workcamp", Effort: 640})
	monumentPlans["suntemple"] = sun
	buildingCatalog = append(buildingCatalog, CatalogEntry{"suntemple", "Sun Temple", 11, 10, 0, "Monuments", 0})
	for variant, kind := range []string{"smallmausoleum", "mediummausoleum", "largemausoleum"} {
		p := monumentPlan{Family: "mausoleum", Size: variant, UpfrontResource: "sandstone", UpfrontAmount: 24000}
		p.Phases = []monumentPhase{{Name: "clearing and foundations", Guild: "workcamp", Effort: 800}, {Name: "first storey", Guild: "stonemasons", Resource: "sandstone", Stone: 12800, Effort: 32}}
		for ramp := 1; ramp <= 4; ramp++ {
			p.Phases = append(p.Phases, monumentPhase{Name: fmt.Sprintf("access ramp %d", ramp), Guild: "carpenters", Timber: CarpenterSupply, Effort: 200, WorkOnly: true})
		}
		p.Phases = append(p.Phases, monumentPhase{Name: "second storey", Guild: "stonemasons", Resource: "sandstone", Stone: 12800, Effort: 32}, monumentPhase{Name: "plazas and ornaments", Guild: "workcamp", Effort: 640})
		monumentPlans[kind] = p
		// Original English names are all Mausoleum; these are art variants, not sizes.
		buildingCatalog = append(buildingCatalog, CatalogEntry{kind, "Mausoleum", 22, 8, 0, "Monuments", 0})
	}
}
func discreteMonumentJobs(kind string) bool {
	f := monumentPlans[kind].Family
	return carvedMonument(kind) || f == "suntemple" || f == "mausoleum" || f == "royaltomb"
}
func monumentGuildLimit(site Building, guild string) int {
	phase, _ := currentMonumentPhase(site)
	if carvedMonument(site.Kind) || phase.WorkOnly {
		return 1
	}
	return pyramidGuildLimit(guild)
}

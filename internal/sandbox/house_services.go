package sandbox

import "slices"

var amenities = []string{"apothecary", "physician", "dentist", "mortuary", "scribalschool", "library", "policestation"}

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"apothecary", "Apothecary", 1, 1, 30, "Health", 5},
		CatalogEntry{"physician", "Physician", 2, 2, 50, "Health", 8},
		CatalogEntry{"dentist", "Dentist", 1, 1, 50, "Health", 2},
		CatalogEntry{"mortuary", "Mortuary", 2, 2, 100, "Health", 8},
		CatalogEntry{"scribalschool", "Scribal school", 2, 2, 150, "Education", 10},
		CatalogEntry{"library", "Library", 3, 3, 450, "Education", 30},
		CatalogEntry{"policestation", "Police station", 1, 1, 30, "Services", 6})
}

func supplyForService(kind string) string {
	switch kind {
	case "scribalschool", "library":
		return "papyrus"
	case "mortuary":
		return "linen"
	}
	return ""
}

func serviceGroup(role string) string {
	switch role {
	case "physician", "apothecary", "dentist", "mortuary":
		return "doctor"
	case "scribalschool", "library":
		return "teacher"
	case "policestation":
		return "policeman"
	case "goodsseller":
		return "seller"
	}
	return role
}

// Service visits are stored per provider. Mortuary linen and teaching papyrus
// must reach the institution before its employees can go out on a round.
func (w *World) visitAmenity(a Walker, b *Building) {
	i := slices.Index(amenities, a.Role)
	if i >= 0 && b.Kind == "hut" && b.Pop > 0 {
		b.ServiceUntil[i] = min(MaxTick, w.Tick+MarketDuration)
	}
}

// The mortality cadence is provisional. Exposure accumulates in occupied,
// untreated households. Contact with a physician improves the household's
// condition, as does malaria protection near the floodplain.
func (w *World) healthTick() {
	if !w.HazardsEnabled || w.Tick%(YearTicks/12) != 0 || w.population() < 200 {
		return
	}
	if w.Campaign != nil && w.Campaign.MissionID < 2 {
		return
	}
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind != "hut" || b.Pop == 0 || b.Damaged != "" {
			continue
		}
		if b.ServiceUntil[1] > w.Tick || b.ServiceUntil[0] > w.Tick && b.X <= 11 {
			b.DiseaseRisk = max(0, b.DiseaseRisk-8)
			continue
		}
		if b.Tier >= 4 {
			b.DiseaseRisk = max(0, b.DiseaseRisk-1)
			continue
		}
		b.DiseaseRisk += 1
		if !b.Food {
			b.DiseaseRisk++
		}
		if b.X <= 11 {
			b.DiseaseRisk++
		}
		if b.DiseaseRisk >= 100 {
			b.DiseaseRisk = 0
			b.Pop--
			w.DiseaseDeaths++
			w.Departures++
		}
	}
}

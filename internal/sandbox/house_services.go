package sandbox

import "slices"

var amenities = []string{"apothecary", "physician", "dentist", "mortuary", "scribalschool", "library", "policestation", "senethouse", "courthouse"}

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"apothecary", "Apothecary", 1, 1, 30, "Health", 5},
		CatalogEntry{"physician", "Physician", 2, 2, 50, "Health", 8},
		CatalogEntry{"dentist", "Dentist", 1, 1, 50, "Health", 2},
		CatalogEntry{"mortuary", "Mortuary", 2, 2, 100, "Health", 8},
		CatalogEntry{"scribalschool", "Scribal school", 2, 2, 150, "Education", 10},
		CatalogEntry{"library", "Library", 3, 3, 450, "Education", 30},
		CatalogEntry{"senethouse", "Senet house", 4, 4, 1500, "Entertainment", 25},
		CatalogEntry{"policestation", "Police station", 1, 1, 30, "Services", 6})
}

func supplyForService(kind string) string {
	switch kind {
	case "senethouse":
		return "beer"
	case "scribalschool", "library":
		return "papyrus"
	case "mortuary":
		return "linen"
	}
	return ""
}

func serviceGroup(role string) string {
	switch role {
	case "senethouse":
		return "entertainer"
	case "physician", "apothecary", "dentist", "mortuary":
		return "doctor"
	case "scribalschool", "library":
		return "teacher"
	case "policestation", "courthouse":
		return "policeman"
	case "goodsseller":
		return "seller"
	}
	return role
}

// Service visits are stored per provider. Mortuary linen and teaching papyrus
// must reach the institution before its employees can go out on a round.
func (w *World) visitAmenity(a Walker, b *Building, duration int64) {
	i := slices.Index(amenities, a.Role)
	if i >= 0 && b.Kind == "hut" && b.Pop > 0 {
		if serviceGroup(a.Role) == "policeman" {
			duration = MarketDuration
		}
		b.ServiceUntil[i] = min(MaxTick, w.Tick+duration)
		if a.Role == "courthouse" || a.Role == "policestation" {
			b.CrimeRisk = max(0, b.CrimeRisk-5)
		}
	}
}

// Retail manual pp.153–155. Disease affects one home; malaria is tied to
// river grass and marshes, and can spread to neighbors. The monthly exposure
// increments are provisional until traced in the original executable.
func (w *World) healthTick() {
	if !w.HazardsEnabled || w.Tick%(YearTicks/12) != 0 || w.population() < 200 {
		return
	}
	if w.Campaign != nil && w.Campaign.MissionID < 2 {
		return
	}
	grass := map[Point]bool{}
	for _, p := range w.Grassland {
		grass[p] = true
	}
	marsh := map[Point]bool{}
	for _, p := range w.ReedBeds {
		for y := p.Y - 2; y <= p.Y+2; y++ {
			for x := p.X - 2; x <= p.X+2; x++ {
				marsh[Point{x, y}] = true
			}
		}
	}
	var outbreaks []Point
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind != "hut" || b.Pop == 0 || b.Damaged != "" {
			continue
		}
		exposure := 0
		for _, p := range b.tiles() {
			if marsh[p] {
				exposure = 3
				break
			}
			if grass[p] && (abs(p.X-w.bank(p.Y)) <= 8 || w.westBank(p.Y) > 0 && abs(p.X-w.westBank(p.Y)) <= 8) {
				exposure = max(exposure, 2)
			}
		}
		if b.Water {
			exposure -= 2
		}
		if b.ServiceUntil[0] > w.Tick {
			exposure -= 5
		}
		if exposure == 0 && !b.Water && b.ServiceUntil[0] <= w.Tick {
			exposure = -1
		}
		b.MalariaRisk = max(0, min(100, b.MalariaRisk+exposure))
		if b.ServiceUntil[1] > w.Tick {
			b.DiseaseRisk = max(0, b.DiseaseRisk-8)
		} else if b.Tier >= 4 {
			b.DiseaseRisk = max(0, b.DiseaseRisk-1)
		} else {
			b.DiseaseRisk++
			if !b.Food {
				b.DiseaseRisk++
			}
		}
		if b.MalariaRisk >= 100 {
			outbreaks = append(outbreaks, b.point())
		}
		if b.MalariaRisk >= 100 || b.DiseaseRisk >= 100 {
			w.Departures += int64(b.Pop)
			w.DiseaseDeaths += int64(b.Pop)
			b.Pop, b.DiseaseRisk, b.MalariaRisk = 0, 0, 0
		}
	}
	for _, p := range outbreaks {
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if b.Kind == "hut" && b.Pop > 0 && nearBuilding(p, *b, 2) && b.ServiceUntil[0] <= w.Tick {
				b.MalariaRisk = min(100, b.MalariaRisk+15)
			}
		}
	}
}

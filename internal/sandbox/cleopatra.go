package sandbox

// Expansion manual pp. 3–4. The tomb freight chain requires locally made
// paint and lamps, including imported oil. Workshop batch sizes and upkeep
// rates remain provisional pending the original expansion data.
func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"hennafarm", "Henna farm", 3, 3, 40, "Industry", 0},
		CatalogEntry{"paintmaker", "Paint maker", 2, 2, 50, "Industry", 12},
		CatalogEntry{"lampmaker", "Lamp maker", 2, 2, 50, "Industry", 12},
		CatalogEntry{"artisans", "Artisans' guild", 2, 2, 150, "Monuments", 10},
		CatalogEntry{"zoo", "Zoo", 6, 6, 2000, "Entertainment", 30},
		CatalogEntry{"smallstatue", "Small statue", 1, 1, 12, "Beautification", 0},
		CatalogEntry{"mediumstatue", "Medium statue", 2, 2, 60, "Beautification", 0},
		CatalogEntry{"largestatue", "Large statue", 3, 3, 150, "Beautification", 0})
}
func expansionInput(kind, r string) bool {
	switch kind {
	case "artisans":
		return r == "clay" || r == "paint"
	case "lampmaker":
		return r == "pottery"
	case "zoo":
		return r == "meat" || r == "straw"
	}
	return false
}
func (w *World) operateZoo(b *Building) {
	if b.Workers < b.WorkersNeeded || w.Tick < b.ShowUntil[0] {
		return
	}
	if inventory(*b, "meat") < 20 || inventory(*b, "straw") < 20 {
		return
	}
	w.ConsumeResource(b.ID, "meat", 20)
	w.ConsumeResource(b.ID, "straw", 20)
	b.ShowUntil[0] = min(MaxTick, w.Tick+YearTicks/12)
}
func (w World) zooCoverage() int {
	capacity := 0
	for _, b := range w.Buildings {
		if b.Kind == "zoo" && b.Connected && b.Damaged == "" && b.Workers == b.WorkersNeeded && b.ShowUntil[0] > w.Tick {
			capacity += 7500
		}
	}
	return min(capacity, w.population())
}

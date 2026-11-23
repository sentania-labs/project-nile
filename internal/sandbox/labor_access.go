package sandbox

// A workplace needs contact with occupied homes before drawing from the city
// workforce. Coverage rises as its representative walks past homes and fades
// between visits. The monthly decay interval is calibrated on Nile's clock.
func (w World) canHire(b Building) bool {
	return b.Connected && b.Damaged == "" && (!w.LaborSeekers || b.HousesCovered > 0)
}

func (w *World) dispatchLaborSeekers() {
	if !w.LaborSeekers || w.population() == 0 {
		return
	}
	for _, b := range w.Buildings {
		if b.WorkersNeeded == 0 || !b.Connected || b.Damaged != "" || b.HousesCovered >= 50 || w.active(b.ID, "laborseeker") {
			continue
		}
		entrances := w.perimeter(b)
		if len(entrances) == 0 {
			continue
		}
		start := entrances[(int64(b.ID)+w.Tick/MigrationInterval)%int64(len(entrances))]
		route := w.patrolFor(start, "laborseeker")
		w.spawn("laborseeker", "patrol", b.ID, b.ID, b.ID, route, 0, 0)
	}
}
func (w *World) visitLaborHomes(a Walker) {
	employer := w.building(a.Origin)
	if employer == nil || employer.Damaged != "" {
		return
	}
	for _, home := range w.Buildings {
		if home.Kind == "hut" && home.Pop > 0 && home.roadAccess(Point{a.X, a.Y}) {
			employer.HousesCovered = min(300, employer.HousesCovered+1)
		}
	}
}

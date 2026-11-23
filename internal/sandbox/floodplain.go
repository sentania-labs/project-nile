package sandbox

import "errors"

// Work Camps and ten-person farm crews follow the original model/manual.
// The compressed 240-tick year and 400-fig yield are explicit prototype tuning.
const FarmCrew = 10
const FigSeasonWork = 144 * FarmCrew

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"figfarm", "Fig Farm", 3, 3, 40, "Food", 0},
		CatalogEntry{"workcamp", "Work Camp", 3, 3, 60, "Labor", 20})
}

func (w World) availableCampWorkers(campID int64) int {
	camp := w.building(campID)
	if camp == nil || camp.Kind != "workcamp" || !camp.Connected || camp.Damaged != "" {
		return 0
	}
	assigned := 0
	for _, a := range w.Walkers {
		if a.Origin == campID {
			assigned++
		}
	}
	return max(0, camp.Workers-assigned)
}

func (w *World) dispatchFarmWorkers() {
	if w.phase() < 48 || w.phase() >= 192 {
		return
	}
	for _, farm := range w.Buildings {
		if farm.Kind != "figfarm" || farm.Damaged != "" {
			continue
		}
		assigned := 0
		for _, a := range w.Walkers {
			if a.Role == "farmworker" && a.Source == farm.ID && a.Stage != "return" {
				assigned++
			}
		}
		for _, camp := range w.Buildings {
			if camp.Kind != "workcamp" || assigned >= FarmCrew {
				continue
			}
			available := min(FarmCrew-assigned, w.availableCampWorkers(camp.ID))
			if available == 0 {
				continue
			}
			route := w.landPath(w.perimeter(camp), w.accessTiles(farm))
			for range available {
				if w.spawn("farmworker", "commute", camp.ID, farm.ID, farm.ID, append([]Point{}, route...), 0, 0) {
					assigned++
				}
			}
		}
	}
}

func (w *World) advanceFarmWorker(a *Walker) bool {
	if a.Next < len(a.Route) {
		p := a.Route[a.Next]
		a.X, a.Y, a.Next = p.X, p.Y, a.Next+1
	}
	if a.Next < len(a.Route) {
		return true
	}
	if a.Stage == "return" {
		return false
	}
	farm := w.building(a.Source)
	if farm == nil || farm.Damaged != "" || w.phase() < 48 || w.phase() >= 192 {
		reverseJourney(a, a.Origin, "return")
		return true
	}
	a.Stage = "work"
	farm.Work = min(FigSeasonWork, farm.Work+1)
	return true
}

func (w *World) harvestFigs() {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind != "figfarm" {
			continue
		}
		if w.phase() == 192 {
			if b.Damaged == "" {
				w.ProduceResource(b.ID, "figs", b.Work*400/FigSeasonWork)
			}
			b.Work = 0
		}
		if w.phase() == 0 {
			b.Work = 0
		}
	}
}

func (w World) validateFarmWorker(a Walker, target Building) error {
	camp, farm := w.building(a.Origin), w.building(a.Source)
	if camp == nil || camp.Kind != "workcamp" || farm == nil || farm.Kind != "figfarm" || a.Cargo != 0 || a.Reserved != 0 || a.Resource != "" {
		return errors.New("invalid farm crew assignment")
	}
	first, last := a.Route[0], a.Route[len(a.Route)-1]
	if a.Stage == "commute" || a.Stage == "work" {
		if target.ID != farm.ID || !camp.adjacent(first) || !farm.adjacent(last) || a.Stage == "work" && a.Next != len(a.Route) {
			return errors.New("invalid farm worker commute or work position")
		}
	} else if a.Stage == "return" {
		if target.ID != camp.ID || !farm.adjacent(first) || !camp.adjacent(last) {
			return errors.New("invalid farm worker return")
		}
	} else {
		return errors.New("unknown farm worker stage")
	}
	return nil
}

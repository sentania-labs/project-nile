package sandbox

import "errors"

// One retained round trip protects every leg from road removal. LoadAt and
// DeliverAt are one-based arrival indices; no leg teleports or loses cargo.
func (w *World) dispatchStoneHaulers() {
	for _, site := range w.Buildings {
		phase, ok := currentMonumentPhase(site)
		if !ok || phase.Stone == 0 || site.Damaged != "" || !site.StagingAccess || w.ResourceFreeSpace(site.ID, phase.Resource) < StoneSledge {
			continue
		}
		for _, camp := range w.Buildings {
			if w.availableCampWorkers(camp.ID) < 1 {
				continue
			}
			sent := false
			for _, yard := range w.Buildings {
				if yard.Kind != "storageyard" || yard.Damaged != "" || !yard.Connected || yard.Workers == 0 || inventory(yard, phase.Resource)-w.reservedResource(yard.ID, phase.Resource) < StoneSledge {
					continue
				}
				collect := w.path(w.perimeter(camp), yard)
				if len(collect) == 0 {
					continue
				}
				deliver := w.path([]Point{collect[len(collect)-1]}, site)
				if len(deliver) == 0 {
					continue
				}
				back := w.path([]Point{deliver[len(deliver)-1]}, camp)
				if len(back) == 0 {
					continue
				}
				route := append(append(append([]Point{}, collect...), deliver[1:]...), back[1:]...)
				if w.spawn("stonehauler", "collect", camp.ID, camp.ID, yard.ID, route, 0, StoneSledge) {
					a := &w.Walkers[len(w.Walkers)-1]
					a.Resource = phase.Resource
					a.Site = site.ID
					a.LoadAt = len(collect)
					a.DeliverAt = len(collect) + len(deliver) - 1
					sent = true
				}
				break
			}
			if sent {
				break
			}
		}
	}
}
func (w *World) stoneHaulerArrival(a *Walker) {
	if a.Stage == "collect" && a.Next >= a.LoadAt {
		yard := w.building(a.Source)
		a.Cargo = a.Reserved
		setInventory(yard, a.Resource, inventory(*yard, a.Resource)-a.Cargo)
		a.Reserved = 0
		a.Stage = "deliver"
	}
	if a.Stage == "deliver" && a.Next >= a.DeliverAt {
		a.Cargo -= w.DepositResource(a.Site, a.Resource, a.Cargo)
		if a.Cargo == 0 {
			a.Stage = "return"
		}
	}
}
func (w *World) advanceStoneHauler(a *Walker) bool {
	w.stoneHaulerArrival(a)
	if a.Stage == "deliver" && a.Next >= a.DeliverAt {
		return true
	}
	if a.Next < len(a.Route) {
		p := a.Route[a.Next]
		a.X, a.Y, a.Next = p.X, p.Y, a.Next+1
	}
	w.stoneHaulerArrival(a)
	return a.Next < len(a.Route) || a.Cargo > 0
}
func (w World) validateStoneHauler(a Walker) error {
	camp, yard, site := w.building(a.Origin), w.building(a.Source), w.building(a.Site)
	if camp == nil || camp.Kind != "workcamp" || yard == nil || yard.Kind != "storageyard" || site == nil || !isPyramid(site.Kind) || a.Target != camp.ID || (a.Resource != "stone" && a.Resource != "limestone" && a.Resource != "bricks") || a.LoadAt < 1 || a.DeliverAt < a.LoadAt || a.DeliverAt > len(a.Route) || !camp.roadAccess(a.Route[0]) || !yard.roadAccess(a.Route[a.LoadAt-1]) || !site.roadAccess(a.Route[a.DeliverAt-1]) {
		return errors.New("invalid stone-hauling round trip")
	}
	switch a.Stage {
	case "collect":
		if a.Cargo != 0 || a.Reserved != StoneSledge || a.Next > a.LoadAt {
			return errors.New("invalid stone reservation")
		}
	case "deliver":
		if a.Cargo != StoneSledge || a.Reserved != 0 || a.Next < a.LoadAt || a.Next > a.DeliverAt {
			return errors.New("invalid stone delivery")
		}
	case "return":
		if a.Cargo != 0 || a.Reserved != 0 || a.Next < a.DeliverAt {
			return errors.New("invalid stone return")
		}
	default:
		return errors.New("unknown stone-hauler stage")
	}
	return nil
}

package sandbox

import "errors"

// Manual pp170–177 verifies self-equipped archers, weapons for infantry,
// timber-made chariots, Academy visits, ranged archery and fast chariot charges.
// Retail Normal rows40/44/94/205 establish costs/staffing. Combat statistics,
// training duration and recruitment cadence are compressed prototype rules.
func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"archerfort", "Archer Fort", 7, 4, 1000, "Military", 0},
		CatalogEntry{"chariotfort", "Charioteer Fort", 7, 4, 1000, "Military", 0},
		CatalogEntry{"chariotmaker", "Chariot Maker", 2, 2, 250, "Military", 30},
		CatalogEntry{"academy", "Military Academy", 4, 4, 1000, "Military", 20})
	goodsRecipes["chariotmaker"] = goodsRecipe{"timber", "chariots", 100, 100}
	// Original Normal building rows40/44/45/94/95/205. Military quarters
	// depress nearby housing; placing them in the neighborhood has a real cost.
	for _, kind := range []string{"fort", "archerfort", "chariotfort"} {
		influences[kind] = influence{-20, 2, 2, 6}
	}
	influences["academy"] = influence{-3, 1, 1, 3}
	influences["recruiter"] = influence{-6, 1, 1, 3}
	influences["chariotmaker"] = influence{-6, 1, 1, 6}
	influences["gatehouse"] = influence{-4, 1, 1, 4}
	influences["tower"] = influence{-6, 1, 1, 6}
}
func isFort(kind string) bool { return kind == "fort" || kind == "archerfort" || kind == "chariotfort" }

// Earlier snapshots predate original fort desirability and counted only landed
// enemies as active. Refresh derived housing once when adopting these rules;
// current snapshots still undergo the strict, non-repairing invariant check.
func (w *World) upgradeMilitary() {
	w.MilitaryVersion = 1
	w.refresh()
	w.syncInvasions()
	alive := w.passengersAtSea(0) + w.hostileWarships(0)
	for _, a := range w.Troops {
		if a.Enemy {
			alive++
		}
	}
	w.Military.InvasionActive = alive > 0
}
func fortUnit(kind string) string {
	if kind == "archerfort" {
		return "archer"
	}
	if kind == "chariotfort" {
		return "charioteer"
	}
	return "infantry"
}
func troopUnit(a Troop) string {
	if a.Kind == "" {
		return "infantry"
	}
	return a.Kind
}
func unitSupply(kind string) string {
	switch kind {
	case "infantry":
		return "weapons"
	case "charioteer":
		return "chariots"
	}
	return ""
}
func unitHP(a Troop) int {
	n := 5
	if troopUnit(a) == "sentry" {
		n = 4
	}
	if troopUnit(a) == "archer" {
		n = 3
	}
	if troopUnit(a) == "charioteer" {
		n = 7
	}
	if a.Enemy {
		n = 3
	}
	if a.Experience > 0 {
		n++
	}
	return n
}
func (w *World) recruitMilitary() {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind != "recruiter" || !goodsOperating(*b) {
			continue
		}
		b.Work = min(160, b.Work+b.Workers)
		if b.Work < 160 {
			continue
		}
		if w.recruitSentry(b) {
			continue
		}
		for _, company := range w.Companies {
			fort := w.building(company.FortID)
			if company.Soldiers >= CompanyLimit || company.Order != "return" || fort.Damaged != "" {
				continue
			}
			kind := fortUnit(fort.Kind)
			resource := unitSupply(kind)
			if resource != "" && w.ResourceAmount(b.ID, resource) < 100 {
				continue
			}
			tiles := w.garrisonTiles(*fort)
			if len(tiles) == 0 {
				continue
			}
			route := w.landPath(w.perimeter(*b), []Point{tiles[company.Soldiers%len(tiles)]})
			if len(route) == 0 || w.NextID >= MaxTick {
				continue
			}
			stage, school := "recruit", int64(0)
			for _, academy := range w.Buildings {
				if academy.Kind != "academy" || !goodsOperating(academy) {
					continue
				}
				to := w.landPath(w.perimeter(*b), w.perimeter(academy))
				onward := w.landPath(w.perimeter(academy), tiles)
				if len(to) > 0 && len(onward) > 0 {
					route = to
					school = academy.ID
					stage = "academy"
					break
				}
			}
			if resource != "" && w.ConsumeResource(b.ID, resource, 100) != 100 {
				break
			}
			a := Troop{ID: w.NextID, Kind: kind, FortID: fort.ID, Origin: b.ID, School: school, X: route[0].X, Y: route[0].Y, Stage: stage, Route: route, Next: 1}
			a.HP = unitHP(a)
			w.Troops = append(w.Troops, a)
			w.NextID++
			w.Military.Recruited++
			if kind == "archer" {
				w.Military.ArchersRecruited++
			}
			if kind == "charioteer" {
				w.Military.ChariotsRecruited++
			}
			b.Work = 0
			w.syncCompanies()
			break
		}
	}
}
func (w *World) trainMilitary(a *Troop) {
	if a.Stage == "academy" && a.Next >= len(a.Route) {
		a.Stage = "training"
		a.Training = 32
	}
	if a.Stage != "training" {
		return
	}
	school := w.building(a.School)
	if school != nil && goodsOperating(*school) && a.Training > 0 {
		a.Training--
	}
	if a.Training > 0 {
		return
	}
	fort := w.building(a.FortID)
	route := w.landPath([]Point{{a.X, a.Y}}, w.garrisonTiles(*fort))
	if len(route) == 0 {
		return
	}
	a.Experience = 1
	a.HP = unitHP(*a)
	a.Stage = "recruit"
	a.School = 0
	a.Route = route
	a.Next = 1
}
func unitRange(a Troop) int {
	if troopUnit(a) == "sentry" {
		return 6
	}
	if troopUnit(a) == "archer" {
		return 5
	}
	return 1
}
func unitMovement(a Troop, tick int64) int {
	if a.Enemy {
		if tick%2 == 0 {
			return 1
		}
		return 0
	}
	if troopUnit(a) == "archer" && tick%3 == 0 {
		return 0
	}
	if troopUnit(a) == "charioteer" {
		return 2
	}
	return 1
}
func (w World) validateUnit(a Troop) error {
	kind := troopUnit(a)
	if kind != "infantry" && kind != "archer" && kind != "charioteer" && kind != "sentry" {
		return errors.New("unknown troop type")
	}
	if a.Experience < 0 || a.Experience > 4 || a.Training < 0 || a.Training > 32 || a.HP > unitHP(a) {
		return errors.New("invalid soldier training or health")
	}
	if a.Stage == "academy" || a.Stage == "training" {
		school := w.building(a.School)
		if school == nil || school.Kind != "academy" || !school.roadAccess(a.Route[len(a.Route)-1]) {
			return errors.New("recruit lost its academy route")
		}
	} else if a.School != 0 || a.Training != 0 {
		return errors.New("soldier retained an invalid training assignment")
	}
	return nil
}

package sandbox

import (
	"errors"
	"slices"
)

// Cleopatra manual pp2–4 verifies cliffs, peasants delivering lamps, masons,
// and artisans supplied with paint and plaster from clay. No expansion binary
// is available here. Footprint, quantities and work cadence below are explicit
// prototype tuning, not transcribed original constants.
func init() {
	p := monumentPlan{Family: "royaltomb"}
	p.Phases = append(p.Phases, monumentPhase{Name: "preparing the cliff entrance", Guild: "workcamp", Effort: 160})
	for _, name := range []string{"excavating the entrance", "excavating the passage", "excavating the burial chamber", "finishing the stonework"} {
		p.Phases = append(p.Phases, monumentPhase{Name: name, Guild: "stonemasons", Resource: "lamps", Stone: 400, Effort: 240})
	}
	for _, name := range []string{"plastering the entrance", "painting the passage", "decorating the burial chamber", "sealing the royal tomb"} {
		p.Phases = append(p.Phases, monumentPhase{Name: name, Guild: "artisans", Supplies: map[string]int{"clay": 20, "paint": 20}, WorkOnly: true, Effort: 240})
	}
	monumentPlans["smallroyaltomb"] = p
	buildingCatalog = append(buildingCatalog, CatalogEntry{"smallroyaltomb", "Small Royal Burial Tomb", 4, 4, 0, "Monuments", 0})
}
func isRoyalTomb(kind string) bool { return monumentPlans[kind].Family == "royaltomb" }
func (w World) cliff(p Point) bool { return slices.Contains(w.Cliffs, p) }
func (w World) royalTombPlacement(b Building) error {
	if !isRoyalTomb(b.Kind) {
		return nil
	}
	for _, p := range b.tiles() {
		if !w.cliff(p) {
			return errors.New("place the whole Royal Tomb in a cliff face")
		}
	}
	if !w.buildable(monumentStaging(b)) {
		return errors.New("the cliff entrance needs open ground for its staging road")
	}
	return nil
}
func (w World) validateCliffs() error {
	seen := map[Point]bool{}
	for _, p := range w.Cliffs {
		if p.X < 0 || p.Y < 0 || p.X >= w.mapSize() || p.Y >= w.mapSize() || w.navigableWater(p) || seen[p] || slices.Contains(w.OreRock, p) || slices.Contains(w.QuarryRock, p) || slices.Contains(w.Woodland, p) || slices.Contains(w.Roads, p) || slices.Contains(w.Canals, p) {
			return errors.New("invalid cliff terrain")
		}
		seen[p] = true
	}
	return nil
}
func (w *World) addValleyCliffs() {
	// New worlds only. Loading an older civilization never changes its terrain.
	for _, base := range []Point{{74, 16}, {78, 64}, {8, 76}} {
		for y := 0; y < 4; y++ {
			for x := 0; x < 8; x++ {
				w.Cliffs = append(w.Cliffs, Point{base.X + x, base.Y + y})
			}
		}
	}
}
func suppliesReady(b Building, required map[string]int) bool {
	for r, n := range required {
		if inventory(b, r) < n {
			return false
		}
	}
	return true
}
func (w *World) depositBuilderSupplies(a *Walker, id int64) bool {
	for r, n := range a.Supplies {
		left := n - w.DepositResource(id, r, n)
		if left == 0 {
			delete(a.Supplies, r)
		} else {
			a.Supplies[r] = left
		}
	}
	return len(a.Supplies) == 0
}
func validBuilderSupplies(a Walker, origin Building) bool {
	if len(a.Supplies) == 0 {
		return true
	}
	if a.Role != "builder" || origin.Kind != "artisans" || a.Stage == "work" || len(a.Supplies) > 2 {
		return false
	}
	for r, n := range a.Supplies {
		if (r != "clay" && r != "paint") || n < 1 || n > 20 {
			return false
		}
	}
	return true
}

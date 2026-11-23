package sandbox

import (
	"errors"
	"fmt"
)

// Original placement0x00463d10 consumes the whole granite stock before work.
// Original jobs at0x005469b0/0x0054fcf0: four/eight carpenters, then six/ten
// masons. Efforts are transcribed facts; their mapping to Nile ticks is tuning.
func init() {
	for size, kind := range []string{"smallobelisk", "largeobelisk"} {
		efforts := []int{320, 320, 600, 600, 580, 580, 520, 520, 340, 340}
		carpenters := 4
		if size == 1 {
			efforts = []int{320, 320, 610, 610, 680, 680, 750, 750, 740, 740, 670, 670, 500, 500, 530, 530, 340, 340}
			carpenters = 8
		}
		plan := monumentPlan{Family: "obelisk", Size: size, UpfrontResource: "granite", UpfrontAmount: 10000 * (size + 1)}
		for i, n := range efforts {
			p := monumentPhase{Name: fmt.Sprintf("carving %d", i-carpenters+1), Guild: "stonemasons", Effort: n, WorkOnly: true}
			if i < carpenters {
				p.Name = fmt.Sprintf("scaffolding %d", i+1)
				p.Guild = "carpenters"
				p.Timber = CarpenterSupply
			}
			plan.Phases = append(plan.Phases, p)
		}
		monumentPlans[kind] = plan
		buildingCatalog = append(buildingCatalog, CatalogEntry{kind, []string{"Small obelisk", "Large obelisk"}[size], 3 + 2*size, 3 + 2*size, 0, "Monuments", 0})
	}
}
func carvedMonument(kind string) bool {
	p := monumentPlans[kind]
	return p.Family == "sphinx" || p.Family == "obelisk"
}
func (w World) monumentStored(resource string) int {
	amount := 0
	for _, b := range w.Buildings {
		if b.Kind == "storageyard" && b.Damaged == "" {
			amount += max(0, inventory(b, resource)-w.reservedResource(b.ID, resource))
		}
	}
	return amount
}
func (w World) monumentUpfrontCheck(b Building) error {
	p, ok := monumentPlans[b.Kind]
	if !ok || p.UpfrontAmount == 0 {
		return nil
	}
	if p.Family == "obelisk" {
		for _, old := range w.Buildings {
			if monumentPlans[old.Kind].Family == "obelisk" && !monumentComplete(old) {
				return errors.New("finish the current obelisk before placing another")
			}
		}
	}
	if p.Family == "suntemple" {
		for _, old := range w.Buildings {
			if old.Kind == "suntemple" && monumentComplete(old) {
				return errors.New("this city already has a completed Sun Temple")
			}
		}
	}
	if w.monumentStored(p.UpfrontResource) < p.UpfrontAmount {
		return fmt.Errorf("store %d %s in storage yards before placing this monument", p.UpfrontAmount, p.UpfrontResource)
	}
	return nil
}

// Caller has completed every placement/funds check. No partial debit is possible.
func (w *World) monumentUpfrontCommit(b *Building) {
	p := monumentPlans[b.Kind]
	left := p.UpfrontAmount
	if left == 0 {
		return
	}
	for _, yard := range w.Buildings {
		if yard.Kind != "storageyard" || yard.Damaged != "" {
			continue
		}
		available := max(0, inventory(yard, p.UpfrontResource)-w.reservedResource(yard.ID, p.UpfrontResource))
		left -= w.ConsumeResource(yard.ID, p.UpfrontResource, min(left, available))
		if left == 0 {
			break
		}
	}
	b.MonumentMaterials = map[string]int{p.UpfrontResource: p.UpfrontAmount}
}

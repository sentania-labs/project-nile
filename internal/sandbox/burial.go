package sandbox

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

// Original0x004094d0 increments the delivered-load ledger and debits
// selected loads*100 from storage. Requirements come from the mission,
// never invented per pyramid. Partial dispatch is a deliberate player action.
func (w World) burialRequirements() map[string]int {
	goals := map[string]int{}
	if w.Campaign == nil {
		return goals
	}
	mission, ok := campaign.Definition(w.Campaign.MissionID)
	if !ok {
		return goals
	}
	for raw, n := range mission.BurialGoods {
		id, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			continue
		}
		r := campaign.ResourceID(id)
		if r != "" {
			goals[r] = n * 100
		}
	}
	return goals
}
func (w World) burialComplete() bool {
	for r, n := range w.burialRequirements() {
		if w.BurialDelivered[r] < n {
			return false
		}
	}
	return true
}
func (w *World) DispatchBurial(resource string, loads int) error {
	required := w.burialRequirements()[resource]
	if !validResource(resource) || required == 0 || loads < 1 || loads > required/100 {
		return errors.New("choose a required burial good and a positive load count")
	}
	amount := loads * 100
	if amount > required-w.BurialDelivered[resource] {
		return errors.New("that exceeds the remaining burial provisions")
	}
	if w.requestStock(resource) < amount {
		return fmt.Errorf("need %d available %s in connected storage", amount, resource)
	}
	left := amount
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind != "storageyard" || !b.Connected || b.Damaged != "" {
			continue
		}
		n := min(left, max(0, inventory(*b, resource)-w.reservedResource(b.ID, resource)))
		left -= w.ConsumeResource(b.ID, resource, n)
		if left == 0 {
			break
		}
	}
	if w.BurialDelivered == nil {
		w.BurialDelivered = map[string]int{}
	}
	w.BurialDelivered[resource] += amount
	w.refresh()
	return nil
}
func (w World) burialView(v *View) {
	v.BurialRequirements = w.burialRequirements()
	v.BurialDelivered = w.BurialDelivered
	v.BurialAvailable = map[string]int{}
	for r := range v.BurialRequirements {
		v.BurialAvailable[r] = w.requestStock(r)
	}
}
func (w World) validateBurial() error {
	required := w.burialRequirements()
	for r, n := range w.BurialDelivered {
		if n < 0 || n%100 != 0 || n > required[r] || required[r] == 0 {
			return errors.New("invalid burial provisions ledger")
		}
		_, consumed := w.resourceCounters(r)
		if int64(n) > consumed {
			return errors.New("burial provisions were not removed from inventory")
		}
	}
	return nil
}

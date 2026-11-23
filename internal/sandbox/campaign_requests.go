package sandbox

import (
	"errors"
	"fmt"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

// Campaign requests use real city stock and their own conservation sink. They
// do not imply that a commercial trade route has opened.
func (w World) requestStock(resource string) int {
	if resource == "debens" {
		return int(max(int64(0), min(w.Treasury, 1<<30)))
	}
	stock := 0
	for _, b := range w.Buildings {
		if b.Kind == "storageyard" && b.Connected && b.Damaged == "" {
			stock += max(0, inventory(b, resource)-w.reservedResource(b.ID, resource))
		}
	}
	return stock
}

func (w *World) DispatchCampaignRequest(id int) error {
	if w.Campaign == nil {
		return errors.New("there is no active campaign")
	}
	r, err := w.Campaign.Request(id)
	if err != nil {
		return err
	}
	if r.Resource != "debens" && !validResource(r.Resource) {
		return errors.New("this request requires a capability that is not yet available")
	}
	if r.Quantity <= 0 || w.requestStock(r.Resource) < r.Quantity {
		return fmt.Errorf("request needs %d %s in available storage", r.Quantity, r.Resource)
	}
	if err = w.Campaign.DispatchRequest(id); err != nil {
		return err
	}
	if r.Resource == "debens" {
		w.Treasury -= int64(r.Quantity)
		w.Expenses += int64(r.Quantity)
		w.RequestSpent += int64(r.Quantity)
	} else {
		remaining := r.Quantity
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if b.Kind != "storageyard" || !b.Connected || b.Damaged != "" {
				continue
			}
			n := min(remaining, max(0, inventory(*b, r.Resource)-w.reservedResource(b.ID, r.Resource)))
			setInventory(b, r.Resource, inventory(*b, r.Resource)-n)
			remaining -= n
			if remaining == 0 {
				break
			}
		}
		if w.RequestSent == nil {
			w.RequestSent = map[string]int64{}
		}
		w.RequestSent[r.Resource] += int64(r.Quantity)
	}
	w.refresh()
	return nil
}

func (w *World) RefuseCampaignRequest(id int) error {
	if w.Campaign == nil {
		return errors.New("there is no active campaign")
	}
	return w.Campaign.RefuseRequest(id)
}

func (w World) campaignRequestView(v *campaign.View) {
	for i := range v.Requests {
		r := &v.Requests[i]
		r.Available = w.requestStock(r.Resource)
		r.CanDispatch = (r.Status == "requested" || r.Status == "overdue") && r.Quantity > 0 && r.Available >= r.Quantity && (r.Resource == "debens" || validResource(r.Resource))
	}
}

func (w World) validateCampaignRequests() error {
	if w.RequestSpent < 0 || w.RequestSpent > w.Expenses {
		return errors.New("invalid campaign request expense")
	}
	if w.Campaign == nil && (w.RequestSpent != 0 || len(w.RequestSent) > 0) {
		return errors.New("campaign request ledger without a campaign")
	}
	for resource, n := range w.RequestSent {
		if !validResource(resource) || n < 0 {
			return errors.New("invalid campaign goods sent")
		}
	}
	if w.Campaign != nil && w.Campaign.EventsInitialized && w.Campaign.EventMonth > w.Tick/(YearTicks/12) {
		return errors.New("campaign event clock is ahead of the simulation")
	}
	return nil
}

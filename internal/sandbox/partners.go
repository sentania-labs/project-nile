package sandbox

import (
	"errors"
	"slices"
	"strconv"
	"sync"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

type partner struct {
	Limits             map[string]int
	ID, Name           string
	Imports, Exports   []string
	Sea, InitiallyOpen bool
	Cost               int
}

var partnerCache sync.Map

func campaignPartners(id int) []partner {
	if v, ok := partnerCache.Load(id); ok {
		return v.([]partner)
	}
	var result []partner
	for _, city := range campaign.Cities(id) {
		if city.Type != 1 && city.Type != 3 && city.Type != 5 {
			continue
		}
		p := partner{ID: "city_" + strconv.Itoa(city.ID), Limits: map[string]int{}, Name: city.Name, Sea: city.Sea, InitiallyOpen: city.Open, Cost: city.Cost}
		for i, n := range city.AnnualLimits {
			if r := campaign.ResourceID(i); validResource(r) {
				p.Limits[r] = n
			}
		}
		for _, i := range city.Sells {
			if r := campaign.ResourceID(i); validResource(r) {
				p.Imports = append(p.Imports, r)
			}
		}
		for _, i := range city.Buys {
			if r := campaign.ResourceID(i); validResource(r) {
				p.Exports = append(p.Exports, r)
			}
		}
		if len(p.Exports)+len(p.Imports) > 0 {
			result = append(result, p)
		}
	}
	partnerCache.Store(id, result)
	return result
}
func (w World) partners() []partner {
	if w.Campaign == nil {
		return nil
	}
	return campaignPartners(w.Campaign.MissionID)
}
func (w *World) enablePartner(p partner) {
	if w.OpenPartners == nil {
		w.OpenPartners = map[string]bool{}
	}
	w.OpenPartners[p.ID] = true
	if p.Sea {
		w.SeaTradeOpened = true
	} else {
		w.TradeOpened = true
	}
	w.resetTradeYear()
}
func (w *World) openPartner(id string) error {
	for _, p := range w.partners() {
		if p.ID != id {
			continue
		}
		if w.OpenPartners[id] {
			return nil
		}
		if w.Treasury < int64(p.Cost) {
			return errors.New("not enough debens to open this route")
		}
		w.spend(int64(p.Cost))
		w.enablePartner(p)
		return nil
	}
	return errors.New("this trading partner is unavailable")
}
func (w World) partnerAccepts(p partner, mode, r string) bool {
	return mode == "import" && slices.Contains(p.Imports, r) || mode == "export" && slices.Contains(p.Exports, r)
}

// The visit belongs to one partner even if another route opens mid-journey.
func (w World) routeAllowance(sea bool, r, mode, assigned string) (string, int) {
	if w.Campaign == nil {
		if sea {
			return "", max(0, tradeAnnualLimit-w.SeaTradeUsed[r])
		}
		return "", max(0, tradeAnnualLimit-w.TradeUsed[r])
	}
	for _, p := range w.partners() {
		if p.Sea != sea || !w.OpenPartners[p.ID] || assigned != "" && assigned != p.ID || !w.partnerAccepts(p, mode, r) {
			continue
		}
		available := p.Limits[r] - w.PartnerUsed[p.ID][r]
		if available > 0 {
			return p.ID, available
		}
	}
	return "", 0
}
func (w *World) usePartner(id, r string, n int) {
	if id == "" {
		return
	}
	if w.PartnerUsed == nil {
		w.PartnerUsed = map[string]map[string]int{}
	}
	if w.PartnerUsed[id] == nil {
		w.PartnerUsed[id] = map[string]int{}
	}
	w.PartnerUsed[id][r] += n
}

func (w World) validatePartners() error {
	known := map[string]partner{}
	for _, p := range w.partners() {
		known[p.ID] = p
	}
	for id, open := range w.OpenPartners {
		p, ok := known[id]
		if !ok || !open || p.Sea && !w.SeaTradeOpened || !p.Sea && !w.TradeOpened {
			return errors.New("invalid campaign trade route")
		}
	}
	for id, used := range w.PartnerUsed {
		if !w.OpenPartners[id] {
			return errors.New("trade quota on a closed route")
		}
		p := known[id]
		for r, n := range used {
			if n < 0 || n > p.Limits[r] || (!slices.Contains(p.Imports, r) && !slices.Contains(p.Exports, r)) {
				return errors.New("invalid trading partner quota")
			}
		}
	}
	return nil
}

func (w World) aggregateTradeLimit(sea bool, r string) int {
	if w.Campaign == nil {
		return tradeAnnualLimit
	}
	n := 0
	for _, p := range w.partners() {
		if p.Sea == sea && w.OpenPartners[p.ID] {
			n += p.Limits[r]
		}
	}
	return n
}

package sandbox

import (
	"errors"
	"fmt"
	"slices"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

// The non-military outcome stays in the same snapshot as the originating
// campaign event. Completing the event and applying its effect are one tick.
type CityEvents struct {
	Relations        map[string]string         `json:"relations,omitempty"`
	Gifts            map[string]int64          `json:"gifts,omitempty"`
	Prices           map[string]TradePrice     `json:"prices,omitempty"`
	Limits           map[string]map[string]int `json:"limits,omitempty"`
	LandBlockedUntil int64                     `json:"land_blocked_until,omitempty"`
	SeaBlockedUntil  int64                     `json:"sea_blocked_until,omitempty"`
	FloodYear        int64                     `json:"flood_year,omitempty"`
	FloodQuality     int                       `json:"flood_quality,omitempty"`
	BloodUntil       int64                     `json:"blood_until,omitempty"`
	Last             string                    `json:"last,omitempty"`
	LastTick         int64                     `json:"last_tick,omitempty"`
}

func (w World) prevailingWage() int {
	if w.KingdomWage == 0 {
		return 30
	}
	return w.KingdomWage
}
func (w World) marketPrice(r string) (TradePrice, bool) {
	if p, ok := w.CityEvents.Prices[r]; ok {
		return p, true
	}
	return priceFor(r)
}
func (w World) partnerLimit(p partner, r string) int {
	if n, ok := w.CityEvents.Limits[p.ID][r]; ok {
		return n
	}
	return p.Limits[r]
}
func (w *World) changeDemand(resource string, direction int) {
	steps := []int{0, 1500, 2500, 4000}
	for _, p := range w.partners() {
		if !w.OpenPartners[p.ID] || (!slices.Contains(p.Imports, resource) && !slices.Contains(p.Exports, resource)) {
			continue
		}
		i := slices.Index(steps, w.partnerLimit(p, resource))
		if i < 0 {
			continue
		}
		if w.CityEvents.Limits == nil {
			w.CityEvents.Limits = map[string]map[string]int{}
		}
		if w.CityEvents.Limits[p.ID] == nil {
			w.CityEvents.Limits[p.ID] = map[string]int{}
		}
		w.CityEvents.Limits[p.ID][resource] = steps[min(len(steps)-1, max(0, i+direction))]
	}
}
func (w *World) applyCivilianEvents(events []campaign.EventEffect) {
	for _, e := range events {
		if !w.civilianEffect(e) {
			continue
		}
		_ = w.Campaign.CompleteEffect(e.ID)
		w.CityEvents.LastTick = w.Tick
		w.refresh()
	}
}
func (w *World) civilianEffect(e campaign.EventEffect) bool {
	c := &w.CityEvents
	r := campaign.ResourceID(e.Item)
	switch e.Type {
	case 6, 7:
		until := min(MaxTick, w.Tick+3*(YearTicks/12))
		if e.Type == 6 {
			c.SeaBlockedUntil = until
			c.Last = "Storms have delayed Nile merchants."
		} else {
			c.LandBlockedUntil = until
			c.Last = "The caravan road is temporarily closed."
		}
	case 8, 9:
		change := 1 + (e.ID & 3)
		if e.Type == 9 {
			change = -change
		}
		w.KingdomWage = min(45, max(5, w.prevailingWage()+change))
		c.Last = "The Kingdom has changed its wage."
	case 10:
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if b.Kind == "hut" && b.Pop > 0 {
				b.DiseaseRisk = min(99, b.DiseaseRisk+25)
			}
		}
		c.Last = "Contaminated water is making residents ill."
	case 11, 12:
		kind := "goldmine"
		c.Last = "A mine has collapsed."
		if e.Type == 12 {
			kind = "claypit"
			c.Last = "Floodwater has damaged the clay pits."
		}
		n := max(1, e.Amount)
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if b.Kind == kind && b.Damaged == "" && n > 0 {
				b.Damaged = "collapse"
				w.CollapseCount++
				n--
			}
		}
	case 13, 14:
		direction := 1
		if e.Type == 14 {
			direction = -1
		}
		w.changeDemand(r, direction)
		c.Last = "Trading partners have revised their annual demand for " + r + "."
	case 15, 16:
		p, ok := w.marketPrice(r)
		if !ok {
			return false
		}
		delta := e.Amount
		if e.Type == 16 {
			delta = -delta
		}
		p.Sell = max(0, p.Sell+delta)
		p.Buy = max(2, p.Buy+delta)
		if c.Prices == nil {
			c.Prices = map[string]TradePrice{}
		}
		c.Prices[r] = p
		c.Last = "Merchants have announced a new price for " + r + "."
	case 19:
		return w.applyCityStatus(e)
	case 21, 22:
		c.FloodYear = (w.Tick+SeasonOffset)/YearTicks + 1
		c.FloodQuality = 0
		c.Last = "The next inundation will fail."
		if e.Type == 22 {
			c.FloodQuality = 100
			c.Last = "An abundant inundation is predicted."
		}
	case 23:
		if e.Amount <= 0 {
			return false
		}
		if r == "debens" {
			amount := int64(e.Amount)
			w.Treasury += amount
			w.Income += amount
			w.Household.RoyalDebens += amount
			c.Last = fmt.Sprintf("Pharaoh has sent %d debens.", amount)
			break
		}
		if !validResource(r) {
			return false
		}
		offered := e.Amount * 100
		remaining := offered
		for _, b := range w.Buildings {
			if b.Kind != "storageyard" || b.Damaged != "" || !storageAccepts(b, r) {
				continue
			}
			n := min(remaining, w.ResourceFreeSpace(b.ID, r))
			remaining -= w.DepositResource(b.ID, r, n)
			if remaining == 0 {
				break
			}
		}
		received := offered - remaining
		if received > 0 {
			if c.Gifts == nil {
				c.Gifts = map[string]int64{}
			}
			c.Gifts[r] += int64(received)
		}
		c.Last = fmt.Sprintf("Pharaoh sent %d %s. Your storage yards accepted %d.", offered, r, received)
	case 24:
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if cropResource(b.Kind) != "" {
				b.Work = 0
			}
		}
		c.Last = "Locusts have stripped the fields."
	case 25, 26:
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if b.Kind == "hut" && b.Pop > 0 && (int64(i)+int64(e.ID))%4 == 0 {
				n := max(1, b.Pop/8)
				b.Pop -= n
				w.Departures += int64(n)
			}
		}
		if e.Type == 25 {
			c.Last = "Frogs have driven families from their homes."
		} else {
			c.Last = "A hailstorm has struck the city."
		}
	case 27:
		c.BloodUntil = min(MaxTick, w.Tick+YearTicks/4)
		c.Last = "The Nile has turned red. Fishing has stopped."
	case 28:
		for i := range w.Buildings {
			b := &w.Buildings[i]
			if b.Kind == "hut" && b.Pop > 0 {
				b.MoodSet = true
				b.Happiness = 20
				b.CrimeRisk = 70
			}
		}
		c.Last = "A crime wave has unsettled the city."
	default:
		return false
	}
	return true
}
func (w World) validateCivilianEvents() error {
	c := w.CityEvents
	if w.KingdomWage < 0 || w.KingdomWage > 100 || c.LastTick < 0 || c.LastTick > w.Tick || c.FloodQuality < 0 || c.FloodQuality > 100 {
		return errors.New("invalid city event state")
	}
	for _, until := range []int64{c.LandBlockedUntil, c.SeaBlockedUntil, c.BloodUntil} {
		if until < 0 || until > MaxTick {
			return errors.New("invalid event duration")
		}
	}
	for r, n := range c.Gifts {
		if !validResource(r) || n <= 0 {
			return errors.New("invalid royal gift ledger")
		}
	}
	for id, limits := range c.Limits {
		found := false
		for _, p := range w.partners() {
			if p.ID != id {
				continue
			}
			found = true
			for r, n := range limits {
				if !slices.Contains([]int{0, 1500, 2500, 4000}, n) || (!slices.Contains(p.Imports, r) && !slices.Contains(p.Exports, r)) {
					return errors.New("invalid event trade quota")
				}
			}
		}
		if !found {
			return errors.New("unknown event trading partner")
		}
	}
	for r, p := range c.Prices {
		base, ok := priceFor(r)
		if !ok || p.Resource != r || p.Quantity != base.Quantity || p.Buy < 2 || p.Sell < 0 {
			return errors.New("invalid market price change")
		}
	}
	return nil
}

// Original dispatcher 0x00441c50 passes the selected location to the route-ID
// lookup at 0x004385a0. Directory slot and warning lead time are separate.
func (w *World) applyCityStatus(e campaign.EventEffect) bool {
	if w.Campaign == nil {
		return false
	}
	for _, p := range campaignPartners(w.Campaign.MissionID) {
		if p.Route != e.Location {
			continue
		}
		var status, message string
		switch e.Subtype {
		case 0:
			status = "closed"
			message = p.Name + " has fallen to an enemy. Trade has stopped."
		case 1:
			w.CityEvents.Last = "Pharaoh has captured " + p.Name + "."
			return true
		case 2:
			status = "open"
			message = "Pharaoh has opened trade with " + p.Name + "."
		case 3:
			status = "closed"
			message = "The trade route to " + p.Name + " has closed."
		default:
			return false
		}
		if w.CityEvents.Relations == nil {
			w.CityEvents.Relations = map[string]string{}
		}
		w.CityEvents.Relations[p.ID] = status
		if status == "open" {
			w.enablePartner(p)
		} else {
			delete(w.OpenPartners, p.ID)
		}
		w.CityEvents.Last = message
		return true
	}
	return false
}

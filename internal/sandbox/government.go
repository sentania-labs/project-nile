package sandbox

import "errors"

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"familymansion", "Family mansion", 4, 4, 400, "Government", 0},
		CatalogEntry{"dynastymansion", "Dynasty mansion", 4, 4, 700, "Government", 0},
		CatalogEntry{"townpalace", "Town palace", 5, 5, 1000, "Government", 25},
		CatalogEntry{"citypalace", "City palace", 6, 6, 1200, "Government", 30})
}
func isResidence(kind string) bool {
	return kind == "residence" || kind == "familymansion" || kind == "dynastymansion"
}
func isPalace(kind string) bool {
	return kind == "palace" || kind == "townpalace" || kind == "citypalace"
}

// Gift costs and diminishing favor follow Akhenaten's kingdom data tables.
// The ordinary treasury and a governor's personal savings are separate ledgers.
type RoyalHousehold struct {
	Interest    int64 `json:"interest,omitempty"`
	GiftsSent   int   `json:"gifts_sent,omitempty"`
	LastGift    int64 `json:"last_gift,omitempty"`
	Reputation  *int  `json:"reputation,omitempty"`
	Donated     int64 `json:"donated,omitempty"`
	RoyalDebens int64 `json:"royal_debens,omitempty"`
}

func (w World) kingdomStanding() int {
	if w.Campaign != nil {
		return w.Campaign.Kingdom
	}
	if w.Household.Reputation != nil {
		return *w.Household.Reputation
	}
	return 50
}
func (w World) giftCosts() [3]int64 {
	return [3]int64{w.Savings/8 + 20, w.Savings/4 + 50, w.Savings/2 + 100}
}
func (w *World) SendRoyalGift(size int) error {
	if size < 0 || size > 2 {
		return errors.New("choose a modest, generous or lavish gift")
	}
	cost := w.giftCosts()[size]
	if w.Savings < cost {
		return errors.New("insufficient personal savings for this gift")
	}
	if w.Tick-w.Household.LastGift >= YearTicks {
		w.Household.GiftsSent = 0
	}
	gains := [5][3]int{{3, 5, 10}, {1, 3, 5}, {0, 1, 3}, {0, 0, 1}, {}}
	gain := gains[w.Household.GiftsSent][size]
	w.Savings -= cost
	w.Household.LastGift = w.Tick
	w.Household.GiftsSent = min(4, w.Household.GiftsSent+1)
	if w.Campaign != nil {
		w.Campaign.ChangeKingdom(gain)
	} else {
		rating := min(100, w.kingdomStanding()+gain)
		w.Household.Reputation = &rating
	}
	return nil
}
func (w World) validateRoyalHousehold() error {
	h := w.Household
	if h.Interest < 0 || h.Interest > w.Expenses || h.Donated < 0 || h.RoyalDebens < 0 || h.Donated+h.RoyalDebens > w.Income || h.LastGift < 0 || h.LastGift > w.Tick || h.GiftsSent < 0 || h.GiftsSent > 4 || h.Reputation != nil && (w.Campaign != nil || *h.Reputation < 0 || *h.Reputation > 100) {
		return errors.New("invalid governor accounts")
	}
	return nil
}

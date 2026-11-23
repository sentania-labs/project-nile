package sandbox

import (
	"errors"
	"slices"
)

const tradeRouteID = "kingdom_road"
const tradeOpeningCost = 200 // Authored sandbox route, not an original campaign route.
const tradeAnnualLimit = 1500
const tradeLoad = 100

type TradePolicy struct {
	Mode  string `json:"mode"`
	Limit int    `json:"limit"`
}
type TradePrice struct {
	Resource string `json:"resource"`
	Buy      int    `json:"buy"`
	Sell     int    `json:"sell"`
	Quantity int    `json:"quantity"`
}
type TradeRouteView struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Open        bool           `json:"open"`
	Available   bool           `json:"available"`
	OpenCost    int            `json:"open_cost"`
	AnnualLimit int            `json:"annual_limit"`
	Used        map[string]int `json:"used"`
}

// Original English retail Pharaoh.exe SHA256
// b6169404b9e22f228d07d5cabf889d25b8cd46d53023af9c9d0d2d6034cb741a:
// file offset 0x166cb8, little-endian uint32 buy/sell price pairs. Prices are
// per100 inventory units. "meat" means game meat; jewelry is luxury goods.
// Route identity, opening charge, visit cadence and partial-load rounding below
// are original-authored prototype rules, not verified original behavior.
var tradePrices = []TradePrice{
	{"grain", 28, 21, 100}, {"meat", 44, 34, 100}, {"figs", 33, 25, 100},
	{"straw", 21, 16, 100}, {"clay", 38, 29, 100}, {"bricks", 150, 120, 100},
	{"pottery", 140, 105, 100}, {"barley", 48, 37, 100}, {"beer", 185, 140, 100},
	{"flax", 54, 42, 100}, {"linen", 210, 160, 100}, {"jewelry", 310, 150, 100},
	{"timber", 225, 170, 100}, {"reeds", 31, 23, 100}, {"papyrus", 200, 165, 100},
	{"stone", 38, 29, 100}, {"limestone", 46, 35, 100}, {"granite", 60, 45, 100}, {"sandstone", 40, 32, 100},
}

func priceFor(r string) (TradePrice, bool) {
	for _, p := range tradePrices {
		if p.Resource == r {
			return p, true
		}
	}
	return TradePrice{}, false
}
func (w World) tradeAvailable() bool { return w.Campaign == nil }
func (w *World) OpenTrade(route string) error {
	if route != tradeRouteID {
		return errors.New("unknown trade route")
	}
	if !w.tradeAvailable() {
		return errors.New("this mission has no available foreign trade route")
	}
	if w.TradeOpened {
		return nil
	}
	if w.Treasury < tradeOpeningCost {
		return errors.New("not enough debens to open this route")
	}
	w.TradeOpened = true
	w.Treasury -= tradeOpeningCost
	w.Expenses += tradeOpeningCost
	w.resetTradeYear()
	return nil
}
func (w *World) SetTradePolicy(r, mode string, limit int) error {
	if _, ok := priceFor(r); !ok || !validResource(r) {
		return errors.New("resource is not traded on this route")
	}
	if mode != "off" && mode != "import" && mode != "export" || limit < 0 || limit > 3200 {
		return errors.New("trade mode must be off, import, or export; stock limit must be 0..3200")
	}
	if !w.tradeAvailable() {
		return errors.New("this mission has no available foreign trade route")
	}
	if w.TradePolicies == nil {
		w.TradePolicies = map[string]TradePolicy{}
	}
	w.TradePolicies[r] = TradePolicy{mode, limit}
	return nil
}
func (w *World) resetTradeYear() {
	year := (w.Tick+SeasonOffset)/YearTicks + 1
	if w.TradeOpened && (w.TradeYear != year || w.TradeUsed == nil) {
		w.TradeYear = year
		w.TradeUsed = map[string]int{}
	}
}

// Traders own their cargo until the import is unloaded, and own exported goods
// from the moment of payment. Their load is never part of city inventory.
func (w World) cityResource(r string) int {
	n := 0
	for _, b := range w.Buildings {
		n += inventory(b, r)
	}
	for _, a := range w.Walkers {
		ar := a.Resource
		if ar == "" && (a.Role == "cart" || a.Role == "buyer" || a.Role == "seller") {
			ar = "grain"
		}
		if a.Role != "trader" && ar == r {
			n += a.Cargo
		}
	}
	return n
}
func (w *World) dispatchTrader() {
	if !w.TradeOpened || !w.tradeAvailable() || w.Tick < w.NextTraderTick {
		return
	}
	for _, a := range w.Walkers {
		if a.Role == "trader" {
			return
		}
	}
	for offset := 0; offset < len(tradePrices); offset++ {
		index := (w.TradeCursor + offset) % len(tradePrices)
		price := tradePrices[index]
		r := price.Resource
		policy := w.TradePolicies[r]
		quota := tradeAnnualLimit - w.TradeUsed[r]
		if quota <= 0 || policy.Mode == "off" || policy.Mode == "" {
			continue
		}
		for _, b := range w.Buildings {
			if b.Kind != "storageyard" || !goodsOperating(b) {
				continue
			}
			route := w.path([]Point{w.entry()}, b)
			if len(route) == 0 {
				continue
			}
			amount := 0
			if policy.Mode == "import" && storageAccepts(b, r) {
				amount = min(tradeLoad, quota, max(0, policy.Limit-w.cityResource(r)), w.ResourceFreeSpace(b.ID, r), int(max(int64(0), min(w.Treasury, 1000000))*100/int64(price.Buy)))
			} else if policy.Mode == "export" {
				amount = min(tradeLoad, quota, max(0, w.cityResource(r)-policy.Limit), inventory(b, r)-w.reservedResource(b.ID, r))
				if amount*price.Sell/100 == 0 {
					amount = 0
				}
			}
			if amount <= 0 {
				continue
			}
			cargo, reserved := amount, 0
			if policy.Mode == "export" {
				cargo, reserved = 0, amount
			}
			if w.spawn("trader", policy.Mode, 0, b.ID, b.ID, route, cargo, reserved) {
				w.Walkers[len(w.Walkers)-1].Resource = r
				w.TradeCursor = (index + 1) % len(tradePrices)
				w.NextTraderTick = min(MaxTick, w.Tick+32)
				return
			}
		}
	}
}
func (w *World) advanceTrader(a *Walker) bool {
	if a.Next < len(a.Route) {
		p := a.Route[a.Next]
		a.X, a.Y = p.X, p.Y
		a.Next++
	}
	if a.Next < len(a.Route) {
		return true
	}
	if a.Stage == "leave" {
		return false
	}
	b := w.building(a.Target)
	r := a.Resource
	p, _ := priceFor(r)
	policy := w.TradePolicies[r]
	quota := max(0, tradeAnnualLimit-w.TradeUsed[r])
	if goodsOperating(*b) && a.Stage == policy.Mode {
		if a.Stage == "import" && storageAccepts(*b, r) {
			amount := min(a.Cargo, quota, max(0, policy.Limit-w.cityResource(r)), max(0, resourceLimit(*b, r)-inventory(*b, r)), int(max(int64(0), min(w.Treasury, 1000000))*100/int64(p.Buy)))
			if amount > 0 {
				cost := int64((amount*p.Buy + 99) / 100)
				w.DepositResource(b.ID, r, amount)
				a.Cargo -= amount
				w.Treasury -= cost
				w.Expenses += cost
				w.ImportSpent += cost
				if w.ResourceImported == nil {
					w.ResourceImported = map[string]int64{}
				}
				w.ResourceImported[r] += int64(amount)
				w.TradeUsed[r] += amount
			}
		} else if a.Stage == "export" {
			amount := min(a.Reserved, quota, max(0, w.cityResource(r)-policy.Limit), inventory(*b, r))
			earned := int64(amount * p.Sell / 100)
			if earned > 0 {
				setInventory(b, r, inventory(*b, r)-amount)
				a.Cargo = amount
				w.Treasury += earned
				w.Income += earned
				w.ExportEarned += earned
				if w.ResourceExported == nil {
					w.ResourceExported = map[string]int64{}
				}
				w.ResourceExported[r] += int64(amount)
				w.TradeUsed[r] += amount
			}
		}
	}
	a.Reserved = 0
	a.Stage = "leave"
	// Return along the protected arrival route. This also works with legacy saves
	// whose original entry predates the through-going Kingdom Road.
	a.Route = slices.Clone(a.Route)
	slices.Reverse(a.Route)
	a.Next = 1
	return true
}
func (w World) tradeView(v *View) {
	v.TradeRoutes = []TradeRouteView{{tradeRouteID, "Kingdom Road caravan", w.TradeOpened, w.tradeAvailable(), tradeOpeningCost, tradeAnnualLimit, w.TradeUsed}}
	v.TradePolicies = w.TradePolicies
	v.TradePrices = tradePrices
	v.ResourceImported = w.ResourceImported
	v.ResourceExported = w.ResourceExported
	v.ImportSpent = w.ImportSpent
	v.ExportEarned = w.ExportEarned
	if v.TradeRoutes[0].Used == nil {
		v.TradeRoutes[0].Used = map[string]int{}
	}
	if v.TradePolicies == nil {
		v.TradePolicies = map[string]TradePolicy{}
	}
	if v.ResourceImported == nil {
		v.ResourceImported = map[string]int64{}
	}
	if v.ResourceExported == nil {
		v.ResourceExported = map[string]int64{}
	}
}
func (w World) validateTrade() error {
	if w.ImportSpent < 0 || w.ExportEarned < 0 || w.ImportSpent > w.Expenses || w.ExportEarned > w.Income || w.NextTraderTick < 0 || w.NextTraderTick > MaxTick || w.TradeCursor < 0 || w.TradeCursor >= len(tradePrices) || w.TradeYear < 0 || w.TradeYear > (w.Tick+SeasonOffset)/YearTicks+1 {
		return errors.New("invalid trade ledger")
	}
	if !w.TradeOpened && (w.ImportSpent != 0 || w.ExportEarned != 0 || len(w.ResourceImported) > 0 || len(w.ResourceExported) > 0 || len(w.TradeUsed) > 0) {
		return errors.New("trade ledger without an open route")
	}
	for _, ledger := range []map[string]int64{w.ResourceImported, w.ResourceExported} {
		for r, n := range ledger {
			if _, ok := priceFor(r); !ok || n < 0 {
				return errors.New("invalid resource trade ledger")
			}
		}
	}
	for r, n := range w.TradeUsed {
		if _, ok := priceFor(r); !ok || n < 0 || n > tradeAnnualLimit || int64(n) > w.ResourceImported[r]+w.ResourceExported[r] {
			return errors.New("invalid annual trade use")
		}
	}
	for r, p := range w.TradePolicies {
		if _, ok := priceFor(r); !ok || (p.Mode != "off" && p.Mode != "import" && p.Mode != "export") || p.Limit < 0 || p.Limit > 3200 {
			return errors.New("invalid trade policy")
		}
	}
	count := 0
	for _, a := range w.Walkers {
		if a.Role == "trader" {
			count++
		}
	}
	if count > 1 {
		return errors.New("overlapping route caravans")
	}
	return nil
}
func (w World) validateTrader(a Walker, target Building) error {
	if _, ok := priceFor(a.Resource); !ok || !w.TradeOpened || target.Kind != "storageyard" || a.Origin != 0 || a.Source != target.ID || a.Cargo > tradeLoad || a.Reserved > tradeLoad {
		return errors.New("invalid trader")
	}
	if a.Stage == "leave" {
		if a.Reserved != 0 || a.Route[len(a.Route)-1] != w.entry() || !target.roadAccess(a.Route[0]) {
			return errors.New("invalid departing trader")
		}
		return nil
	}
	if a.Route[0] != w.entry() {
		return errors.New("trader did not enter at Kingdom Road")
	}
	if a.Stage == "import" {
		if a.Cargo <= 0 || a.Reserved != 0 {
			return errors.New("invalid import load")
		}
	} else if a.Stage == "export" {
		if a.Cargo != 0 || a.Reserved <= 0 {
			return errors.New("invalid export reservation")
		}
	} else {
		return errors.New("invalid trader stage")
	}
	return nil
}

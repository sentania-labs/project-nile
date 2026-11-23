package sandbox

import (
	"errors"
	"slices"
)

const seaRouteID = "nile_route"
const seaOpeningCost = 300 // Authored sandbox partner.
// Original reference ship capacity is1200 units. A single-resource prototype
// visit uses the same hold; dock workers carry it in100-unit road journeys.
const merchantLoad = 1200

func (w *World) openSeaTrade() error {
	if !w.tradeAvailable() {
		return errors.New("this mission has no sea trade partner")
	}
	if w.SeaTradeOpened {
		return nil
	}
	if w.Treasury < seaOpeningCost {
		return errors.New("not enough debens to open the Nile route")
	}
	w.SeaTradeOpened = true
	w.Treasury -= seaOpeningCost
	w.Expenses += seaOpeningCost
	w.resetTradeYear()
	return nil
}
func (w World) seaEntry() Point { return Point{w.westBank(0), 0} }
func (w *World) dispatchMerchantShip() {
	if !w.SeaTradeOpened || !w.tradeAvailable() || w.Tick < w.NextMerchantTick {
		return
	}
	for _, a := range w.Boats {
		if a.Kind == "merchantship" {
			return
		}
	}
	for offset := 0; offset < len(tradePrices); offset++ {
		index := (w.SeaCursor + offset) % len(tradePrices)
		price := tradePrices[index]
		r := price.Resource
		policy := w.TradePolicies[r]
		partner, quota := w.routeAllowance(true, r, policy.Mode, "")
		if quota <= 0 {
			continue
		}
		for _, dock := range w.Buildings {
			if dock.Kind != "dock" || !goodsOperating(dock) {
				continue
			}
			route := w.waterPath(w.seaEntry(), w.berth(dock))
			if len(route) == 0 {
				continue
			}
			amount := 0
			if policy.Mode == "import" {
				amount = min(merchantLoad, quota, max(0, policy.Limit-w.cityResource(r)), w.ResourceFreeSpace(dock.ID, r), int(max(int64(0), min(w.Treasury, 1000000))*100/int64(price.Buy)))
			} else if policy.Mode == "export" {
				amount = min(merchantLoad, quota, max(0, w.cityResource(r)-policy.Limit), inventory(dock, r)-w.reservedResource(dock.ID, r))
				if amount*price.Sell/100 == 0 {
					amount = 0
				}
			}
			if amount <= 0 || w.NextID >= MaxTick {
				continue
			}
			cargo, reserved := amount, 0
			if policy.Mode == "export" {
				cargo, reserved = 0, amount
			}
			w.Boats = append(w.Boats, Boat{Partner: partner, ID: w.NextID, Kind: "merchantship", Home: dock.ID, X: route[0].X, Y: route[0].Y, Stage: policy.Mode, Resource: r, Cargo: cargo, Reserved: reserved, Route: route, Next: 1})
			w.NextID++
			w.SeaCursor = (index + 1) % len(tradePrices)
			w.NextMerchantTick = min(MaxTick, w.Tick+96)
			return
		}
	}
}
func (w *World) arriveMerchantShip(a *Boat) bool {
	if a.Stage == "leave" {
		return false
	}
	dock := w.building(a.Home)
	r := a.Resource
	p, _ := priceFor(r)
	policy := w.TradePolicies[r]
	_, quota := w.routeAllowance(true, r, a.Stage, a.Partner)
	if goodsOperating(*dock) && a.Stage == policy.Mode {
		if a.Stage == "import" {
			n := min(a.Cargo, quota, max(0, policy.Limit-w.cityResource(r)), max(0, resourceLimit(*dock, r)-inventory(*dock, r)), int(max(int64(0), min(w.Treasury, 1000000))*100/int64(p.Buy)))
			if n > 0 {
				cost := int64((n*p.Buy + 99) / 100)
				w.DepositResource(dock.ID, r, n)
				a.Cargo -= n
				w.Treasury -= cost
				w.Expenses += cost
				w.ImportSpent += cost
				if w.ResourceImported == nil {
					w.ResourceImported = map[string]int64{}
				}
				w.ResourceImported[r] += int64(n)
				w.SeaTradeUsed[r] += n
				w.usePartner(a.Partner, r, n)
			}
		} else if a.Stage == "export" {
			n := min(a.Reserved, quota, max(0, w.cityResource(r)-policy.Limit), inventory(*dock, r))
			earned := int64(n * p.Sell / 100)
			if earned > 0 {
				setInventory(dock, r, inventory(*dock, r)-n)
				a.Cargo = n
				w.Treasury += earned
				w.Income += earned
				w.ExportEarned += earned
				if w.ResourceExported == nil {
					w.ResourceExported = map[string]int64{}
				}
				w.ResourceExported[r] += int64(n)
				w.SeaTradeUsed[r] += n
				w.usePartner(a.Partner, r, n)
			}
		}
	}
	a.Reserved = 0
	a.Stage = "leave"
	a.Route = slices.Clone(a.Route)
	slices.Reverse(a.Route)
	a.Next = 1
	return true
}
func (w World) dockNeeds(b Building, r string) int {
	if b.Kind != "dock" || !w.SeaTradeOpened || w.TradePolicies[r].Mode != "export" {
		return 0
	}
	waiting := 0
	for _, dock := range w.Buildings {
		if dock.Kind == "dock" {
			waiting += inventory(dock, r) + w.incomingResource(dock.ID, r)
		}
	}
	_, quota := w.routeAllowance(true, r, "export", "")
	return min(max(0, w.cityResource(r)-w.TradePolicies[r].Limit-waiting), quota, w.ResourceFreeSpace(b.ID, r))
}
func (w World) validateMerchantShip(a Boat, home Building) error {
	if _, ok := priceFor(a.Resource); !ok || !w.SeaTradeOpened || home.Kind != "dock" || a.Source != 0 || a.WaitUntil != 0 {
		return errors.New("invalid merchant ship")
	}
	if a.Stage == "leave" {
		if a.Reserved != 0 || a.Route[len(a.Route)-1] != w.seaEntry() {
			return errors.New("invalid departing merchant ship")
		}
		return nil
	}
	if a.Route[0] != w.seaEntry() || a.Route[len(a.Route)-1] != w.berth(home) {
		return errors.New("merchant ship missed its dock")
	}
	if a.Stage == "import" {
		if a.Cargo <= 0 || a.Reserved != 0 {
			return errors.New("invalid sea import")
		}
	} else if a.Stage == "export" {
		if a.Cargo != 0 || a.Reserved <= 0 {
			return errors.New("invalid sea export")
		}
	} else {
		return errors.New("invalid merchant ship stage")
	}
	return nil
}
func (w World) validateSeaTrade() error {
	if w.SeaCursor < 0 || w.SeaCursor >= len(tradePrices) || w.NextMerchantTick < 0 || w.NextMerchantTick > MaxTick {
		return errors.New("invalid sea trade schedule")
	}
	for r, n := range w.SeaTradeUsed {
		if _, ok := priceFor(r); !ok || !w.SeaTradeOpened || n < 0 || n > w.aggregateTradeLimit(true, r) || int64(n) > w.ResourceImported[r]+w.ResourceExported[r] {
			return errors.New("invalid sea trade quota")
		}
	}
	count := 0
	for _, a := range w.Boats {
		if a.Kind == "merchantship" {
			count++
		}
	}
	if count > 1 {
		return errors.New("duplicate merchant convoy")
	}
	return nil
}

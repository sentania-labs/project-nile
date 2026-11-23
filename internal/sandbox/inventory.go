package sandbox

import (
	"errors"
	"maps"
	"sync"
)

type ResourceInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
}

var foodKinds = []string{"grain", "meat", "figs", "pomegranates", "lettuce", "chickpeas", "beef", "fish"}
var bazaarGoods = []string{"grain", "meat", "figs", "pomegranates", "lettuce", "chickpeas", "beef", "fish", "pottery", "beer", "linen", "jewelry"}
var resourceCatalog = []ResourceInfo{
	{"chariots", "Chariots", "units"},
	{"pomegranates", "Pomegranates", "units"}, {"lettuce", "Lettuce", "units"}, {"chickpeas", "Chickpeas", "units"}, {"beef", "Beef", "units"}, {"fish", "Fish", "units"},
	{"gems", "Gemstones", "units"}, {"copper", "Copper", "units"}, {"weapons", "Weapons", "units"},
	{"stone", "Plain stone", "units"}, {"timber", "Timber", "units"}, {"limestone", "Limestone", "units"}, {"granite", "Granite", "units"}, {"sandstone", "Sandstone", "units"},
	{"grain", "Grain", "units"}, {"meat", "Game meat", "units"}, {"figs", "Figs", "units"}, {"barley", "Barley", "units"}, {"beer", "Beer", "units"},
	{"reeds", "Reeds", "units"}, {"papyrus", "Papyrus", "units"}, {"clay", "Clay", "units"}, {"pottery", "Pottery", "units"},
	{"flax", "Flax", "units"}, {"linen", "Linen", "units"}, {"jewelry", "Jewelry", "units"}, {"straw", "Straw", "units"}, {"bricks", "Bricks", "units"},
}

func validResource(resource string) bool {
	for _, r := range resourceCatalog {
		if r.ID == resource {
			return true
		}
	}
	return false
}
func newResource(resource string) bool {
	return validResource(resource) && resource != "grain" && resource != "clay" && resource != "straw" && resource != "bricks"
}
func inventory(b Building, r string) int {
	switch r {
	case "grain":
		return b.Stock
	case "clay":
		return b.Clay
	case "straw":
		return b.Straw
	case "bricks":
		return b.Bricks
	}
	return b.Goods[r]
}
func setInventory(b *Building, r string, n int) {
	switch r {
	case "grain":
		b.Stock = n
	case "clay":
		b.Clay = n
	case "straw":
		b.Straw = n
	case "bricks":
		b.Bricks = n
	default:
		if b.Goods == nil {
			b.Goods = map[string]int{}
		}
		if n == 0 {
			delete(b.Goods, r)
		} else {
			b.Goods[r] = n
		}
	}
}
func resourceLimit(b Building, r string) int {
	if !validResource(r) {
		return 0
	}
	if extracted[b.Kind] == r || b.Kind == "woodcutter" && r == "timber" {
		return 400
	}
	if b.Kind == "dock" {
		return 1200
	}
	if b.Kind == "shipwright" && r == "timber" {
		return 800
	}
	if b.Kind == "fishingwharf" && r == "fish" {
		return 400
	}
	if n := monumentResourceCapacity(b, r); n > 0 {
		return n
	}
	if r == supplyForService(b.Kind) {
		return 200
	}
	if r == campCrops[b.Kind] {
		return 400
	}
	if b.Kind == "cattleranch" && r == "straw" {
		return 32
	}
	if r == "grain" && b.AgricultureVersion > 0 {
		switch b.Kind {
		case "farm":
			return 400
		case "granary":
			return 1600
		case "storageyard":
			return 800
		case "bazaar":
			return 400
		case "hut":
			return 32 * b.lotSide() * b.lotSide()
		}
		return 0
	}
	if r == "grain" {
		return stockCapacity(b.Kind) * b.lotSide() * b.lotSide()
	}
	if r == "clay" || r == "straw" || r == "bricks" {
		if b.Kind == "potter" && r == "clay" {
			return 24
		}
		return materialCapacity(b, r)
	}
	switch b.Kind {
	case "storageyard":
		return 800
	case "recruiter":
		if r == "weapons" || r == "chariots" {
			return 400
		}
	case "coppermine":
		if r == "copper" {
			return 400
		}
	case "huntinglodge":
		if r == "meat" {
			return 400
		}
	case "granary":
		if foodResource(r) {
			return 1600
		}
	case "bazaar":
		if foodResource(r) {
			return 400
		}
		if householdGood(r) {
			return 200
		}
	case "hut":
		if foodResource(r) || householdGood(r) {
			return 32 * b.lotSide() * b.lotSide()
		}
	case "figfarm":
		if r == "figs" {
			return 400
		}
	case "barleyfarm":
		if r == "barley" {
			return 400
		}
	case "flaxfarm":
		if r == "flax" {
			return 400
		}
	case "reedgatherer":
		if r == "reeds" {
			return 400
		}
	}
	if recipe, ok := goodsRecipes[b.Kind]; ok {
		if r == recipe.Input {
			return 200
		}
		if r == recipe.Output {
			return 400
		}
	}
	return 0
}
func foodResource(r string) bool {
	for _, kind := range foodKinds {
		if r == kind {
			return true
		}
	}
	return false
}
func householdGood(r string) bool {
	return r == "pottery" || r == "beer" || r == "linen" || r == "jewelry"
}
func (w World) ResourceAmount(id int64, r string) int {
	b := w.building(id)
	if b == nil {
		return 0
	}
	return inventory(*b, r)
}
func (w World) HouseGoods(id int64, r string) int { return w.ResourceAmount(id, r) }
func (w World) incomingResource(id int64, r string) int {
	n := w.boatIncoming(id, r)
	for _, a := range w.Walkers {
		if a.Resource != r {
			continue
		}
		switch a.Role {
		case "builder":
			if a.Source == id || a.Origin == id {
				n += a.Cargo
			}
		case "stonehauler":
			if a.Site == id {
				n += a.Cargo + a.Reserved
			}
		case "trader":
			if a.Stage == "import" && a.Target == id {
				n += a.Cargo
			}
		case "freight":
			if a.Stage == "deliver" && a.Target == id {
				n += a.Cargo
			}
		case "goodsbuyer":
			if a.Origin == id {
				n += a.Cargo + a.Reserved
			}
		case "goodsseller":
			if a.Origin == id {
				n += a.Cargo
			}
		case "hunter", "reedharvester", "lumberjack":
			if a.Origin == id {
				n += max(a.Cargo, a.Reserved)
			}
		}
	}
	if r == "grain" {
		n += w.reservedSpace(id)
	}
	if r == "clay" || r == "straw" || r == "bricks" {
		n += w.materialReserved(id, r)
	}
	return n
}
func (w World) reservedResource(id int64, r string) int {
	n := w.boatReserved(id, r)
	for _, a := range w.Walkers {
		if ((a.Role == "stonehauler" && a.Stage == "collect") || (a.Role == "goodsbuyer" && a.Stage == "fetch") || (a.Role == "trader" && a.Stage == "export")) && a.Source == id && a.Resource == r {
			n += a.Reserved
		}
	}
	if r == "grain" {
		n += w.reservedStock(id)
	}
	return n
}
func (w World) ResourceFreeSpace(id int64, r string) int {
	b := w.building(id)
	if b == nil {
		return 0
	}
	return max(0, resourceLimit(*b, r)-inventory(*b, r)-w.incomingResource(id, r))
}
func (w *World) RecordResourceProduction(r string, n int) int {
	if !validResource(r) || n <= 0 {
		return 0
	}
	switch r {
	case "grain":
		w.GrainProduced += int64(n)
	case "clay":
		w.ClayProduced += int64(n)
	case "straw":
		w.StrawProduced += int64(n)
	case "bricks":
		w.BricksProduced += int64(n)
	default:
		if w.GoodsProduced == nil {
			w.GoodsProduced = map[string]int64{}
		}
		w.GoodsProduced[r] += int64(n)
	}
	return n
}

// DepositResource transfers already-produced cargo into a store. The caller
// holds a capacity reservation and must retain any unaccepted cargo.
func (w *World) DepositResource(id int64, r string, n int) int {
	b := w.building(id)
	if b == nil || !validResource(r) || n <= 0 {
		return 0
	}
	n = min(n, max(0, resourceLimit(*b, r)-inventory(*b, r)))
	setInventory(b, r, inventory(*b, r)+n)
	return n
}
func (w *World) ProduceResource(id int64, r string, n int) int {
	n = min(n, w.ResourceFreeSpace(id, r))
	n = w.DepositResource(id, r, n)
	w.RecordResourceProduction(r, n)
	return n
}
func (w *World) ConsumeResource(id int64, r string, n int) int {
	b := w.building(id)
	if b == nil || !validResource(r) || n <= 0 {
		return 0
	}
	n = min(n, max(0, inventory(*b, r)-w.reservedResource(id, r)))
	setInventory(b, r, inventory(*b, r)-n)
	switch r {
	case "grain":
		w.GrainConsumed += int64(n)
	case "clay":
		w.ClayConsumed += int64(n)
	case "straw":
		w.StrawConsumed += int64(n)
	case "bricks":
		w.BricksConsumed += int64(n)
	default:
		if w.GoodsConsumed == nil {
			w.GoodsConsumed = map[string]int64{}
		}
		w.GoodsConsumed[r] += int64(n)
	}
	return n
}
func (w World) hasInventory(b Building) bool {
	if b.Gold > 0 {
		return true
	}
	for _, r := range resourceCatalog {
		if inventory(b, r.ID) > 0 {
			return true
		}
	}
	return false
}
func storageOrder(b Building, r string) string {
	if o := b.StorageOrders[r]; o != "" {
		return o
	}
	return "accept"
}
func storageAccepts(b Building, r string) bool {
	o := storageOrder(b, r)
	return o == "accept" || o == "get"
}
func (w *World) SetStorageOrder(x, y int, r, order string) error {
	if !validResource(r) || (order != "accept" && order != "refuse" && order != "get" && order != "empty") {
		return errors.New("choose a valid resource and accept, refuse, get, or empty")
	}
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind == "storageyard" && b.contains(Point{x, y}) {
			if resourceLimit(*b, r) == 0 {
				return errors.New("this storage yard cannot store that resource")
			}
			if b.StorageOrders == nil {
				b.StorageOrders = map[string]string{}
			}
			b.StorageOrders[r] = order
			w.refresh()
			return nil
		}
	}
	return errors.New("select a storage yard")
}

// Completed ordinary buildings have fixed storage limits for their kind,
// housing footprint, and agricultural rules. Monuments consume capacity as
// construction advances and must recompute their limits at every refresh.
type capacityKey struct {
	Kind              string
	Side, Agriculture int
}

var ordinaryLimits sync.Map

func buildingResourceLimits(b *Building) map[string]int {
	key := capacityKey{b.Kind, b.lotSide(), int(b.AgricultureVersion)}
	fixed := !isMonument(b.Kind)
	if fixed {
		if limits, ok := ordinaryLimits.Load(key); ok {
			return maps.Clone(limits.(map[string]int))
		}
	}
	limits := map[string]int{}
	for _, r := range resourceCatalog {
		if n := resourceLimit(*b, r.ID); n > 0 {
			limits[r.ID] = n
		}
	}
	if fixed {
		ordinaryLimits.Store(key, maps.Clone(limits))
	}
	return limits
}
func (w World) refreshGoods(b *Building) {
	b.ResourceLimits = buildingResourceLimits(b)
	b.Inventories = map[string]int{}
	for _, r := range []string{"grain", "clay", "straw", "bricks"} {
		if n := inventory(*b, r); n > 0 {
			b.Inventories[r] = n
		}
	}
	for r, n := range b.Goods {
		if n > 0 && newResource(r) {
			b.Inventories[r] = n
		}
	}
}
func (w World) goodsView(v *View) {
	v.ResourceCatalog = resourceCatalog
	v.ReedBeds = append([]Point{}, w.ReedBeds...)
	v.ResourceTotals = map[string]int{}
	v.ResourceProduced = map[string]int64{}
	v.ResourceConsumed = map[string]int64{}
	for _, r := range resourceCatalog {
		n := 0
		for _, b := range w.Buildings {
			n += inventory(b, r.ID)
		}
		for _, a := range w.Walkers {
			ar := a.Resource
			if ar == "" && (a.Role == "cart" || a.Role == "buyer" || a.Role == "seller") {
				ar = "grain"
			}
			if a.Role != "trader" && ar == r.ID {
				n += a.Cargo
			}
		}
		v.ResourceTotals[r.ID] = n + w.boatInventory(r.ID)
		v.ResourceProduced[r.ID], v.ResourceConsumed[r.ID] = w.resourceCounters(r.ID)
	}
}
func (w World) resourceCounters(r string) (int64, int64) {
	switch r {
	case "grain":
		return w.GrainProduced, w.GrainConsumed
	case "clay":
		return w.ClayProduced, w.ClayConsumed
	case "straw":
		return w.StrawProduced, w.StrawConsumed
	case "bricks":
		return w.BricksProduced, w.BricksConsumed
	}
	return w.GoodsProduced[r], w.GoodsConsumed[r]
}
func (w World) validateGoods() error {
	for r, n := range w.GoodsProduced {
		if !newResource(r) || n < 0 {
			return errors.New("invalid produced resource counter")
		}
	}
	for r, n := range w.GoodsConsumed {
		if !newResource(r) || n < 0 {
			return errors.New("invalid consumed resource counter")
		}
	}
	totals := map[string]int{}
	for _, b := range w.Buildings {
		for r, n := range b.Goods {
			if !newResource(r) || n < 0 {
				return errors.New("invalid resource inventory")
			}
		}
		for r, o := range b.StorageOrders {
			if b.Kind != "storageyard" || !validResource(r) || (o != "accept" && o != "get" && o != "refuse" && o != "empty") {
				return errors.New("invalid storage order")
			}
		}
		for _, r := range resourceCatalog {
			n := inventory(b, r.ID)
			if n < 0 || n+w.incomingResource(b.ID, r.ID) > resourceLimit(b, r.ID) || w.reservedResource(b.ID, r.ID) > n {
				return errors.New("resource capacity or reservation exceeded: " + r.ID)
			}
			totals[r.ID] += n
		}
	}
	for _, a := range w.Walkers {
		if a.Role != "trader" && newResource(a.Resource) {
			totals[a.Resource] += a.Cargo
		}
	}
	for _, r := range resourceCatalog {
		if !newResource(r.ID) {
			continue
		}
		p, c := w.resourceCounters(r.ID)
		if p+w.ResourceImported[r.ID] != c+int64(totals[r.ID]+w.boatInventory(r.ID))+w.ResourceExported[r.ID]+w.RequestSent[r.ID] {
			return errors.New("resource conservation failed: " + r.ID)
		}
	}
	return nil
}

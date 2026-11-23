package sandbox

import (
	"errors"
	"slices"
)

// Manual pp170-173: staffed road-connected Recruiter, copper weapons for
// infantry, forts without employees or road requirements, companies commanded
// into the field. Normal model rows45/95/112/217 provide costs and staffing.
// Sixteen soldiers and100 weapon units/recruit are reference-corroborated.
// This first battle uses authored recruitment timing, hit points and combat
// cadence; it does not claim the original damage/morale/experience formulas.
const CompanyLimit = 16

type Company struct {
	Morale      int    `json:"morale,omitempty"`
	ChargeUntil int64  `json:"charge_until,omitempty"`
	RestUntil   int64  `json:"rest_until,omitempty"`
	Rotated     bool   `json:"rotated,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Experience  int    `json:"experience,omitempty"`
	FortID      int64  `json:"fort_id"`
	Order       string `json:"order"`
	Target      Point  `json:"target"`
	Soldiers    int    `json:"soldiers"`
	Arriving    int    `json:"arriving"`
	Stationed   int    `json:"stationed"`
}
type Troop struct {
	Fought     bool    `json:"fought,omitempty"`
	InvasionID int     `json:"invasion_id,omitempty"`
	Kind       string  `json:"kind,omitempty"`
	School     int64   `json:"school,omitempty"`
	Training   int     `json:"training,omitempty"`
	Experience int     `json:"experience,omitempty"`
	ID         int64   `json:"id"`
	FortID     int64   `json:"fort_id"`
	Origin     int64   `json:"origin"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Enemy      bool    `json:"enemy"`
	HP         int     `json:"hp"`
	Stage      string  `json:"stage"`
	Route      []Point `json:"route"`
	Next       int     `json:"next"`
}
type MilitaryStats struct {
	SentriesRecruited int64 `json:"sentries_recruited,omitempty"`
	ArchersRecruited  int64 `json:"archers_recruited,omitempty"`
	ChariotsRecruited int64 `json:"chariots_recruited,omitempty"`
	Recruited         int64 `json:"recruited"`
	Lost              int64 `json:"lost"`
	Invaders          int64 `json:"invaders"`
	Defeated          int64 `json:"defeated"`
	Breaches          int64 `json:"breaches"`
	InvasionActive    bool  `json:"invasion_active"`
}

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"coppermine", "Copper Mine", 2, 2, 150, "Military", 10},
		CatalogEntry{"weaponsmith", "Weaponsmith", 2, 2, 120, "Military", 12},
		CatalogEntry{"recruiter", "Recruiter", 3, 3, 150, "Military", 10},
		CatalogEntry{"fort", "Infantry Fort", 7, 4, 1000, "Military", 0})
	goodsRecipes["weaponsmith"] = goodsRecipe{"copper", "weapons", 100, 100}
}
func (w World) militaryPlacement(b Building) error {
	if b.Kind == "coppermine" && !w.mineTouchesOre(b) {
		return errors.New("place a copper mine beside exposed metallic rock")
	}
	if b.Kind == "recruiter" {
		for _, other := range w.Buildings {
			if other.Kind == "recruiter" {
				return errors.New("the city can have only one Recruiter")
			}
		}
	}
	if isFort(b.Kind) {
		n := 0
		for _, other := range w.Buildings {
			if isFort(other.Kind) {
				n++
			}
		}
		if n >= 6 {
			return errors.New("the city already has six companies")
		}
	}
	return nil
}
func (w World) militaryReserved(p Point) bool {
	for _, d := range w.Deployments {
		if !closedDeployment(d) && (d.Landing != nil && *d.Landing == p || d.Boarding != nil && *d.Boarding == p) {
			return true
		}
	}
	for _, a := range w.Troops {
		if p == (Point{a.X, a.Y}) || slices.Contains(a.Route, p) {
			return true
		}
	}
	return false
}
func (w World) militaryBuildingInUse(id int64) bool {
	for _, a := range w.Troops {
		if a.FortID == id || a.Origin == id || a.School == id {
			return true
		}
	}
	return false
}
func (w *World) syncCompanies() {
	companies := make([]Company, 0)
	for _, b := range w.Buildings {
		if !isFort(b.Kind) {
			continue
		}
		company := Company{Morale: 60, Kind: fortUnit(b.Kind), FortID: b.ID, Order: "return", Target: Point{b.X, b.Y - 1}}
		for _, old := range w.Companies {
			if old.FortID == b.ID {
				company = old
				break
			}
		}
		company.Soldiers, company.Arriving, company.Stationed = 0, 0, 0
		company.Experience = 0
		for _, a := range w.Troops {
			if a.FortID != b.ID {
				continue
			}
			company.Soldiers++
			company.Experience += a.Experience
			if a.Stage == "recruit" || a.Stage == "academy" || a.Stage == "training" {
				company.Arriving++
			}
			if a.Stage == "garrison" {
				company.Stationed++
			}
		}
		if company.Soldiers > 0 {
			company.Experience /= company.Soldiers
		}
		companies = append(companies, company)
	}
	w.Companies = companies
}
func (w World) garrisonTiles(fort Building) []Point {
	blocked, roads := w.occupied(), w.roadSet()
	out := []Point{}
	for _, p := range w.accessTiles(fort) {
		if w.buildable(p) && (!blocked[p] || roads[p]) {
			out = append(out, p)
		}
	}
	return out
}
func (w *World) MilitaryOrder(fortID int64, order string, p Point) error {
	if w.deployedFort(fortID) {
		return errors.New("this company is serving a kingdom deployment")
	}
	fort := w.building(fortID)
	if fort == nil || !isFort(fort.Kind) || fort.Damaged != "" {
		return errors.New("select an intact Fort")
	}
	rotated := false
	current := w.company(fortID)
	if current != nil {
		rotated = current.Rotated
	}
	if order == "rotate" {
		if current == nil {
			return errors.New("company not found")
		}
		rotated = !rotated
		order = current.Order
		p = current.Target
	}
	if !validCompanyOrder(order) || order == "deployment" {
		return errors.New("unknown company order")
	}
	c := w.company(fortID)
	if order != "return" && c != nil && morale(*c) < 15 {
		return errors.New("this company must recover its morale at the fort")
	}
	if order == "loose" && fort.Kind == "chariotfort" {
		return errors.New("charioteers cannot form loose ranks")
	}
	if order == "charge" && (fort.Kind != "chariotfort" || c != nil && c.RestUntil > w.Tick) {
		return errors.New("only rested charioteers can charge")
	}
	if order != "return" && (!w.buildable(p) || w.occupied()[p] && !w.roadSet()[p]) {
		return errors.New("muster on open dry ground")
	}
	routes := map[int64][]Point{}
	slot := 0
	for _, a := range w.Troops {
		if a.FortID != fortID || a.Stage == "recruit" || a.Stage == "academy" || a.Stage == "training" || a.Stage == "sentry_recruit" {
			continue
		}
		goals := []Point{p}
		if order == "return" {
			tiles := w.garrisonTiles(*fort)
			if len(tiles) == 0 {
				return errors.New("fort has no accessible staging tiles")
			}
			goals = []Point{tiles[slot%len(tiles)]}
		} else {
			spacing := 1
			if order == "loose" {
				spacing = 2
			}
			dx, dy := slot%4*spacing, slot/4*spacing
			if rotated {
				dx, dy = dy, dx
			}
			formation := Point{p.X + dx, p.Y + dy}
			if w.buildable(formation) && (!w.occupied()[formation] || w.roadSet()[formation]) {
				goals = []Point{formation}
			}
		}
		slot++
		route := w.landPath([]Point{{a.X, a.Y}}, goals)
		if len(route) == 0 {
			return errors.New("company cannot reach that location")
		}
		routes[a.ID] = route
	}
	for i := range w.Companies {
		if w.Companies[i].FortID == fortID {
			w.Companies[i].Order = order
			w.Companies[i].Rotated = rotated
			if order == "charge" {
				w.Companies[i].ChargeUntil = min(MaxTick, w.Tick+24)
				w.Companies[i].RestUntil = min(MaxTick, w.Tick+120)
			}
			if order != "return" {
				w.Companies[i].Target = p
			}
		}
	}
	for i := range w.Troops {
		a := &w.Troops[i]
		if route, ok := routes[a.ID]; ok {
			a.Route = route
			a.Next = 1
			a.Stage = "march"
			if order == "return" {
				a.Stage = "return"
			}
		}
	}
	w.syncCompanies()
	return nil
}
func (w *World) TestInvasion() error {
	if w.Campaign != nil {
		return errors.New("test invasions are available only in the sandbox")
	}
	if w.Military.InvasionActive {
		return errors.New("an invasion is already active")
	}
	for y := 15; y < w.mapSize(); y++ {
		if err := w.SpawnInvasion(int(w.NextID), "test", 4, Point{w.mapSize() - 1, y}); err == nil {
			return nil
		}
	}
	return errors.New("no land route into the settlement")
}

func (w *World) militaryTick() {
	wasBattle := w.Military.InvasionActive
	w.syncCompanies()
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if !goodsOperating(*b) {
			continue
		}
		if b.Kind == "coppermine" && w.ResourceFreeSpace(b.ID, "copper") >= 100 {
			b.Work = min(480, b.Work+b.Workers)
			if b.Work == 480 {
				w.ProduceResource(b.ID, "copper", 100)
				b.Work = 0
			}
		}
	}
	w.recruitMilitary()

	// Resolve all damage from the same pre-combat positions. No render ordering
	// or map iteration order can affect who strikes first.
	hits := map[int64]int{}
	engaged := map[int64]bool{}
	for i := range w.Troops {
		a := &w.Troops[i]
		if a.HP <= 0 || a.Stage == "kingdom_away" || a.Stage == "recruit" || a.Stage == "garrison" || a.Stage == "aboard" || a.Stage == "embark" || a.Stage == "boarding" || a.Stage == "academy" || a.Stage == "training" || a.Stage == "sentry_recruit" {
			continue
		}
		if a.Stage == "guard" {
			tower := w.building(a.FortID)
			if tower == nil || !goodsOperating(*tower) {
				continue
			}
		}
		best := -1
		distance := 100000
		for j, b := range w.Troops {
			if a.Enemy == b.Enemy || b.HP <= 0 || b.Stage == "kingdom_away" || b.Stage == "recruit" || b.Stage == "aboard" || b.Stage == "academy" || b.Stage == "training" || b.Stage == "sentry_recruit" {
				continue
			}
			d := abs(a.X-b.X) + abs(a.Y-b.Y)
			if d < distance {
				best = j
				distance = d
			}
		}
		if best < 0 {
			continue
		}
		target := w.Troops[best]
		if distance <= unitRange(*a) {
			engaged[a.ID] = true
			if w.Tick%4 == 0 {
				damage := w.battleDamage(*a, target, distance)
				a.Fought = true
				hits[target.ID] += damage
			}
			continue
		}
		if !a.Enemy && distance <= w.pursuitRadius(*a) && a.Stage != "return" && a.Stage != "guard" {
			route := w.landPath([]Point{{a.X, a.Y}}, []Point{{target.X, target.Y}})
			if len(route) > 0 {
				a.Route = route
				a.Next = 1
				a.Stage = "march"
			}
		}
	}
	for i := range w.Troops {
		w.Troops[i].HP -= hits[w.Troops[i].ID]
	}
	active := make([]Troop, 0, len(w.Troops))
	invaders := 0
	for _, a := range w.Troops {
		if a.Stage == "aboard" || a.Stage == "kingdom_away" {
			active = append(active, a)
			continue
		}
		if a.HP <= 0 {
			if a.Enemy {
				w.Military.Defeated++
				w.recordInvader(a, false)
			} else {
				w.Military.Lost++
				w.changeMorale(a.FortID, -8)
			}
			continue
		}
		if !engaged[a.ID] && a.Stage != "training" && !w.attackDefense(&a) {
			for range w.tacticalMovement(a) {
				if a.Next < len(a.Route) {
					p := a.Route[a.Next]
					a.X, a.Y = p.X, p.Y
					a.Next++
				}
			}
		}
		w.trainMilitary(&a)
		w.arriveSentry(&a)

		if a.Next >= len(a.Route) {
			if a.Stage == "kingdom_march" {
				a.Stage = "kingdom_away"
				a.Route = pointSlice(a.X, a.Y)
				a.Next = 1
			}
			if a.Stage == "embark" {
				a.Stage = "boarding"
			}
			if a.Enemy && !engaged[a.ID] {
				w.Military.Breaches++
				w.recordInvader(a, true)
				if home := w.building(a.Origin); home != nil && home.Pop > 0 {
					home.Pop--
					w.Departures++
				}
				continue
			}
			if a.Stage == "recruit" || a.Stage == "return" {
				wasRecruit := a.Stage == "recruit"
				a.Stage = "garrison"
				a.Origin = 0
				if wasRecruit {
					for _, company := range w.Companies {
						if company.FortID == a.FortID && company.Order != "return" && company.Order != "deployment" {
							onward := w.landPath([]Point{{a.X, a.Y}}, []Point{company.Target})
							if len(onward) > 0 {
								a.Route = onward
								a.Next = 1
								a.Stage = "march"
							}
						}
					}
				}
			}
			if a.Stage == "march" && a.Next >= len(a.Route) {
				a.Stage = "field"
			}
		}
		if a.Stage == "garrison" || a.Stage == "field" || a.Stage == "boarding" {
			a.Route = []Point{{a.X, a.Y}}
			a.Next = 1
		}
		if a.Enemy {
			invaders++
		}
		active = append(active, a)
	}
	active = slices.DeleteFunc(active, func(a Troop) bool {
		if a.Stage == "guard" {
			tower := w.building(a.FortID)
			if tower == nil || tower.Damaged != "" {
				w.Military.Lost++
				return true
			}
		}
		return false
	})
	w.Troops = active
	w.reconcileBoarding()
	w.advanceLandDeployments()
	w.syncInvasions()
	w.Military.InvasionActive = invaders > 0 || w.passengersAtSea(0) > 0 || w.hostileWarships(0) > 0
	w.tacticsTick(wasBattle)
	w.syncCompanies()
}
func (w World) militaryView(v *View) {
	v.Companies = append([]Company{}, w.Companies...)
	v.Military = w.Military
	v.Invasions = append([]Invasion{}, w.Invasions...)
	for _, a := range w.Troops {
		if a.Stage == "aboard" || a.Stage == "kingdom_away" {
			continue
		}
		role := troopUnit(a)
		if a.Enemy {
			role = "invader"
		}
		v.Walkers = append(v.Walkers, WalkerView{ID: a.ID, X: a.X, Y: a.Y, Role: role, Origin: a.Origin, Source: a.FortID, Target: a.FortID, Stage: a.Stage})
	}
}
func (w World) validateMilitary() error {
	if err := w.validateArmyDispatches(); err != nil {
		return err
	}
	if w.MilitaryVersion < 0 || w.MilitaryVersion > 1 {
		return errors.New("unsupported military rules version")
	}
	if err := w.validateInvasions(); err != nil {
		return err
	}
	m := w.Military
	if m.Recruited < 0 || m.Recruited > MaxTick || m.SentriesRecruited < 0 || m.ArchersRecruited < 0 || m.ChariotsRecruited < 0 || m.SentriesRecruited+m.ArchersRecruited+m.ChariotsRecruited > m.Recruited || w.GoodsConsumed["weapons"] < (m.Recruited-m.ArchersRecruited-m.ChariotsRecruited-m.SentriesRecruited)*100 || w.GoodsConsumed["chariots"] < m.ChariotsRecruited*100 || m.Lost < 0 || m.Invaders < 0 || m.Defeated < 0 || m.Breaches < 0 {
		return errors.New("invalid military ledger")
	}
	ids := map[int64]bool{}
	for _, b := range w.Buildings {
		ids[b.ID] = true
	}
	for _, a := range w.Walkers {
		ids[a.ID] = true
	}
	friendly, enemies := int64(0), int64(0)
	roads := w.roadSet()
	blocked := w.occupied()
	for _, a := range w.Troops {
		if a.ID < 1 || a.ID >= w.NextID || ids[a.ID] || a.HP < 1 || a.HP > unitHP(a) || a.Next < 1 || a.Next > len(a.Route) || a.Route[a.Next-1] != (Point{a.X, a.Y}) {
			return errors.New("invalid military figure")
		}
		ids[a.ID] = true
		if err := w.validateUnit(a); err != nil {
			return err
		}
		if (a.Stage == "kingdom_march" || a.Stage == "kingdom_away") && !w.landDeploymentOwns(a) {
			return errors.New("soldier left city without an army dispatch")
		}
		for i, p := range a.Route {
			if a.Stage == "aboard" {
				if !w.aboard(a.ID) || !w.water(p) {
					return errors.New("invalid embarked soldier")
				}
				continue
			}
			allowedDefense := a.Enemy && w.defenseAt(p) != nil
			if a.Stage == "guard" {
				tower := w.building(a.FortID)
				allowedDefense = tower != nil && tower.Kind == "tower" && tower.contains(p)
			}
			if !w.walkable(p) || blocked[p] && !roads[p] && !allowedDefense || i > 0 && !adjacent(p, a.Route[i-1]) {
				return errors.New("invalid military land route")
			}
		}
		if a.Enemy {
			enemies++
			if a.FortID != 0 || a.HP > 3 || a.Stage != "raid" || w.building(a.Origin) == nil {
				return errors.New("invalid invading soldier")
			}
		} else {
			friendly++
			fort, origin := w.building(a.FortID), w.building(a.Origin)
			if fort == nil || (!isFort(fort.Kind) && fort.Kind != "tower") || (a.Stage == "recruit" && (origin == nil || origin.Kind != "recruiter")) || !slices.Contains([]string{"recruit", "garrison", "march", "field", "return", "embark", "boarding", "aboard", "academy", "training", "sentry_recruit", "guard", "kingdom_march", "kingdom_away"}, a.Stage) {
				return errors.New("invalid infantry company")
			}
			if (a.Stage == "recruit" || a.Stage == "return" || a.Stage == "garrison") && !fort.roadAccess(a.Route[len(a.Route)-1]) {
				return errors.New("recruit did not reach its fort")
			}
		}
	}
	if friendly+m.Lost != m.Recruited || enemies+int64(w.passengersAtSea(0))+m.Defeated+m.Breaches != m.Invaders || m.InvasionActive != (enemies > 0 || w.passengersAtSea(0) > 0 || w.hostileWarships(0) > 0) {
		return errors.New("military conservation failed")
	}
	derived := w
	derived.syncCompanies()
	if !slices.Equal(derived.Companies, w.Companies) {
		return errors.New("inconsistent company state")
	}
	for _, c := range w.Companies {
		if c.Soldiers > CompanyLimit || !validCompanyOrder(c.Order) || c.Morale < 0 || c.Morale > 100 || c.ChargeUntil < 0 || c.ChargeUntil > MaxTick || c.RestUntil < c.ChargeUntil || c.RestUntil > MaxTick || c.Order != "return" && c.Order != "deployment" && !w.buildable(c.Target) {
			return errors.New("invalid company orders")
		}
	}
	return nil
}

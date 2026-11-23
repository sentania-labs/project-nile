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
	Kind       string `json:"kind,omitempty"`
	Experience int    `json:"experience,omitempty"`
	FortID     int64  `json:"fort_id"`
	Order      string `json:"order"`
	Target     Point  `json:"target"`
	Soldiers   int    `json:"soldiers"`
	Arriving   int    `json:"arriving"`
	Stationed  int    `json:"stationed"`
}
type Troop struct {
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
		company := Company{Kind: fortUnit(b.Kind), FortID: b.ID, Order: "return", Target: Point{b.X, b.Y - 1}}
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
	if order != "muster" && order != "return" {
		return errors.New("company order must be muster or return")
	}
	if order == "muster" && (!w.buildable(p) || w.occupied()[p] && !w.roadSet()[p]) {
		return errors.New("muster on open dry ground")
	}
	routes := map[int64][]Point{}
	slot := 0
	for _, a := range w.Troops {
		if a.FortID != fortID || a.Stage == "recruit" || a.Stage == "academy" || a.Stage == "training" {
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
			formation := Point{p.X + slot%4, p.Y + slot/4}
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
			if order == "muster" {
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
	if w.NextID > MaxTick-4 {
		return errors.New("entity identifier limit reached")
	}
	var target *Building
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind == "hut" && b.Pop > 0 {
			target = b
			break
		}
	}
	if target == nil {
		return errors.New("settle a town before testing its defenses")
	}
	var route []Point
	for y := 15; y < w.mapSize(); y++ {
		route = w.landPath([]Point{{w.mapSize() - 1, y}}, w.accessTiles(*target))
		if len(route) > 0 {
			break
		}
	}
	if len(route) == 0 {
		return errors.New("no land route into the settlement")
	}
	for i := range 4 {
		separate := w.landPath([]Point{{route[0].X, route[0].Y + i}}, w.accessTiles(*target))
		if len(separate) == 0 {
			separate = route
		}
		w.Troops = append(w.Troops, Troop{ID: w.NextID, Origin: target.ID, X: separate[0].X, Y: separate[0].Y, Enemy: true, HP: 3, Stage: "raid", Route: slices.Clone(separate), Next: 1})
		w.NextID++
	}
	w.Military.Invaders += 4
	w.Military.InvasionActive = true
	return nil
}
func (w *World) militaryTick() {
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
		if a.HP <= 0 || a.Stage == "recruit" || a.Stage == "garrison" || a.Stage == "aboard" || a.Stage == "embark" || a.Stage == "boarding" || a.Stage == "academy" || a.Stage == "training" {
			continue
		}
		best := -1
		distance := 100000
		for j, b := range w.Troops {
			if a.Enemy == b.Enemy || b.HP <= 0 || b.Stage == "recruit" || b.Stage == "aboard" || b.Stage == "academy" || b.Stage == "training" {
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
				damage := 1
				if a.Experience > 0 && w.Tick%8 == 0 {
					damage++
				}
				if troopUnit(*a) == "charioteer" {
					damage++
				}
				hits[target.ID] += damage
			}
			continue
		}
		if !a.Enemy && distance <= 8 && a.Stage != "return" {
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
		if a.Stage == "aboard" {
			active = append(active, a)
			continue
		}
		if a.HP <= 0 {
			if a.Enemy {
				w.Military.Defeated++
			} else {
				w.Military.Lost++
			}
			continue
		}
		if !engaged[a.ID] && a.Stage != "training" {
			for range unitMovement(a, w.Tick) {
				if a.Next < len(a.Route) {
					p := a.Route[a.Next]
					a.X, a.Y = p.X, p.Y
					a.Next++
				}
			}
		}
		w.trainMilitary(&a)

		if a.Next >= len(a.Route) {
			if a.Stage == "embark" {
				a.Stage = "boarding"
			}
			if a.Enemy && !engaged[a.ID] {
				w.Military.Breaches++
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
						if company.FortID == a.FortID && company.Order == "muster" {
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
	w.Troops = active
	w.reconcileBoarding()
	w.Military.InvasionActive = invaders > 0
	w.syncCompanies()
}
func (w World) militaryView(v *View) {
	v.Companies = append([]Company{}, w.Companies...)
	v.Military = w.Military
	for _, a := range w.Troops {
		if a.Stage == "aboard" {
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
	m := w.Military
	if m.Recruited < 0 || m.Recruited > MaxTick || m.ArchersRecruited < 0 || m.ChariotsRecruited < 0 || m.ArchersRecruited+m.ChariotsRecruited > m.Recruited || w.GoodsConsumed["weapons"] < (m.Recruited-m.ArchersRecruited-m.ChariotsRecruited)*100 || w.GoodsConsumed["chariots"] < m.ChariotsRecruited*100 || m.Lost < 0 || m.Invaders < 0 || m.Defeated < 0 || m.Breaches < 0 {
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
		for i, p := range a.Route {
			if a.Stage == "aboard" {
				if !w.aboard(a.ID) || !w.water(p) {
					return errors.New("invalid embarked soldier")
				}
				continue
			}
			if !w.buildable(p) || blocked[p] && !roads[p] || i > 0 && !adjacent(p, a.Route[i-1]) {
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
			if fort == nil || !isFort(fort.Kind) || (a.Stage == "recruit" && (origin == nil || origin.Kind != "recruiter")) || !slices.Contains([]string{"recruit", "garrison", "march", "field", "return", "embark", "boarding", "aboard", "academy", "training"}, a.Stage) {
				return errors.New("invalid infantry company")
			}
			if (a.Stage == "recruit" || a.Stage == "return" || a.Stage == "garrison") && !fort.roadAccess(a.Route[len(a.Route)-1]) {
				return errors.New("recruit did not reach its fort")
			}
		}
	}
	if friendly+m.Lost != m.Recruited || enemies+m.Defeated+m.Breaches != m.Invaders || m.InvasionActive != (enemies > 0) {
		return errors.New("military conservation failed")
	}
	derived := w
	derived.syncCompanies()
	if !slices.Equal(derived.Companies, w.Companies) {
		return errors.New("inconsistent company state")
	}
	for _, c := range w.Companies {
		if c.Soldiers > CompanyLimit || (c.Order != "return" && c.Order != "muster" && c.Order != "deployment") || c.Order == "muster" && !w.buildable(c.Target) {
			return errors.New("invalid company orders")
		}
	}
	return nil
}

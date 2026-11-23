// Package sandbox implements an original deterministic neighborhood prototype.
// Housing capacities and the water/market/food dependency follow extracted model
// rows. Quantities, cadence, staffing, reach, and seasonal timing are provisional.
package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/sentania-labs/project-nile/internal/campaign"
	"reflect"
)

const Size = 32
const MaxTick int64 = 1<<53 - 1
const WellReach = 2
const HousingRoadReach = 2
const MigrationInterval = 8
const SnapshotVersion = 10
const YearTicks int64 = 240
const SeasonOffset int64 = 144
const ConsumptionInterval int64 = 48
const MarketDuration int64 = 128

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Entry is the arrival point used by pre-Kingdom-Road saved cities.
var Entry = Point{6, 15}
var directions = []Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

type Building struct {
	ServiceSupplyCredit       bool              `json:"service_supply_credit,omitempty"`
	TempleUpgrades            uint8             `json:"temple_upgrades,omitempty"`
	Facing                    int               `json:"facing,omitempty"`
	HousesCovered             int               `json:"houses_covered,omitempty"`
	DefenseHP                 int               `json:"defense_hp,omitempty"`
	SoilFertility             int               `json:"soil_fertility,omitempty"`
	MonumentParts             []MonumentPart    `json:"monument_parts,omitempty"`
	AgricultureVersion        int               `json:"agriculture_version,omitempty"`
	FarmType                  string            `json:"farm_type,omitempty"`
	Fertility                 int               `json:"fertility,omitempty"`
	Irrigated                 bool              `json:"irrigated,omitempty"`
	ShipTarget                int64             `json:"ship_target,omitempty"`
	LotSide                   int               `json:"lot_side,omitempty"`
	WantedSide                int               `json:"wanted_side,omitempty"`
	ShrineAccess              bool              `json:"shrine_access,omitempty"`
	Happiness                 int               `json:"happiness,omitempty"`
	MoodSet                   bool              `json:"mood_set,omitempty"`
	CrimeRisk                 int               `json:"crime_risk,omitempty"`
	ServiceUntil              [9]int64          `json:"service_until"`
	MalariaRisk               int               `json:"malaria_risk,omitempty"`
	DiseaseRisk               int               `json:"disease_risk"`
	ShowUntil                 [3]int64          `json:"show_until"`
	EntertainmentUntil        [3]int64          `json:"entertainment_until"`
	GodUntil                  [5]int64          `json:"god_until"`
	Desirability              int               `json:"desirability"`
	Goods                     map[string]int    `json:"goods,omitempty"`
	StorageOrders             map[string]string `json:"storage_orders,omitempty"`
	Inventories               map[string]int    `json:"inventories"`
	ResourceLimits            map[string]int    `json:"resource_limits"`
	ID                        int64             `json:"id"`
	Kind                      string            `json:"kind"`
	X                         int               `json:"x"`
	Y                         int               `json:"y"`
	Stock                     int               `json:"stock"`
	Pop                       int               `json:"pop"`
	Tier                      int               `json:"tier"`
	Water                     bool              `json:"water"`
	WaterUntil                int64             `json:"water_until"`
	Capacity                  int               `json:"capacity"`
	TierName                  string            `json:"tier_name"`
	Connected                 bool              `json:"connected"`
	Width                     int               `json:"width"`
	Height                    int               `json:"height"`
	StockCapacity             int               `json:"stock_capacity"`
	Workers                   int               `json:"workers"`
	WorkersNeeded             int               `json:"workers_needed"`
	Food                      bool              `json:"food"`
	Market                    bool              `json:"market"`
	MarketUntil               int64             `json:"market_until"`
	Work                      int               `json:"work"`
	WorkCapacity              int               `json:"work_capacity"`
	Clay                      int               `json:"clay"`
	Straw                     int               `json:"straw"`
	Bricks                    int               `json:"bricks"`
	Gold                      int               `json:"gold"`
	GoldCapacity              int               `json:"gold_capacity"`
	FireRisk                  int               `json:"fire_risk"`
	CollapseRisk              int               `json:"collapse_risk"`
	Damaged                   string            `json:"damaged"`
	FireUntil                 int64             `json:"fire_until"`
	CollapseUntil             int64             `json:"collapse_until"`
	TaxUntil                  int64             `json:"tax_until"`
	FireCovered               bool              `json:"fire_covered"`
	CollapseCovered           bool              `json:"collapse_covered"`
	TaxCovered                bool              `json:"tax_covered"`
	ConstructionDone          int               `json:"construction_done"`
	ConstructionRequired      int               `json:"construction_required"`
	MonumentPhase             int               `json:"monument_phase"`
	MonumentWork              int               `json:"monument_work"`
	MonumentFamily            string            `json:"monument_family,omitempty"`
	MonumentNeed              string            `json:"monument_need,omitempty"`
	MonumentGuild             string            `json:"monument_guild,omitempty"`
	MonumentPercent           int               `json:"monument_percent,omitempty"`
	MonumentStaging           Point             `json:"monument_staging"`
	MonumentRequiredMaterials map[string]int    `json:"monument_required_materials,omitempty"`
	MonumentMaterials         map[string]int    `json:"monument_materials,omitempty"`
	Stage                     string            `json:"stage"`
	RepairCost                int64             `json:"repair_cost"`
	CrewOnSite                int               `json:"crew_on_site"`
	CrewEnRoute               int               `json:"crew_en_route"`
	CrewReturning             int               `json:"crew_returning"`
	StagingAccess             bool              `json:"staging_access"`
	LastTax                   int64             `json:"last_tax"`
}
type Walker struct {
	Supplies   map[string]int `json:"supplies,omitempty"`
	Partner    string         `json:"partner,omitempty"`
	Site       int64          `json:"site,omitempty"`
	LoadAt     int            `json:"load_at,omitempty"`
	DeliverAt  int            `json:"deliver_at,omitempty"`
	People     int            `json:"people,omitempty"`
	ID         int64          `json:"id"`
	X          int            `json:"x"`
	Y          int            `json:"y"`
	Cargo      int            `json:"cargo"`
	Role       string         `json:"role"`
	Target     int64          `json:"target"`
	Route      []Point        `json:"route"`
	Next       int            `json:"next"`
	Origin     int64          `json:"origin"`
	Source     int64          `json:"source"`
	Stage      string         `json:"stage"`
	Reserved   int            `json:"reserved"`
	Resource   string         `json:"resource"`
	ShiftUntil int64          `json:"shift_until"`
}
type World struct {
	SandboxRatings     *campaign.RatingState     `json:"sandbox_ratings,omitempty"`
	Cliffs             []Point                   `json:"cliffs,omitempty"`
	Ferries            []FerryLine               `json:"ferries,omitempty"`
	RoyalCredit        bool                      `json:"royal_credit,omitempty"`
	Household          RoyalHousehold            `json:"royal_household,omitempty"`
	LaborSeekers       bool                      `json:"labor_seekers,omitempty"`
	Naval              NavalStats                `json:"naval,omitempty"`
	KingdomWage        int                       `json:"kingdom_wage,omitempty"`
	CityEvents         CityEvents                `json:"city_events"`
	Invasions          []Invasion                `json:"invasions,omitempty"`
	OpenPartners       map[string]bool           `json:"open_partners,omitempty"`
	PartnerUsed        map[string]map[string]int `json:"partner_used,omitempty"`
	BurialDelivered    map[string]int            `json:"burial_delivered,omitempty"`
	SalaryRank         int                       `json:"salary_rank,omitempty"`
	Savings            int64                     `json:"savings,omitempty"`
	Stolen             int64                     `json:"stolen,omitempty"`
	Arrests            int64                     `json:"arrests,omitempty"`
	AgricultureVersion int                       `json:"agriculture_version,omitempty"`
	Canals             []Point                   `json:"canals,omitempty"`
	Bridges            []Bridge                  `json:"bridges,omitempty"`
	Deployments        []KingdomDeployment       `json:"deployments,omitempty"`
	ArmyDispatches     []ArmyDispatch            `json:"army_dispatches,omitempty"`
	RequestSent        map[string]int64          `json:"request_sent,omitempty"`
	RequestSpent       int64                     `json:"request_spent,omitempty"`
	Boats              []Boat                    `json:"boats,omitempty"`
	ShipsBuilt         int64                     `json:"ships_built,omitempty"`
	FishingReady       map[string]int64          `json:"fishing_ready,omitempty"`
	SeaTradeOpened     bool                      `json:"sea_trade_opened,omitempty"`
	SeaTradeUsed       map[string]int            `json:"sea_trade_used,omitempty"`
	SeaCursor          int                       `json:"sea_cursor,omitempty"`
	NextMerchantTick   int64                     `json:"next_merchant_tick,omitempty"`
	Companies          []Company                 `json:"companies,omitempty"`
	Troops             []Troop                   `json:"troops,omitempty"`
	Military           MilitaryStats             `json:"military"`
	MilitaryVersion    int                       `json:"military_version,omitempty"`
	MapExtent          int                       `json:"map_extent,omitempty"`
	Banks              []int                     `json:"banks,omitempty"`
	Festival           *Festival                 `json:"festival,omitempty"`
	LastFestival       [5]int64                  `json:"last_festival"`
	FestivalsHeld      int64                     `json:"festivals_held"`
	TradeOpened        bool                      `json:"trade_opened,omitempty"`
	TradePolicies      map[string]TradePolicy    `json:"trade_policies,omitempty"`
	ResourceImported   map[string]int64          `json:"resource_imported,omitempty"`
	ResourceExported   map[string]int64          `json:"resource_exported,omitempty"`
	ImportSpent        int64                     `json:"import_spent,omitempty"`
	ExportEarned       int64                     `json:"export_earned,omitempty"`
	TradeUsed          map[string]int            `json:"trade_used,omitempty"`
	TradeYear          int64                     `json:"trade_year,omitempty"`
	TradeCursor        int                       `json:"trade_cursor,omitempty"`
	NextTraderTick     int64                     `json:"next_trader_tick,omitempty"`
	DiseaseDeaths      int64                     `json:"disease_deaths"`
	Favor              [5]int                    `json:"favor"`
	RoadFeatures       []RoadFeature             `json:"road_features,omitempty"`
	Campaign           *campaign.State           `json:"campaign,omitempty"`
	HuntingGrounds     []Point                   `json:"hunting_grounds,omitempty"`
	GoodsProduced      map[string]int64          `json:"goods_produced,omitempty"`
	GoodsConsumed      map[string]int64          `json:"goods_consumed,omitempty"`
	ReedBeds           []Point                   `json:"reed_beds,omitempty"`
	Grassland          []Point                   `json:"grassland,omitempty"`
	WestBanks          []int                     `json:"west_banks,omitempty"`
	Woodland           []Point                   `json:"woodland,omitempty"`
	QuarryRock         []Point                   `json:"quarry_rock,omitempty"`
	OreRock            []Point                   `json:"ore_rock,omitempty"`
	GoldProduced       int64                     `json:"gold_produced"`
	GoldDelivered      int64                     `json:"gold_delivered"`
	LegacyTaxes        bool                      `json:"legacy_taxes,omitempty"`
	EntryPoint         *Point                    `json:"entry,omitempty"`
	ExitPoint          *Point                    `json:"exit,omitempty"`
	KingdomRoad        []Point                   `json:"kingdom_road,omitempty"`
	Tick               int64                     `json:"tick"`
	Roads              []Point                   `json:"roads"`
	Buildings          []Building                `json:"buildings"`
	Walkers            []Walker                  `json:"walkers"`
	NextID             int64                     `json:"next_id"`
	Arrivals           int64                     `json:"arrivals"`
	Version            int                       `json:"version"`
	Departures         int64                     `json:"departures"`
	GrainProduced      int64                     `json:"grain_produced"`
	GrainConsumed      int64                     `json:"grain_consumed"`
	Treasury           int64                     `json:"treasury"`
	Income             int64                     `json:"income"`
	Expenses           int64                     `json:"expenses"`
	LastIncome         int64                     `json:"last_income"`
	LastWages          int64                     `json:"last_wages"`
	BudgetEnabled      bool                      `json:"budget_enabled"`
	HazardsEnabled     bool                      `json:"hazards_enabled"`
	FireCount          int                       `json:"fire_count"`
	CollapseCount      int                       `json:"collapse_count"`
	ClayProduced       int64                     `json:"clay_produced"`
	ClayConsumed       int64                     `json:"clay_consumed"`
	StrawProduced      int64                     `json:"straw_produced"`
	StrawConsumed      int64                     `json:"straw_consumed"`
	BricksProduced     int64                     `json:"bricks_produced"`
	BricksConsumed     int64                     `json:"bricks_consumed"`
	MonthlyFinance     bool                      `json:"monthly_finance,omitempty"`
	TaxRate            int                       `json:"tax_rate"`
	WageRate           int                       `json:"wage_rate"`
	LaborPriority      string                    `json:"labor_priority"`
}

// Rendering needs positions and job status, not every persisted route vertex.
// Keep transport snapshots bounded while saves retain complete replay state.
type WalkerView struct {
	Supplies    map[string]int `json:"supplies,omitempty"`
	AboardFerry bool           `json:"aboard_ferry,omitempty"`
	People      int            `json:"people,omitempty"`
	ID          int64          `json:"id"`
	X           int            `json:"x"`
	Y           int            `json:"y"`
	Cargo       int            `json:"cargo"`
	Role        string         `json:"role"`
	Target      int64          `json:"target"`
	Origin      int64          `json:"origin"`
	Source      int64          `json:"source"`
	Stage       string         `json:"stage"`
	Resource    string         `json:"resource"`
	ShiftUntil  int64          `json:"shift_until"`
}
type View struct {
	Culture            int                    `json:"culture"`
	Prosperity         int                    `json:"prosperity"`
	Cliffs             []Point                `json:"cliffs,omitempty"`
	ConstructionCredit int64                  `json:"construction_credit"`
	InterestPaid       int64                  `json:"interest_paid"`
	RoyalGiftCosts     [3]int64               `json:"royal_gift_costs"`
	KingdomStanding    int                    `json:"kingdom_standing"`
	LaborSeekers       bool                   `json:"labor_seekers,omitempty"`
	Naval              NavalStats             `json:"naval,omitempty"`
	KingdomWage        int                    `json:"kingdom_wage"`
	CityEvents         CityEvents             `json:"city_events"`
	ZooCoverage        int                    `json:"zoo_coverage,omitempty"`
	Invasions          []Invasion             `json:"invasions,omitempty"`
	MonumentRating     int                    `json:"monument_rating"`
	BurialRequirements map[string]int         `json:"burial_requirements"`
	BurialDelivered    map[string]int         `json:"burial_delivered"`
	BurialAvailable    map[string]int         `json:"burial_available"`
	SalaryRank         int                    `json:"salary_rank"`
	Savings            int64                  `json:"savings"`
	Stolen             int64                  `json:"stolen"`
	Arrests            int64                  `json:"arrests"`
	Sentiment          int                    `json:"sentiment"`
	AgricultureVersion int                    `json:"agriculture_version,omitempty"`
	Canals             []Point                `json:"canals,omitempty"`
	WateredCanals      []Point                `json:"watered_canals,omitempty"`
	Bridges            []Bridge               `json:"bridges,omitempty"`
	Deployments        []KingdomDeployment    `json:"deployments"`
	ArmyDispatches     []ArmyDispatch         `json:"army_dispatches"`
	Boats              []BoatView             `json:"boats"`
	ShipsBuilt         int64                  `json:"ships_built"`
	FishingGrounds     []Point                `json:"fishing_grounds"`
	Companies          []Company              `json:"companies"`
	Military           MilitaryStats          `json:"military"`
	MapExtent          int                    `json:"map_extent,omitempty"`
	Banks              []int                  `json:"banks,omitempty"`
	Festival           *Festival              `json:"festival,omitempty"`
	LastFestival       [5]int64               `json:"last_festival"`
	FestivalsHeld      int64                  `json:"festivals_held"`
	TradeRoutes        []TradeRouteView       `json:"trade_routes"`
	TradePolicies      map[string]TradePolicy `json:"trade_policies"`
	TradePrices        []TradePrice           `json:"trade_prices"`
	ResourceImported   map[string]int64       `json:"resource_imported"`
	ResourceExported   map[string]int64       `json:"resource_exported"`
	ImportSpent        int64                  `json:"import_spent"`
	ExportEarned       int64                  `json:"export_earned"`
	DiseaseDeaths      int64                  `json:"disease_deaths"`
	Favor              [5]int                 `json:"favor"`
	RoadFeatures       []RoadFeature          `json:"road_features,omitempty"`
	Campaign           *campaign.View         `json:"campaign,omitempty"`
	HuntingGrounds     []Point                `json:"hunting_grounds"`
	ResourceTotals     map[string]int         `json:"resource_totals"`
	ResourceProduced   map[string]int64       `json:"resource_produced"`
	ResourceConsumed   map[string]int64       `json:"resource_consumed"`
	ResourceCatalog    []ResourceInfo         `json:"resource_catalog"`
	ReedBeds           []Point                `json:"reed_beds"`
	Grassland          []Point                `json:"grassland"`
	WestBanks          []int                  `json:"west_banks,omitempty"`
	Woodland           []Point                `json:"woodland,omitempty"`
	QuarryRock         []Point                `json:"quarry_rock,omitempty"`
	OreRock            []Point                `json:"ore_rock"`
	Gold               int                    `json:"gold"`
	GoldInTransit      int                    `json:"gold_in_transit"`
	GoldProduced       int64                  `json:"gold_produced"`
	GoldDelivered      int64                  `json:"gold_delivered"`
	TaxIncome          int64                  `json:"tax_income"`
	PalaceActive       bool                   `json:"palace_active"`
	TaxesEnabled       bool                   `json:"taxes_enabled"`
	Tick               int64                  `json:"tick"`
	Roads              []Point                `json:"roads"`
	Buildings          []Building             `json:"buildings"`
	Walkers            []WalkerView           `json:"walkers"`
	Population         int                    `json:"population"`
	Grain              int                    `json:"grain"`
	Entry              Point                  `json:"entry"`
	Exit               *Point                 `json:"exit,omitempty"`
	KingdomRoad        []Point                `json:"kingdom_road"`
	HousingRoadReach   int                    `json:"housing_road_reach"`
	GrainProduced      int64                  `json:"grain_produced"`
	GrainConsumed      int64                  `json:"grain_consumed"`
	Departures         int64                  `json:"departures"`
	Season             string                 `json:"season"`
	FloodLevel         float64                `json:"flood_level"`
	Year               int64                  `json:"year"`
	HarvestIn          int64                  `json:"harvest_in"`
	Workforce          int                    `json:"workforce"`
	Employed           int                    `json:"employed"`
	Treasury           int64                  `json:"treasury"`
	Income             int64                  `json:"income"`
	Expenses           int64                  `json:"expenses"`
	LastIncome         int64                  `json:"last_income"`
	LastWages          int64                  `json:"last_wages"`
	FireCount          int                    `json:"fire_count"`
	CollapseCount      int                    `json:"collapse_count"`
	Clay               int                    `json:"clay"`
	Straw              int                    `json:"straw"`
	Bricks             int                    `json:"bricks"`
	ClayProduced       int64                  `json:"clay_produced"`
	ClayConsumed       int64                  `json:"clay_consumed"`
	StrawProduced      int64                  `json:"straw_produced"`
	StrawConsumed      int64                  `json:"straw_consumed"`
	BricksProduced     int64                  `json:"bricks_produced"`
	BricksConsumed     int64                  `json:"bricks_consumed"`
	MonumentProgress   int                    `json:"monument_progress"`
	MonumentGoal       int                    `json:"monument_goal"`
	BuildingCatalog    []CatalogEntry         `json:"building_catalog"`
	MonthlyFinance     bool                   `json:"monthly_finance,omitempty"`
	TaxRate            int                    `json:"tax_rate"`
	WageRate           int                    `json:"wage_rate"`
	LaborPriority      string                 `json:"labor_priority"`
	PotentialWorkforce int                    `json:"potential_workforce"`
	MigrationInterval  int64                  `json:"migration_interval"`
}

func neighborhood() World {
	w := World{MilitaryVersion: 1, Favor: neutralFavor(), Roads: []Point{}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1, Version: SnapshotVersion, TaxRate: 2, WageRate: 1, LaborPriority: "balanced"}
	for x := 10; x <= 22; x++ {
		w.Roads = append(w.Roads, Point{x, 10}, Point{x, 20})
	}
	for y := 11; y < 20; y++ {
		w.Roads = append(w.Roads, Point{10, y}, Point{22, y})
	}
	for x := 6; x < 10; x++ {
		w.Roads = append(w.Roads, Point{x, 15})
	}
	_ = w.Build("hut", 14, 21)
	_ = w.Build("hut", 20, 21)
	return w
}
func Initial() World {
	w := neighborhood()
	w.Roads = slices.DeleteFunc(w.Roads, func(p Point) bool { return p.X < 10 && p.Y == 15 })
	w.addKingdomRoad()
	w.addResourceTerrain()
	w.Buildings[0].Pop = 5
	w.Buildings[1].Pop = 5
	w.Arrivals = 10
	_ = w.Build("hut", 15, 21)
	_ = w.Build("hut", 19, 21)
	w.Buildings[2].Pop = 3
	w.Buildings[3].Pop = 3
	w.Arrivals = 16
	for _, b := range []struct {
		kind string
		x, y int
	}{{"well", 14, 23}, {"well", 20, 23}, {"farm", 7, 10}, {"granary", 14, 6}, {"bazaar", 19, 8}, {"firehouse", 12, 9}, {"architect", 21, 9}, {"taxoffice", 11, 17}, {"palace", 11, 24}} {
		_ = w.Build(b.kind, b.x, b.y)
	}
	for i := range w.Buildings {
		if w.Buildings[i].Kind == "farm" {
			w.Buildings[i].Work = 194
		}
	}
	w.BudgetEnabled = true
	w.HazardsEnabled = true
	w.Treasury = 2500
	w.refresh()
	return w
}
func valid(p Point) bool { return p.X >= 6 && p.X < Size && p.Y >= 0 && p.Y < Size }
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func adjacent(a, b Point) bool  { return abs(a.X-b.X)+abs(a.Y-b.Y) == 1 }
func (b Building) point() Point { return Point{b.X, b.Y} }
func dimensions(kind string) (int, int) {
	if entry, ok := catalog(kind); ok {
		return entry.Width, entry.Height
	}
	return 1, 1
}
func stockCapacity(kind string) int {
	switch kind {
	case "dock":
		return 1200
	case "farm":
		return 48
	case "granary", "storageyard":
		return 192
	case "bazaar":
		return 48
	case "hut":
		return 12
	}
	return 0
}
func requiredWorkers(kind string) int { entry, _ := catalog(kind); return entry.Workers }
func validKind(kind string) bool      { _, ok := catalog(kind); return ok && kind != "road" }
func (b Building) contains(p Point) bool {
	for _, part := range b.MonumentParts {
		if part.contains(p) {
			return true
		}
	}
	return p.X >= b.X && p.X < b.X+b.Width && p.Y >= b.Y && p.Y < b.Y+b.Height
}
func (b Building) adjacent(p Point) bool {
	return (p.X == b.X-1 || p.X == b.X+b.Width) && p.Y >= b.Y && p.Y < b.Y+b.Height || (p.Y == b.Y-1 || p.Y == b.Y+b.Height) && p.X >= b.X && p.X < b.X+b.Width
}
func (b Building) tiles() []Point {
	out := []Point{}
	for y := b.Y; y < b.Y+b.Height; y++ {
		for x := b.X; x < b.X+b.Width; x++ {
			out = append(out, Point{x, y})
		}
	}
	for _, part := range b.MonumentParts {
		for y := part.Y; y < part.Y+part.Height; y++ {
			for x := part.X; x < part.X+part.Width; x++ {
				out = append(out, Point{x, y})
			}
		}
	}
	return out
}
func (w World) roadSet() map[Point]bool {
	out := map[Point]bool{}
	for _, p := range w.Roads {
		out[p] = true
	}
	return out
}
func (w World) occupied() map[Point]bool {
	out := w.roadSet()
	for _, p := range w.QuarryRock {
		out[p] = true
	}
	for _, p := range w.Woodland {
		out[p] = true
	}
	for _, p := range w.OreRock {
		out[p] = true
	}
	for _, b := range w.Buildings {
		for _, p := range b.tiles() {
			out[p] = true
		}
	}
	return out
}
func (w *World) building(id int64) *Building {
	for i := range w.Buildings {
		if w.Buildings[i].ID == id {
			return &w.Buildings[i]
		}
	}
	return nil
}
func (w World) perimeter(b Building) []Point {
	roads := w.roadSet()
	out := []Point{}
	for _, p := range w.accessTiles(b) {
		if roads[p] {
			out = append(out, p)
		}
	}
	return out
}
func (w World) path(starts []Point, destination Building) []Point {
	return w.pathWithFerries(starts, destination, false)
}
func (w World) pathWithFerries(starts []Point, destination Building, unstaffed bool) []Point {
	edges := w.ferryEdges(unstaffed)
	roads := w.roadSet()
	queue := []Point{}
	previous := map[Point]Point{}
	for _, p := range starts {
		if roads[p] {
			if _, seen := previous[p]; !seen {
				previous[p] = p
				queue = append(queue, p)
			}
		}
	}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		if destination.roadAccess(p) {
			out := []Point{p}
			for previous[p] != p {
				p = previous[p]
				out = append(out, p)
			}
			slices.Reverse(out)
			return out
		}
		if q, ok := edges[p]; ok && roads[q] {
			if _, seen := previous[q]; !seen {
				previous[q] = p
				queue = append(queue, q)
			}
		}
		for _, d := range directions {
			next := Point{p.X + d.X, p.Y + d.Y}
			if _, seen := previous[next]; roads[next] && !seen {
				previous[next] = p
				queue = append(queue, next)
			}
		}
	}
	return nil
}
func (w World) route(destination Point) []Point {
	return w.path([]Point{w.entry()}, Building{X: destination.X, Y: destination.Y, Width: 1, Height: 1})
}
func (w World) population() int {
	n := 0
	for _, b := range w.Buildings {
		n += b.Pop
	}
	return n
}
func (w *World) refresh() {
	w.syncCompanies()
	scores := w.desirabilityMap()
	roads := w.roadSet()
	wells := map[Point]bool{}
	for _, b := range w.Buildings {
		if b.Kind == "well" {
			wells[b.point()] = true
		}
	}
	edges := w.ferryEdges(true)
	reached := map[Point]bool{}
	queue := []Point{}
	if roads[w.entry()] {
		reached[w.entry()] = true
		queue = append(queue, w.entry())
	}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		if q, ok := edges[p]; ok && roads[q] && !reached[q] {
			reached[q] = true
			queue = append(queue, q)
		}
		for _, d := range directions {
			next := Point{p.X + d.X, p.Y + d.Y}
			if roads[next] && !reached[next] {
				reached[next] = true
				queue = append(queue, next)
			}
		}
	}
	for i := range w.Buildings {
		b := &w.Buildings[i]
		b.Width, b.Height = b.footprintDimensions()
		b.AgricultureVersion = w.AgricultureVersion
		if w.AgricultureVersion > 0 {
			b.FarmType = w.farmType(*b)
		}
		b.StockCapacity = resourceLimit(*b, "grain")
		w.refreshGoods(b)
		b.WorkersNeeded = requiredWorkers(b.Kind)
		if b.AgricultureVersion > 0 && cropResource(b.Kind) != "" {
			b.WorkersNeeded = 0
			if b.FarmType == "meadow" {
				b.WorkersNeeded = 10
			}
		}
		b.Workers = 0
		b.RepairCost = max(int64(5), w.sitePrice(b.Kind)/2)
		b.WorkCapacity = buildingWorkCapacity(*b)
		b.GoldCapacity = 0
		if b.Kind == "goldmine" {
			b.GoldCapacity = GoldMineCapacity
		}
		b.FireCovered = b.FireUntil > w.Tick
		b.CollapseCovered = b.CollapseUntil > w.Tick
		b.TaxCovered = b.TaxUntil > w.Tick
		b.Stage = ""
		monumentRefresh(b)
		b.Connected = false
		for _, p := range w.accessTiles(*b) {
			if reached[p] {
				b.Connected = true
				break
			}
		}
		b.CrewOnSite, b.CrewEnRoute, b.CrewReturning = 0, 0, 0
		b.StagingAccess = isMonument(b.Kind) && b.Connected
		if isMonument(b.Kind) || w.managedFloodFarm(*b) {
			for _, walker := range w.Walkers {
				if (walker.Role == "builder" || walker.Role == "farmworker") && walker.Source == b.ID {
					switch walker.Stage {
					case "commute":
						b.CrewEnRoute++
					case "work":
						b.CrewOnSite++
					case "return":
						b.CrewReturning++
					}
				}
			}
		}
		b.Water = b.Kind == "hut" && b.WaterUntil > w.Tick
		b.Food = b.Kind == "hut" && slices.ContainsFunc(foodKinds, func(resource string) bool { return inventory(*b, resource) > 0 })
		b.Market = b.Kind == "hut" && b.MarketUntil > w.Tick
		previousTier := b.Tier
		b.Tier, b.Capacity, b.TierName = 0, 0, ""
		if b.Kind != "hut" {
			continue
		}
		for dx := -WellReach; dx <= WellReach && !b.Water; dx++ {
			for dy := -WellReach; dy <= WellReach; dy++ {
				if wells[Point{b.X + dx, b.Y + dy}] {
					b.Water = true
					break
				}
			}
		}
		if w.buildable(b.point()) {
			b.Desirability = scores[b.point()]
		}
		b.ShrineAccess = false // Older saves recorded proximity as household religion.
		w.refreshHome(b, previousTier)
		if b.Tier < 10 {
			b.Capacity *= b.lotSide() * b.lotSide()
		}
		if b.Damaged != "" {
			b.Tier = 0
			b.Capacity = 0
			b.TierName = "Damaged home"
		}
		if b.Pop > b.Capacity {
			w.Departures += int64(b.Pop - b.Capacity)
			b.Pop = b.Capacity
		}
	}
	workforce := w.availableWorkforce()
	// Reserve a minimal administration crew, then fill the Palace's remaining
	// jobs after other employers. This partial-staffing policy is provisional.
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if isPalace(b.Kind) && w.canHire(*b) && workforce > 0 {
			b.Workers = 1
			workforce--
		}
	}
	order := make([]int, len(w.Buildings))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return staffingRank(w.LaborPriority, w.Buildings[a].Kind) - staffingRank(w.LaborPriority, w.Buildings[b].Kind)
	})
	for _, i := range order {
		b := &w.Buildings[i]
		if !isPalace(b.Kind) && w.canHire(*b) {
			b.Workers = min(workforce, b.WorkersNeeded)
			workforce -= b.Workers
		}
	}
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if isPalace(b.Kind) && w.canHire(*b) {
			more := min(workforce, b.WorkersNeeded-b.Workers)
			b.Workers += more
			workforce -= more
		}
	}

	w.refreshAgriculture()
}
func (w *World) clear(p Point) error {
	if i := slices.Index(w.Canals, p); i >= 0 {
		w.Canals = slices.Delete(w.Canals, i, i+1)
		w.refresh()
		return nil
	}
	if p == w.entry() || (w.ExitPoint != nil && p == *w.ExitPoint) {
		return errors.New("the Kingdom Road entry and exit must stay open")
	}
	if i := slices.Index(w.Roads, p); i >= 0 {
		return w.clearRoad(i, p)
	}
	for i, b := range w.Buildings {
		if !b.contains(p) {
			continue
		}
		if b.Pop > 0 {
			return errors.New("this home has residents")
		}
		if w.riverBuildingInUse(b.ID) {
			return errors.New("this building has an active vessel or ship order")
		}
		if w.militaryBuildingInUse(b.ID) {
			return errors.New("this building has active soldiers")
		}
		if w.hasInventory(b) {
			return errors.New("this building still contains goods")
		}
		for _, walker := range w.Walkers {
			if walker.Target == b.ID || walker.Origin == b.ID || walker.Source == b.ID || walker.Site == b.ID {
				return errors.New("this building has an active delivery or arriving migrant")
			}
		}
		if b.Kind == "ferry" {
			if !w.removeFerryAllowed(b) {
				return errors.New("passengers are using this ferry line")
			}
			w.Ferries = slices.DeleteFunc(w.Ferries, func(f FerryLine) bool { return slices.Contains(f.Landings[:], b.ID) })
		}
		w.Buildings = slices.Delete(w.Buildings, i, i+1)
		w.refresh()
		return nil
	}
	return errors.New("the tile is already empty")
}

// ZoneHousing reserves every unoccupied land tile in an inclusive rectangle.
func (w *World) ZoneHousing(a, b Point) error {
	occupied := w.occupied()
	var plots []Point
	for y := max(0, min(a.Y, b.Y)); y <= min(w.mapSize()-1, max(a.Y, b.Y)); y++ {
		for x := max(0, min(a.X, b.X)); x <= min(w.mapSize()-1, max(a.X, b.X)); x++ {
			p := Point{x, y}
			if w.buildable(p) && !occupied[p] && !slices.Contains(w.Canals, p) && !w.offRoadReserved(p) && !slices.Contains(w.HuntingGrounds, p) {
				plots = append(plots, p)
			}
		}
	}
	if len(plots) == 0 {
		return errors.New("no vacant land in selected area")
	}
	if w.cannotBuildFor(int64(len(plots)) * constructionCost("hut")) {
		return errors.New("not enough treasury to zone this housing area")
	}
	if w.NextID > MaxTick-int64(len(plots)) {
		return errors.New("entity identifier limit reached")
	}
	if w.Version == 0 {
		w.Version = SnapshotVersion
	}
	if w.LaborPriority == "" {
		w.policyDefaults()
	}
	w.spendConstruction(int64(len(plots)) * constructionCost("hut"))
	for _, p := range plots {
		w.Buildings = append(w.Buildings, Building{ID: w.NextID, Kind: "hut", X: p.X, Y: p.Y, Width: 1, Height: 1})
		w.NextID++
	}
	w.refresh()
	return nil
}
func (w *World) Build(kind string, x, y int) error { return w.BuildFacing(kind, x, y, 0) }
func (w *World) BuildFacing(kind string, x, y, facing int) error {
	isComplex := slices.Contains(complexKinds, kind)
	if facing < 0 || facing > 3 || facing != 0 && !isComplex {
		return errors.New("invalid building orientation")
	}
	if isComplex && w.templeComplex() != nil {
		return errors.New("only one temple complex may stand in a city")
	}
	if isResidence(kind) && slices.ContainsFunc(w.Buildings, func(b Building) bool { return isResidence(b.Kind) }) {
		return errors.New("the city already has a personal residence")
	}
	if !w.Campaign.Allows(kind) {
		return errors.New("this building has not opened in the current mission")
	}
	p := Point{x, y}
	if kind == "complexaltar" || kind == "complexoracle" {
		return w.addTempleUpgrade(kind, p)
	}
	if kind == "bridge" || kind == "shipbridge" {
		return w.buildBridge(kind, p)
	}
	if kind == "clear" {
		if bridge, ok := w.crossing(p); ok {
			return w.clearBridge(bridge)
		}
	}
	if !w.buildable(p) && !((isRoyalTomb(kind) || kind == "clear") && w.cliff(p)) && !(coastal(kind) && w.water(p)) {
		return errors.New("choose dry land inside the city map")
	}
	if kind == "roadblock" || kind == "plaza" {
		return w.buildRoadFeature(kind, p)
	}
	if kind == "clear" {
		if i := w.roadFeature(p); i >= 0 {
			w.RoadFeatures = slices.Delete(w.RoadFeatures, i, i+1)
			w.refresh()
			return nil
		}
		return w.clear(p)
	}
	if kind != "road" && !validKind(kind) {
		return errors.New("available buildings: road, hut, well, farm, granary, bazaar")
	}
	width, height := dimensions(kind)
	if facing%2 != 0 {
		width, height = height, width
	}
	candidate := Building{Facing: facing, ID: w.NextID, Kind: kind, X: x, Y: y, Width: width, Height: height, DefenseHP: defenseHealth(kind)}
	if err := w.royalTombPlacement(candidate); err != nil {
		return err
	}
	if err := w.prepareMonumentPlacement(&candidate); err != nil {
		return err
	}
	occupied := w.occupied()
	if err := w.defensePlacement(candidate, occupied); err != nil {
		return err
	}
	if err := w.venuePlacement(candidate); err != nil {
		return err
	}
	if slices.Contains(venues, kind) {
		for _, road := range w.Roads {
			delete(occupied, road)
		}
		for _, old := range w.Buildings {
			for _, p := range old.tiles() {
				occupied[p] = true
			}
		}
	}
	for _, tile := range candidate.tiles() {
		if w.riverReserved(tile) {
			return errors.New("a vessel is using this river position")
		}
		if !w.buildingTileValid(candidate, tile) {
			return errors.New("the entire building must fit on dry land")
		}
		if kind != "road" && w.offRoadReserved(tile) {
			return errors.New("wait until the worker returns before building on this path")
		}
		if occupied[tile] || (kind != "road" && slices.Contains(w.Canals, tile)) {
			return errors.New("the building footprint overlaps an occupied tile")
		}
		if slices.Contains(w.HuntingGrounds, tile) {
			return errors.New("leave the hunting grounds open for wildlife")
		}
		if (kind == "claypit" || w.AgricultureVersion == 0 && (kind == "farm" || campCrops[kind] != "")) && !w.floodplain(tile) {
			return errors.New("this building must fit in the fertile strip beside the Nile")
		}
	}
	if err := w.agriculturePlacement(candidate); err != nil {
		return err
	}
	if err := w.coastalPlacement(candidate); err != nil {
		return err
	}
	if w.riverReserved(candidate.point()) {
		return errors.New("a vessel is using this river position")
	}
	if err := w.militaryPlacement(candidate); err != nil {
		return err
	}
	if err := w.goodsPlacement(candidate); err != nil {
		return err
	}
	if err := w.resourcePlacement(candidate); err != nil {
		return err
	}
	if kind == "watersupply" && !w.palaceGroundwater(candidate) {
		return errors.New("a Water Supply needs groundwater beneath its footprint")
	}
	if w.cannotBuildFor(w.sitePrice(kind)) {
		return errors.New("not enough treasury for this construction")
	}
	if w.NextID >= MaxTick {
		return errors.New("entity identifier limit reached")
	}
	if err := w.monumentUpfrontCheck(candidate); err != nil {
		return err
	}
	if w.Version == 0 {
		w.Version = SnapshotVersion
	}
	if w.LaborPriority == "" {
		w.policyDefaults()
	}
	w.monumentUpfrontCommit(&candidate)
	w.spendConstruction(w.sitePrice(kind))
	if kind == "road" {
		w.Roads = append(w.Roads, p)
	} else {
		w.replaceTowerWalls(candidate)
		w.Buildings = append(w.Buildings, candidate)
		if kind == "ferry" {
			w.pairFerry(candidate)
		}
		w.NextID++
		if isPalace(kind) {
			w.LegacyTaxes = false
		}
	}
	w.refresh()
	return nil
}
func (w *World) Step(ticks int) error {
	if ticks < 1 || ticks > 1000 || w.Tick > MaxTick-int64(ticks) {
		return errors.New("ticks must be 1..1000 within the exact counter range")
	}
	for range ticks {
		w.Tick++
		if w.LaborSeekers && w.Tick%(YearTicks/12) == 0 {
			for i := range w.Buildings {
				b := &w.Buildings[i]
				b.HousesCovered = max(0, b.HousesCovered-1)
			}
		}
		w.resetTradeYear()
		w.moveWalkers()
		w.refresh()
		w.produceAndConsume()
		w.cityTick()
		w.refresh()
		if w.Tick%MigrationInterval == 0 {
			w.dispatch()
			w.refresh()
		}
		if w.Tick%MigrationInterval == 0 {
			w.resizeHousing()
		}
		w.riverTick()
		w.militaryTick()
		w.healthTick()
		w.justiceTick()
		w.refresh()
		w.religionTick()
		w.festivalTick()
		w.campaignTick()
	}
	return nil
}
func (w World) phase() int64 { return (w.Tick + SeasonOffset) % YearTicks }

// A deterministic presentation profile for the provisional seasonal clock.
// Rise during the first quarter, hold at peak, recede over the second half.
func (w World) floodLevel() float64 {
	phase := w.phase()
	if w.CityEvents.FloodYear > 0 && (w.Tick+SeasonOffset)/YearTicks == w.CityEvents.FloodYear && w.CityEvents.FloodQuality == 0 {
		return 0
	}
	if phase >= 48 {
		return 0
	}
	return min(float64(phase)/12, 1, float64(48-phase)/24)
}
func (w World) View() View {
	v := View{Cliffs: w.Cliffs, ConstructionCredit: w.constructionCredit(), InterestPaid: w.Household.Interest, RoyalGiftCosts: w.giftCosts(), KingdomStanding: w.kingdomStanding(), LaborSeekers: w.LaborSeekers, KingdomWage: w.prevailingWage(), CityEvents: w.CityEvents, ZooCoverage: w.zooCoverage(), SalaryRank: w.SalaryRank, Savings: w.Savings, Stolen: w.Stolen, Arrests: w.Arrests, Sentiment: w.sentiment(), Bridges: w.Bridges, MapExtent: w.mapSize(), Banks: w.Banks, Festival: w.Festival, LastFestival: w.LastFestival, FestivalsHeld: w.FestivalsHeld, DiseaseDeaths: w.DiseaseDeaths, Favor: w.Favor, Tick: w.Tick, Roads: w.Roads, Buildings: w.Buildings, Walkers: []WalkerView{}, Entry: w.entry(), Exit: w.ExitPoint, KingdomRoad: append([]Point{}, w.KingdomRoad...), HousingRoadReach: HousingRoadReach, FloodLevel: w.floodLevel(), Population: w.population(), GrainProduced: w.GrainProduced, GrainConsumed: w.GrainConsumed, Departures: w.Departures, Year: (w.Tick+SeasonOffset)/YearTicks + 1, HarvestIn: (192 - w.phase() + YearTicks) % YearTicks}
	v.Culture = w.cultureRating()
	if r := w.ratingState(); r != nil {
		v.Prosperity = r.Prosperity
	}
	v.Season = "Harvest"
	if w.phase() < 48 {
		v.Season = "Inundation"
	} else if w.phase() < 192 {
		v.Season = "Growing"
	}
	for _, b := range w.Buildings {
		v.Grain += b.Stock
		v.Employed += b.Workers
	}
	for _, walker := range w.Walkers {
		v.Walkers = append(v.Walkers, WalkerView{Supplies: walker.Supplies, People: walker.People, ID: walker.ID, X: walker.X, Y: walker.Y, Cargo: walker.Cargo, Role: walker.Role, Target: walker.Target, Origin: walker.Origin, Source: walker.Source, Stage: walker.Stage, Resource: walker.Resource, ShiftUntil: walker.ShiftUntil})
		if walker.Role != "trader" && (walker.Resource == "" || walker.Resource == "grain") {
			v.Grain += walker.Cargo
		}
	}
	v.Workforce = w.availableWorkforce()
	v.PotentialWorkforce = v.Population / 2
	v.MigrationInterval = w.migrationInterval()
	v.RoadFeatures = w.RoadFeatures
	w.cityView(&v)
	w.burialView(&v)
	v.MonumentRating, _ = w.monumentObservation()
	w.miningView(&v)
	w.campaignView(&v)
	w.goodsView(&v)
	w.tradeView(&v)
	w.militaryView(&v)
	w.riverView(&v)
	w.ferryView(&v)
	w.agricultureView(&v)
	return v
}
func (w World) Hash() string {
	b, _ := json.Marshal(w)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
func (w *World) upgrade() error {
	if w.Version == SnapshotVersion {
		if w.MilitaryVersion == 0 {
			w.upgradeMilitary()
		}
		return nil
	}
	if w.Version >= 0 && w.Version < SnapshotVersion && w.MilitaryVersion == 0 {
		w.MilitaryVersion = 1
	}
	if w.Version == 7 || w.Version == 8 || w.Version == 9 {
		if err := w.validateCliffs(); err != nil {
			return err
		}
		if err := w.validateGeography(); err != nil {
			return err
		}
		for _, b := range w.Buildings {
			if b.LotSide < 0 || b.LotSide > 4 {
				return errors.New("invalid historic housing footprint")
			}
		}
		w.Version = SnapshotVersion
		w.refresh()
		return nil
	}
	if w.Version < 7 {
		w.Favor = neutralFavor()
	}
	if w.Version == 6 {
		w.Version = SnapshotVersion
		w.addGoodsTerrain()
		w.addHuntingGrounds()
		w.refresh()
		return nil
	}
	if w.Version >= 0 && w.Version <= 5 {
		w.addGoodsTerrain()
		// Terrain is added only to empty candidate tiles. Existing tax flows
		// continue until a legacy city constructs its first Palace.
		w.addResourceTerrain()
		w.LegacyTaxes = !w.hasPalace()
	}
	if w.Version == 5 {
		w.Version = SnapshotVersion
		w.refresh()
		return nil
	}
	if w.Version == 4 {
		// Keep the old arrival point and all roads/jobs. Never carve a road
		// through an existing saved city to introduce a new starter scenario.
		w.Version = SnapshotVersion
		w.refresh()
		return nil
	}
	if w.Version == 3 {
		w.Version = SnapshotVersion
		w.policyDefaults()
		w.refresh()
		return nil
	}
	if w.Version == 2 {
		w.enableCity()
		return nil
	}
	if w.Version != 0 {
		return errors.New("unsupported snapshot version")
	}
	if w.Departures != 0 || w.GrainProduced != 0 || w.GrainConsumed != 0 {
		return errors.New("invalid legacy snapshot counters")
	}
	for _, b := range w.Buildings {
		if (b.Kind != "hut" && b.Kind != "well") || b.Stock != 0 {
			return errors.New("unsupported legacy building")
		}
	}
	for i := range w.Walkers {
		if w.Walkers[i].Role != "immigrant" || w.Walkers[i].Cargo != 0 {
			return errors.New("unsupported legacy walker")
		}
		w.Walkers[i].Stage = "travel"
	}
	w.enableCity()
	return nil
}
func (w World) Validate() error {
	if err := w.validateCliffs(); err != nil {
		return err
	}
	if err := w.validateGeography(); err != nil {
		return err
	}
	if err := w.validateRoadFeatures(); err != nil {
		return err
	}
	if err := w.validateBurial(); err != nil {
		return err
	}
	if err := w.validateCampaign(); err != nil {
		return err
	}
	for _, mood := range w.Favor {
		if mood < 0 || mood > 100 {
			return errors.New("invalid divine favor")
		}
	}
	if w.Festival != nil && (slices.Index(gods, w.Festival.God) < 0 || w.Festival.Size < 1 || w.Festival.Size > 3 || w.Festival.Due < 1 || w.Festival.Due > MaxTick) {
		return errors.New("invalid festival plan")
	}
	if w.DiseaseDeaths < 0 || w.DiseaseDeaths > w.Departures || w.FestivalsHeld < 0 {
		return errors.New("invalid city history")
	}
	if w.Version != SnapshotVersion || w.Tick < 0 || w.Tick > MaxTick || w.NextID < 1 || w.NextID > MaxTick || w.Arrivals < 0 || w.Departures < 0 || w.GrainProduced < 0 || w.GrainConsumed < 0 || w.Roads == nil || w.Buildings == nil || w.Walkers == nil {
		return errors.New("invalid world header")
	}
	if err := w.validateMapAccess(); err != nil {
		return err
	}
	if err := w.validateAgriculture(); err != nil {
		return err
	}
	if err := w.validateFerries(); err != nil {
		return err
	}
	if err := w.validateRiver(); err != nil {
		return err
	}
	if err := w.validateSeaTrade(); err != nil {
		return err
	}
	if err := w.validateRoyalHousehold(); err != nil {
		return err
	}
	if w.SalaryRank < 0 || w.SalaryRank >= len(salaryByRank) || w.Savings < 0 || w.Arrests < 0 || w.Stolen < 0 {
		return errors.New("invalid personal treasury")
	}
	if err := w.validateBridges(); err != nil {
		return err
	}
	if err := w.validateDefenses(); err != nil {
		return err
	}
	if err := w.validateMilitary(); err != nil {
		return err
	}
	if err := w.validateTrade(); err != nil {
		return err
	}
	if err := w.validateGoods(); err != nil {
		return err
	}
	if err := w.validateMining(); err != nil {
		return err
	}
	occupied := map[Point]bool{}
	for _, ps := range [][]Point{w.QuarryRock, w.Woodland} {
		for _, p := range ps {
			if !w.buildable(p) || occupied[p] {
				return errors.New("invalid resource terrain")
			}
			occupied[p] = true
		}
	}
	for _, p := range w.OreRock {
		occupied[p] = true
	}
	buildingFootprints := map[Point]bool{}
	venueRoads := w.roadSet()
	ids := map[int64]bool{}
	for _, p := range w.Roads {
		if !w.walkable(p) || occupied[p] {
			return errors.New("invalid road")
		}
		occupied[p] = true
	}
	complexFound := false
	for _, b := range w.Buildings {
		if slices.Contains(complexKinds, b.Kind) {
			if complexFound {
				return errors.New("multiple temple complexes")
			}
			complexFound = true
		}
		if err := w.royalTombPlacement(b); err != nil {
			return err
		}
		if err := w.validateMonumentParts(b); err != nil {
			return err
		}
		width, height := b.footprintDimensions()
		if b.MalariaRisk < 0 || b.MalariaRisk > 100 || b.DiseaseRisk < 0 || b.DiseaseRisk > 100 {
			return errors.New("invalid household health risk")
		}
		if b.ServiceSupplyCredit && supplyForService(b.Kind) == "" {
			return errors.New("service credit at unsupported site")
		}
		if b.Facing < 0 || b.Facing > 3 || b.TempleUpgrades > 3 || ((b.TempleUpgrades != 0 || b.Facing != 0) && !slices.Contains(complexKinds, b.Kind)) || b.Kind == "complexaltar" || b.Kind == "complexoracle" || b.HousesCovered < 0 || b.HousesCovered > 300 || (!w.LaborSeekers && b.HousesCovered != 0) || b.Happiness < 0 || b.Happiness > 100 || b.CrimeRisk < 0 || b.CrimeRisk > 100 || b.LotSide < 0 || b.LotSide > 4 || (b.Kind != "hut" && b.LotSide != 0) || !validKind(b.Kind) || b.Width != width || b.Height != height || b.ID < 1 || b.ID >= w.NextID || ids[b.ID] || b.Stock < 0 || b.Stock > b.StockCapacity || b.Pop < 0 || b.Pop > b.Capacity || (b.Kind != "hut" && b.Pop != 0) || b.Work < 0 || b.Work > buildingWorkCapacity(b) || b.MarketUntil < 0 {
			return errors.New("invalid building")
		}
		for _, p := range b.tiles() {
			if !w.buildingTileValid(b, p) || buildingFootprints[p] || (occupied[p] && !((slices.Contains(venues, b.Kind) || b.Kind == "gatehouse") && venueRoads[p])) || ((b.Kind == "claypit" || w.AgricultureVersion == 0 && (b.Kind == "farm" || campCrops[b.Kind] != "")) && !w.floodplain(p)) {
				return errors.New("invalid building footprint")
			}
			occupied[p] = true
			buildingFootprints[p] = true
		}
		ids[b.ID] = true
	}
	if int64(w.population()) != w.Arrivals-w.Departures {
		return errors.New("population does not match arrivals and departures")
	}
	if int64(w.View().Grain)+w.GrainConsumed+w.ResourceExported["grain"]+w.RequestSent["grain"] != w.GrainProduced+w.ResourceImported["grain"]+w.CityEvents.Gifts["grain"] {
		return errors.New("grain conservation failed")
	}
	derived := w
	derived.Buildings = slices.Clone(w.Buildings)
	derived.refresh()
	if derived.Departures != w.Departures {
		return errors.New("invalid population capacity")
	}
	for i := range w.Buildings {
		if !reflect.DeepEqual(w.Buildings[i], derived.Buildings[i]) {
			return fmt.Errorf("inconsistent derived state for %s at (%d, %d)", w.Buildings[i].Kind, w.Buildings[i].X, w.Buildings[i].Y)
		}
	}
	roads := w.roadSet()
	immigrantHomes := map[int64]bool{}
	for _, walker := range w.Walkers {
		if walker.Role != "stonehauler" && (walker.Site != 0 || walker.LoadAt != 0 || walker.DeliverAt != 0) {
			return errors.New("stone journey fields on another role")
		}
		if walker.People < 0 || walker.People > MigrantBatch || walker.Role != "immigrant" && walker.People != 0 {
			return errors.New("invalid migrant party size")
		}
		if walker.Role != "builder" && walker.ShiftUntil != 0 {
			return errors.New("invalid shift on a non-builder walker")
		}
		if walker.Role != "builder" && walker.Role != "stonehauler" && walker.Role != "porter" && walker.Role != "goldcart" && walker.Role != "trader" && !isGoodsRole(walker.Role) && walker.Role != "hunter" && walker.Resource != "" && walker.Resource != "grain" {
			return errors.New("invalid walker resource")
		}
		if walker.Role != "builder" && len(walker.Supplies) > 0 {
			return errors.New("supplies on unrelated walker")
		}
		if walker.ID < 1 || walker.ID >= w.NextID || ids[walker.ID] || walker.Cargo < 0 || walker.Cargo > walkerCapacity(walker.Role) || walker.Reserved < 0 || walker.Reserved > reservationCapacity(walker.Role) || len(walker.Route) == 0 || walker.Next < 1 || walker.Next > len(walker.Route) || walker.Route[walker.Next-1] != (Point{walker.X, walker.Y}) {
			return errors.New("invalid walker")
		}
		ids[walker.ID] = true
		target := w.building(walker.Target)
		if target == nil || (!offRoadOutbound(walker) && !(walker.Role == "trader" && walker.Stage == "leave") && !target.roadAccess(walker.Route[len(walker.Route)-1])) {
			return errors.New("invalid walker destination")
		}
		switch walker.Role {
		case "laborseeker":
			if !w.LaborSeekers || target.ID != walker.Origin || target.ID != walker.Source || target.WorkersNeeded == 0 || (walker.Stage != "patrol" && walker.Stage != "return") || walker.Cargo != 0 || walker.Reserved != 0 {
				return errors.New("invalid labor representative")
			}
		case "robber":
			origin := w.building(walker.Origin)
			if origin == nil || origin.Kind != "hut" || !crimeTarget(*target) || walker.Stage != "robbery" || walker.Cargo != 0 || walker.Reserved != 0 || !origin.roadAccess(walker.Route[0]) {
				return errors.New("invalid robbery")
			}
		case "stonehauler":
			if err := w.validateStoneHauler(walker); err != nil {
				return err
			}
		case "immigrant":
			if immigrantHomes[walker.Target] {
				return errors.New("multiple migrant parties assigned to one home")
			}
			immigrantHomes[walker.Target] = true
			if walker.Cargo != 0 || walker.Reserved != 0 || target.Kind != "hut" || walker.Route[0] != w.entry() {
				return errors.New("invalid migrant")
			}
		case "cart":
			origin, source := w.building(walker.Origin), w.building(walker.Source)
			if origin == nil || origin.Kind != "farm" || source == nil || source.Kind != "granary" || walker.Reserved != 0 || walker.Cargo > 16 {
				return errors.New("invalid farm cart")
			}
			if walker.Stage == "deliver" {
				if target.ID != source.ID || walker.Cargo == 0 || !origin.adjacent(walker.Route[0]) {
					return errors.New("invalid cart delivery")
				}
			} else if walker.Stage == "return" {
				if target.ID != origin.ID || walker.Cargo != 0 || !source.adjacent(walker.Route[0]) {
					return errors.New("invalid cart return")
				}
			} else {
				return errors.New("invalid cart stage")
			}
		case "buyer":
			origin, source := w.building(walker.Origin), w.building(walker.Source)
			if origin == nil || origin.Kind != "bazaar" || source == nil || source.Kind != "granary" || walker.Cargo > 16 {
				return errors.New("invalid bazaar buyer")
			}
			if walker.Stage == "fetch" {
				if target.ID != source.ID || walker.Cargo != 0 || walker.Reserved == 0 || !origin.adjacent(walker.Route[0]) {
					return errors.New("invalid buyer reservation")
				}
			} else if walker.Stage == "return" {
				if target.ID != origin.ID || walker.Reserved != 0 || walker.Cargo == 0 || !source.adjacent(walker.Route[0]) {
					return errors.New("invalid buyer return")
				}
			} else {
				return errors.New("invalid buyer stage")
			}
		case "freight", "goodsbuyer", "goodsseller", "reedharvester", "lumberjack":
			if err := w.validateGoodsWalker(walker, *target); err != nil {
				return err
			}
		case "trader":
			if err := w.validateTrader(walker, *target); err != nil {
				return err
			}
		case "farmworker":
			if err := w.validateFarmWorker(walker, *target); err != nil {
				return err
			}
		case "hunter":
			if err := w.validateHunter(walker, *target); err != nil {
				return err
			}
		case "goldcart":
			if err := w.validateGoldWalker(walker, *target); err != nil {
				return err
			}
		case "performer":
			if err := w.validatePerformer(walker, *target); err != nil {
				return err
			}
		case "courthouse", "physician", "apothecary", "dentist", "mortuary", "scribalschool", "library", "policestation", "senethouse", "entertainer", "porter", "firefighter", "architect", "taxcollector", "watercarrier", "priest", "builder":
			if err := w.validateCityWalker(walker, *target); err != nil {
				return err
			}
		case "seller":
			if target.Kind != "bazaar" || target.ID != walker.Origin || walker.Source != 0 || (walker.Stage != "patrol" && walker.Stage != "return") || walker.Reserved != 0 || (walker.Stage == "patrol" && walker.Route[0] != walker.Route[len(walker.Route)-1]) {
				return errors.New("invalid seller patrol")
			}
		default:
			return errors.New("unknown walker role")
		}
		ferryRoads := w.ferryEdges(true)
		for i, p := range walker.Route {
			if (!offRoadRole(walker.Role) && !roads[p]) || (offRoadRole(walker.Role) && occupied[p] && !roads[p]) || !w.walkable(p) || (i > 0 && !adjacent(p, walker.Route[i-1]) && !ferryRoadsConnect(ferryRoads, p, walker.Route[i-1])) {
				return fmt.Errorf("invalid route at step %d", i)
			}
		}
	}
	for _, b := range w.Buildings {
		if w.reservedStock(b.ID) > b.Stock || b.Stock+w.reservedSpace(b.ID) > b.StockCapacity {
			return errors.New("inventory reservations exceed capacity")
		}
	}
	return w.validateCity()
}

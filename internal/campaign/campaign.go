// Package campaign keeps original campaign configuration separate from the
// simulation. Its state consumes simulation observations, never render events.
package campaign

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
)

//go:embed data/base-campaign.json
var sourceData []byte

const StateVersion = 1
const Nubt = 0
const Thinis = 1
const Perwadjyt = 2
const Nekhen = 3
const Mennefer = 4
const FoodPopulation = 150
const WaterMeat = 400

type Criterion struct {
	Value   int `json:"value"`
	Enabled int `json:"enabled_raw"`
}

type Mission struct {
	ID             int                  `json:"id"`
	Name           string               `json:"name"`
	Period         string               `json:"period"`
	StartYear      int                  `json:"start_year_signed"`
	Rank           int                  `json:"player_rank_raw"`
	EnemyID        int                  `json:"enemy_id_raw"`
	MilestoneYears [3]int               `json:"milestone_years_raw"`
	RescueFunds    int64                `json:"rescue_loan_raw"`
	InitialFunds   int64                `json:"initial_funds"`
	Goals          map[string]Criterion `json:"goals"`
	NextMissionID  []int                `json:"next_mission_ids"`
	Map            MapSize              `json:"map"`
	ComplexFlags   [5]int               `json:"temple_complex_flags_raw"`
	KnownGods      [5]int               `json:"gods_known_raw"`
	MonumentCodes  [3]int               `json:"monument_codes_raw"`
	BurialGoods    map[string]int       `json:"burial_requirements_raw"`
	EditorFlags    [47]int              `json:"editor_allow_flags_raw"`
}

type MapSize struct {
	Width  int `json:"width_tiles"`
	Height int `json:"height_tiles"`
}

var definitions = sync.OnceValue(func() []Mission {
	var catalog struct {
		Missions []Mission `json:"missions"`
	}
	if err := json.Unmarshal(sourceData, &catalog); err != nil {
		panic("invalid embedded original campaign facts: " + err.Error())
	}
	if len(catalog.Missions) != 38 {
		panic("incomplete embedded original campaign facts")
	}
	return catalog.Missions
})

func Definition(id int) (Mission, bool) {
	if id < 0 || id >= len(definitions()) {
		return Mission{}, false
	}
	original := definitions()[id]
	result := original
	result.NextMissionID = slices.Clone(original.NextMissionID)
	result.Goals = make(map[string]Criterion, len(original.Goals))
	for name, criterion := range original.Goals {
		result.Goals[name] = criterion
	}
	result.BurialGoods = make(map[string]int, len(original.BurialGoods))
	for resource, quantity := range original.BurialGoods {
		result.BurialGoods[resource] = quantity
	}
	return result, true
}

// State is included in the world's SQLite snapshot. Unlocks survive later
// population/stock losses, and an achieved victory remains achieved.
type State struct {
	Ratings              *RatingState `json:"ratings,omitempty"`
	EducationOpened      bool         `json:"education_opened,omitempty"`
	TradeOpened          bool         `json:"trade_opened,omitempty"`
	MonumentOpened       bool         `json:"monument_opened,omitempty"`
	FinanceOpened        bool         `json:"finance_opened,omitempty"`
	EventsInitialized    bool         `json:"events_initialized,omitempty"`
	Events               []EventState `json:"events,omitempty"`
	EventRNG             uint32       `json:"event_rng,omitempty"`
	EventMonth           int64        `json:"event_month,omitempty"`
	Kingdom              int          `json:"kingdom,omitempty"`
	RescueReceived       int64        `json:"rescue_received"`
	Version              int          `json:"version"`
	MissionID            int          `json:"mission_id"`
	FoodUnlocked         bool         `json:"food_unlocked"`
	WaterUnlocked        bool         `json:"water_unlocked"`
	FireUnlocked         bool         `json:"fire_unlocked"`
	ArchitectOpened      bool         `json:"architect_opened"`
	ReligionOpened       bool         `json:"religion_opened"`
	EntertainmentOpened  bool         `json:"entertainment_opened"`
	HealthOpened         bool         `json:"health_opened"`
	IndustryOpened       bool         `json:"industry_opened"`
	BeautificationOpened bool         `json:"beautification_opened"`
	Completed            []int        `json:"completed,omitempty"`
	Won                  bool         `json:"won"`
	VictoryTick          int64        `json:"victory_tick"`
}

// Observation uses actual connected/staffed stocks, not lifetime production.
// QualifyingHomes counts occupied houses at or above the scenario housing tier.
type Observation struct {
	Culture           int
	Prosperity        int
	Monuments         int
	Kingdom           int
	BurialComplete    bool
	Tick              int64
	Population        int
	EligibleMeat      int
	EligibleFigs      int
	StoredPottery     int
	StoredBeer        int
	StoredBricks      int
	ApartmentHomes    int
	TeacherDispatched bool
	DiseaseDeaths     int64
	QualifyingHomes   int
	FireCount         int
	CollapseCount     int
	ActiveFirehouses  int
	ActiveArchitects  int
	GoldDelivered     int64
	BastFavor         int
}

func New(id int) (*State, error) {
	if !Playable(id) {
		return nil, errors.New("this campaign mission is not playable yet")
	}
	return NewScenario(id)
}

// NewScenario initializes a source scenario for development verification. Public
// mission starts and handoffs use New, which requires a proven Playable mission.
func NewScenario(id int) (*State, error) {
	if _, ok := Definition(id); !ok {
		return nil, errors.New("unknown campaign scenario")
	}
	s := &State{Version: StateVersion, MissionID: id, Ratings: &RatingState{}}
	if id >= Thinis {
		s.FoodUnlocked, s.WaterUnlocked, s.FireUnlocked, s.ArchitectOpened = true, true, true, true
	}
	if id >= Perwadjyt {
		s.ReligionOpened, s.EntertainmentOpened = true, true
	}
	return s, nil
}

// Observe applies original-supported early tutorial gates. It does not
// import additional victory prerequisites from a modern reconstruction script.
func (s *State) Observe(o Observation) {
	if s == nil {
		return
	}
	if s.MissionID == Nubt {
		s.FireUnlocked = s.FireUnlocked || o.FireCount > 0
		s.ArchitectOpened = s.ArchitectOpened || o.CollapseCount > 0
		s.FoodUnlocked = s.FoodUnlocked || o.Population >= FoodPopulation
		s.WaterUnlocked = s.WaterUnlocked || (s.FoodUnlocked && o.EligibleMeat >= WaterMeat)
	}
	if s.MissionID == Thinis {
		s.ReligionOpened = s.ReligionOpened || o.GoldDelivered >= 500
		s.EntertainmentOpened = s.EntertainmentOpened || (s.ReligionOpened && o.BastFavor >= 40)
	}
	if s.MissionID == Perwadjyt {
		s.HealthOpened = s.HealthOpened || o.DiseaseDeaths > 0
		s.IndustryOpened = s.IndustryOpened || o.EligibleFigs >= 800
		s.BeautificationOpened = s.BeautificationOpened || o.StoredPottery >= 200
	}
	if s.MissionID == Nekhen {
		s.FinanceOpened = s.FinanceOpened || o.StoredBeer >= 300
	}
	if s.MissionID == Mennefer {
		s.EducationOpened = s.EducationOpened || o.ApartmentHomes > 0
		s.TradeOpened = s.TradeOpened || s.EducationOpened && o.TeacherDispatched
		s.MonumentOpened = s.MonumentOpened || s.TradeOpened && o.StoredBricks >= 800
	}
	mission, _ := Definition(s.MissionID)
	if !s.Won && mission.GoalsMet(o.Progress()) {
		s.Won = true
		s.VictoryTick = o.Tick
	}
}

func (s *State) Allows(kind string) bool {
	if s == nil {
		return true
	}
	if s.MissionID >= Nekhen {
		m, _ := Definition(s.MissionID)
		if s.MissionID == Mennefer {
			switch kind {
			case "scribalschool", "papyrusmaker", "reedgatherer":
				return s.EducationOpened
			case "mastaba", "smallmastaba", "bricklayers":
				return s.MonumentOpened
			}
		}
		if s.MissionID == Nekhen && (kind == "courthouse" || kind == "residence") {
			return s.FinanceOpened
		}
		return m.BaseAllows(kind)
	}
	switch kind {
	case "road", "hut", "clear":
		return true
	case "huntinglodge":
		return s.MissionID <= Thinis && s.FoodUnlocked
	case "granary", "bazaar":
		return s.FoodUnlocked
	case "watersupply":
		return s.WaterUnlocked
	case "firehouse":
		return s.FireUnlocked
	case "architect":
		return s.ArchitectOpened
	case "goldmine":
		return s.MissionID == Thinis
	case "palace", "policestation":
		return s.MissionID >= Thinis
	case "temple_bast", "shrine_bast":
		return s.MissionID == Thinis && s.ReligionOpened
	case "temple_osiris", "shrine_osiris", "figfarm", "workcamp":
		return s.MissionID == Perwadjyt
	case "festival_square":
		return s.MissionID >= Thinis && s.ReligionOpened
	case "booth", "jugglerschool":
		return s.MissionID >= Thinis && s.EntertainmentOpened
	case "claypit", "potter", "storageyard":
		return s.MissionID == Perwadjyt && s.IndustryOpened
	case "apothecary", "physician":
		return s.MissionID == Perwadjyt && s.HealthOpened
	case "garden", "plaza", "roadblock":
		return s.MissionID == Perwadjyt && s.BeautificationOpened
	}
	return false
}

func (s *State) Validate(tick int64) error {
	if s == nil {
		return nil
	}
	if s.Version != StateVersion || s.MissionID < Nubt || s.MissionID >= len(definitions()) || s.VictoryTick < 0 || s.VictoryTick > tick {
		return errors.New("invalid campaign state")
	}
	if (s.WaterUnlocked && !s.FoodUnlocked) || (!s.Won && s.VictoryTick != 0) {
		return errors.New("inconsistent campaign milestones")
	}
	mission, _ := Definition(s.MissionID)
	if s.RescueReceived != 0 && s.RescueReceived != mission.RescueFunds {
		return errors.New("invalid campaign rescue loan")
	}
	if s.EntertainmentOpened && !s.ReligionOpened {
		return errors.New("entertainment opened before religion")
	}
	if s.MissionID == Mennefer && (s.TradeOpened && !s.EducationOpened || s.MonumentOpened && !s.TradeOpened) {
		return errors.New("inconsistent Men-nefer tutorial sequence")
	}
	for i, id := range s.Completed {
		if id < 0 || id >= s.MissionID || (i > 0 && s.Completed[i-1] >= id) {
			return errors.New("invalid completed campaign history")
		}
		previous, _ := Definition(id)
		next := s.MissionID
		if i+1 < len(s.Completed) {
			next = s.Completed[i+1]
		}
		if !slices.Contains(previous.NextMissionID, next) {
			return errors.New("completed campaign history does not follow original branches")
		}
	}
	return s.validateEvents()
}

type Objective struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Current  int    `json:"current"`
	Target   int    `json:"target"`
	Complete bool   `json:"complete"`
}

type NextMission struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Playable bool   `json:"playable"`
}

type View struct {
	Requests         []Request     `json:"requests"`
	Kingdom          int           `json:"kingdom"`
	RescueReceived   int64         `json:"rescue_received"`
	MissionID        int           `json:"mission_id"`
	Name             string        `json:"name"`
	StartYear        int           `json:"start_year"`
	Stage            string        `json:"stage"`
	Guidance         string        `json:"guidance"`
	Won              bool          `json:"won"`
	VictoryTick      int64         `json:"victory_tick"`
	Objectives       []Objective   `json:"objectives"`
	AllowedBuildings []string      `json:"allowed_buildings"`
	NextMissions     []NextMission `json:"next_missions"`
	Completed        []int         `json:"completed"`
}

func (s *State) View(o Observation) *View {
	if s == nil {
		return nil
	}
	mission, _ := Definition(s.MissionID)
	v := &View{RescueReceived: s.RescueReceived, MissionID: s.MissionID, Name: mission.Name, StartYear: mission.StartYear,
		Requests: s.Requests(), Kingdom: s.Kingdom,
		Won: s.Won, VictoryTick: s.VictoryTick, AllowedBuildings: []string{}, NextMissions: []NextMission{},
		Objectives: []Objective{{ID: "housing", Label: "Occupied Meager Shanties or better", Current: o.QualifyingHomes,
			Target: mission.Goals["housing_count"].Value, Complete: o.QualifyingHomes >= mission.Goals["housing_count"].Value}}}
	v.Completed = append([]int{}, s.Completed...)
	v.Objectives = mission.Objectives(o.Progress())
	for _, kind := range buildingKinds() {
		if s.Allows(kind) {
			v.AllowedBuildings = append(v.AllowedBuildings, kind)
		}
	}
	for _, id := range mission.NextMissionID {
		next, _ := Definition(id)
		v.NextMissions = append(v.NextMissions, NextMission{ID: id, Name: next.Name, Playable: Playable(id)})
	}
	switch {
	case s.Won:
		v.Stage, v.Guidance = "victory", mission.Name+" has reached its goals. You can continue tending this city or advance to the next available mission."
	case s.MissionID == Nekhen:
		v.Stage, v.Guidance = "apartments", "Grow grain and barley, manufacture pottery and beer, and supply physicians and musicians to ten Modest Apartments. Establish a city of 1,000 residents."
	case s.MissionID == Mennefer && !s.EducationOpened:
		v.Stage, v.Guidance = "apartments", "Develop an occupied Spacious Apartment with food, pottery, beer, clean water, worship, physicians and entertainment to open education."
	case s.MissionID == Mennefer && !s.TradeOpened:
		v.Stage, v.Guidance = "education", "Deliver locally made papyrus to a staffed Scribal School. Its first teacher opens regional commerce."
	case s.MissionID == Mennefer && !s.MonumentOpened:
		v.Stage, v.Guidance = "bricks", "Open a trade route and import 800 bricks into Storage Yards to begin the Small Mastaba."
	case s.MissionID == Mennefer:
		v.Stage, v.Guidance = "monument", "Complete the Small Mastaba and sustain the city's population, culture, prosperity and relations with the Kingdom."
	case s.MissionID == Perwadjyt && !s.IndustryOpened:
		v.Stage, v.Guidance = "floodplain", "Build Fig Farms on the floodplain and staff Work Camps. Store 800 figs in staffed granaries to open pottery production."
		v.Objectives = append(v.Objectives, Objective{"figs", "Figs in connected granaries with at least half staffing", o.EligibleFigs, 800, false})
	case s.MissionID == Perwadjyt && !s.BeautificationOpened:
		v.Stage, v.Guidance = "pottery", "Deliver clay to Potters and store 200 pottery. Bazaar buyers bring household goods from Storage Yards."
		v.Objectives = append(v.Objectives, Objective{"pottery", "Pottery in Storage Yards", o.StoredPottery, 200, false})
	case s.MissionID == Perwadjyt:
		v.Stage, v.Guidance = "housing_evolution", "Supply Modest Homesteads with figs, pottery, clean water, worship and jugglers. Reach 600 residents."
	case s.MissionID == Thinis && !s.ReligionOpened:
		v.Stage, v.Guidance = "gold", "Build a Palace on grassland and a Gold Mine beside metallic rock. Deliver 500 gold to the treasury."
		v.Objectives = append(v.Objectives, Objective{"gold", "Gold delivered to the Palace", int(o.GoldDelivered), 500, false})
	case s.MissionID == Thinis && !s.EntertainmentOpened:
		v.Stage, v.Guidance = "religion", "Honor Bast with a temple and shrine. Entertainment opens when her favor reaches 40."
		v.Objectives = append(v.Objectives, Objective{"bast", "Bast's favor", o.BastFavor, 40, false})
	case s.MissionID == Thinis:
		v.Stage, v.Guidance = "entertainment", "Train jugglers and keep a Booth performing. Supply ten homes with food, clean water, worship and entertainment."
	case s.FireUnlocked && o.ActiveFirehouses == 0:
		v.Stage, v.Guidance = "fire_protection", "Build and staff a Firehouse on the road to protect your village."
	case s.ArchitectOpened && o.ActiveArchitects == 0:
		v.Stage, v.Guidance = "maintenance", "Build and staff an Architect's Post to maintain roadside buildings."
	case !s.FoodUnlocked:
		v.Stage, v.Guidance = "housing", "Connect housing plots to the Kingdom Road and welcome 150 residents."
		v.Objectives = append(v.Objectives, Objective{"population", "Residents to establish food distribution", o.Population, FoodPopulation, false})
	case !s.WaterUnlocked:
		v.Stage, v.Guidance = "food", "Hunters supply meat to staffed granaries. Store 400 meat to open the Water Supply."
		v.Objectives = append(v.Objectives, Objective{"meat", "Meat in connected granaries with at least half staffing", o.EligibleMeat, WaterMeat, false})
	default:
		v.Stage, v.Guidance = "housing_evolution", "Let Water Supply and Bazaar walkers reach six occupied homes so they can become Meager Shanties."
	}
	return v
}

// Next validates branch selection before a caller replaces the current world.
// Unsupported successors leave the won city intact and available to keep playing.
func (s *State) Next(id int) (*State, error) {
	if s == nil || !s.Won {
		return nil, errors.New("complete the current mission before advancing")
	}
	mission, _ := Definition(s.MissionID)
	if !slices.Contains(mission.NextMissionID, id) {
		return nil, fmt.Errorf("mission %d does not follow %s", id, mission.Name)
	}
	next, err := New(id)
	if err != nil {
		return nil, err
	}
	next.Completed = append(slices.Clone(s.Completed), s.MissionID)
	return next, nil
}

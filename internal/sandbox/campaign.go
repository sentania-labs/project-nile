package sandbox

import (
	"errors"
	"slices"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

// NewCampaign uses the original scenario extent and an independently authored Nile.
func NewCampaign(id int) (World, error) {
	state, err := campaign.New(id)
	if err != nil {
		return World{}, err
	}
	w := NewTown()
	w.Campaign = state
	w.Campaign.InitEvents()
	for _, p := range w.partners() {
		if p.InitiallyOpen {
			w.enablePartner(p)
		}
	}
	if id == campaign.Thinis {
		w.Favor = [5]int{30, 30, 30, 30, 30}
	}
	w.addHuntingGrounds()
	mission, _ := campaign.Definition(id)
	w.extendCampaignSite(id, mission.Map.Width)
	w.seedCampaignMaterials(id)
	w.Treasury = mission.InitialFunds
	w.refresh()
	return w, nil
}

func (w World) startingTreasury() int64 {
	if w.Campaign != nil {
		if mission, ok := campaign.Definition(w.Campaign.MissionID); ok {
			return mission.InitialFunds
		}
	}
	return 2500
}

func (w World) campaignObservation() campaign.Observation {
	o := campaign.Observation{Tick: w.Tick, Population: w.population(), FireCount: w.FireCount, CollapseCount: w.CollapseCount, GoldDelivered: w.GoldDelivered, BastFavor: w.Favor[4], DiseaseDeaths: w.DiseaseDeaths}
	o.Monuments, o.BurialComplete = w.monumentObservation()
	level := 2
	if w.Campaign != nil {
		o.Kingdom = w.Campaign.Kingdom
		mission, _ := campaign.Definition(w.Campaign.MissionID)
		level = mission.Goals["housing_level_index"].Value
	}
	for _, b := range w.Buildings {
		if b.Kind == "hut" && b.Pop > 0 && b.Damaged == "" && b.Tier >= level {
			o.QualifyingHomes++
		}
		if !b.Connected || b.Damaged != "" {
			continue
		}
		if b.Kind == "granary" && b.Workers*2 >= b.WorkersNeeded {
			o.EligibleMeat += w.ResourceAmount(b.ID, "meat")
			o.EligibleFigs += w.ResourceAmount(b.ID, "figs")
		}
		if b.Kind == "storageyard" {
			o.StoredPottery += w.ResourceAmount(b.ID, "pottery")
			o.StoredBeer += w.ResourceAmount(b.ID, "beer")
		}
		if b.Workers > 0 {
			if b.Kind == "firehouse" {
				o.ActiveFirehouses++
			}
			if b.Kind == "architect" {
				o.ActiveArchitects++
			}
		}
	}
	return o
}

func (w *World) campaignTick() {
	if w.Campaign != nil {
		w.Campaign.AdvanceEvents(w.Tick / (YearTicks / 12))
		if w.Treasury < 0 && w.Campaign.RescueReceived == 0 {
			mission, _ := campaign.Definition(w.Campaign.MissionID)
			if mission.RescueFunds > 0 {
				w.Campaign.RescueReceived = mission.RescueFunds
				w.Treasury += mission.RescueFunds
				w.Income += mission.RescueFunds
			}
		}
		w.Campaign.Observe(w.campaignObservation())
	}
}

func (w World) campaignView(v *View) {
	v.HuntingGrounds = append([]Point{}, w.HuntingGrounds...)
	if w.Campaign == nil {
		return
	}
	v.Campaign = w.Campaign.View(w.campaignObservation())
	w.campaignRequestView(v.Campaign)
	v.BuildingCatalog = nil
	for _, entry := range buildingCatalog {
		if w.Campaign.Allows(entry.Kind) {
			v.BuildingCatalog = append(v.BuildingCatalog, entry)
		}
	}
}

func (w World) validateCampaign() error {
	if err := w.validateCampaignRequests(); err != nil {
		return err
	}
	if err := w.Campaign.Validate(w.Tick); err != nil {
		return err
	}
	seen := map[Point]bool{}
	occupied := w.occupied()
	for _, p := range w.HuntingGrounds {
		if !w.buildable(p) || occupied[p] || seen[p] {
			return errors.New("invalid hunting grounds")
		}
		seen[p] = true
	}
	for _, b := range w.Buildings {
		if b.WaterUntil < 0 || b.WaterUntil > MaxTick || (b.Kind != "hut" && b.WaterUntil != 0) {
			return errors.New("invalid water service expiration")
		}
		if w.Campaign != nil && !w.Campaign.Allows(b.Kind) {
			return errors.New("building is not unlocked in this campaign")
		}
	}
	for _, f := range w.RoadFeatures {
		if !w.Campaign.Allows(f.Kind) {
			return errors.New("street feature is not unlocked in this campaign")
		}
	}
	return nil
}

func (w *World) AdvanceCampaign(id int) (World, error) {
	state, err := w.Campaign.Next(id)
	if err != nil {
		return World{}, err
	}
	next, err := NewCampaign(id)
	if err != nil {
		return World{}, err
	}
	next.Campaign = state
	return next, nil
}

func (w *World) addHuntingGrounds() {
	if w.HuntingGrounds != nil {
		return
	}
	w.HuntingGrounds = []Point{}
	occupied := w.occupied()
	for _, p := range []Point{{27, 3}, {29, 9}} {
		if !occupied[p] {
			w.HuntingGrounds = append(w.HuntingGrounds, p)
		}
	}
}

// Hunt loads and duration are prototype tuning. The route and cargo are real
// simulation state; meat enters inventory accounting at capture, before return.
const HuntLoad = 100

func (w *World) dispatchHunters() {
	for _, b := range w.Buildings {
		if b.Kind != "huntinglodge" || !b.Connected || b.Damaged != "" || b.Workers < b.WorkersNeeded || w.active(b.ID, "hunter") || w.ResourceFreeSpace(b.ID, "meat") < HuntLoad {
			continue
		}
		route := w.landPath(w.perimeter(b), w.HuntingGrounds)
		if w.spawn("hunter", "hunt", b.ID, b.ID, b.ID, route, 0, HuntLoad) {
			w.Walkers[len(w.Walkers)-1].Resource = "meat"
		}
	}
}

func (w *World) advanceHunter(walker *Walker) bool {
	if walker.Next < len(walker.Route) {
		p := walker.Route[walker.Next]
		walker.X, walker.Y = p.X, p.Y
		walker.Next++
	}
	if walker.Next < len(walker.Route) {
		return true
	}
	if walker.Stage == "hunt" {
		walker.Cargo = w.RecordResourceProduction("meat", HuntLoad)
		walker.Reserved = 0
		reverseJourney(walker, walker.Origin, "return")
		return true
	}
	deposited := w.DepositResource(walker.Origin, "meat", walker.Cargo)
	walker.Cargo -= deposited
	return walker.Cargo > 0
}

func (w World) validateHunter(walker Walker, target Building) error {
	if target.Kind != "huntinglodge" || walker.Origin != target.ID || walker.Source != target.ID || walker.Target != target.ID || walker.Resource != "meat" {
		return errors.New("invalid hunting job")
	}
	first, last := walker.Route[0], walker.Route[len(walker.Route)-1]
	if walker.Stage == "hunt" {
		if walker.Cargo != 0 || walker.Reserved != HuntLoad || !target.adjacent(first) || !slices.Contains(w.HuntingGrounds, last) {
			return errors.New("invalid outbound hunter")
		}
	} else if walker.Stage == "return" {
		if walker.Cargo <= 0 || walker.Cargo > HuntLoad || walker.Reserved != 0 || !slices.Contains(w.HuntingGrounds, first) || !target.adjacent(last) {
			return errors.New("invalid returning hunter")
		}
	} else {
		return errors.New("unknown hunting job stage")
	}
	return nil
}

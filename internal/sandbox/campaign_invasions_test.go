package sandbox

import (
	"testing"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

func TestOriginalLandAndRiverInvasionScaling(t *testing.T) {
	for _, row := range []struct {
		mission, source, soldiers, escorts int
		sea                                bool
	}{{5, 8, 12, 0, false}, {16, 8, 10, 2, true}, {16, 3, 0, 1, true}} {
		w := NewTown()
		if err := w.Build("hut", 9, 2); err != nil {
			t.Fatal(err)
		}
		w.Buildings[0].Pop = 4
		w.Arrivals = 4
		w.refresh()
		var definition campaign.EventDefinition
		for _, d := range campaign.Events(row.mission) {
			if d.ID == row.source {
				definition = d
				break
			}
		}
		if definition.Type != 2 || definition.Amount.Fixed < 0 || definition.Item.Fixed < 0 || definition.Location.Fixed < 0 {
			t.Fatal("fixture requires a fixed original invasion record")
		}
		w.Campaign = &campaign.State{Version: campaign.StateVersion, MissionID: row.mission, EventsInitialized: true, EventRNG: 1, EventMonth: -1, Kingdom: 50,
			Events: []campaign.EventState{{ID: 1, SourceID: row.source, Item: definition.Item.Fixed, Amount: definition.Amount.Fixed, Location: definition.Location.Fixed, Status: "scheduled"}}}
		mission, _ := campaign.Definition(row.mission)
		w.Treasury = mission.InitialFunds + w.Income - w.Expenses
		effects := w.Campaign.AdvanceEvents(0)
		w.applyCampaignInvasions(effects)
		if len(w.Invasions) != 1 || w.Invasions[0].Amount != row.soldiers || w.Invasions[0].Sea != row.sea || w.Campaign.Events[0].Status != "completed" {
			t.Fatalf("original mission%d source%d invasion was not instantiated: %+v effects%+v", row.mission, row.source, w.Invasions, effects)
		}
		escorts := 0
		for _, ship := range w.Boats {
			if ship.Kind == "enemywarship" {
				escorts++
			}
		}
		if escorts != row.escorts {
			t.Fatalf("naval escorts%d, want%d", escorts, row.escorts)
		}
		before := w.Hash()
		w.applyCampaignInvasions(effects)
		if w.Hash() != before {
			t.Fatal("acknowledged invasion duplicated its force or chain")
		}
		if err := w.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

func TestOriginalArmyRequestDispatchesCompaniesResolvesAndReturnsAfterReload(t *testing.T) {
	w := riverFixture(t)
	addGoodsBuilding(t, &w, "recruiter", 11, 12)
	first := addGoodsBuilding(t, &w, "archerfort", 17, 24)
	second := addGoodsBuilding(t, &w, "archerfort", 17, 2)
	runValid(t, &w, 700)
	// Isolate Selima Oasis's actual conditional army request after recruitment.
	// Its chosen seven-strength target is in the original[5,10) range; the
	// fixture schedules activation next month instead of replaying its parent.
	month := w.Tick / (YearTicks / 12)
	w.Campaign = &campaign.State{Version: campaign.StateVersion, MissionID: 8, EventsInitialized: true, EventRNG: 1, EventMonth: month, Kingdom: 50,
		Events: []campaign.EventState{{ID: 1, SourceID: 8, Chained: true, DueMonth: month + 1, Item: 32, Amount: 7, Location: 3, Status: "scheduled"}}}
	w.refresh()
	runValid(t, &w, int(YearTicks/12))
	if destination := os.Getenv("NILE_ARMY_CAPTURE_SAVE"); destination != "" {
		capture, err := Open(destination)
		if err != nil {
			t.Fatal(err)
		}
		if err = capture.Save(w); err != nil {
			t.Fatal(err)
		}
		if err = capture.Close(); err != nil {
			t.Fatal(err)
		}
	}
	before := w.Hash()
	if w.DispatchCampaignRequest(1) == nil || w.Hash() != before {
		t.Fatal("military request accepted goods dispatch")
	}
	if w.DispatchCampaignArmy(1, []int64{first, first}, false) == nil || w.Hash() != before {
		t.Fatal("invalid selection partly consumed request or company")
	}
	if err := w.DispatchCampaignArmy(1, []int64{first, second}, false); err != nil {
		t.Fatal(err)
	}
	if w.ArmyDeploymentStatus(1) != "outbound" || w.Campaign.Events[0].Status != "army_outbound" {
		t.Fatal("army skipped its physical departure or source calendar")
	}
	store, err := Open(filepath.Join(t.TempDir(), "campaign-army.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for range 400 {
		runValid(t, &w, 1)
		runValid(t, &restored, 1)
		seen[w.ArmyDeploymentStatus(1)] = true
		if w.Hash() != restored.Hash() {
			t.Fatal("source army request diverged after reload")
		}
		if w.Campaign.Events[0].Status == "completed" && w.ArmyDeploymentStatus(1) == "returned" {
			break
		}
	}
	if w.Campaign.Events[0].Status != "completed" || w.ArmyDeploymentStatus(1) != "returned" || w.Military.Lost != 2 {
		t.Fatalf("army outcome failed: event%s physical%s losses%d", w.Campaign.Events[0].Status, w.ArmyDeploymentStatus(1), w.Military.Lost)
	}
	for _, state := range []string{"outbound", "awaits_outcome", "awaits_return", "returning", "returned"} {
		if !seen[state] {
			t.Fatalf("army skipped physical state%s", state)
		}
	}
}

package sandbox

import (
	"encoding/json"
	"testing"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

func TestBurialPartialDispatchRespectsOriginalLoadsAndReservations(t *testing.T) {
	w := NewTown()
	w.BudgetEnabled = false
	yard := addGoodsBuilding(t, &w, "storageyard", 11, 10)
	w.Campaign = &campaign.State{MissionID: 10}
	var err error
	if w.burialRequirements()["pottery"] != 1000 || w.burialRequirements()["timber"] != 1000 || w.burialRequirements()["granite"] != 1500 {
		t.Fatal("raw burial loads not converted to100units")
	}
	w.ProduceResource(yard, "pottery", 800)
	if err = w.DispatchBurial("pottery", 3); err != nil {
		t.Fatal(err)
	}
	if w.ResourceAmount(yard, "pottery") != 500 || w.BurialDelivered["pottery"] != 300 || w.burialComplete() {
		t.Fatal("partialdispatchlostinventoryorfinishedgoal")
	}
	before := w.Hash()
	if err = w.DispatchBurial("pottery", 8); err == nil || w.Hash() != before {
		t.Fatal("invaliddispatchmutatedstate")
	}
	if err = w.DispatchBurial("pottery", 5); err != nil {
		t.Fatal(err)
	}
	w.ProduceResource(yard, "pottery", 200)
	if err = w.DispatchBurial("pottery", 2); err != nil {
		t.Fatal(err)
	}
	if w.BurialDelivered["pottery"] != 1000 {
		t.Fatal("completionlostloads")
	}
	if err = w.validateBurial(); err != nil {
		t.Fatal(err)
	}
	// Saqqara's prerequisites are not all playable yet. Exercise the exact
	// snapshot payload without weakening the campaign's load capability gate.
	payload, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err = json.Unmarshal(payload, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Hash() != w.Hash() {
		t.Fatal("burialledgerchangedonrestore")
	}
	if err = loaded.validateBurial(); err != nil {
		t.Fatal(err)
	}

}
func TestOriginalMonumentRatingExamples(t *testing.T) {
	w := NewTown()
	w.Buildings = []Building{{Kind: "mastaba", ConstructionDone: 64}}
	rating, _ := w.monumentObservation()
	if rating != 9 {
		t.Fatalf("smallmastaba rating%d", rating)
	}
	w.Buildings = append(w.Buildings, w.Buildings[0], w.Buildings[0])
	rating, _ = w.monumentObservation()
	if rating != 15 {
		t.Fatalf("three smallmastaba rating%d", rating)
	}
}

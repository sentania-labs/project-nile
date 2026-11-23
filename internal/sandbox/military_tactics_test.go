package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestCompanyMoraleRecoversAtHomeAndRecallsExhaustedFieldCompany(t *testing.T) {
	w, fort := armyFixture(t)
	runValid(t, &w, 1200)
	if w.Companies[0].Stationed != 16 {
		t.Fatal("company did not recruit")
	}
	initial := morale(w.Companies[0])
	if err := w.MilitaryOrder(fort, "loose", Point{25, 22}); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 70)
	if morale(w.Companies[0]) >= initial {
		t.Fatal("extended field duty had no morale cost")
	}
	if err := w.MilitaryOrder(fort, "rotate", Point{}); err != nil {
		t.Fatal(err)
	}
	if !w.Companies[0].Rotated {
		t.Fatal("rank rotation was not retained")
	}
	before := w.Hash()
	if w.MilitaryOrder(fort, "charge", Point{25, 22}) == nil || w.Hash() != before {
		t.Fatal("infantry charged or rejected order mutated state")
	}
	// A tired company is recalled by the actual simulation, then recovers.
	w.Companies[0].Morale = 15
	runValid(t, &w, 100)
	if w.Companies[0].Order != "return" || w.Companies[0].Stationed != w.Companies[0].Soldiers || morale(w.Companies[0]) <= 15 {
		t.Fatalf("tired company did not return/recover %+v", w.Companies[0])
	}
}

func TestMilitaryRulesMigrationRefreshesHistoricalHousingOnce(t *testing.T) {
	w, _ := armyFixture(t)
	// A pre-military-rules city could keep seven residents next to a fort.
	b := w.building(1)
	for i := range w.Buildings {
		if w.Buildings[i].Kind == "hut" && w.Buildings[i].Y == 21 {
			b = &w.Buildings[i]
			break
		}
	}
	w.Arrivals += int64(7 - b.Pop)
	b.Pop = 7
	b.Capacity = 7
	b.Tier = 1
	b.TierName = "Sturdy Hut"
	b.Desirability = 0
	w.MilitaryVersion = 0
	store, err := Open(filepath.Join(t.TempDir(), "old-military.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	payload, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(payload)
	if _, err = store.db.Exec("INSERT INTO snapshots(payload,checksum) VALUES(?,?)", payload, hex.EncodeToString(hash[:])); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.MilitaryVersion != 1 || loaded.population()+int(loaded.Departures) != int(loaded.Arrivals) {
		t.Fatal("rules migration lost population accounting")
	}
	if err = store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	copy, err := store.Load()
	if err != nil || copy.Hash() != loaded.Hash() {
		t.Fatal("current military save changed on reload")
	}
	loaded.building(b.ID).Pop = 1000
	if store.Save(loaded) == nil {
		t.Fatal("current save silently repaired corrupt population")
	}
}

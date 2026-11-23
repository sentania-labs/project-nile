package campaign

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOriginalNubtGatesAndLatchedProgress(t *testing.T) {
	s, _ := New(Nubt)
	if s.Allows("granary") || s.Allows("well") || s.Allows("farm") || !s.Allows("hut") {
		t.Fatal("Nubt must start with roads and housing only")
	}
	s.Observe(Observation{Tick: 1, Population: 149, EligibleMeat: 400})
	if s.FoodUnlocked || s.WaterUnlocked {
		t.Fatal("meat cannot bypass the population tutorial gate")
	}
	s.Observe(Observation{Tick: 2, Population: 150, EligibleMeat: 399, FireCount: 1})
	if !s.Allows("huntinglodge") || !s.Allows("firehouse") || s.Allows("watersupply") || s.Allows("architect") {
		t.Fatal("independent unlock gates were not respected")
	}
	s.Observe(Observation{Tick: 3, Population: 140, EligibleMeat: 400, CollapseCount: 1})
	if !s.WaterUnlocked || !s.ArchitectOpened {
		t.Fatal("water or collapse gate failed")
	}
	s.Observe(Observation{Tick: 4})
	if !s.FoodUnlocked || !s.WaterUnlocked || !s.FireUnlocked || !s.ArchitectOpened {
		t.Fatal("progress was lost when conditions later changed")
	}
}

func TestVictoryUsesSourceHousingGoalAndPersists(t *testing.T) {
	s, _ := New(Nubt)
	s.Observe(Observation{Tick: 100, QualifyingHomes: 5})
	if s.Won {
		t.Fatal("five homes cannot satisfy a six-home goal")
	}
	// No reconstruction-only firehouse/granary/delay prerequisite is applied.
	s.Observe(Observation{Tick: 101, QualifyingHomes: 6})
	if !s.Won || s.VictoryTick != 101 {
		t.Fatal("source housing goal did not trigger victory")
	}
	payload, _ := json.Marshal(s)
	var restored State
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatal(err)
	}
	restored.Observe(Observation{Tick: 102})
	if !restored.Won || restored.VictoryTick != 101 || restored.Validate(102) != nil {
		t.Fatal("restored victory was not stable")
	}
	before, _ := json.Marshal(restored)
	if next, err := restored.Next(1); err != nil || next.MissionID != Thinis || !reflect.DeepEqual(next.Completed, []int{Nubt}) {
		t.Fatal("Thinis transition did not preserve completed mission history")
	}
	if _, err := restored.Next(2); err == nil {
		t.Fatal("branch selection may not skip the original successor")
	}
	after, _ := json.Marshal(restored)
	if string(before) != string(after) {
		t.Fatal("transition mutated the previous campaign state")
	}
}

func TestCampaignFactsKeepDistinctBranchesAndRepeatedNames(t *testing.T) {
	for _, sample := range []struct {
		id   int
		next []int
	}{{0, []int{1}}, {5, []int{6, 7}}, {8, []int{10}}, {9, []int{10}}, {34, []int{36}}, {35, []int{37}}, {37, []int{}}} {
		mission, ok := Definition(sample.id)
		if !ok || !reflect.DeepEqual(mission.NextMissionID, sample.next) {
			t.Fatalf("mission %d successors %v", sample.id, mission.NextMissionID)
		}
	}
	first, _ := Definition(1)
	later, _ := Definition(23)
	if first.Name != later.Name || first.StartYear == later.StartYear {
		t.Fatal("two Thinis scenarios must remain distinct by ID")
	}
	first.Goals["population"] = Criterion{Value: 99999}
	first.NextMissionID[0] = 37
	again, _ := Definition(1)
	if again.Goals["population"].Value == 99999 || again.NextMissionID[0] != 2 {
		t.Fatal("caller mutated shared campaign definitions")
	}
}

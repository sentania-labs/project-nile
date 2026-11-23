package main

import (
	"testing"
	"time"
)

func TestCampaignProtocolPreservesSaveAndRejectsUnavailableMissions(t *testing.T) {
	requests := []string{
		`{"id":1,"op":"save"}`,
		`{"id":2,"op":"new_campaign","mission_id":0}`,
		`{"id":3,"op":"new_campaign","mission_id":37}`,
		`{"id":4,"op":"new_campaign"}`,
		`{"id":5,"op":"advance_campaign","mission_id":1}`,
		`{"id":6,"op":"build","kind":"granary","x":11,"y":21}`,
		`{"id":7,"op":"load"}`,
	}
	responses := runTimedSandbox(t, requests, make([]time.Duration, len(requests)))
	if !responses[1].OK || responses[1].Running || responses[1].State.Campaign == nil || responses[1].State.Campaign.Name != "Nubt" || responses[1].State.Treasury != 2500 || responses[1].State.Population != 0 {
		t.Fatal("new campaign did not start a paused empty Nubt village")
	}
	for i := 2; i <= 5; i++ {
		if responses[i].OK || responses[i].Hash != responses[1].Hash {
			t.Fatalf("request %d bypassed campaign validation or mutated state", i)
		}
	}
	if !responses[6].OK || responses[6].State.Campaign != nil || responses[6].Hash != responses[0].Hash {
		t.Fatal("starting a campaign overwrote the existing save")
	}
}

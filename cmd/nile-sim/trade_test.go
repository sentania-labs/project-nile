package main

import (
	"testing"
	"time"
)

func TestTradeProtocolPoliciesAndPersistence(t *testing.T) {
	requests := []string{
		`{"id":1,"op":"open_trade","route":"kingdom_road"}`,
		`{"id":2,"op":"trade_policy","resource":"stone","mode":"import","limit":400}`,
		`{"id":3,"op":"save"}`,
		`{"id":4,"op":"trade_policy","resource":"stone","mode":"export","limit":50}`,
		`{"id":5,"op":"load"}`,
		`{"id":6,"op":"trade_policy","resource":"gold","mode":"import","limit":100}`,
		`{"id":7,"op":"new_campaign","mission_id":0}`,
		`{"id":8,"op":"open_trade","route":"kingdom_road"}`,
	}
	responses := runTimedSandbox(t, requests, make([]time.Duration, len(requests)))
	for i := 0; i <= 4; i++ {
		if !responses[i].OK {
			t.Fatalf("request%d: %s", i, responses[i].Error)
		}
	}
	if !responses[0].State.TradeRoutes[0].Open || responses[0].State.Treasury != 2300 {
		t.Fatal("route opening not charged")
	}
	if responses[4].Hash != responses[2].Hash || responses[4].State.TradePolicies["stone"].Mode != "import" {
		t.Fatal("trade policy was not restored")
	}
	if responses[5].OK || responses[5].Hash != responses[4].Hash {
		t.Fatal("invalid resource changed state")
	}
	if responses[7].OK || responses[7].Hash != responses[6].Hash {
		t.Fatal("campaign gained an unverified trade route")
	}
}

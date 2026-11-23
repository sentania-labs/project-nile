package main

import (
	"strings"
	"testing"
	"time"
)

func TestArmyCommandValidatesRequestWithoutChangingCity(t *testing.T) {
	r := runTimedSandbox(t, []string{
		`{"id":1,"op":"state"}`,
		`{"id":2,"op":"dispatch_army","fort_ids":[1,2],"naval":true}`,
		`{"id":3,"op":"dispatch_army","request_id":1,"fort_ids":[1,2],"naval":true}`,
	}, []time.Duration{0, 0, 0})
	if r[1].OK || !strings.Contains(r[1].Error, "request_id") || r[2].OK || !strings.Contains(r[2].Error, "campaign") {
		t.Fatalf("invalid army command was accepted or fields rejected: %s / %s", r[1].Error, r[2].Error)
	}
	if r[0].Hash != r[1].Hash || r[0].Hash != r[2].Hash {
		t.Fatal("invalid army dispatch changed city")
	}
}

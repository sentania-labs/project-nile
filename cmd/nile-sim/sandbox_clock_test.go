package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sentania-labs/project-nile/internal/sandbox"
)

func runTimedSandbox(t *testing.T, requests []string, elapsed []time.Duration) []sandboxResponse {
	t.Helper()
	store, err := sandbox.Open(filepath.Join(t.TempDir(), "clock.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	base := time.Now()
	index := 0
	now := func() time.Time {
		if index >= len(elapsed) {
			t.Fatal("unexpected clock read")
		}
		result := base.Add(elapsed[index])
		index++
		return result
	}
	var output bytes.Buffer
	if err := serveSandboxWithClock(strings.NewReader(strings.Join(requests, "\n")+"\n"), &output, store, now); err != nil {
		t.Fatal(err)
	}
	responses := []sandboxResponse{}
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var response sandboxResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatal(err)
		}
		responses = append(responses, response)
	}
	return responses
}

func TestSandboxClockRetainsFractionsAcrossPause(t *testing.T) {
	requests := []string{
		`{"id":1,"op":"state"}`, `{"id":2,"op":"resume"}`,
		`{"id":3,"op":"state"}`, `{"id":4,"op":"state"}`,
		`{"id":5,"op":"pause"}`, `{"id":6,"op":"state"}`,
		`{"id":7,"op":"resume"}`, `{"id":8,"op":"state"}`,
		`{"id":9,"op":"step","ticks":1}`,
	}
	responses := runTimedSandbox(t, requests, []time.Duration{0, 0, 599 * time.Millisecond, 600 * time.Millisecond, 750 * time.Millisecond, 20 * time.Second, 30 * time.Second, 30*time.Second + 50*time.Millisecond, 30*time.Second + 50*time.Millisecond})
	wantTicks := []int64{0, 0, 2, 3, 3, 3, 3, 4, 4}
	wantRunning := []bool{false, true, true, true, false, false, true, true, true}
	for i, response := range responses {
		if response.State.Tick != wantTicks[i] || response.Running != wantRunning[i] || response.OK != (i != 8) {
			t.Fatalf("response %d: %+v", i, response)
		}
	}
}

func TestSandboxClockCatchUpDoesNotTruncateLongGap(t *testing.T) {
	responses := runTimedSandbox(t, []string{`{"id":1,"op":"resume"}`, `{"id":2,"op":"state"}`, `{"id":3,"op":"pause"}`}, []time.Duration{0, 1000*time.Second + 350*time.Millisecond, 1000*time.Second + 400*time.Millisecond})
	if responses[1].State.Tick != 5001 || responses[2].State.Tick != 5002 || responses[2].Running {
		t.Fatalf("long gap was truncated: %+v", responses)
	}
}

func TestSandboxLoadResetsClockAndPausedTimeDoesNotAccumulate(t *testing.T) {
	requests := []string{
		`{"id":1,"op":"step","ticks":8}`, `{"id":2,"op":"save"}`,
		`{"id":3,"op":"resume"}`, `{"id":4,"op":"pause"}`,
		`{"id":5,"op":"load"}`, `{"id":6,"op":"state"}`,
		`{"id":7,"op":"resume"}`, `{"id":8,"op":"state"}`, `{"id":9,"op":"state"}`,
	}
	responses := runTimedSandbox(t, requests, []time.Duration{0, 0, 0, 350 * time.Millisecond, time.Second, 2 * time.Second, 3 * time.Second, 3*time.Second + 50*time.Millisecond, 3*time.Second + 200*time.Millisecond})
	wantTicks := []int64{8, 8, 8, 9, 8, 8, 8, 8, 9}
	for i, response := range responses {
		if !response.OK || response.State.Tick != wantTicks[i] {
			t.Fatalf("response %d: %+v", i, response)
		}
	}
	if responses[4].Running || responses[4].Hash != responses[1].Hash {
		t.Fatal("paused load changed state or resumed simulation")
	}
}

func TestSandboxClockIndependentOfPollingFrequency(t *testing.T) {
	coarse := runTimedSandbox(t, []string{`{"id":1,"op":"resume"}`, `{"id":2,"op":"state"}`}, []time.Duration{0, 3200 * time.Millisecond})
	requests := []string{`{"id":1,"op":"resume"}`}
	elapsed := []time.Duration{0}
	for i := 1; i <= 64; i++ {
		requests = append(requests, `{"id":2,"op":"state"}`)
		elapsed = append(elapsed, time.Duration(i)*50*time.Millisecond)
	}
	fine := runTimedSandbox(t, requests, elapsed)
	if coarse[1].Hash != fine[len(fine)-1].Hash {
		t.Fatal("poll frequency changed simulation state")
	}
}

func TestSandboxSpeedChangePreservesElapsedFraction(t *testing.T) {
	requests := []string{`{"id":1,"op":"resume"}`, `{"id":2,"op":"set_speed","speed":4}`, `{"id":3,"op":"state"}`, `{"id":4,"op":"set_speed","speed":3}`}
	responses := runTimedSandbox(t, requests, []time.Duration{0, 100 * time.Millisecond, 125 * time.Millisecond, 125 * time.Millisecond})
	if responses[1].State.Tick != 0 || responses[1].Speed != 4 || responses[2].State.Tick != 1 || responses[3].OK || responses[3].Speed != 4 {
		t.Fatalf("speed timing: %+v", responses)
	}
}

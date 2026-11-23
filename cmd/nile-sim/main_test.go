package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sentania-labs/project-nile/internal/probe"
)

func TestProtocolErrorsPreserveState(t *testing.T) {
	store, err := probe.Open(filepath.Join(t.TempDir(), "fixture.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	input := `{"id":1,"op":"step","ticks":9}
{"id":2,"op":"step","ticks":-1}
{"id":3,"op":"load"}
{"id":4,"op":"missing"}
{"id":5,"op":"step","ticks":1,"unexpected":true}
{"id":6,"op":"step","ticks":1} {}
not-json
{"id":8,"op":"quit"}
{"id":9,"op":"step","ticks":1}
`
	var output bytes.Buffer
	if err := serve(strings.NewReader(input), &output, store); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 8 {
		t.Fatalf("unexpected response count: %d", len(lines))
	}
	for i, line := range lines {
		var response response
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatal(err)
		}
		if response.State.Tick != 9 || response.State.Grain != 24 {
			t.Fatalf("request %d mutated state: %+v", i, response)
		}
		wantOK := i == 0 || i == 7
		if response.OK != wantOK {
			t.Fatalf("unexpected response %d: %+v", i, response)
		}
	}
}

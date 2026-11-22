package probe

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestDatabaseURI(t *testing.T) {
	for path, want := range map[string]string{
		"/tmp/a ?#.nile":          "file:///tmp/a%20%3F%23.nile",
		"C:/Users/Test/save.nile": "file:///C:/Users/Test/save.nile",
	} {
		if got := databaseURI(path); got != want {
			t.Fatalf("databaseURI(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestReplayAcrossSaveAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.nile")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	continuous := Initial()
	if err := continuous.Step(137); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(continuous); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := continuous.Step(863); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	resumed, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.Step(863); err != nil {
		t.Fatal(err)
	}
	if resumed != continuous || resumed.Hash() != continuous.Hash() {
		t.Fatalf("replay differs: %+v / %+v", resumed, continuous)
	}
}

func TestSingleWriterAndPreviousGeneration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.nile")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	state := Initial()
	for range 3 {
		if err := s.Save(state); err != nil {
			t.Fatal(err)
		}
		if err := state.Step(1); err != nil {
			t.Fatal(err)
		}
	}
	other, err := Open(path)
	if err == nil {
		other.Close()
		t.Fatal("second writer acquired save")
	}
	var count int
	if err := s.db.QueryRow("SELECT count(*) FROM snapshots").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("retained %d snapshots", count)
	}
	var previous []byte
	if err := s.db.QueryRow("SELECT payload FROM snapshots ORDER BY generation ASC LIMIT 1").Scan(&previous); err != nil {
		t.Fatal(err)
	}
	var saved State
	if err := json.Unmarshal(previous, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Tick != 1 {
		t.Fatalf("wrong previous snapshot: %+v", saved)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	other, err = Open(path)
	if err != nil {
		t.Fatalf("lease not released: %v", err)
	}
	other.Close()
}

func TestLoadRejectsCorruptionAndInvalidInvariants(t *testing.T) {
	for _, kind := range []string{"checksum", "invariants", "unknown", "trailing"} {
		t.Run(kind, func(t *testing.T) {
			s, err := Open(filepath.Join(t.TempDir(), "save.nile"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			state := Initial()
			if err := s.Save(state); err != nil {
				t.Fatal(err)
			}
			payload, _ := json.Marshal(state)
			hash := state.Hash()
			switch kind {
			case "checksum":
				hash = "corrupt"
			case "invariants":
				state.Grain = -1
				payload, _ = json.Marshal(state)
				hash = state.Hash()
			case "unknown":
				payload = append(payload[:len(payload)-1], []byte(",\"future\":1}")...)
			case "trailing":
				payload = append(payload, []byte(" {}")...)
			}
			if _, err := s.db.Exec("UPDATE snapshots SET payload=?,checksum=?", payload, hash); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Load(); err == nil {
				t.Fatal("corrupt state loaded")
			}
		})
	}
}

func TestRejectsUnsupportedSchemaWithoutChangingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.nile")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("future format accepted")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 99 {
		t.Fatalf("future schema was modified: %d", version)
	}
}

func TestInvalidStepDoesNotMutate(t *testing.T) {
	for _, ticks := range []int{-1, 0, 10001} {
		state := Initial()
		if state.Step(ticks) == nil || state != Initial() {
			t.Fatalf("invalid step changed state: %d %+v", ticks, state)
		}
	}
	state := Initial()
	state.Tick = MaxTick
	before := state
	if state.Step(1) == nil || state != before {
		t.Fatal("counter overflow accepted")
	}
}

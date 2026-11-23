package sandbox

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestAutosavesRetainTwoManualGenerations(t *testing.T) {
	file := filepath.Join(t.TempDir(), "family city.nile")
	db, err := sql.Open("sqlite", databaseURI(file))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE snapshots(generation INTEGER PRIMARY KEY AUTOINCREMENT,payload BLOB NOT NULL,checksum TEXT NOT NULL);PRAGMA application_id=1313624898;PRAGMA user_version=1"); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	w := NewTown()
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	if err = w.Step(3); err != nil {
		t.Fatal(err)
	}
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		if err = w.Step(10); err != nil {
			t.Fatal(err)
		}
		if err = store.Autosave(w); err != nil {
			t.Fatal(err)
		}
	}
	if restored, err := store.Load(); err != nil || restored.Tick != 43 {
		t.Fatalf("latest %d: %v", restored.Tick, err)
	}
	if restored, err := store.LastManual(); err != nil || restored.Tick != 3 {
		t.Fatalf("last manual %d: %v", restored.Tick, err)
	}
	if restored, err := store.PreviousManual(); err != nil || restored.Tick != 0 {
		t.Fatalf("previous manual %d: %v", restored.Tick, err)
	}
	var count int
	if err = store.db.QueryRow("SELECT count(*) FROM snapshots").Scan(&count); err != nil || count != 4 {
		t.Fatalf("retained %d generations: %v", count, err)
	}
}

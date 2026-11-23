package sandbox

import (
	"path/filepath"
	"testing"
)

func TestValleyRoadShoreAndRestore(t *testing.T) {
	w := NewValley()
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	if w.mapSize() != 96 || w.ExitPoint.Y != 95 {
		t.Fatal("wrong valley extent")
	}
	const y = 78
	x := w.bank(y) + 5
	if err := w.Build("hut", x+1, y); err != nil {
		t.Fatal(err)
	}
	if err := w.Build("hut", w.bank(y)-1, y); err == nil {
		t.Fatal("housing built in river")
	}
	if err := w.Build("farm", w.bank(y)+2, y); err == nil {
		t.Fatal("farm extended outside fertile terrain")
	}
	if err := w.Step(300); err != nil {
		t.Fatal(err)
	}
	if w.population() == 0 {
		t.Fatal("upper valley homes unreachable")
	}
	db, err := Open(filepath.Join(t.TempDir(), "valley.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := db.Load()
	if err != nil || restored.Hash() != w.Hash() {
		t.Fatalf("valley restore failed: %v", err)
	}
}

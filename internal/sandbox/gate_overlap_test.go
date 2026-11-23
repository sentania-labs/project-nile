package sandbox

import "testing"

func TestGatehouseCannotReplaceGateOnRoad(t *testing.T) {
	w := NewValley()
	for _, p := range []Point{{80, 55}, {81, 55}, {80, 56}, {81, 56}} {
		if err := w.Build("road", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Build("gatehouse", 80, 55); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	if err := w.Build("gatehouse", 80, 55); err == nil || w.Hash() != before {
		t.Fatal("duplicate gatehouse changed the city")
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

package sandbox

import "testing"

func TestSeaInvasionCanLandBesideWesternSettlement(t *testing.T) {
	w := NewValley()
	// Geometry fixture: a western home is the only landing target. No claim
	// about city development is made by this isolated route regression.
	home := Building{ID: w.NextID, Kind: "hut", X: 29, Y: 31, Width: 1, Height: 1, Pop: 1}
	w.Buildings = append(w.Buildings, home)
	w.NextID++
	if err := w.SpawnSeaInvasion(1, "foreign", 8, w.seaEntry(), 0); err != nil {
		t.Fatal(err)
	}
	boat := w.Boats[0]
	if boat.Landing.X >= w.westBank(boat.Landing.Y) {
		t.Fatal("western settlement could only be invaded from the east bank")
	}
	for _, p := range boat.Route {
		if !w.navigableWater(p) {
			t.Fatal("invasion boat crossed land", p)
		}
	}
	if abs(boat.Route[len(boat.Route)-1].X-boat.Landing.X)+abs(boat.Route[len(boat.Route)-1].Y-boat.Landing.Y) != 1 {
		t.Fatal("transport did not stop beside landing shore")
	}
}

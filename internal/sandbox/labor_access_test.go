package sandbox

import (
	"path/filepath"
	"testing"
)

func TestLaborSeekerBlockedThenPermittedFromWorkplace(t *testing.T) {
	w := goodsFixture(t)
	w.LaborSeekers = true
	yard := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	entrance := Point{18, 9}
	if err := w.BuildRoads([]Point{entrance}); err != nil {
		t.Fatal(err)
	}
	if err := w.Build("roadblock", entrance.X, entrance.Y); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 120)
	if w.building(yard).Workers != 0 {
		t.Fatal("remote workforce hired through a closed roadblock")
	}
	if err := w.RoadPermission(entrance, "laborseeker", true); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100 && !w.active(yard, "laborseeker"); i++ {
		runValid(t, &w, 1)
	}
	if !w.active(yard, "laborseeker") {
		t.Fatal("no labor seeker on road")
	}
	s, err := Open(filepath.Join(t.TempDir(), "labor.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Save(w); err != nil {
		t.Fatal(err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 200)
	runValid(t, &reloaded, 200)
	if w.Hash() != reloaded.Hash() || w.building(yard).Workers == 0 || w.building(yard).HousesCovered == 0 {
		t.Fatal("physical labor contact did not restore staffing deterministically")
	}
}

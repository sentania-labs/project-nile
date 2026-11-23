package sandbox

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func mastabaFixture(t *testing.T) (World, int64, int64) {
	t.Helper()
	w, id, yard := pyramidFixture(t)
	w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool { return b.ID == id || b.Kind == "stonemasons" || b.Kind == "carpenters" })
	w.TradePolicies = map[string]TradePolicy{}
	addGoodsBuilding(t, &w, "bricklayers", 19, 12)
	if e := w.SetTradePolicy("bricks", "import", 800); e != nil {
		t.Fatal(e)
	}
	w.refresh()
	if dir := os.Getenv("NILE_MASTABA_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "mastaba-prepared.nile")
	}
	id = addGoodsBuilding(t, &w, "smallmastaba", 23, 18)
	if e := w.Validate(); e != nil {
		t.Fatal(e)
	}
	return w, id, yard
}
func TestMastabaOriginalDimensionsMaterialsAndStaging(t *testing.T) {
	for i, kind := range []string{"smallmastaba", "mediummastaba", "largemastaba"} {
		p := monumentPlans[kind]
		sum := 0
		for _, phase := range p.Phases {
			sum += phase.Stone
			if phase.Timber != 0 || phase.Resource != "" && phase.Resource != "bricks" {
				t.Fatalf("mastaba requested non-brick supplies: %+v", phase)
			}
		}
		width, height := dimensions(kind)
		if width != 4+i*2 || height != 10+i*4 || sum != []int{10800, 24000, 74400}[i] || monumentCode(kind) != 18+i {
			t.Fatalf("wrong original %s footprint or quantities: %dx%d, %d", kind, width, height, sum)
		}
		want := Point{2, height}
		if i == 0 {
			want.X = 0
		}
		if monumentStaging(Building{Kind: kind, Width: width, Height: height}) != want {
			t.Fatal("wrong staging point", kind)
		}
	}
}
func TestSmallMastabaPhysicalBricksGuildsAndReplay(t *testing.T) {
	w, id, _ := mastabaFixture(t)
	seen := map[string]bool{}
	sledges := map[int64]bool{}
	s, e := Open(filepath.Join(t.TempDir(), "mastaba.nile"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var replay *World
	for range 14000 {
		if e = w.Step(1); e != nil {
			t.Fatal(e)
		}
		if w.Tick%16 == 0 {
			if e = w.Validate(); e != nil {
				t.Fatalf("tick%d: %v", w.Tick, e)
			}
		}
		for _, a := range w.Walkers {
			if a.Role == "builder" {
				seen[w.building(a.Origin).Kind+"/"+a.Stage] = true
			}
			if a.Role == "stonehauler" {
				seen["sledge/"+a.Stage] = true
				if a.Resource != "bricks" {
					t.Fatal("mastaba requested stone")
				}
				if a.Stage == "deliver" {
					sledges[a.ID] = true
					if a.Cargo != 400 {
						t.Fatal("not an original four-load sledge")
					}
				}
				if replay == nil && a.Stage == "deliver" {
					if e = s.Save(w); e != nil {
						t.Fatal(e)
					}
					r, err := s.Load()
					if err != nil {
						t.Fatal(err)
					}
					replay = &r
				}
			}
		}
		if replay != nil && replay.Tick < w.Tick {
			if e = replay.Step(1); e != nil {
				t.Fatal(e)
			}
			if replay.Hash() != w.Hash() {
				t.Fatalf("in-flight mastaba diverged at tick%d", w.Tick)
			}
		}
		if monumentComplete(*w.building(id)) {
			break
		}
	}
	b := w.building(id)
	if !monumentComplete(*b) || b.ConstructionDone != 10800 || b.MonumentMaterials["bricks"] != 10800 || len(sledges) != 27 {
		t.Fatalf("unfinished original mastaba: stage%s progress%d sledges%d inventory%d", b.Stage, b.ConstructionDone, len(sledges), b.Bricks)
	}
	for _, role := range []string{"workcamp/work", "bricklayers/work", "sledge/collect", "sledge/deliver", "sledge/return"} {
		if !seen[role] {
			t.Fatal("missing physical role", role)
		}
	}
	if seen["carpenters/work"] || seen["stonemasons/work"] || w.BricksConsumed != 10800 || w.GoodsConsumed["timber"] != 0 {
		t.Fatal("wrong mastaba labor or consumption")
	}
	if rating, _ := w.monumentObservation(); rating != 9 {
		t.Fatalf("small mastaba should yield original rating9, got%d", rating)
	}
	if dir := os.Getenv("NILE_MASTABA_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "mastaba-complete.nile")
	}
	t.Logf("original small mastaba completed at tick%d: 27 physical sledges,10800 bricks", w.Tick)
}

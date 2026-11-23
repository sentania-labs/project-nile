package sandbox

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func royalTombFixture(t *testing.T) (World, int64) {
	t.Helper()
	w, id, _ := pyramidFixture(t)
	w.MapExtent = 64
	w.OreRock = nil
	w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool { return b.ID == id || b.Kind == "carpenters" })
	w.TradePolicies = map[string]TradePolicy{}
	for _, y := range []int{4, 7, 10} {
		for x := 28; x <= 52; x++ {
			if e := w.Build("road", x, y); e != nil {
				t.Fatal(e)
			}
		}
	}
	for _, y := range []int{3, 6, 9} {
		for x := 33; x <= 50; x++ {
			h := addGoodsBuilding(t, &w, "hut", x, y)
			w.building(h).Pop = 5
			w.Arrivals += 5
		}
	}
	for x := 23; x <= 25; x++ {
		if e := w.Build("road", x, 24); e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []Point{{9, 15}, {9, 16}} {
		if e := w.Build("road", p.X, p.Y); e != nil {
			t.Fatal(e)
		}
	}
	addGoodsBuilding(t, &w, "artisans", 16, 12)
	addGoodsBuilding(t, &w, "lampmaker", 18, 12)
	addGoodsBuilding(t, &w, "paintmaker", 20, 12)
	addGoodsBuilding(t, &w, "potter", 24, 12)
	addGoodsBuilding(t, &w, "claypit", 7, 15)
	addGoodsBuilding(t, &w, "hennafarm", 7, 10)
	for y := 20; y < 24; y++ {
		for x := 23; x < 31; x++ {
			w.Cliffs = append(w.Cliffs, Point{x, y})
		}
	}
	if e := w.SetTradePolicy("oil", "import", 800); e != nil {
		t.Fatal(e)
	}
	w.refresh()
	if e := w.Validate(); e != nil {
		t.Fatal(e)
	}
	if dir := os.Getenv("NILE_ROYAL_TOMB_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "royaltomb-prepared.nile")
	}
	id = addGoodsBuilding(t, &w, "smallroyaltomb", 23, 20)
	return w, id
}
func TestRoyalTombCliffsAndAtomicPlacement(t *testing.T) {
	w, id := royalTombFixture(t)
	if !w.building(id).StagingAccess {
		t.Fatal("tomb staging inaccessible")
	}
	before := w.Hash()
	for _, kind := range []string{"road", "hut", "smallroyaltomb"} {
		x, y := 27, 20
		if kind == "smallroyaltomb" {
			x, y = 24, 25
		}
		if e := w.Build(kind, x, y); e == nil || w.Hash() != before {
			t.Fatalf("invalid cliff placement accepted or mutated: %s", kind)
		}
	}
	if w.walkable(Point{28, 21}) {
		t.Fatal("people can cross cliffs")
	}
	if e := w.Build("clear", 23, 20); e != nil {
		t.Fatal("empty tomb site cannot be cleared:", e)
	}
	if !w.cliff(Point{23, 20}) || w.walkable(Point{23, 20}) {
		t.Fatal("clearing a tomb removed its underlying cliff")
	}
	v := NewValley()
	if len(v.Cliffs) == 0 {
		t.Fatal("new valley has no cliff sites")
	}
	if e := v.Validate(); e != nil {
		t.Fatal(e)
	}
}
func TestRoyalTombPhysicalProductionSuppliesAndReplay(t *testing.T) {
	w, id := royalTombFixture(t)
	seen := map[string]bool{}
	deliveries := map[int64]bool{}
	var replay *World
	store, e := Open(filepath.Join(t.TempDir(), "tomb.nile"))
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	for range 18000 {
		if e = w.Step(1); e != nil {
			t.Fatal(e)
		}
		if w.Tick%32 == 0 {
			if e = w.Validate(); e != nil {
				t.Fatalf("tick%d stage%s: %v", w.Tick, w.building(id).Stage, e)
			}
		}
		for _, a := range w.Walkers {
			if a.Role == "stonehauler" && a.Stage == "deliver" {
				if a.Resource != "lamps" || a.Cargo != 400 {
					t.Fatal("wrong tomb cargo")
				}
				deliveries[a.ID] = true
			}
			if a.Role == "builder" {
				origin := w.building(a.Origin)
				seen[origin.Kind+"/"+a.Stage] = true
				if origin.Kind == "artisans" && a.Stage == "commute" && replay == nil {
					if a.Supplies["paint"] != 20 || a.Supplies["clay"] != 20 {
						t.Fatal("artisan left without physical supplies")
					}
					if e = store.Save(w); e != nil {
						t.Fatal(e)
					}
					r, err := store.Load()
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
				t.Fatalf("supplied artisan replay diverged at%d", w.Tick)
			}
		}
		if monumentComplete(*w.building(id)) {
			break
		}
	}
	b := w.building(id)
	if !monumentComplete(*b) || b.MonumentMaterials["lamps"] != 1600 || b.MonumentMaterials["paint"] != 80 || b.MonumentMaterials["clay"] != 80 || len(deliveries) != 4 {
		t.Fatalf("tomb incomplete at%d stage%s,materials%v,roles%v,goods%v", w.Tick, b.Stage, b.MonumentMaterials, seen, w.View().ResourceTotals)
	}
	for _, role := range []string{"workcamp/work", "stonemasons/work", "artisans/commute", "artisans/work", "artisans/return"} {
		if !seen[role] {
			t.Fatal("missing physical role", role)
		}
	}
	if w.GoodsProduced["lamps"] < 1600 || w.GoodsProduced["paint"] < 80 || w.GoodsProduced["pottery"] < 1600 || w.ResourceImported["oil"] < 1600 || replay == nil {
		t.Fatal("tomb bypassed local industry or imported oil")
	}
	if e = w.Validate(); e != nil {
		t.Fatal(e)
	}
	if dir := os.Getenv("NILE_ROYAL_TOMB_DIR"); dir != "" {
		saveMonumentFixture(t, w, dir, "royaltomb-complete.nile")
	}
	t.Logf("small Royal Tomb complete at%d, physical lamp deliveries%d, consumed%v", w.Tick, len(deliveries), b.MonumentMaterials)
}

func TestRoyalTombArtisansNeedBothInputsAndReachSite(t *testing.T) {
	w, id := royalTombFixture(t)
	// The full-chain test supplies the real industries. Here isolate the final
	// gate with a consistent already-excavated site and an empty guild.
	b := w.building(id)
	b.MonumentPhase = 5
	b.ConstructionDone = 1600
	b.MonumentMaterials = map[string]int{"lamps": 1600}
	w.RecordResourceProduction("lamps", 1600)
	w.GoodsConsumed = map[string]int64{"lamps": 1600}
	w.Buildings = slices.DeleteFunc(w.Buildings, func(b Building) bool {
		return b.Kind == "lampmaker" || b.Kind == "paintmaker" || b.Kind == "potter" || b.Kind == "claypit" || b.Kind == "hennafarm"
	})
	guild := int64(0)
	for _, v := range w.Buildings {
		if v.Kind == "artisans" {
			guild = v.ID
		}
	}
	w.ProduceResource(guild, "paint", 80)
	w.refresh()
	runValid(t, &w, 100)
	if w.building(id).MonumentPhase != 5 || w.builderCount(id, guild) != 0 {
		t.Fatal("artisan bypassed missing plaster")
	}
	w.ProduceResource(guild, "clay", 80)
	runValid(t, &w, int(MigrationInterval-w.Tick%MigrationInterval))
	if w.building(id).MonumentWork != 0 {
		t.Fatal("artisan worked before walking to the cliff")
	}
	supplied := false
	for _, a := range w.Walkers {
		if a.Origin == guild && a.Stage == "commute" {
			supplied = len(a.Supplies) == 2
		}
	}
	if !supplied {
		t.Fatal("guild did not send its supplies with the artisan")
	}
}

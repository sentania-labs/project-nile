package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"
)

func builderFixture(t *testing.T) World {
	t.Helper()
	w := World{Version: SnapshotVersion, Roads: []Point{}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1}
	for x := 6; x <= 15; x++ {
		if err := w.Build("road", x, 10); err != nil {
			t.Fatal(err)
		}
	}
	for y := 11; y <= 15; y++ {
		if err := w.Build("road", 6, y); err != nil {
			t.Fatal(err)
		}
	}
	for _, b := range []struct {
		kind string
		x, y int
	}{{"hut", 7, 11}, {"well", 7, 13}, {"mastaba", 12, 11}} {
		if err := w.Build(b.kind, b.x, b.y); err != nil {
			t.Fatal(err)
		}
	}
	w.Buildings[0].Pop = 6
	w.Arrivals = 6
	w.Buildings[2].Bricks = 8
	w.BricksProduced = 8
	w.refresh()
	return w
}
func TestBuildersCommuteBeforeConstructionWorkAndReturnHome(t *testing.T) {
	w := builderFixture(t)
	w.Step(8)
	site := w.Buildings[2]
	if site.ConstructionDone != 0 || site.CrewOnSite != 0 || site.CrewEnRoute != 3 {
		t.Fatalf("construction before arrival: %+v", site)
	}
	seenWork, seenReturn := false, false
	for range 180 {
		previous := map[int64]Point{}
		for _, walker := range w.Walkers {
			if walker.Role == "builder" {
				previous[walker.ID] = Point{walker.X, walker.Y}
			}
		}
		before := w.Buildings[2].ConstructionDone
		w.Step(1)
		for _, walker := range w.Walkers {
			if walker.Role != "builder" {
				continue
			}
			if p, ok := previous[walker.ID]; ok && abs(p.X-walker.X)+abs(p.Y-walker.Y) > 1 {
				t.Fatal("builder teleported")
			}
			if walker.Stage == "work" {
				seenWork = true
			}
			if walker.Stage == "return" {
				seenReturn = true
			}
		}
		if w.Buildings[2].ConstructionDone > before && w.Buildings[2].CrewOnSite == 0 {
			t.Fatal("construction without arrived crew")
		}
		if err := w.Validate(); err != nil {
			t.Fatalf("tick%d: %v", w.Tick, err)
		}
	}
	if !seenWork || !seenReturn || w.Buildings[2].ConstructionDone != 8 {
		t.Fatal("shift or brick consumption missing")
	}
}
func TestOnSiteBuilderShiftSaveReplayAndProtectedRoute(t *testing.T) {
	w := builderFixture(t)
	for w.Buildings[2].CrewOnSite == 0 {
		w.Step(1)
	}
	before := w.Hash()
	if w.Build("clear", 10, 10) == nil || w.Hash() != before {
		t.Fatal("removed a builder return route")
	}
	s, err := Open(filepath.Join(t.TempDir(), "crew.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	w.Step(300)
	restored.Step(300)
	if w.Hash() != restored.Hash() {
		t.Fatal("builder shift replay diverged")
	}
}
func TestBuilderCancellationReturnsAlongAlreadyWalkedPath(t *testing.T) {
	w := builderFixture(t)
	w.Step(9)
	old := map[int64]Point{}
	for _, walker := range w.Walkers {
		if walker.Role == "builder" {
			old[walker.ID] = Point{walker.X, walker.Y}
		}
	}
	// A sudden local labor loss cancels the active commute without snapping to
	// the unvisited staging point. Residents remain represented in the world.
	w.Buildings[0].Pop = 1
	w.Departures = 5
	w.refresh()
	w.Step(1)
	for _, walker := range w.Walkers {
		if walker.Role == "builder" {
			if walker.Stage != "return" || old[walker.ID] != (Point{walker.X, walker.Y}) {
				t.Fatal("cancelled builder jumped position")
			}
		}
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestFiscalAndLaborPolicyHasRealEffectsAndPersists(t *testing.T) {
	w := Initial()
	buildMonumentDistrict(t, &w)
	if err := w.SetPolicy(2, 1, "construction"); err != nil {
		t.Fatal(err)
	}
	farmWorkers, monumentWorkers := 0, 0
	for _, b := range w.Buildings {
		if b.Kind == "farm" {
			farmWorkers = b.Workers
		}
		if b.Kind == "mastaba" {
			monumentWorkers = b.Workers
		}
	}
	if farmWorkers != 0 || monumentWorkers == 0 {
		t.Fatal("construction priority did not reassign labor")
	}
	if err := w.SetPolicy(2, 1, "food"); err != nil {
		t.Fatal(err)
	}
	for _, b := range w.Buildings {
		if b.Kind == "farm" && b.Workers != 2 {
			t.Fatal("food priority did not restore farm workers")
		}
	}
	before := w.Hash()
	if w.SetPolicy(-1, 6, "unknown") == nil || w.Hash() != before {
		t.Fatal("invalid policy changed state")
	}
	w.Step(600)
	if err := w.SetPolicy(0, 0, "food"); err != nil {
		t.Fatal(err)
	}
	income, expenses := w.Income, w.Expenses
	w.Step(48)
	if w.Income != income || w.Expenses != expenses {
		t.Fatal("zero rates still charged tax or wages")
	}
	if err := w.SetPolicy(10, 5, "services"); err != nil {
		t.Fatal(err)
	}
	w.Step(48)
	if w.LastIncome == 0 || w.LastWages == 0 {
		t.Fatal("positive rates did not affect ledger")
	}
	s, err := Open(filepath.Join(t.TempDir(), "policy.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if restored.TaxRate != 10 || restored.WageRate != 5 || restored.LaborPriority != "services" || restored.Hash() != w.Hash() {
		t.Fatal("policy did not persist")
	}
}
func TestVersionThreeCityMigrationPreservesTreasury(t *testing.T) {
	w := Initial()
	w.Step(200)
	w.Version = 3
	raw, _ := json.Marshal(w)
	var old map[string]json.RawMessage
	json.Unmarshal(raw, &old)
	for _, key := range []string{"tax_rate", "wage_rate", "labor_priority"} {
		delete(old, key)
	}
	raw, _ = json.Marshal(old)
	sum := sha256.Sum256(raw)
	s, err := Open(filepath.Join(t.TempDir(), "city-v3.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.db.Exec("INSERT INTO snapshots(payload,checksum) VALUES(?,?)", raw, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	restored, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if restored.Treasury != w.Treasury || restored.Income != w.Income || restored.Expenses != w.Expenses || restored.TaxRate != 2 || restored.WageRate != 1 || restored.LaborPriority != "balanced" {
		t.Fatal("migration changed ledger or omitted defaults")
	}
}

func TestZeroWagesStopHiringAndHighTaxesSlowMigration(t *testing.T) {
	w := Initial()
	if err := w.SetPolicy(2, 0, "balanced"); err != nil {
		t.Fatal(err)
	}
	if w.View().Workforce != 0 || w.View().PotentialWorkforce == 0 {
		t.Fatal("zero-wage workforce feedback missing")
	}
	w.Step(16)
	for _, b := range w.Buildings {
		if b.Workers != 0 {
			t.Fatal("unpaid labor still allocated")
		}
	}
	for _, walker := range w.Walkers {
		if walker.Role == "immigrant" {
			t.Fatal("zero wages still recruited migrants")
		}
	}
	if err := w.SetPolicy(10, 1, "balanced"); err != nil {
		t.Fatal(err)
	}
	if w.View().Workforce == 0 || w.View().MigrationInterval != 48 {
		t.Fatal("wage restoration or high-tax delay missing")
	}
	w.Step(24)
	for _, walker := range w.Walkers {
		if walker.Role == "immigrant" {
			t.Fatal("high taxes failed to delay dispatch")
		}
	}
	w.Step(8)
	found := false
	for _, walker := range w.Walkers {
		if walker.Role == "immigrant" {
			found = true
		}
	}
	if !found {
		t.Fatal("high-tax migration did not resume at its interval")
	}
}

func TestDensePatrolWorldKeepsPresentationBelowOneMiB(t *testing.T) {
	w := World{Version: SnapshotVersion, Roads: []Point{}, Buildings: []Building{}, Walkers: []Walker{}, NextID: 1, TaxRate: 2, WageRate: 1, LaborPriority: "balanced"}
	for x := 6; x < Size; x++ {
		for y := 0; y < Size; y++ {
			if x%2 == 0 || y == 0 {
				w.Roads = append(w.Roads, Point{x, y})
			}
		}
	}
	count := 0
	for x := 7; x < Size; x += 2 {
		for y := 1; y < Size; y++ {
			kind := "firehouse"
			if count%2 == 0 {
				kind = "hut"
			}
			if err := w.Build(kind, x, y); err != nil {
				t.Fatal(err)
			}
			count++
		}
	}
	for i := range w.Buildings {
		if w.Buildings[i].Kind == "hut" {
			w.Buildings[i].Pop = 5
			w.Arrivals += 5
		}
	}
	w.refresh()
	w.dispatch()
	w.refresh()
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	presentation, err := json.Marshal(w.View())
	if err != nil {
		t.Fatal(err)
	}
	if len(presentation) > 1024*1024-1024 {
		t.Fatalf("presentation too large: %d", len(presentation))
	}
	if len(snapshot) <= len(presentation)*2 {
		t.Fatal("fixture did not exercise large saved routes")
	}
	t.Logf("%d patrols: save%d bytes, presentation%d bytes", len(w.Walkers), len(snapshot), len(presentation))
}

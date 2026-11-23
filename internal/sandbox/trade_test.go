package sandbox

import (
	"path/filepath"
	"testing"
)

func tradeFixture(t *testing.T) (World, int64) {
	t.Helper()
	w := goodsFixture(t)
	id := addGoodsBuilding(t, &w, "storageyard", 18, 6)
	if err := w.BuildRoads([]Point{{18, 9}}); err != nil {
		t.Fatal(err)
	}
	if err := w.OpenTrade(tradeRouteID); err != nil {
		t.Fatal(err)
	}
	return w, id
}
func trader(w World, stage string) *Walker {
	for i := range w.Walkers {
		if w.Walkers[i].Role == "trader" && w.Walkers[i].Stage == stage {
			return &w.Walkers[i]
		}
	}
	return nil
}
func TestTradeImportPaysAtStorageAndReplaysInFlight(t *testing.T) {
	w, yard := tradeFixture(t)
	before := w.Treasury
	if err := w.SetTradePolicy("stone", "import", 150); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		runValid(t, &w, 1)
		if trader(w, "import") != nil {
			break
		}
	}
	a := trader(w, "import")
	if a == nil || a.Cargo != 100 || w.ResourceImported["stone"] != 0 || w.Treasury != before || w.View().ResourceTotals["stone"] != 0 {
		t.Fatal("foreign cargo changed city accounts before delivery")
	}
	store, err := Open(filepath.Join(t.TempDir(), "trade.nile"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Save(w); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 150)
	runValid(t, &restored, 150)
	if restored.Hash() != w.Hash() {
		t.Fatal("in-flight trade replay diverged")
	}
	if w.ResourceImported["stone"] != 150 || w.ResourceAmount(yard, "stone") != 150 || w.ImportSpent != 57 || w.Treasury != before-57 {
		t.Fatalf("wrong physical import settlement: imported=%v amount=%d spent=%d treasury=%d", w.ResourceImported, w.ResourceAmount(yard, "stone"), w.ImportSpent, w.Treasury)
	}
	if err := w.SetTradePolicy("stone", "export", 50); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		runValid(t, &w, 1)
		if trader(w, "export") != nil {
			break
		}
	}
	if trader(w, "export") == nil || w.ExportEarned != 0 {
		t.Fatal("export paid before storage pickup")
	}
	runValid(t, &w, 100)
	if w.ResourceAmount(yard, "stone") != 50 || w.ResourceExported["stone"] != 100 || w.ExportEarned != 29 || w.View().TaxIncome != w.Income-w.ExportEarned-w.GoldDelivered {
		t.Fatal("wrong export settlement or stock threshold")
	}
}
func TestTradeHonorsOrdersFundsAndChangedPolicies(t *testing.T) {
	w, yard := tradeFixture(t)
	if err := w.SetTradePolicy("timber", "import", 100); err != nil {
		t.Fatal(err)
	}
	if err := w.SetStorageOrder(18, 6, "timber", "refuse"); err != nil {
		t.Fatal(err)
	}
	runValid(t, &w, 48)
	if trader(w, "import") != nil {
		t.Fatal("trader ignored storage refusal")
	}
	w.SetStorageOrder(18, 6, "timber", "accept")
	for range 48 {
		runValid(t, &w, 1)
		if trader(w, "import") != nil {
			break
		}
	}
	if trader(w, "import") == nil {
		t.Fatal("no trader dispatched")
	}
	w.SetTradePolicy("timber", "off", 100)
	runValid(t, &w, 70)
	if w.ImportSpent != 0 || w.ResourceAmount(yard, "timber") != 0 {
		t.Fatal("cancelled policy still purchased cargo")
	}
	w.SetTradePolicy("timber", "import", 100)
	w.Treasury = 2
	runValid(t, &w, 80)
	if w.ImportSpent != 0 {
		t.Fatal("purchased unaffordable unit")
	}
	w.Treasury = 23
	runValid(t, &w, 90)
	if w.ResourceImported["timber"] != 10 || w.ImportSpent != 23 || w.Treasury != 0 {
		t.Fatalf("partial affordable load wrong: imported=%v spent=%d treasury=%d", w.ResourceImported, w.ImportSpent, w.Treasury)
	}
}
func TestTradeLegacyResourceConservationAndCapacity(t *testing.T) {
	for _, r := range []string{"grain", "clay", "straw", "bricks"} {
		t.Run(r, func(t *testing.T) {
			w, yard := tradeFixture(t)
			cap := resourceLimit(*w.building(yard), r)
			w.SetTradePolicy(r, "import", 3200)
			runValid(t, &w, 220)
			if got := w.ResourceAmount(yard, r); got != cap || w.ResourceImported[r] != int64(cap) {
				t.Fatalf("finite capacity wrong %s: stock%d cap%d imports%d", r, got, cap, w.ResourceImported[r])
			}
			w.SetTradePolicy(r, "export", 0)
			runValid(t, &w, 220)
			if w.ResourceAmount(yard, r) != 0 || w.ResourceExported[r] != int64(cap) {
				t.Fatal("legacy export lost or duplicated inventory")
			}
		})
	}
}
func TestTradeAnnualQuotaAndAtomicPolicyValidation(t *testing.T) {
	w, _ := tradeFixture(t)
	w.SetTradePolicy("stone", "import", 200)
	before := w.Hash()
	for _, f := range []func() error{func() error { return w.OpenTrade("missing") }, func() error { return w.SetTradePolicy("gold", "import", 100) }, func() error { return w.SetTradePolicy("stone", "buy", 100) }, func() error { return w.SetTradePolicy("stone", "export", -1) }} {
		if f() == nil || w.Hash() != before {
			t.Fatal("invalid trade request changed state")
		}
	}
	// A completed year's history with already-consumed stone tests the boundary
	// without manufacturing 1500 units through unrelated building fixtures.
	w.ResourceImported = map[string]int64{"stone": 1500}
	w.GoodsConsumed = map[string]int64{"stone": 1500}
	w.TradeUsed = map[string]int{"stone": 1500}
	runValid(t, &w, 20)
	if trader(w, "import") != nil {
		t.Fatal("annual quota exceeded")
	}
	for w.TradeYear == 1 {
		runValid(t, &w, 1)
	}
	for range 32 {
		runValid(t, &w, 1)
		if trader(w, "import") != nil {
			break
		}
	}
	if trader(w, "import") == nil {
		t.Fatal("annual quota did not renew")
	}
	a := trader(w, "import")
	if err := w.Build("clear", a.Route[0].X, a.Route[0].Y); err == nil {
		t.Fatal("active route could be cleared")
	}
	if err := w.Build("clear", 18, 6); err == nil {
		t.Fatal("active destination could be cleared")
	}
	w.ResourceImported["stone"]++
	if w.Validate() == nil {
		t.Fatal("corrupt import ledger accepted")
	}
}

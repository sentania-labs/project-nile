package sandbox

import (
	"path/filepath"
	"testing"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

func TestGiftAndChangedPricePreserveIncomingDelivery(t *testing.T) {
	w, yard := tradeFixture(t)
	if err := w.SetTradePolicy("stone", "import", 100); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100 && trader(w, "import") == nil; i++ {
		runValid(t, &w, 1)
	}
	if trader(w, "import") == nil {
		t.Fatal("expected incoming stone")
	}
	free := w.ResourceFreeSpace(yard, "stone")
	if !w.civilianEffect(campaign.EventEffect{Type: 23, Item: 24, Amount: 100}) {
		t.Fatal("gift ignored")
	}
	if w.CityEvents.Gifts["stone"] != int64(free) || w.ResourceAmount(yard, "stone") != free || w.ResourceFreeSpace(yard, "stone") != 0 {
		t.Fatal("gift consumed reserved capacity")
	}
	before, _ := w.marketPrice("stone")
	w.civilianEffect(campaign.EventEffect{Type: 15, Item: 24, Amount: 3})
	price, _ := w.marketPrice("stone")
	if price.Buy != before.Buy+3 || price.Sell != before.Sell+3 {
		t.Fatal("incorrect market update")
	}
	w.refresh()
	s, err := Open(filepath.Join(t.TempDir(), "gift.nile"))
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
	runValid(t, &w, 90)
	runValid(t, &restored, 90)
	if w.Hash() != restored.Hash() {
		t.Fatal("gift or market update diverged after restoration")
	}
}

func TestTimnaRoyalTradeRouteTargetsMenNefer(t *testing.T) {
	w := NewTown()
	w.Campaign = &campaign.State{MissionID: 5}
	const city = "city_1"
	if err := w.openPartner(city); err == nil {
		t.Fatal("dormant route opened early")
	}
	funds := w.Treasury
	if !w.civilianEffect(campaign.EventEffect{Type: 19, Location: 5, Subtype: 2}) || !w.OpenPartners[city] || w.Treasury != funds {
		t.Fatal("royal opening did not unlock original route")
	}
	if id, quota := w.routeAllowance(false, "pottery", "import", city); id != city || quota != 1500 {
		t.Fatal("wrong latent pottery quota", id, quota)
	}
	w.usePartner(city, "pottery", 100)
	w.TradeUsed["pottery"] = 100
	w.civilianEffect(campaign.EventEffect{Type: 19, Location: 5, Subtype: 3})
	if _, n := w.routeAllowance(false, "pottery", "import", city); n != 0 || w.PartnerUsed[city]["pottery"] != 100 || w.openPartner(city) == nil {
		t.Fatal("closed route lost its ledger or kept trading")
	}
	if err := w.validatePartners(); err != nil {
		t.Fatal(err)
	}
	w.civilianEffect(campaign.EventEffect{Type: 19, Location: 5, Subtype: 2})
	if _, n := w.routeAllowance(false, "pottery", "import", city); n != 1400 {
		t.Fatal("route regained spent quota after reopening", n)
	}
}

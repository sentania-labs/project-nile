package sandbox

import "testing"

func TestAnnualRoyalCreditAndPersonalGift(t *testing.T) {
	w := World{RoyalCredit: true, BudgetEnabled: true, Treasury: -4900, Savings: 420}
	if w.cannotBuildFor(100) || !w.cannotBuildFor(101) {
		t.Fatal("construction crossed the credit limit")
	}
	w.Tick = YearTicks / 12
	w.settleMonthlyAccounts()
	if w.Treasury != -4900 {
		t.Fatal("charged annual interest early")
	}
	w.Buildings = []Building{{Kind: "complex_ra", Workers: 1, Connected: true}}
	w.Tick = YearTicks
	w.settleMonthlyAccounts()
	if w.Treasury != -5145 || w.Household.Interest != 245 {
		t.Fatalf("interest on debt: %v", w.Household)
	}
	if err := w.SendRoyalGift(1); err != nil {
		t.Fatal(err)
	}
	if w.kingdomStanding() != 55 || w.Savings != 265 || w.Treasury != -5145 {
		t.Fatal("incorrect gift account")
	}
	if err := w.DonateSavings(10); err != nil {
		t.Fatal(err)
	}
	if w.Savings != 255 || w.Household.Donated != 10 || w.Treasury != -5135 {
		t.Fatal("donation failed")
	}
	if err := w.validateRoyalHousehold(); err != nil {
		t.Fatal(err)
	}
}

func TestComplexBenefitsRequireWorkersAndPriests(t *testing.T) {
	w := World{Tick: ConsumptionInterval, Buildings: []Building{
		{ID: 1, Kind: "complex_osiris", Workers: 50, Connected: true, TempleUpgrades: 1, Width: 7, Height: 13},
		{ID: 2, Kind: "hut", Pop: 40, Stock: 10, X: 9, Y: 12, Width: 1, Height: 1},
	}}
	w.visitService(Walker{Role: "priest", Origin: 1, X: 8, Y: 12})
	w.produceGoods()
	if w.Buildings[1].Stock != 4 || w.GrainConsumed != 6 {
		t.Fatal("household did not stretch food after a Sebek visit")
	}
	w.Buildings[0].Workers = 0
	w.Buildings[1].Stock = 10
	w.produceGoods()
	if w.Buildings[1].Stock != 2 {
		t.Fatal("unemployed complex still granted benefit")
	}
	w.Buildings[0].Kind = "complex_bast"
	w.Buildings[0].Workers = 50
	w.Buildings[1].DiseaseRisk = 82
	w.visitService(Walker{Role: "priest", Origin: 1, X: 8, Y: 12})
	if w.Buildings[1].DiseaseRisk != 0 {
		t.Fatal("Isis priest failed to treat household")
	}
}

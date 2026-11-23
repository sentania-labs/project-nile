package sandbox

import (
	"path/filepath"
	"testing"
)

func TestCampaignRequestUsesAvailableStockAndPersistsInFlight(t *testing.T) {
	w, err := NewCampaign(4)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.Build("storageyard", 11, 2); err != nil {
		t.Fatal(err)
	}
	if err = w.Build("storageyard", 11, 7); err != nil {
		t.Fatal(err)
	}
	var target int
	var due int64
	for _, e := range w.Campaign.Events {
		if e.SourceID == 2 {
			target, due = e.ID, e.DueMonth
		}
	}
	w.Tick = due * (YearTicks / 12)
	w.Campaign.AdvanceEvents(due)
	r, err := w.Campaign.Request(target)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.DispatchCampaignRequest(target); err == nil {
		t.Fatal("a request consumed missing goods")
	}
	if len(w.RequestSent) != 0 {
		t.Fatal("rejected request changed its conservation ledger")
	}
	left := r.Quantity
	for _, b := range w.Buildings {
		if b.Kind == "storageyard" {
			left -= w.ProduceResource(b.ID, r.Resource, left)
		}
	}
	if left != 0 {
		t.Fatal("fixture storage too small")
	}
	if err = w.DispatchCampaignRequest(target); err != nil {
		t.Fatal(err)
	}
	if w.requestStock(r.Resource) != 0 || w.RequestSent[r.Resource] != int64(r.Quantity) {
		t.Fatal("request did not remove its exact cargo")
	}
	if err = w.DispatchCampaignRequest(target); err == nil {
		t.Fatal("a dispatched request consumed goods twice")
	}
	w.refresh()
	if err = w.Validate(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(filepath.Join(t.TempDir(), "request.nile"))
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
	if restored.Hash() != w.Hash() {
		t.Fatal("request delivery or stock ledger changed on reload")
	}
	if err = w.Step(100); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		if err = restored.Step(1); err != nil {
			t.Fatal(err)
		}
	}
	if restored.Hash() != w.Hash() {
		t.Fatal("request outcomes depend on tick batching after reload")
	}
	if err = w.Validate(); err != nil {
		t.Fatal(err)
	}
}

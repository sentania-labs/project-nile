package sandbox

import "testing"

func TestMenneferCreditsCompletedMastabaInEitherOrder(t *testing.T) {
	w, err := NewCampaign(4)
	if err != nil {
		t.Fatal(err)
	}
	empty := Building{Kind: "smallmastaba"}
	complete := empty
	complete.MonumentPhase = len(monumentPlans[complete.Kind].Phases)
	for _, sites := range [][]Building{{empty, complete}, {complete, empty}, {complete, complete}} {
		w.Buildings = sites
		if rating, _ := w.monumentObservation(); rating != 9 {
			t.Fatalf("earned %d monument rating, want 9", rating)
		}
	}
}

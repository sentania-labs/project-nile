package sandbox

import (
	"math"
	"strconv"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

// Original scenario-code families and weights:table0x00565a60, stride16.
// Ratings use the original0x004db190 sqrt(points)*6.32+.5, capped100.
var monumentWeights = []int{0, 4, 10, 4, 12, 26, 48, 80, 3, 9, 20, 36, 60, 4, 12, 26, 48, 80, 2, 3, 11, 10, 1, 3, 4, 5, 5, 5}

func monumentCode(kind string) int {
	if kind == "mastaba" {
		return 18
	}
	if kind == "sphinx" {
		return 21
	}
	p, ok := monumentPlans[kind]
	if !ok {
		return 0
	}
	switch p.Family {
	case "bent":
		return 1 + p.Size
	case "brick":
		return 3 + p.Size
	case "stepped":
		return 8 + p.Size
	case "pyramid":
		return 13 + p.Size
	}
	return 0
}
func monumentPercent(b Building) int {
	if monumentComplete(b) {
		return 100
	}
	if b.ConstructionRequired <= 0 {
		return 0
	}
	if b.Kind == "mastaba" {
		return b.ConstructionDone * 100 / 64
	}
	if b.Kind == "sphinx" {
		return min(99, b.ConstructionDone*100/b.ConstructionRequired)
	}
	if b.MonumentPhase < 3 {
		return b.MonumentPhase * 5 / 3
	}
	return min(99, 5+b.ConstructionDone*95/b.ConstructionRequired)
}
func (w World) monumentObservation() (int, bool) {
	allowed := map[int]int{}
	if w.Campaign != nil {
		m, _ := campaign.Definition(w.Campaign.MissionID)
		for _, code := range m.MonumentCodes {
			if code > 0 {
				allowed[code]++
			}
		}
	}
	points := 0
	for _, b := range w.Buildings {
		code := monumentCode(b.Kind)
		if code <= 0 || code >= len(monumentWeights) {
			continue
		}
		if w.Campaign != nil {
			if allowed[code] <= 0 {
				continue
			}
			allowed[code]--
		}
		points += monumentWeights[code] * monumentPercent(b) / 100
	}
	if w.Campaign != nil {
		mission, _ := campaign.Definition(w.Campaign.MissionID)
		for raw, n := range mission.BurialGoods {
			if n <= 0 {
				continue
			}
			id, _ := strconv.Atoi(raw)
			loads := w.BurialDelivered[campaign.ResourceID(id)] / 100
			switch id {
			case 1, 8, 18, 20, 24, 25, 26, 30:
				points += loads / 32
			case 10, 13, 15, 17, 19, 23, 28:
				points += loads / 16
			}
		}
	}
	return min(100, int(math.Sqrt(float64(points))*6.32+0.5)), w.burialComplete()
}

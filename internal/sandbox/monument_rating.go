package sandbox

import (
	"math"
	"slices"
	"strconv"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

// Original scenario-code families and weights:table0x00565a60, stride16.
// Ratings use the original0x004db190 sqrt(points)*6.32+.5, capped100.
var monumentWeights = [...]int{0, 4, 10, 4, 12, 26, 48, 80, 3, 9, 20, 36, 60, 4, 12, 26, 48, 80, 2, 3, 11, 10, 1, 3, 4, 5, 5, 5}

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
	case "suntemple":
		return 24
	case "mausoleum":
		return 25 + p.Size
	case "obelisk":
		return 22 + p.Size
	case "mastaba":
		return 18 + p.Size
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
	if monumentPlans[b.Kind].Family == "obelisk" {
		return min(99, 33+b.ConstructionDone*66/b.ConstructionRequired)
	}
	if isRoyalTomb(b.Kind) {
		return min(99, b.MonumentPhase*100/len(monumentPlans[b.Kind].Phases))
	}
	if b.Kind == "sphinx" {
		return min(99, b.ConstructionDone*100/b.ConstructionRequired)
	}
	if monumentPlans[b.Kind].Family == "mausoleum" {
		if b.MonumentPhase == 0 {
			return 0
		}
		return min(99, 5+b.ConstructionDone*95/b.ConstructionRequired)
	}
	if b.Kind == "suntemple" {
		if b.MonumentPhase < 13 {
			return b.MonumentPhase * 6 / 13
		}
		return min(99, 6+max(0, b.ConstructionDone-13)*94/16000)
	}
	if monumentPlans[b.Kind].Family == "mastaba" {
		if b.MonumentPhase == 0 {
			return 0
		}
		return min(99, 5+b.ConstructionDone*95/b.ConstructionRequired)
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
	var credited [len(monumentWeights)][]int
	for _, b := range w.Buildings {
		if b.Kind == "mastaba" && w.Campaign != nil {
			continue
		}
		code := monumentCode(b.Kind)
		if code <= 0 || code >= len(monumentWeights) {
			continue
		}
		value := monumentWeights[code] * monumentPercent(b) / 100
		if w.Campaign == nil {
			points += value
			continue
		}
		if allowed[code] > 0 {
			credited[code] = append(credited[code], value)
		}
	}
	if w.Campaign != nil {
		for code, values := range credited {
			slices.Sort(values)
			for _, value := range values[max(0, len(values)-allowed[code]):] {
				points += value
			}
		}
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

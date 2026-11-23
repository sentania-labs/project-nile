package sandbox

import "github.com/sentania-labs/project-nile/internal/campaign"

// A campaign location selects an independently authored edge approach. These
// positions deliberately do not contain the proprietary scenario's map data.
func (w World) campaignInvasionEntry(location int) Point {
	n := w.mapSize()
	positions := []Point{{n - 1, n / 4}, {n - 1, n * 3 / 4}, {n / 2, 0}, {n / 2, n - 1}, {n - 1, n / 2}, {n * 3 / 4, 0}, {n * 3 / 4, n - 1}, {n - 1, 0}}
	if location < 1 || location > len(positions) {
		location = 1
	}
	return positions[location-1]
}

func (w *World) applyCampaignInvasions(effects []campaign.EventEffect) {
	definition, _ := campaign.Definition(w.Campaign.MissionID)
	for _, effect := range effects {
		if effect.Type != 2 {
			continue
		}
		class := map[int]string{1: "foreign", 2: "egyptian", 3: "kingdom", 4: "bedouin"}[effect.Item]
		if class == "" {
			continue
		}
		// Original 0x0043ab20 and table0x00565a48: Normal difficulty80%,
		// both paths cap at150; naval forces skip difficulty scaling.
		naval := effect.Location >= 9 && effect.Item != 4
		amount := min(150, effect.Amount*80/100)
		if naval {
			amount = min(150, effect.Amount)
		}
		if amount == 0 && (!naval || effect.Escorts == 0) {
			_ = w.Campaign.CompleteEffect(effect.ID)
			continue
		}
		if w.invasion(effect.ID) == nil {
			var err error
			if naval {
				y := 0
				if effect.Location%2 == 0 {
					y = w.mapSize() - 1
				}
				err = w.SpawnSeaInvasion(effect.ID, class, amount, Point{(w.bank(y) + w.westBank(y)) / 2, y}, effect.Escorts)
			} else {
				err = w.SpawnInvasion(effect.ID, class, amount, w.campaignInvasionEntry(effect.Location), definition.EnemyID)
			}
			if err != nil {
				continue
			}
		}
		// The original starts the completion chain when the invasion appears;
		// its subsequent military result is independent (0x0043a110).
		_ = w.Campaign.CompleteEffect(effect.ID)
	}
}

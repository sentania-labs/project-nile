package sandbox

// Original Normal Figure_model_normal.txt, ALL ENEMIES, five rows per culture.
// Loader0x004a6320 maps its infantry/missile/chariot rows onto figure44/43/45;
// spawn0x0043ab20 uses their frequency field, assigning rounding to infantry.
// The same field for Egyptian figures54..56 is60/40/0. Bedouin99 is100% foot.
// These are factual composition values; local combat cadence is still separate.
var enemyShares = [10][3]int{
	{60, 20, 20}, // Hyksos
	{50, 50, 0},  // Sea People
	{40, 20, 40}, // Hittite
	{60, 40, 0},  // Mitani
	{75, 25, 0},  // Kushite
	{40, 60, 0},  // Libyan
	{70, 30, 0},  // Nubian
	{50, 50, 0},  // Canaanite
	{60, 40, 0},  // Egyptian / Kingdom
	{100, 0, 0},  // Bedouin
}

func invasionCulture(class string, requested []int) (int, bool) {
	culture := 7
	if len(requested) > 1 {
		return 0, false
	}
	if len(requested) == 1 {
		culture = requested[0]
	}
	if class == "egyptian" || class == "kingdom" {
		culture = 8
	}
	if class == "bedouin" {
		culture = 9
	}
	return culture, culture >= 0 && culture < len(enemyShares)
}

func invasionUnits(class string, culture, amount int) []string {
	archers, chariots := 0, 0
	if class == "archer" {
		archers = amount
	} else if class == "charioteer" {
		chariots = amount
	} else if class != "test" {
		archers = amount * enemyShares[culture][1] / 100
		chariots = amount * enemyShares[culture][2] / 100
	}
	units := make([]string, 0, amount)
	for i, count := range []int{amount - archers - chariots, archers, chariots} {
		for range count {
			units = append(units, []string{"infantry", "archer", "charioteer"}[i])
		}
	}
	return units
}

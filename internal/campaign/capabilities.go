package campaign

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
	"sync"
)

//go:embed data/base-campaign-economy.json
var economyData []byte

//go:embed data/cleopatra-campaign-economy.json
var expansionEconomyData []byte

type City struct {
	ArmyTravelMonths int         `json:"army_travel_months"`
	PotentialBuys    []int       `json:"potential_buys"`
	PotentialSells   []int       `json:"potential_sells"`
	EmpireObject     int         `json:"empire_object_id"`
	Name             string      `json:"name"`
	AnnualLimits     map[int]int `json:"annual_limits"`
	ID               int         `json:"id"`
	Type             int         `json:"type_raw"`
	NameID           int         `json:"name_id_raw"`
	Route            int         `json:"route_id"`
	Open             bool        `json:"initially_open"`
	Buys             []int       `json:"buys"`
	Sells            []int       `json:"sells"`
	Cost             int         `json:"cost_to_open"`
	Sea              bool        `json:"sea_trade"`
}

var economies = sync.OnceValue(func() [][]City {
	var data struct {
		Missions []struct {
			ID     int    `json:"id"`
			Cities []City `json:"cities"`
		} `json:"missions"`
	}
	result := make([][]City, len(definitions()))
	for _, source := range [][]byte{economyData, expansionEconomyData} {
		// Do not reuse decoded backing arrays: base city slices are retained
		// by the result while the expansion catalog is decoded.
		data.Missions = nil
		if err := json.Unmarshal(source, &data); err != nil {
			panic(err)
		}
		for _, m := range data.Missions {
			result[m.ID] = m.Cities
		}
	}
	return result
})

func Cities(id int) []City {
	if id < 0 || id >= len(economies()) {
		return nil
	}
	result := slices.Clone(economies()[id])
	for i := range result {
		limits := map[int]int{}
		for r, n := range result[i].AnnualLimits {
			limits[r] = n
		}
		result[i].AnnualLimits = limits
		result[i].Buys = slices.Clone(result[i].Buys)
		result[i].Sells = slices.Clone(result[i].Sells)
		result[i].PotentialBuys = slices.Clone(result[i].PotentialBuys)
		result[i].PotentialSells = slices.Clone(result[i].PotentialSells)
	}
	return result
}

func (m Mission) HasLocalResource(resource int) bool {
	for _, c := range economies()[m.ID] {
		if c.Type == 0 && slices.Contains(c.Sells, resource) {
			return true
		}
	}
	return false
}

func (m Mission) CanImportResource(resource int) bool {
	for _, c := range economies()[m.ID] {
		if (c.Type == 1 || c.Type == 3 || c.Type == 5) && slices.Contains(c.Sells, resource) {
			return true
		}
	}
	return false
}

// Direct menu flag joins are corroborated by original0x00403000, not by
// aligning the shorter list of English editor labels with 47 numeric slots.
var menuFlags = map[string]int{
	"waterlift": 3, "irrigationditch": 4, "smallstatue": 32, "mediumstatue": 32, "largestatue": 32,
	"wall": 33, "tower": 34, "gatehouse": 35,
	"watersupply": 18, "granary": 7, "bazaar": 8, "storageyard": 9, "workcamp": 6,
	"scribalschool": 16, "library": 17, "physician": 21, "apothecary": 20, "dentist": 19, "mortuary": 22,
	"ferry": 29, "bridge": 28, "shipbridge": 28, "goldmine": 2, "booth": 11, "jugglerschool": 11, "bandstand": 12, "conservatory": 12,
	"pavilion": 13, "danceschool": 13, "senethouse": 14, "festival_square": 15, "garden": 30, "plaza": 31,
	"taxoffice": 27, "courthouse": 23, "policestation": 24, "dock": 10, "shipwright": 5,
	"recruiter": 36, "fort": 37, "militaryacademy": 40, "weaponsmith": 41, "chariotsmith": 42,
	"archerfort": 38, "chariotfort": 39, "academy": 40, "chariotmaker": 42,
	"warshipwharf": 43, "transportwharf": 44, "palace": 25, "townpalace": 25, "citypalace": 25, "residence": 26, "familymansion": 26, "dynastymansion": 26,
}

var localProducers = map[string]int{
	"farm": 1, "cattleranch": 2, "lettucefarm": 3, "chickpeafarm": 4, "pomegranatefarm": 5, "figfarm": 6, "fishingwharf": 7, "huntinglodge": 8,
	"claypit": 11, "barleyfarm": 14, "flaxfarm": 16, "gemstonemine": 18, "woodcutter": 20, "reedgatherer": 22,
	"stonequarry": 24, "limestonequarry": 25, "granitequarry": 26, "coppermine": 29, "sandstonequarry": 30,
	"hennafarm": 32, "marblequarry": 35,
}

var workshops = map[string]int{"potter": 11, "brewery": 14, "weaver": 16, "jeweler": 18, "papyrusmaker": 22, "weaponsmith": 29, "chariotsmith": 20, "chariotmaker": 20}

func MonumentCode(kind string) int {
	switch kind {
	case "mastaba", "smallmastaba":
		return 18
	case "mediummastaba":
		return 19
	case "largemastaba":
		return 20
	case "sphinx":
		return 21
	case "smallobelisk":
		return 22
	case "largeobelisk":
		return 23
	case "suntemple":
		return 24
	case "smallmausoleum":
		return 25
	case "mediummausoleum":
		return 26
	case "largemausoleum":
		return 27
	case "pharos":
		return 28
	case "alexandrialibrary":
		return 29
	case "caesareum":
		return 30
	case "smallroyaltomb":
		return 33
	case "mediumroyaltomb":
		return 34
	case "largeroyaltomb":
		return 35
	case "grandroyaltomb":
		return 36
	case "abusimbel":
		return 37
	}
	for i, prefix := range []string{"small", "medium", "large", "complex", "grand"} {
		for family, base := range map[string]int{"bent": 1, "brick": 3, "stepped": 8, "pyramid": 13} {
			if prefix+family == kind && (family != "bent" || i < 2) {
				return base + i
			}
		}
	}
	return 0
}

func MonumentKind(code int) string {
	for _, kind := range []string{"smallmastaba", "mediummastaba", "largemastaba", "sphinx", "smallobelisk", "largeobelisk", "suntemple", "smallmausoleum", "mediummausoleum", "largemausoleum", "pharos", "alexandrialibrary", "caesareum", "smallroyaltomb", "mediumroyaltomb", "largeroyaltomb", "grandroyaltomb", "abusimbel"} {
		if MonumentCode(kind) == code {
			return kind
		}
	}
	for _, prefix := range []string{"small", "medium", "large", "complex", "grand"} {
		for _, family := range []string{"bent", "brick", "stepped", "pyramid"} {
			if code > 0 && MonumentCode(prefix+family) == code {
				return prefix + family
			}
		}
	}
	return ""
}

func (m Mission) BaseAllows(kind string) bool {
	if kind == "zoo" {
		// Gold0x00404ce1 reads0x00784d3a, allowance slot45 from
		// base0x00784ce0. The language label alone is not the join.
		if m.SourceVersion() >= 160 {
			return m.EditorFlags[45] != 0
		}
		return m.ID == 31 || m.ID == 37
	}
	if kind == "paintmaker" || kind == "lampmaker" {
		if m.SourceVersion() < 160 {
			return false
		}
		input := 32
		if kind == "lampmaker" {
			input = 31
			// Gold0x004045c8 requires oil plus pottery-production access;
			// its0x00443d90 helper maps pottery13 to clay11.
			if !m.HasLocalResource(11) && !m.CanImportResource(11) {
				return false
			}
		}
		return m.HasLocalResource(input) || m.CanImportResource(input)
	}
	if kind == "artisans" {
		return m.SourceVersion() >= 160 && slices.ContainsFunc(m.MonumentCodes[:], func(code int) bool { return code >= 33 && code <= 36 })
	}
	if code := MonumentCode(kind); code > 0 {
		return slices.Contains(m.MonumentCodes[:], code)
	}
	if kind == "bricklayers" || kind == "stonemasons" || kind == "carpenters" {
		for _, code := range m.MonumentCodes {
			if code >= 33 && code <= 36 {
				if kind == "stonemasons" {
					return true
				}
				continue
			}
			if kind == "bricklayers" && (code >= 3 && code <= 7 || code >= 18 && code <= 20) {
				return true
			}
			if kind == "stonemasons" && code > 0 && !(code >= 18 && code <= 20) {
				return true
			}
			if kind == "carpenters" && code > 0 && !(code >= 18 && code <= 20) {
				return true
			}
		}
		return false
	}
	switch kind {
	case "road", "hut", "clear", "firehouse", "architect", "roadblock":
		return true
	}
	// Original0x00403000 chooses government size by player rank and reads
	// independent per-deity complex flags at scenario offsets1240..1248.
	for _, pair := range [][2]string{{"palace", "residence"}, {"townpalace", "familymansion"}, {"citypalace", "dynastymansion"}} {
		if kind == pair[0] || kind == pair[1] {
			band := 0
			if m.Rank >= 8 {
				band = 2
			} else if m.Rank >= 6 {
				band = 1
			}
			want := [3][2]string{{"palace", "residence"}, {"townpalace", "familymansion"}, {"citypalace", "dynastymansion"}}[band]
			return (kind == want[0] || kind == want[1]) && m.EditorFlags[menuFlags[kind]] != 0
		}
	}
	if kind == "complexaltar" || kind == "complexoracle" {
		return slices.ContainsFunc(m.ComplexFlags[:], func(flag int) bool { return flag != 0 })
	}
	if strings.HasPrefix(kind, "complex_") {
		for i, god := range []string{"osiris", "ra", "ptah", "seth", "bast"} {
			if kind == "complex_"+god {
				return m.ComplexFlags[i] != 0
			}
		}
		return false
	}
	if strings.HasPrefix(kind, "temple_") || strings.HasPrefix(kind, "shrine_") {
		god := strings.TrimPrefix(strings.TrimPrefix(kind, "temple_"), "shrine_")
		for i, name := range []string{"osiris", "ra", "ptah", "seth", "bast"} {
			if god == name {
				return m.KnownGods[i] > 0
			}
		}
	}
	if resource, ok := localProducers[kind]; ok {
		if resource >= 31 && m.SourceVersion() < 160 {
			return false
		}
		if kind == "fishingwharf" && m.EditorFlags[5] == 0 {
			return false
		}
		return m.HasLocalResource(resource)
	}
	if resource, ok := workshops[kind]; ok {
		if index, hasFlag := menuFlags[kind]; hasFlag && m.EditorFlags[index] == 0 {
			return false
		}
		return m.HasLocalResource(resource) || m.CanImportResource(resource)
	}
	if kind == "brickworks" {
		return m.ID >= 7 && (m.HasLocalResource(11) || m.CanImportResource(11)) && (m.HasLocalResource(1) || m.HasLocalResource(9) || m.CanImportResource(9))
	}
	if index, ok := menuFlags[kind]; ok {
		return m.EditorFlags[index] != 0
	}
	return false
}

func buildingKinds() []string {
	result := []string{"zoo", "road", "hut", "clear", "firehouse", "architect", "roadblock", "brickworks", "smallmastaba", "mediummastaba", "largemastaba", "sphinx", "smallobelisk", "largeobelisk", "suntemple", "smallmausoleum", "mediummausoleum", "largemausoleum", "bricklayers", "stonemasons", "carpenters", "pharos", "alexandrialibrary", "caesareum", "smallroyaltomb", "mediumroyaltomb", "largeroyaltomb", "grandroyaltomb", "abusimbel", "artisans", "paintmaker", "lampmaker"}
	for _, prefix := range []string{"small", "medium", "large", "complex", "grand"} {
		for _, family := range []string{"bent", "brick", "stepped", "pyramid"} {
			result = append(result, prefix+family)
		}
	}
	for kind := range menuFlags {
		result = append(result, kind)
	}
	for kind := range localProducers {
		result = append(result, kind)
	}
	for kind := range workshops {
		result = append(result, kind)
	}
	for _, god := range []string{"osiris", "ra", "ptah", "seth", "bast"} {
		result = append(result, "temple_"+god, "shrine_"+god, "complex_"+god)
	}
	result = append(result, "complexaltar", "complexoracle")
	slices.Sort(result)
	return slices.Compact(result)
}

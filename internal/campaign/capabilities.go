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

type City struct {
	Name         string      `json:"name"`
	AnnualLimits map[int]int `json:"annual_limits"`
	ID           int         `json:"id"`
	Type         int         `json:"type_raw"`
	NameID       int         `json:"name_id_raw"`
	Route        int         `json:"route_id"`
	Open         bool        `json:"initially_open"`
	Buys         []int       `json:"buys"`
	Sells        []int       `json:"sells"`
	Cost         int         `json:"cost_to_open"`
	Sea          bool        `json:"sea_trade"`
}

var economies = sync.OnceValue(func() [][]City {
	var data struct {
		Missions []struct {
			ID     int    `json:"id"`
			Cities []City `json:"cities"`
		} `json:"missions"`
	}
	if err := json.Unmarshal(economyData, &data); err != nil {
		panic(err)
	}
	result := make([][]City, 38)
	for _, m := range data.Missions {
		result[m.ID] = m.Cities
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
	"watersupply": 18, "granary": 7, "bazaar": 8, "storageyard": 9, "workcamp": 6,
	"scribalschool": 16, "library": 17, "physician": 21, "apothecary": 20, "dentist": 22, "mortuary": 24,
	"ferry": 29, "bridge": 28, "shipbridge": 28, "goldmine": 2, "booth": 11, "jugglerschool": 11, "bandstand": 12, "conservatory": 12,
	"pavilion": 13, "danceschool": 13, "senethouse": 14, "festival_square": 15, "garden": 30, "plaza": 31,
	"taxoffice": 27, "courthouse": 23, "policestation": 19, "dock": 10, "shipwright": 5,
	"recruiter": 36, "fort": 37, "militaryacademy": 40, "weaponsmith": 41, "chariotsmith": 42,
	"warshipwharf": 43, "transportwharf": 44, "palace": 25, "residence": 26,
}

var localProducers = map[string]int{
	"farm": 1, "cattleranch": 2, "lettucefarm": 3, "chickpeafarm": 4, "pomegranatefarm": 5, "figfarm": 6, "fishingwharf": 7, "huntinglodge": 8,
	"claypit": 11, "barleyfarm": 14, "flaxfarm": 16, "gemstonemine": 18, "woodcutter": 20, "reedgatherer": 22,
	"stonequarry": 24, "limestonequarry": 25, "granitequarry": 26, "coppermine": 29, "sandstonequarry": 30,
}

var workshops = map[string]int{"potter": 11, "brewery": 14, "weaver": 16, "jeweler": 18, "papyrusmaker": 22, "weaponsmith": 29, "chariotsmith": 20}

func (m Mission) BaseAllows(kind string) bool {
	switch kind {
	case "road", "hut", "clear", "firehouse", "architect", "roadblock":
		return true
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
	result := []string{"road", "hut", "clear", "firehouse", "architect", "roadblock", "brickworks"}
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
		result = append(result, "temple_"+god, "shrine_"+god)
	}
	slices.Sort(result)
	return slices.Compact(result)
}

package sandbox

import (
	"errors"
	"github.com/sentania-labs/project-nile/internal/campaign"
	"slices"
)

var extracted = map[string]string{"stonequarry": "stone", "limestonequarry": "limestone", "granitequarry": "granite", "sandstonequarry": "sandstone", "gemstonemine": "gems"}

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"stonequarry", "Plain stone quarry", 2, 2, 75, "Industry", 12},
		CatalogEntry{"limestonequarry", "Limestone quarry", 2, 2, 75, "Industry", 12},
		CatalogEntry{"granitequarry", "Granite quarry", 2, 2, 100, "Industry", 12},
		CatalogEntry{"sandstonequarry", "Sandstone quarry", 2, 2, 50, "Industry", 12},
		CatalogEntry{"gemstonemine", "Gemstone mine", 2, 2, 400, "Industry", 8},
		CatalogEntry{"woodcutter", "Wood cutter", 2, 2, 50, "Industry", 8},
		CatalogEntry{"jeweler", "Jeweler", 2, 2, 90, "Industry", 12})
}

func (w World) extractionPlacement(b Building) error {
	if b.Kind == "woodcutter" && len(w.Woodland) == 0 {
		return errors.New("no woodland to harvest in this region")
	}
	if b.Kind == "gemstonemine" {
		if !w.mineTouchesOre(b) {
			return errors.New("place a gemstone mine beside mineral-bearing rock")
		}
		return nil
	}
	if extracted[b.Kind] != "" && !slices.ContainsFunc(w.QuarryRock, b.adjacent) {
		return errors.New("place a quarry beside exposed stone")
	}
	return nil
}

// Woodcutters stop beside a standing tree. A timber load travels home along
// the same walkable route. Tree regrowth timing awaits original-game evidence.
func (w World) gatheringSite(kind string) (role, resource string, tiles []Point) {
	switch kind {
	case "reedgatherer":
		return "reedharvester", "reeds", w.ReedBeds
	case "woodcutter":
		for _, tree := range w.Woodland {
			for _, d := range directions {
				tiles = append(tiles, Point{tree.X + d.X, tree.Y + d.Y})
			}
		}
		return "lumberjack", "timber", tiles
	}
	return "", "", nil
}

// Scenario files determine which raw materials can be produced. These terrain
// patches are independently authored on the extended portion of the city map.
func (w *World) seedCampaignMaterials(id int) {
	m, ok := campaign.Definition(id)
	if !ok || w.mapSize() <= Size {
		return
	}
	occupied := w.occupied()
	add := func(dest *[]Point, base Point) {
		for _, o := range []Point{{0, 0}, {1, 0}, {2, 1}, {0, 2}, {1, 2}} {
			p := Point{base.X + o.X, base.Y + o.Y}
			if w.buildable(p) && !occupied[p] && !slices.Contains(w.HuntingGrounds, p) {
				*dest = append(*dest, p)
				occupied[p] = true
			}
		}
	}
	for y := 39; y < w.mapSize()-4; y += 27 {
		x := w.bank(y) + 12
		if m.HasLocalResource(20) {
			add(&w.Woodland, Point{x, y})
		}
		if m.HasLocalResource(24) || m.HasLocalResource(25) || m.HasLocalResource(26) || m.HasLocalResource(30) {
			add(&w.QuarryRock, Point{x + 9, y + 2})
		}
	}
}

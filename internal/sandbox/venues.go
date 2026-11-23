package sandbox

import (
	"errors"
	"slices"
)

var schools = []string{"jugglerschool", "conservatory", "danceschool"}
var venues = []string{"booth", "bandstand", "pavilion"}

func init() {
	buildingCatalog = append(buildingCatalog,
		CatalogEntry{"jugglerschool", "Juggler school", 2, 2, 50, "Entertainment", 5},
		CatalogEntry{"conservatory", "Conservatory", 3, 3, 100, "Entertainment", 8},
		CatalogEntry{"danceschool", "Dance school", 4, 4, 150, "Entertainment", 10},
		CatalogEntry{"booth", "Booth", 2, 2, 50, "Entertainment", 8},
		CatalogEntry{"bandstand", "Bandstand", 3, 3, 150, "Entertainment", 12},
		CatalogEntry{"pavilion", "Pavilion", 4, 4, 500, "Entertainment", 20})
}

func (w World) venuePlacement(b Building) error {
	if !slices.Contains(venues, b.Kind) {
		return nil
	}
	roads := w.roadSet()
	for _, p := range b.tiles() {
		if !roads[p] {
			continue
		}
		horizontal := roads[Point{p.X + 1, p.Y}] || roads[Point{p.X - 1, p.Y}]
		vertical := roads[Point{p.X, p.Y + 1}] || roads[Point{p.X, p.Y - 1}]
		if horizontal && vertical {
			return nil
		}
	}
	return errors.New("place the venue over a road bend or intersection")
}

func (w *World) dispatchPerformers() {
	for _, school := range w.Buildings {
		style := slices.Index(schools, school.Kind)
		if style < 0 || school.Workers < school.WorkersNeeded || !school.Connected || school.Damaged != "" || w.active(school.ID, "performer") {
			continue
		}
		var next *Building
		var route []Point
		for i := range w.Buildings {
			v := &w.Buildings[i]
			if slices.Index(venues, v.Kind) < style || !v.Connected || v.Damaged != "" || v.Workers < v.WorkersNeeded || v.ShowUntil[style] > w.Tick+MarketDuration/2 {
				continue
			}
			if next != nil && next.ShowUntil[style] < v.ShowUntil[style] {
				continue
			}
			path := w.path(w.perimeter(school), *v)
			if len(path) > 0 && (next == nil || next.ShowUntil[style] > v.ShowUntil[style] || len(path) < len(route)) {
				next, route = v, path
			}
		}
		if next != nil {
			w.spawn("performer", "travel", school.ID, next.ID, 0, route, 0, 0)
		}
	}
}

func (w World) validatePerformer(a Walker, target Building) error {
	source := w.building(a.Origin)
	if source == nil || a.Cargo != 0 || a.Reserved != 0 || a.Source != 0 || a.Resource != "" || a.Stage != "travel" || !source.roadAccess(a.Route[0]) {
		return errors.New("invalid performer journey")
	}
	style := slices.Index(schools, source.Kind)
	if style < 0 || slices.Index(venues, target.Kind) < style {
		return errors.New("invalid performance venue")
	}
	return nil
}

func entertainmentScore(b Building, tick int64) int {
	score := 0
	if b.ServiceUntil[7] > tick {
		score += int(min(MarketDuration, b.ServiceUntil[7]-tick) * 96 * 10 / MarketDuration / 25)
	}
	for i, divider := range []int64{50, 40, 30} {
		remaining := min(MarketDuration, max(int64(0), b.EntertainmentUntil[i]-tick))
		score += int(remaining * 96 * 10 / MarketDuration / divider)
	}
	return score
}

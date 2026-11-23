package sandbox

import (
	"errors"
	"slices"
)

type Festival struct {
	God  string `json:"god"`
	Size int    `json:"size"`
	Due  int64  `json:"due"`
}

func init() {
	buildingCatalog = append(buildingCatalog, CatalogEntry{"festival_square", "Festival square", 5, 5, 500, "Religion", 0})
	influences["festival_square"] = influence{16, 2, -3, 5}
}

func (w World) festivalSite() *Building {
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind == "festival_square" && b.Damaged == "" && b.Connected {
			return b
		}
	}
	return nil
}

// Costs and preparation times use the secondary reconstruction's festival
// calculation. Treasury is debited when the player commissions the festival.
func (w *World) PlanFestival(god string, size int) error {
	i := slices.Index(gods, god)
	if i < 0 || size < 1 || size > 3 {
		return errors.New("choose a god and festival size")
	}
	if !w.Campaign.Allows(temples[i]) {
		return errors.New("this city does not worship that god")
	}
	if w.Festival != nil {
		return errors.New("a festival is already being prepared")
	}
	if w.festivalSite() == nil {
		return errors.New("build a connected festival square")
	}
	pop := w.population()
	cost := int64(pop/([]int{20, 10, 5}[size-1]) + ([]int{10, 20, 40}[size-1]))
	months := size + 1 + pop/([]int{1000, 1500, 2000}[size-1])
	due := w.Tick + YearTicks/12*int64(months)
	if due > MaxTick {
		return errors.New("festival exceeds the time limit")
	}
	if w.Treasury < cost {
		return errors.New("not enough debens for this festival")
	}
	if size == 3 {
		needed := pop/50 + 1
		total := 0
		for _, b := range w.Buildings {
			if b.Kind == "storageyard" && b.Connected && b.Damaged == "" {
				total += max(0, inventory(b, "beer")-w.reservedResource(b.ID, "beer"))
			}
		}
		if total < needed {
			return errors.New("a lavish festival requires beer in connected storage yards")
		}
		for _, b := range w.Buildings {
			if b.Kind == "storageyard" && b.Connected && b.Damaged == "" && needed > 0 {
				needed -= w.ConsumeResource(b.ID, "beer", needed)
			}
		}
	}
	w.spend(cost)
	w.Festival = &Festival{God: god, Size: size, Due: due}
	w.refresh()
	return nil
}

func (w *World) festivalTick() {
	f := w.Festival
	if f == nil || w.Tick < f.Due || w.festivalSite() == nil {
		return
	}
	i := slices.Index(gods, f.God)
	w.Favor[i] = min(100, w.Favor[i]+5*f.Size)
	w.LastFestival[i] = w.Tick
	w.FestivalsHeld++
	w.Festival = nil
}

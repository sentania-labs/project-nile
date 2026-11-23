package sandbox

import (
	"errors"
	"slices"
)

type Invasion struct {
	Culture  int    `json:"culture,omitempty"`
	Sea      bool   `json:"sea,omitempty"`
	EventID  int    `json:"event_id"`
	Class    string `json:"class"`
	Entry    Point  `json:"entry"`
	Amount   int    `json:"amount"`
	Defeated int    `json:"defeated"`
	Breaches int    `json:"breaches"`
	Status   string `json:"status"`
	Started  int64  `json:"started"`
}

func (w *World) invasion(id int) *Invasion {
	for i := range w.Invasions {
		if w.Invasions[i].EventID == id {
			return &w.Invasions[i]
		}
	}
	return nil
}

// Campaign code owns recorded dates, culture IDs and force-size interpretation.
// Amount here is a literal number of physical soldiers, not a historical troop scale.
func (w *World) SpawnInvasion(eventID int, class string, amount int, entry Point, cultures ...int) error {
	culture, valid := invasionCulture(class, cultures)
	if !valid || eventID < 1 || w.invasion(eventID) != nil || amount < 1 || amount > 200 || w.NextID > MaxTick-int64(amount) || !slices.Contains([]string{"foreign", "egyptian", "kingdom", "bedouin", "archer", "charioteer", "test"}, class) {
		return errors.New("invalid or duplicate invasion")
	}
	if !w.buildable(entry) || (entry.X != 0 && entry.X != w.mapSize()-1 && entry.Y != 0 && entry.Y != w.mapSize()-1) {
		return errors.New("invaders must enter on a dry map edge")
	}
	var target *Building
	var route []Point
	for i := range w.Buildings {
		b := &w.Buildings[i]
		if b.Kind == "hut" && b.Pop > 0 {
			path := w.invasionPath(entry, w.accessTiles(*b))
			if len(path) > 0 {
				target = b
				route = path
				break
			}
		}
	}
	if target == nil {
		return errors.New("no land route reaches an inhabited settlement")
	}
	for _, kind := range invasionUnits(class, culture, amount) {
		a := Troop{ID: w.NextID, InvasionID: eventID, Kind: kind, Origin: target.ID, X: entry.X, Y: entry.Y, Enemy: true, HP: 3, Stage: "raid", Route: slices.Clone(route), Next: 1}
		w.Troops = append(w.Troops, a)
		w.NextID++
	}
	w.Invasions = append(w.Invasions, Invasion{Culture: culture, EventID: eventID, Class: class, Entry: entry, Amount: amount, Status: "active", Started: w.Tick})
	w.Military.Invaders += int64(amount)
	w.Military.InvasionActive = true
	return nil
}
func (w World) invasionPath(entry Point, goals []Point) []Point {
	return w.siegePath(entry, goals)
}
func (w *World) recordInvader(a Troop, breach bool) {
	if record := w.invasion(a.InvasionID); record != nil {
		if breach {
			record.Breaches++
		} else {
			record.Defeated++
		}
	}
}
func (w *World) syncInvasions() {
	for i := range w.Invasions {
		d := &w.Invasions[i]
		if d.Defeated+d.Breaches == d.Amount && w.hostileWarships(d.EventID) == 0 {
			d.Status = "repelled"
			if d.Breaches > 0 {
				d.Status = "breached"
			}
		}
	}
}
func (w World) validateInvasions() error {
	seen := map[int]bool{}
	for _, d := range w.Invasions {
		maxAmount := 200
		if d.Sea {
			maxAmount = 65535
		}
		if d.Culture < 0 || d.Culture >= len(enemyShares) || d.EventID < 1 || seen[d.EventID] || d.Amount < 0 || d.Amount == 0 && !d.Sea || d.Amount > maxAmount || d.Defeated < 0 || d.Breaches < 0 || d.Started < 0 || d.Started > w.Tick || !slices.Contains([]string{"foreign", "egyptian", "kingdom", "bedouin", "archer", "charioteer", "test"}, d.Class) {
			return errors.New("invalid invasion ledger")
		}
		seen[d.EventID] = true
		alive := w.passengersAtSea(d.EventID)
		for _, a := range w.Troops {
			if a.Enemy && a.InvasionID == d.EventID {
				alive++
			}
		}
		if alive+d.Defeated+d.Breaches != d.Amount {
			return errors.New("invasion casualty conservation failed")
		}
		status := "active"
		if alive == 0 && w.hostileWarships(d.EventID) == 0 {
			status = "repelled"
			if d.Breaches > 0 {
				status = "breached"
			}
		}
		if d.Status != status {
			return errors.New("invasion status disagrees with its soldiers")
		}
	}
	for _, a := range w.Troops {
		if a.InvasionID != 0 && (!a.Enemy || !seen[a.InvasionID]) {
			return errors.New("soldier has an unknown invasion")
		}
	}
	return nil
}

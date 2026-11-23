package sandbox

// Original executable 0x44b6c0 sends parties of up to four to vacant homes and
// homes with ample room; 0x44ba70 stores their people count and a per-home slot.
// One four-person batch per prototype migration interval is provisional timing.
const MigrantBatch = 4

func migrantPeople(a Walker) int {
	if a.People == 0 {
		return 1 // In-flight walkers from earlier snapshot versions carried one.
	}
	return a.People
}

func (w *World) dispatchMigrants() {
	if w.BudgetEnabled && w.WageRate == 0 || w.Tick%w.migrationInterval() != 0 {
		return
	}
	reserved := map[int64]bool{}
	for _, a := range w.Walkers {
		if a.Role == "immigrant" {
			reserved[a.Target] = true
		}
	}
	remaining := MigrantBatch
	// Original0x0044b6c0 prioritizes vacant lots, then occupied homes with
	// more than seven vacancies, then the remaining small vacancies. Without
	// the middle pass, a fluctuating early neighborhood starves later homes.
	for pass := 0; pass < 3 && remaining > 0; pass++ {
		for _, home := range w.Buildings {
			if remaining == 0 {
				break
			}
			if home.Kind != "hut" || !home.Connected || home.Damaged != "" || reserved[home.ID] || home.Pop >= home.Capacity || (pass == 0 && home.Pop != 0) || (pass == 1 && (home.Pop == 0 || home.Capacity-home.Pop <= 7)) {
				continue
			}
			people := min(remaining, home.Capacity-home.Pop)
			if w.spawn("immigrant", "travel", 0, home.ID, 0, w.pathWithFerries([]Point{w.entry()}, home, true), 0, 0) {
				w.Walkers[len(w.Walkers)-1].People = people
				reserved[home.ID] = true
				remaining -= people
			}
		}
	}
}

func (w *World) arriveMigrants(a Walker, home *Building) {
	people := migrantPeople(a)
	accepted := min(people, max(0, home.Capacity-home.Pop))
	w.Arrivals += int64(people)
	home.Pop += accepted
	w.Departures += int64(people - accepted)
}

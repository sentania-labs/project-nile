package main

import (
	"testing"
	"time"
)

func TestUndoRoadRestoresUntouchedCityAtCurrentTick(t *testing.T) {
	const fresh = `{"id":1,"op":"new_town","scenario":"empty"}`
	const build = `{"id":2,"op":"road","tiles":[{"x":11,"y":4},{"x":12,"y":4}]}`
	const failed = `{"id":3,"op":"build","kind":"road","x":0,"y":0}`
	const advance = `{"id":4,"op":"step","ticks":7}`
	const undo = `{"id":5,"op":"undo"}`
	run := func(actions ...string) []sandboxResponse {
		return runTimedSandbox(t, actions, make([]time.Duration, len(actions)))
	}
	edited := run(fresh, build, failed, advance, undo, undo)
	control := run(fresh, advance)
	if !edited[1].CanUndo || edited[2].OK || !edited[2].CanUndo || !edited[3].CanUndo || !edited[4].OK || edited[4].CanUndo || edited[5].OK || edited[4].Hash != control[1].Hash {
		t.Fatal("undo did not restore independent simulation", edited[4].Error)
	}
	for _, change := range []string{`{"id":5,"op":"step","ticks":17}`, `{"id":5,"op":"set_policy","tax_rate":2,"wage_rate":1,"labor_priority":"balanced"}`, fresh} {
		r := run(fresh, build, change, undo)
		if !r[2].OK || r[2].CanUndo || r[3].OK || r[2].Hash != r[3].Hash {
			t.Fatal("undo survived later action", change)
		}
	}
}

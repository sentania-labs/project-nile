package main

import (
	"encoding/json"
	"errors"

	"github.com/sentania-labs/project-nile/internal/sandbox"
)

// Retail 0x4758e0 and 0x4762d0 maintain one ready edit and a finite
// countdown. Nile measures its window in simulation ticks independently of FPS.
const undoTicks = 16

type sandboxUndo struct {
	tick   int64
	before []byte
}

func constructionEdit(op string) bool {
	switch op {
	case "build", "zone", "road", "canal":
		return true
	}
	return false
}

func preservesUndo(op string) bool {
	switch op {
	case "state", "step", "pause", "resume", "set_speed", "set_pace", "autosave", "save", "save_as":
		return true
	}
	return false
}

func (u sandboxUndo) available(w sandbox.World) bool {
	return len(u.before) > 0 && w.Tick >= u.tick && w.Tick-u.tick <= undoTicks
}

func (u sandboxUndo) restore(w *sandbox.World) error {
	if !u.available(*w) {
		return errors.New("no recent construction to undo")
	}
	var restored sandbox.World
	if err := json.Unmarshal(u.before, &restored); err != nil {
		return err
	}
	// Keep the elapsed clock and rerun all unrelated simulation activity from
	// its saved random state. No intervening player action may be discarded.
	if elapsed := int(w.Tick - u.tick); elapsed > 0 {
		if err := restored.Step(elapsed); err != nil {
			return err
		}
	}
	if err := restored.Validate(); err != nil {
		return err
	}
	*w = restored
	return nil
}

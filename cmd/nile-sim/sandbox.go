package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/sentania-labs/project-nile/internal/sandbox"
)

type sandboxRequest struct {
	TaxRate       *int   `json:"tax_rate,omitempty"`
	WageRate      *int   `json:"wage_rate,omitempty"`
	LaborPriority string `json:"labor_priority,omitempty"`
	ID            int64  `json:"id"`
	Op            string `json:"op"`
	Kind          string `json:"kind,omitempty"`
	X             int    `json:"x,omitempty"`
	Y             int    `json:"y,omitempty"`
	X2            int    `json:"x2,omitempty"`
	Y2            int    `json:"y2,omitempty"`
	Ticks         *int   `json:"ticks,omitempty"`
	Speed         int    `json:"speed,omitempty"`
	Scenario      string `json:"scenario,omitempty"`
}

type sandboxResponse struct {
	ID      int64        `json:"id"`
	OK      bool         `json:"ok"`
	Error   string       `json:"error,omitempty"`
	State   sandbox.View `json:"state"`
	Hash    string       `json:"hash"`
	Running bool         `json:"running"`
	Speed   int          `json:"speed"`
}

const sandboxTickPeriod = 200 * time.Millisecond

// This clock belongs to the simulation process. Polling/render frequency only
// controls when accumulated complete ticks are materialized, never their count.
// The monotonic component of time.Now handles wall-clock adjustments.
type sandboxClock struct {
	running   bool
	anchor    time.Time
	remainder time.Duration
	speed     int
}

func (c *sandboxClock) catchUp(world *sandbox.World, now time.Time) error {
	if !c.running {
		return nil
	}
	elapsed := now.Sub(c.anchor)
	if elapsed < 0 {
		return errors.New("simulation monotonic clock moved backwards")
	}
	// Divide first so a very long elapsed duration cannot overflow when the
	// retained fraction is added. No complete ticks are dropped or clamped.
	speed := max(1, c.speed)
	ticks := int64(elapsed/sandboxTickPeriod) * int64(speed)
	fraction := (elapsed%sandboxTickPeriod)*time.Duration(speed) + c.remainder
	ticks += int64(fraction / sandboxTickPeriod)
	c.remainder = fraction % sandboxTickPeriod
	c.anchor = now
	for ticks > 0 {
		batch := int64(1000)
		if ticks < batch {
			batch = ticks
		}
		if err := world.Step(int(batch)); err != nil {
			c.running = false
			return err
		}
		ticks -= batch
		if ticks > 0 {
			runtime.Gosched()
		}
	}
	return nil
}

func serveSandbox(in io.Reader, out io.Writer, store *sandbox.Store) error {
	return serveSandboxWithClock(in, out, store, time.Now)
}

func serveSandboxWithClock(in io.Reader, out io.Writer, store *sandbox.Store, now func() time.Time) error {
	world := sandbox.Initial()
	clock := sandboxClock{speed: 1} // Manual steps and tests start paused.
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1<<20)
	enc := json.NewEncoder(out)
	for scan.Scan() {
		received := now()
		clockErr := clock.catchUp(&world, received)
		var req sandboxRequest
		dec := json.NewDecoder(bytes.NewReader(scan.Bytes()))
		dec.DisallowUnknownFields()
		err := dec.Decode(&req)
		if err == nil && dec.Decode(new(any)) != io.EOF {
			err = errors.New("one JSON request required per line")
		}
		if err == nil && (req.ID < 0 || req.ID > sandbox.MaxTick) {
			err = errors.New("id outside exact nonnegative counter range")
		}
		if err == nil {
			err = clockErr
		}
		quit := false
		if err == nil {
			switch req.Op {
			case "state":
			case "resume":
				if !clock.running {
					clock.running = true
					clock.anchor = received
				}
			case "pause":
				clock.running = false
			case "set_speed":
				if req.Speed != 1 && req.Speed != 2 && req.Speed != 4 {
					err = errors.New("speed must be 1, 2, or 4")
				} else {
					clock.speed = req.Speed
				}
			case "new_town":
				if req.Scenario != "" && req.Scenario != "empty" && req.Scenario != "starter" {
					err = errors.New("scenario must be empty or starter")
				} else {
					world = sandbox.NewTown()
					if req.Scenario == "starter" {
						world = sandbox.Initial()
					}
					clock.running = false
					clock.anchor = received
					clock.remainder = 0
				}
			case "set_policy":
				if req.TaxRate == nil || req.WageRate == nil {
					err = errors.New("tax_rate and wage_rate are required")
				} else {
					err = world.SetPolicy(*req.TaxRate, *req.WageRate, req.LaborPriority)
				}
			case "repair":
				err = world.Repair(req.X, req.Y)
			case "zone":
				err = world.ZoneHousing(sandbox.Point{X: req.X, Y: req.Y}, sandbox.Point{X: req.X2, Y: req.Y2})
			case "build":
				err = world.Build(req.Kind, req.X, req.Y)
			case "step":
				if clock.running {
					err = errors.New("pause the simulation before an explicit step")
					break
				}
				ticks := 1
				if req.Ticks != nil {
					ticks = *req.Ticks
				}
				err = world.Step(ticks)
			case "save":
				err = store.Save(world)
			case "load":
				var restored sandbox.World
				restored, err = store.Load()
				if err == nil {
					world = restored
					clock.anchor = received
					clock.remainder = 0
				}
			case "quit":
				quit = true
			default:
				err = errors.New("unknown operation")
			}
		}
		resp := sandboxResponse{ID: req.ID, OK: err == nil, State: world.View(), Hash: world.Hash(), Running: clock.running, Speed: clock.speed}
		if err != nil {
			resp.Error = err.Error()
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
		if quit {
			return nil
		}
	}
	return scan.Err()
}

func runSandbox(path string) error {
	store, err := sandbox.Open(path)
	if err != nil {
		return err
	}
	defer store.Close()
	return serveSandbox(os.Stdin, os.Stdout, store)
}

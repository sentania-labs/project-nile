package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sentania-labs/project-nile/internal/sandbox"
)

type sandboxRequest struct {
	Rank          *int            `json:"rank,omitempty"`
	Amount        int64           `json:"amount,omitempty"`
	RequestID     *int            `json:"request_id,omitempty"`
	BoatID        int64           `json:"boat_id,omitempty"`
	Path          string          `json:"path,omitempty"`
	FortID        int64           `json:"fort_id,omitempty"`
	FortIDs       []int64         `json:"fort_ids,omitempty"`
	Naval         bool            `json:"naval,omitempty"`
	God           string          `json:"god,omitempty"`
	FestivalSize  int             `json:"festival_size,omitempty"`
	Route         string          `json:"route,omitempty"`
	Mode          string          `json:"mode,omitempty"`
	Limit         int             `json:"limit,omitempty"`
	Role          string          `json:"role,omitempty"`
	Allow         *bool           `json:"allow,omitempty"`
	Resource      string          `json:"resource,omitempty"`
	Order         string          `json:"order,omitempty"`
	Tiles         []sandbox.Point `json:"tiles,omitempty"`
	TaxRate       *int            `json:"tax_rate,omitempty"`
	WageRate      *int            `json:"wage_rate,omitempty"`
	LaborPriority string          `json:"labor_priority,omitempty"`
	ID            int64           `json:"id"`
	Op            string          `json:"op"`
	Kind          string          `json:"kind,omitempty"`
	X             int             `json:"x,omitempty"`
	Y             int             `json:"y,omitempty"`
	X2            int             `json:"x2,omitempty"`
	Y2            int             `json:"y2,omitempty"`
	Ticks         *int            `json:"ticks,omitempty"`
	Facing        int             `json:"facing,omitempty"`
	Pace          string          `json:"pace,omitempty"`
	Speed         int             `json:"speed,omitempty"`
	Scenario      string          `json:"scenario,omitempty"`
	MissionID     *int            `json:"mission_id,omitempty"`
}

type sandboxResponse struct {
	SavePath   string       `json:"save_path,omitempty"`
	ID         int64        `json:"id"`
	OK         bool         `json:"ok"`
	Error      string       `json:"error,omitempty"`
	State      sandbox.View `json:"state"`
	Hash       string       `json:"hash"`
	Running    bool         `json:"running"`
	TickMillis int64        `json:"tick_ms"`
	Speed      int          `json:"speed"`
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
	period    time.Duration
}

func (c sandboxClock) tickPeriod() time.Duration {
	if c.period == 0 {
		return sandboxTickPeriod
	}
	return c.period
}
func (c *sandboxClock) setPace(pace string) error {
	var next time.Duration
	switch pace {
	case "classic80":
		next = 1520 * time.Millisecond
	case "classic50":
		next = 4580 * time.Millisecond
	case "playtest":
		next = sandboxTickPeriod
	case "classic100":
		next = 86 * time.Millisecond
	default:
		return errors.New("unknown calendar pace")
	}
	c.remainder = c.remainder * time.Duration(next.Milliseconds()) / time.Duration(c.tickPeriod().Milliseconds())
	c.period = next
	return nil
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
	period := c.tickPeriod()
	speed := max(1, c.speed)
	ticks := int64(elapsed/period) * int64(speed)
	fraction := (elapsed%period)*time.Duration(speed) + c.remainder
	ticks += int64(fraction / period)
	c.remainder = fraction % period
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
	original := store
	defer func() {
		if store != original {
			store.Close()
		}
	}()
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
			case "set_pace":
				err = clock.setPace(req.Pace)
			case "set_speed":
				if req.Speed != 1 && req.Speed != 2 && req.Speed != 4 {
					err = errors.New("speed must be 1, 2, or 4")
				} else {
					clock.speed = req.Speed
				}
			case "new_town":
				if req.Scenario != "" && req.Scenario != "empty" && req.Scenario != "starter" && req.Scenario != "valley" {
					err = errors.New("scenario must be empty, valley or starter")
				} else {
					if req.Path != "" {
						if !strings.EqualFold(filepath.Ext(req.Path), ".nile") {
							err = errors.New("select a .nile save file")
							break
						}
						slot, e := sandbox.OpenNew(req.Path)
						if e != nil {
							err = e
							break
						}
						store.Close()
						store = slot
					}
					world = sandbox.NewTown()
					if req.Scenario == "valley" {
						world = sandbox.NewValley()
					}
					if req.Scenario == "starter" {
						world = sandbox.Initial()
					}
					clock.running = false
					clock.anchor = received
					clock.remainder = 0
				}
			case "new_campaign", "advance_campaign":
				if req.MissionID == nil {
					err = errors.New("mission_id is required")
				} else {
					var next sandbox.World
					if req.Op == "advance_campaign" {
						next, err = world.AdvanceCampaign(*req.MissionID)
					} else {
						next, err = sandbox.NewCampaign(*req.MissionID)
					}
					if err == nil {
						world = next
						clock.running = false
						clock.anchor = received
						clock.remainder = 0
					}
				}
			case "dispatch_burial":
				err = world.DispatchBurial(req.Resource, req.Limit)
			case "dispatch_army":
				if req.RequestID == nil {
					err = errors.New("request_id is required")
				} else {
					err = world.DispatchCampaignArmy(*req.RequestID, req.FortIDs, req.Naval)
				}
			case "dispatch_request", "refuse_request":
				if req.RequestID == nil {
					err = errors.New("request_id is required")
				} else if req.Op == "dispatch_request" {
					err = world.DispatchCampaignRequest(*req.RequestID)
				} else {
					err = world.RefuseCampaignRequest(*req.RequestID)
				}
			case "set_salary":
				if req.Rank == nil {
					err = errors.New("rank is required")
				} else {
					err = world.SetSalary(*req.Rank)
				}
			case "royal_gift":
				if req.Rank == nil {
					err = errors.New("gift rank is required")
				} else {
					err = world.SendRoyalGift(*req.Rank)
				}
			case "donate":
				err = world.DonateSavings(req.Amount)
			case "set_policy":
				if req.TaxRate == nil || req.WageRate == nil {
					err = errors.New("tax_rate and wage_rate are required")
				} else {
					err = world.SetPolicy(*req.TaxRate, *req.WageRate, req.LaborPriority)
				}
			case "festival":
				err = world.PlanFestival(req.God, req.FestivalSize)
			case "road_permission":
				if req.Allow == nil {
					err = errors.New("allow is required")
				} else {
					err = world.RoadPermission(sandbox.Point{X: req.X, Y: req.Y}, req.Role, *req.Allow)
				}
			case "naval_order":
				err = world.NavalOrder(req.BoatID, req.Order, sandbox.Point{X: req.X, Y: req.Y})
			case "transport_company":
				err = world.TransportCompany(req.BoatID, req.FortID, sandbox.Point{X: req.X, Y: req.Y})
			case "military_order":
				err = world.MilitaryOrder(req.FortID, req.Order, sandbox.Point{X: req.X, Y: req.Y})
			case "test_sea_invasion":
				err = world.TestSeaInvasion()
			case "test_invasion":
				err = world.TestInvasion()
			case "open_trade":
				err = world.OpenTrade(req.Route)
			case "trade_policy":
				err = world.SetTradePolicy(req.Resource, req.Mode, req.Limit)
			case "storage_order":
				err = world.SetStorageOrder(req.X, req.Y, req.Resource, req.Order)
			case "repair":
				err = world.Repair(req.X, req.Y)
			case "zone":
				err = world.ZoneHousing(sandbox.Point{X: req.X, Y: req.Y}, sandbox.Point{X: req.X2, Y: req.Y2})
			case "canal":
				err = world.BuildCanals(req.Tiles)
			case "road":
				err = world.BuildRoads(req.Tiles)
			case "build":
				err = world.BuildFacing(req.Kind, req.X, req.Y, req.Facing)
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
			case "save_as", "load_from":
				var target *sandbox.Store
				path, pathErr := filepath.Abs(req.Path)
				if pathErr != nil || req.Path == "" || !strings.EqualFold(filepath.Ext(path), ".nile") {
					err = errors.New("select a .nile save file")
					break
				}
				if req.Op == "load_from" {
					if _, e := os.Stat(path); e != nil {
						err = e
						break
					}
				}
				target = store
				if !store.Matches(path) {
					target, err = sandbox.Open(path)
					if err != nil {
						break
					}
				}
				var restored sandbox.World
				if req.Op == "save_as" {
					err = target.Save(world)
				} else {
					restored, err = target.Load()
				}
				if err != nil {
					if target != store {
						target.Close()
					}
					break
				}
				if target != store {
					store.Close()
					store = target
				}
				if req.Op == "load_from" {
					world = restored
					clock.running = false
					clock.remainder = 0
					clock.anchor = received
				}
			case "autosave":
				err = store.Autosave(world)
			case "save":
				err = store.Save(world)
			case "load", "load_manual", "load_previous":
				var restored sandbox.World
				switch req.Op {
				case "load_manual":
					restored, err = store.LastManual()
				case "load_previous":
					restored, err = store.PreviousManual()
				default:
					restored, err = store.Load()
				}
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
		resp := sandboxResponse{TickMillis: clock.tickPeriod().Milliseconds(), SavePath: store.Path, ID: req.ID, OK: err == nil, State: world.View(), Hash: world.Hash(), Running: clock.running, Speed: clock.speed}
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

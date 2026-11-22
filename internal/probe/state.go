// Package probe contains a diagnostic fixture, not Pharaoh gameplay rules.
package probe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

// MaxTick keeps counters exact in JSON consumers using IEEE-754 doubles.
const MaxTick int64 = 1<<53 - 1

type State struct {
	Tick   int64  `json:"tick"`
	Seed   uint32 `json:"seed"`
	RNG    uint32 `json:"rng"`
	Walker uint32 `json:"walker"`
	Grain  int64  `json:"grain"`
}

func Initial() State { return State{Seed: 17, RNG: 17, Grain: 24} }

func (s State) Validate() error {
	if s.Tick < 0 || s.Tick > MaxTick || s.Seed != 17 || s.Walker != uint32(s.Tick%32) || s.Grain != 24 {
		return errors.New("invalid diagnostic state")
	}
	return nil
}

func (s *State) Step(ticks int) error {
	if ticks < 1 || ticks > 10000 || s.Tick > MaxTick-int64(ticks) {
		return errors.New("ticks must be 1..10000 within the exact counter range")
	}
	for range ticks {
		s.Tick++
		s.Walker = uint32(s.Tick % 32)
		// Defined unsigned overflow makes this fixture portable across targets.
		s.RNG = 1664525*s.RNG + 1013904223
	}
	return nil
}

func (s State) Hash() string {
	b, _ := json.Marshal(s)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

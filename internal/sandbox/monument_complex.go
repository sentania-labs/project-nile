package sandbox

import (
	"errors"
	"slices"
)

// Complexes have a real composite footprint, independently modelled from the
// original placement0x004d1c20:2x2 mortuary temple, at least one2x2 causeway,
// and a2x2 Valley Temple straddling an east-facing bank. Nothing teleports
// across water; workers still need the site's separate staging road.
type MonumentPart struct {
	Kind   string `json:"kind"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (p MonumentPart) contains(at Point) bool {
	return at.X >= p.X && at.X < p.X+p.Width && at.Y >= p.Y && at.Y < p.Y+p.Height
}
func (w World) complexParts(b Building) ([]MonumentPart, error) {
	if !monumentPlans[b.Kind].Complex {
		return nil, nil
	}
	y := b.Y + b.Height/2
	x := b.X + b.Width
	parts := []MonumentPart{{"mortuary_temple", x, y, 2, 2}}
	x += 2
	for count := 0; count <= 21; count++ {
		west := w.buildable(Point{x, y}) && w.buildable(Point{x, y + 1})
		east := w.navigableWater(Point{x + 1, y}) && w.navigableWater(Point{x + 1, y + 1})
		if west && east {
			if count == 0 {
				return nil, errors.New("leave room for a causeway between the pyramid and the Nile")
			}
			return append(parts, MonumentPart{"valley_temple", x, y, 2, 2}), nil
		}
		if !west || !w.buildable(Point{x + 1, y}) || !w.buildable(Point{x + 1, y + 1}) {
			break
		}
		parts = append(parts, MonumentPart{"causeway", x, y, 2, 2})
		x += 2
	}
	return nil, errors.New("place the complex west of the Nile with a clear east-facing bank for its Valley Temple")
}
func (w World) prepareMonumentPlacement(b *Building) error {
	if b.Kind == "sphinx" {
		for _, old := range w.Buildings {
			if old.Kind == "sphinx" {
				return errors.New("this city already has a Sphinx")
			}
		}
	}
	parts, err := w.complexParts(*b)
	if err != nil {
		return err
	}
	b.MonumentParts = parts
	return nil
}
func monumentWaterTile(b Building, p Point) bool {
	for _, part := range b.MonumentParts {
		if part.Kind == "valley_temple" && p.X == part.X+1 && part.contains(p) {
			return true
		}
	}
	return false
}
func (w World) validateMonumentParts(b Building) error {
	parts, err := w.complexParts(b)
	if err != nil {
		return err
	}
	if !slices.Equal(parts, b.MonumentParts) {
		return errors.New("invalid monument complex footprint")
	}
	return nil
}

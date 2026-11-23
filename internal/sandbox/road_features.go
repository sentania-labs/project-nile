package sandbox

import (
	"errors"
	"slices"
)

type RoadFeature struct {
	Point
	Kind string   `json:"kind"`
	Pass []string `json:"pass,omitempty"`
}

func (w World) roadFeature(p Point) int {
	return slices.IndexFunc(w.RoadFeatures, func(f RoadFeature) bool { return f.Point == p })
}

func (w *World) buildRoadFeature(kind string, p Point) error {
	if !slices.Contains(w.Roads, p) {
		return errors.New("place this on an existing road")
	}
	if w.roadFeature(p) >= 0 {
		return errors.New("this road already has a plaza or roadblock")
	}
	if w.cannotBuildFor(constructionCost(kind)) {
		return errors.New("not enough treasury")
	}
	w.RoadFeatures = append(w.RoadFeatures, RoadFeature{Point: p, Kind: kind})
	w.spendConstruction(constructionCost(kind))
	w.refresh()
	return nil
}

func (w World) blockedStreet(p Point, role string) bool {
	if b := w.defenseAt(p); b != nil && b.Kind == "gatehouse" && b.Damaged == "" {
		return true
	}
	i := w.roadFeature(p)
	if i < 0 || w.RoadFeatures[i].Kind != "roadblock" {
		return false
	}
	role = serviceGroup(role)
	return !slices.Contains(w.RoadFeatures[i].Pass, role)
}

func (w *World) RoadPermission(p Point, role string, allow bool) error {
	i := w.roadFeature(p)
	if i < 0 || w.RoadFeatures[i].Kind != "roadblock" {
		return errors.New("select a roadblock")
	}
	if !slices.Contains(roadGroups, role) {
		return errors.New("unknown roadblock permission")
	}
	group := w.RoadFeatures[i].Pass
	if j := slices.Index(group, role); allow && j < 0 {
		group = append(group, role)
		slices.Sort(group)
	} else if !allow && j >= 0 {
		group = slices.Delete(group, j, j+1)
	}
	w.RoadFeatures[i].Pass = group
	return nil
}

// A new roadblock can interrupt a walk already in progress. A returning
// walker travels directly home with whatever cargo remains.
func (w World) respectRoadblock(a *Walker) {
	if a.Stage != "patrol" || a.Next >= len(a.Route) || !w.blockedStreet(a.Route[a.Next], a.Role) {
		return
	}
	route := slices.Clone(a.Route[:a.Next])
	slices.Reverse(route)
	a.Route = route
	a.Next = 1
	a.Stage = "return"
}

func (w World) validateRoadFeatures() error {
	seen := map[Point]bool{}
	for _, f := range w.RoadFeatures {
		if !slices.Contains(w.Roads, f.Point) || seen[f.Point] || (f.Kind != "roadblock" && f.Kind != "plaza") {
			return errors.New("invalid street feature")
		}
		seen[f.Point] = true
		if f.Kind == "plaza" && len(f.Pass) > 0 {
			return errors.New("plaza cannot filter walkers")
		}
		last := ""
		for _, r := range f.Pass {
			if !slices.Contains(roadGroups, r) || r <= last {
				return errors.New("invalid roadblock permission")
			}
			last = r
		}
	}
	return nil
}

var roadGroups = []string{"laborseeker", "seller", "watercarrier", "firefighter", "architect", "taxcollector", "priest", "doctor", "teacher", "entertainer", "policeman"}

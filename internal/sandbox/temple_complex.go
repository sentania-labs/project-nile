package sandbox

import (
	"errors"
	"slices"
)

var complexKinds = []string{"complex_osiris", "complex_ra", "complex_ptah", "complex_seth", "complex_bast"}
var altarNames = []string{"Sebek", "Ma’at", "Amon", "Anubis", "Isis"}
var oracleNames = []string{"Min", "Horus", "Thoth", "Sekhmet", "Hathor"}

func init() {
	for i, kind := range complexKinds {
		buildingCatalog = append(buildingCatalog, CatalogEntry{kind, "Temple complex of " + gods[i], 7, 13, 2000, "Religion", 50})
		influences[kind] = influence{20, 2, -4, 6}
	}
	buildingCatalog = append(buildingCatalog, CatalogEntry{"complexaltar", "Add an altar", 1, 1, 900, "Religion", 0}, CatalogEntry{"complexoracle", "Add an oracle", 1, 1, 900, "Religion", 0})
}
func templeGod(kind string) int {
	if i := slices.Index(temples, kind); i >= 0 {
		return i
	}
	return slices.Index(complexKinds, kind)
}
func (w World) templeComplex() *Building {
	for i := range w.Buildings {
		if slices.Contains(complexKinds, w.Buildings[i].Kind) {
			return &w.Buildings[i]
		}
	}
	return nil
}
func (w World) templeBenefit(kind string, upgrade uint8) bool {
	return hasTempleBenefit(w.templeComplex(), kind, upgrade)
}
func hasTempleBenefit(b *Building, kind string, upgrade uint8) bool {
	return b != nil && b.Kind == kind && b.Damaged == "" && b.Connected && b.Workers > 0 && b.TempleUpgrades&upgrade == upgrade
}
func (w *World) addTempleUpgrade(kind string, p Point) error {
	b := w.templeComplex()
	if b == nil || !b.contains(p) || b.Damaged != "" {
		return errors.New("select the existing temple complex")
	}
	var flag uint8 = 1
	if kind == "complexoracle" {
		flag = 2
	}
	if b.TempleUpgrades&flag != 0 {
		return errors.New("the temple already has this addition")
	}
	cost := constructionCost(kind)
	if w.cannotBuildFor(cost) {
		return errors.New("not enough debens for this addition")
	}
	w.spendConstruction(cost)
	b.TempleUpgrades |= flag
	return nil
}

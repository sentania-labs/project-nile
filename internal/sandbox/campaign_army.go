package sandbox

import (
	"errors"

	"github.com/sentania-labs/project-nile/internal/campaign"
)

// DispatchCampaignArmy binds selected physical companies to an actual source
// request. Military cargo is never counted as a commercial inventory resource.
func (w *World) DispatchCampaignArmy(id int, fortIDs []int64, naval bool) error {
	if w.Campaign == nil {
		return errors.New("there is no active campaign")
	}
	r, err := w.Campaign.Request(id)
	if err != nil {
		return err
	}
	if r.Resource != "troops" || r.Status != "requested" {
		return errors.New("select an active military request")
	}
	copy := w.armyCopy()
	if err = copy.StartArmyDeployment(id, r.Location, fortIDs, naval); err != nil {
		return err
	}
	if err = copy.Campaign.DispatchArmy(id); err != nil {
		return err
	}
	*w = copy
	return nil
}

func (w *World) campaignArmyTick() {
	if w.Campaign == nil {
		return
	}
	// Copy the list: resolving an outcome can append its original child events.
	events := append([]campaign.EventState{}, w.Campaign.Events...)
	for _, e := range events {
		status := w.ArmyDeploymentStatus(e.ID)
		switch e.Status {
		case "army_battle", "army_late":
			if status == "defeated" {
				_ = w.Campaign.ResolveArmy(e.ID, true)
				continue
			}
			if status != "awaits_outcome" {
				continue
			}
			forces := w.ArmyDeploymentForces(e.ID)
			strength := 0
			for fort, n := range forces {
				b := w.building(fort)
				if b != nil {
					// Training and Seth's one-shot divine aid require separate
					// source validation. Nile's XP is not the original training.
					strength += campaign.CompanyStrength(n, 0, b.Kind == "chariotfort")
				}
			}
			loss := campaign.BattleLossPercent(strength, e.Amount, false)
			if e.Status == "army_late" {
				loss = 0
			}
			survivors := map[int64]int{}
			for fort, n := range forces {
				survivors[fort] = n - n*loss/100
			}
			if err := w.ResolveArmyDeployment(e.ID, survivors, loss == 100); err == nil {
				_ = w.Campaign.ResolveArmy(e.ID, loss == 100)
			}
		case "army_return":
			if err := w.LaunchArmyReturn(e.ID); err == nil {
				_ = w.Campaign.ArmyReturning(e.ID)
			}
		case "army_homebound":
			if status == "returned" || status == "defeated" {
				_ = w.Campaign.ArmyReturned(e.ID)
			}
		}
	}
}

package tui

import (
	"fmt"
)

func (a *tuiApp) refreshProfilesList() {
	a.suspendListSelection.Store(true)
	defer a.suspendListSelection.Store(false)

	a.profilesList.Clear()
	if len(a.profiles) == 0 {
		a.profilesList.AddItem("No profiles available", "", 0, nil)
		return
	}
	profiles := a.sortedProfilesForDisplayLocked()
	selectedIndex := 0
	for idx, profile := range profiles {
		label := fmt.Sprintf("%2d. %s", idx+1, a.profileLabelLocked(profile))
		if profile.ID == a.selectedProfileID {
			selectedIndex = idx
		}
		a.profilesList.AddItem(label, "", 0, nil)
	}
	a.profilesList.SetCurrentItem(selectedIndex)
}

func (a *tuiApp) refreshSubscriptionsList() {
	a.suspendListSelection.Store(true)
	defer a.suspendListSelection.Store(false)

	a.subscriptionsList.Clear()
	if len(a.subscriptions) == 0 {
		a.subscriptionsList.AddItem("No subscriptions available", "", 0, nil)
		return
	}
	selectedIndex := 0
	for idx, sub := range a.subscriptions {
		label := sub.Remarks
		if label == "" {
			label = sub.URL
		}
		label = fmt.Sprintf("%2d. %s", idx+1, label)
		if !sub.Enabled {
			label = "[off] " + label
		}
		if sub.ID == a.selectedSubID {
			selectedIndex = idx
		}
		a.subscriptionsList.AddItem(label, "", 0, nil)
	}
	a.subscriptionsList.SetCurrentItem(selectedIndex)
}

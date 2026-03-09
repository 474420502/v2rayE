package tui

func (a *tuiApp) storeProfiles(profiles []ProfileItem) {
	a.mu.Lock()
	a.profiles = profiles
	if len(profiles) == 0 {
		a.selectedProfileID = ""
		a.mu.Unlock()
		return
	}
	if a.selectedProfileID == "" {
		a.selectedProfileID = profiles[0].ID
		a.mu.Unlock()
		return
	}
	for _, profile := range profiles {
		if profile.ID == a.selectedProfileID {
			a.mu.Unlock()
			return
		}
	}
	a.selectedProfileID = profiles[0].ID
	a.mu.Unlock()
}

func (a *tuiApp) storeSubscriptions(subscriptions []SubscriptionItem) {
	a.mu.Lock()
	a.subscriptions = subscriptions
	if len(subscriptions) == 0 {
		a.selectedSubID = ""
		a.mu.Unlock()
		return
	}
	if a.selectedSubID == "" {
		a.selectedSubID = subscriptions[0].ID
		a.mu.Unlock()
		return
	}
	for _, sub := range subscriptions {
		if sub.ID == a.selectedSubID {
			a.mu.Unlock()
			return
		}
	}
	a.selectedSubID = subscriptions[0].ID
	a.mu.Unlock()
}

func (a *tuiApp) storeSelectedProfileID(profileID string) {
	a.mu.Lock()
	a.selectedProfileID = profileID
	a.mu.Unlock()
}

func (a *tuiApp) storeBatchDelayState(running bool, result *BatchDelayTestResult) {
	a.mu.Lock()
	a.batchRunning = running
	if result != nil {
		a.batchDelay = *result
	}
	a.mu.Unlock()
}

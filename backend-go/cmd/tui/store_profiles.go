package main

func (a *tuiApp) storeProfiles(profiles []ProfileItem) {
	a.mu.Lock()
	a.profiles = profiles
	if a.selectedProfileID == "" && len(profiles) > 0 {
		a.selectedProfileID = profiles[0].ID
	}
	a.mu.Unlock()
}

func (a *tuiApp) storeSubscriptions(subscriptions []SubscriptionItem) {
	a.mu.Lock()
	a.subscriptions = subscriptions
	if a.selectedSubID == "" && len(subscriptions) > 0 {
		a.selectedSubID = subscriptions[0].ID
	}
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

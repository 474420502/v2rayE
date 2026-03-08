package main

func (a *tuiApp) storeOverview(status CoreStatus, config map[string]any, stats StatsResult, availability AvailabilityResult) {
	a.mu.Lock()
	a.status = status
	a.config = config
	a.stats = stats
	a.availability = availability
	a.mu.Unlock()
}

func (a *tuiApp) storeCoreStatus(status CoreStatus) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
}

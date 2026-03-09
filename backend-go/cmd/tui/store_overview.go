package tui

import "fmt"

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

func (a *tuiApp) storeSystemProxyUsers(users []SystemProxyUserCandidate) {
	a.mu.Lock()
	a.systemProxyUsers = users
	a.proxyUsersStatus = fmt.Sprintf("loaded %d candidates", len(users))
	a.mu.Unlock()
}

func (a *tuiApp) currentSystemProxyUsers() []SystemProxyUserCandidate {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]SystemProxyUserCandidate(nil), a.systemProxyUsers...)
}

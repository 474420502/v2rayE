package main

func (a *tuiApp) storeNetwork(routing RoutingConfig, diagnostics RoutingDiagnostics, hits RoutingHitStats) {
	a.mu.Lock()
	a.routing = routing
	a.diagnostics = diagnostics
	a.hits = hits
	a.mu.Unlock()
}

func (a *tuiApp) storeRoutingTestResult(result RoutingTestResult) {
	a.mu.Lock()
	a.routingTest = result
	a.mu.Unlock()
}

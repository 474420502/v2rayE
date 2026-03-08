package main

import "strings"

func (a *tuiApp) storeLogLevelFilter(filter string) {
	a.mu.Lock()
	a.logLevelFilter = filter
	a.mu.Unlock()
}

func (a *tuiApp) storeLogSourceFilter(filter string) {
	a.mu.Lock()
	a.logSourceFilter = filter
	a.mu.Unlock()
}

func (a *tuiApp) storeLogSearchQuery(query string) {
	a.mu.Lock()
	a.logSearchQuery = query
	a.mu.Unlock()
}

func (a *tuiApp) storeIncomingLogLine(line LogLine) {
	a.mu.Lock()
	a.logLines = appendBoundedLogLines(a.logLines, line, 2000)
	a.logs = appendBounded(a.logs, line.Timestamp+" ["+strings.ToUpper(line.Level)+"] "+line.Message, 400)
	a.mu.Unlock()
}

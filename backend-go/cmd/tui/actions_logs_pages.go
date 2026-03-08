package main

import (
	"context"
	"strings"

	"github.com/gcla/gowid"
)

func (a *tuiApp) logLevelAction(level string) func(context.Context) error {
	return func(context.Context) error {
		a.storeLogLevelFilter(level)
		a.refreshLogsWidget()
		return nil
	}
}

func (a *tuiApp) logSourceAction(source string) func(context.Context) error {
	return func(context.Context) error {
		a.storeLogSourceFilter(source)
		a.refreshLogsWidget()
		return nil
	}
}

func (a *tuiApp) applyLogSearchAction(context.Context) error {
	a.storeLogSearchQuery(strings.TrimSpace(a.logsSearchInput.Text()))
	a.refreshLogsWidget()
	return nil
}

func (a *tuiApp) clearLogSearchAction(context.Context) error {
	a.storeLogSearchQuery("")
	a.runUI(func(app gowid.IApp) {
		a.logsSearchInput.SetText("", app)
	})
	a.refreshLogsWidget()
	return nil
}

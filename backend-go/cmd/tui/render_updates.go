package tui

import (
	"strconv"
	"strings"
	"time"

	"github.com/gcla/gowid"
)

func (a *tuiApp) refreshWidgets() {
	a.runUI(func(app gowid.IApp) {
		a.mu.Lock()
		defer a.mu.Unlock()

		a.dashboardSummary.SetText(a.formatDashboardSummary(), app)
		a.dashboardEvents.SetText(strings.Join(a.events, "\n"), app)
		a.logsView.SetText(a.formatFilteredLogs(), app)
		a.logsStatus.SetText(a.formatLogsStatus(), app)
		a.profilesListHolder.SetSubWidget(a.makeProfilesList(), app)
		a.profileBatchStatus.SetText(a.formatBatchDelayStatus(), app)
		a.profileDetail.SetText(a.formatSelectedProfile(), app)
		a.subscriptionsHolder.SetSubWidget(a.makeSubscriptionsList(), app)
		a.subscriptionDetail.SetText(a.formatSelectedSubscription(), app)
		a.networkSummary.SetText(a.formatNetworkSummary(), app)
		a.networkTestResult.SetText(a.formatRoutingTestResult(), app)
		a.settingsSummary.SetText(a.formatSettingsSummary(), app)
		if !a.settingsDirty || !a.settingsFormLoaded {
			a.settingsListenAddr.SetText(stringValue(a.config, "listenAddr"), app)
			a.settingsSocksPort.SetText(strconv.Itoa(intValue(a.config, "socksPort")), app)
			a.settingsHTTPPort.SetText(strconv.Itoa(intValue(a.config, "httpPort")), app)
			a.settingsTunName.SetText(stringValue(a.config, "tunName"), app)
			a.settingsProxyMode.SetText(stringValue(a.config, "systemProxyMode"), app)
			a.settingsProxyExcept.SetText(stringValue(a.config, "systemProxyExceptions"), app)
			a.settingsFormLoaded = true
		}
		a.syncPages(app)
	})
}

func (a *tuiApp) refreshLogsWidget() {
	a.runUI(func(app gowid.IApp) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.logsView.SetText(a.formatFilteredLogs(), app)
		a.logsStatus.SetText(a.formatLogsStatus(), app)
		if a.page == "logs" {
			a.syncPages(app)
		}
	})
}

func (a *tuiApp) pushEvent(line string) {
	a.mu.Lock()
	a.events = appendBounded(a.events, time.Now().Format(time.RFC3339)+" "+line, 200)
	a.mu.Unlock()
	a.runUI(func(app gowid.IApp) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.dashboardEvents.SetText(strings.Join(a.events, "\n"), app)
		if a.page == "dashboard" {
			a.syncPages(app)
		}
	})
}

func (a *tuiApp) setFooter(message string) {
	a.runUI(func(app gowid.IApp) {
		if a.footer != nil {
			a.footer.SetText(message, app)
		}
	})
}

func (a *tuiApp) markSettingsDirty() {
	a.mu.Lock()
	a.settingsDirty = true
	a.mu.Unlock()
	a.runUI(func(app gowid.IApp) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.settingsSummary.SetText(a.formatSettingsSummary(), app)
		if a.page == "settings" {
			a.syncPages(app)
		}
	})
}

func (a *tuiApp) clearSettingsDirty() {
	a.mu.Lock()
	a.settingsDirty = false
	a.settingsFormLoaded = false
	a.mu.Unlock()
}

func (a *tuiApp) setLogsStreamState(state string) {
	a.runUI(func(app gowid.IApp) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.logsStreamState == state {
			return
		}
		a.logsStreamState = state
		a.dashboardSummary.SetText(a.formatDashboardSummary(), app)
		a.logsStatus.SetText(a.formatLogsStatus(), app)
		if a.page == "dashboard" || a.page == "logs" {
			a.syncPages(app)
		}
	})
}

func (a *tuiApp) setEventsStreamState(state string) {
	a.runUI(func(app gowid.IApp) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.eventsStreamState == state {
			return
		}
		a.eventsStreamState = state
		a.dashboardSummary.SetText(a.formatDashboardSummary(), app)
		if a.page == "dashboard" {
			a.syncPages(app)
		}
	})
}

func (a *tuiApp) runUI(fn func(gowid.IApp)) {
	if a.app == nil {
		return
	}
	_ = a.app.Run(gowid.RunFunction(func(app gowid.IApp) {
		fn(app)
	}))
}

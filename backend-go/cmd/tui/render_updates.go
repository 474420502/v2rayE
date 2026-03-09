package tui

import (
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"
)

func (a *tuiApp) refreshWidgets() {
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()

		a.dashboardSummary.SetText(a.formatDashboardSummary(), app)
		a.dashboardEvents.SetText(strings.Join(a.events, "\n"), app)
		a.logsView.SetText(a.formatFilteredLogs(), app)
		a.logsStatus.SetText(fitSingleLineToWidth(a.formatLogsStatus(), a.viewportCols), app)
		a.refreshProfilesList()
		a.profileBatchStatus.SetText(a.formatBatchDelayStatus(), app)
		a.profileEditStatus.SetText(a.formatProfileEditStatus(), app)
		a.profileDetail.SetText(a.formatSelectedProfile(), app)
		if !a.profileEditDirty || !a.profileEditLoaded || a.profileEditForID != a.selectedProfileID {
			a.syncProfileEditorFromSelection(app)
		}
		a.refreshSubscriptionsList()
		a.subscriptionDetail.SetText(a.formatSelectedSubscription(), app)
		a.networkSummary.SetText(a.formatNetworkSummary(), app)
		a.withSuspendedFieldTracking(func() {
			if !a.networkRoutingDirty {
				a.networkRoutingMode.SetText(a.routing.Mode, app)
				a.networkDomainStrategy.SetText(a.routing.DomainStrategy, app)
				localBypass := true
				if a.routing.LocalBypassEnabled != nil {
					localBypass = *a.routing.LocalBypassEnabled
				}
				a.networkLocalBypass.SetText(strconv.FormatBool(localBypass), app)
			}
		})
		a.networkTestResult.SetText(a.formatRoutingTestResult(), app)
		a.settingsSummary.SetText(a.formatSettingsSummary(), app)
		a.withSuspendedFieldTracking(func() {
			if !a.settingsDirty || !a.settingsFormLoaded {
				a.settingsListenAddr.SetText(stringValue(a.config, "listenAddr"), app)
				a.settingsSocksPort.SetText(strconv.Itoa(intValue(a.config, "socksPort")), app)
				a.settingsHTTPPort.SetText(strconv.Itoa(intValue(a.config, "httpPort")), app)
				a.settingsCoreEngine.SetText(stringValue(a.config, "coreEngine"), app)
				a.settingsLogLevel.SetText(stringValue(a.config, "logLevel"), app)
				a.settingsSkipCert.SetText(strconv.FormatBool(boolValue(a.config, "skipCertVerify")), app)
				a.settingsTunName.SetText(stringValue(a.config, "tunName"), app)
				a.settingsTunMode.SetText(stringValue(a.config, "tunMode"), app)
				a.settingsTunMtu.SetText(strconv.Itoa(intValue(a.config, "tunMtu")), app)
				a.settingsTunAutoRoute.SetText(strconv.FormatBool(boolValue(a.config, "tunAutoRoute")), app)
				a.settingsTunStrict.SetText(strconv.FormatBool(boolValue(a.config, "tunStrictRoute")), app)
				a.settingsDNSMode.SetText(stringValue(a.config, "dnsMode"), app)
				a.settingsDNSList.SetText(strings.Join(toStringSlice(a.config["dnsList"]), ","), app)
				a.settingsProxyMode.SetText(stringValue(a.config, "systemProxyMode"), app)
				a.settingsProxyExcept.SetText(stringValue(a.config, "systemProxyExceptions"), app)
				a.settingsFormLoaded = true
			}
		})
	})
}

func (a *tuiApp) syncProfileEditorFromSelection(app *tview.Application) {
	selected := a.selectedProfile()
	a.withSuspendedFieldTracking(func() {
		if selected == nil {
			a.profileEditName.SetText("", app)
			a.profileEditAddress.SetText("", app)
			a.profileEditPort.SetText("", app)
			a.profileEditNetwork.SetText("", app)
			a.profileEditTLS.SetText("", app)
			a.profileEditSNI.SetText("", app)
			a.profileEditWSPath.SetText("", app)
			a.profileEditGRPCSvc.SetText("", app)
			a.profileDeleteConfirm.SetText("", app)
			a.profileEditForID = ""
			a.profileEditLoaded = true
			a.profileEditDirty = false
			a.profileEditMessage = ""
			return
		}

		a.profileEditName.SetText(selected.Name, app)
		a.profileEditAddress.SetText(selected.Address, app)
		a.profileEditPort.SetText(strconv.Itoa(selected.Port), app)
		if selected.Transport != nil {
			a.profileEditNetwork.SetText(selected.Transport.Network, app)
			a.profileEditTLS.SetText(strconv.FormatBool(selected.Transport.TLS), app)
			a.profileEditSNI.SetText(selected.Transport.SNI, app)
			a.profileEditFingerprint.SetText(selected.Transport.Fingerprint, app)
			a.profileEditALPN.SetText(strings.Join(selected.Transport.ALPN, ","), app)
			a.profileEditRealityPK.SetText(selected.Transport.RealityPublicKey, app)
			a.profileEditRealitySID.SetText(selected.Transport.RealityShortID, app)
			a.profileEditWSPath.SetText(selected.Transport.WSPath, app)
			a.profileEditGRPCSvc.SetText(selected.Transport.GRPCServiceName, app)
		} else {
			a.profileEditNetwork.SetText("", app)
			a.profileEditTLS.SetText("false", app)
			a.profileEditSNI.SetText("", app)
			a.profileEditFingerprint.SetText("", app)
			a.profileEditALPN.SetText("", app)
			a.profileEditRealityPK.SetText("", app)
			a.profileEditRealitySID.SetText("", app)
			a.profileEditWSPath.SetText("", app)
			a.profileEditGRPCSvc.SetText("", app)
		}
		a.profileDeleteConfirm.SetText("", app)
		a.profileEditForID = selected.ID
		a.profileEditLoaded = true
		a.profileEditDirty = false
		a.profileEditMessage = ""
	})
}

func (a *tuiApp) refreshLogsWidget() {
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.logsView.SetText(a.formatFilteredLogs(), app)
		a.logsStatus.SetText(fitSingleLineToWidth(a.formatLogsStatus(), a.viewportCols), app)
	})
}

func (a *tuiApp) pushEvent(line string) {
	a.mu.Lock()
	a.events = appendBounded(a.events, time.Now().Format(time.RFC3339)+" "+line, 200)
	a.mu.Unlock()
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.dashboardEvents.SetText(strings.Join(a.events, "\n"), app)
	})
}

func (a *tuiApp) setFooter(message string) {
	a.footerStatus = message
	a.runUI(func(app *tview.Application) {
		a.refreshFooter()
	})
}

func (a *tuiApp) markSettingsDirty() {
	a.mu.Lock()
	a.settingsDirty = true
	a.mu.Unlock()
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.settingsSummary.SetText(a.formatSettingsSummary(), app)
	})
}

func (a *tuiApp) clearSettingsDirty() {
	a.mu.Lock()
	a.settingsDirty = false
	a.settingsFormLoaded = false
	a.mu.Unlock()
}

func (a *tuiApp) markNetworkRoutingDirty() {
	a.mu.Lock()
	a.networkRoutingDirty = true
	a.mu.Unlock()
}

func (a *tuiApp) clearNetworkRoutingDirty() {
	a.mu.Lock()
	a.networkRoutingDirty = false
	a.mu.Unlock()
}

func (a *tuiApp) markProfileEditDirty() {
	a.mu.Lock()
	a.profileEditDirty = true
	a.profileEditMessage = ""
	a.mu.Unlock()
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.profileEditStatus.SetText(a.formatProfileEditStatus(), app)
	})
}

func (a *tuiApp) clearProfileEditDirty() {
	a.mu.Lock()
	a.profileEditDirty = false
	a.mu.Unlock()
}

func (a *tuiApp) setProfileEditMessage(message string) {
	a.mu.Lock()
	a.profileEditMessage = message
	a.mu.Unlock()
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.profileEditStatus.SetText(a.formatProfileEditStatus(), app)
	})
}

func (a *tuiApp) setLogsStreamState(state string) {
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.logsStreamState == state {
			return
		}
		a.logsStreamState = state
		a.dashboardSummary.SetText(a.formatDashboardSummary(), app)
		a.logsStatus.SetText(a.formatLogsStatus(), app)
	})
}

func (a *tuiApp) setEventsStreamState(state string) {
	a.runUI(func(app *tview.Application) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.eventsStreamState == state {
			return
		}
		a.eventsStreamState = state
		a.dashboardSummary.SetText(a.formatDashboardSummary(), app)
	})
}

func (a *tuiApp) runUI(fn func(*tview.Application)) {
	app := a.app
	if app == nil {
		return
	}
	if a.ctx.Err() != nil {
		return
	}
	// tview callbacks run on the UI goroutine. QueueUpdate/QueueUpdateDraw must not
	// be called synchronously from those callbacks, or deadlocks may occur.
	go app.QueueUpdateDraw(func() {
		if a.ctx.Err() != nil {
			return
		}
		fn(app)
	})
}

func (a *tuiApp) refreshFooter() {
	if a.footer == nil {
		return
	}
	footer := footerText(a.page, a.footerStatus)
	if a.viewportCols > 0 {
		footer = fitSingleLineToWidth(footer, a.viewportCols)
	}
	a.footer.SetText(footer, a.app)
}

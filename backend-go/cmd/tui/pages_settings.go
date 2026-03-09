package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildSettingsPage() builtPage {
	saveBtn := a.actionButton("Save Config", a.saveConfigAction)
	clearErrBtn := a.actionButton("Clear Core Error", a.clearCoreErrorAction)
	exitCleanupBtn := a.actionButton("Exit Cleanup", a.exitCleanupAction)
	proxyOn := a.actionButton("Proxy On", a.selectProxyModeForcedChangeAction)
	proxyOff := a.actionButton("Proxy Off", a.selectProxyModeForcedClearAction)
	proxyPac := a.actionButton("Proxy PAC", a.selectProxyModePacAction)
	tunOff := a.actionButton("TUN Off", a.selectTunModeOffAction)
	tunMixed := a.actionButton("TUN Mixed", a.selectTunModeMixedAction)
	tunSystem := a.actionButton("TUN System", a.selectTunModeSystemAction)
	tunGvisor := a.actionButton("TUN gVisor", a.selectTunModeGvisorAction)
	logDebug := a.actionButton("Log Debug", a.selectLogLevelDebugAction)
	logInfo := a.actionButton("Log Info", a.selectLogLevelInfoAction)
	logWarn := a.actionButton("Log Warn", a.selectLogLevelWarningAction)
	logError := a.actionButton("Log Error", a.selectLogLevelErrorAction)
	engineXray := a.actionButton("Engine xray-core", a.selectCoreEngineXrayAction)

	controls := buttonRow(saveBtn, clearErrBtn, exitCleanupBtn)
	proxyActions := buttonRow(proxyOn, proxyOff, proxyPac)
	tunActions := buttonRow(tunOff, tunMixed, tunSystem, tunGvisor)
	logActions := buttonRow(logDebug, logInfo, logWarn, logError)
	engineActions := buttonRow(engineXray)
	if a.useStackedLayout() {
		controls = buttonColumn(saveBtn, clearErrBtn, exitCleanupBtn)
		proxyActions = buttonColumn(proxyOn, proxyOff, proxyPac)
		tunActions = buttonColumn(tunOff, tunMixed, tunSystem, tunGvisor)
		logActions = buttonColumn(logDebug, logInfo, logWarn, logError)
		engineActions = buttonColumn(engineXray)
	}

	form := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, primitive := range []tview.Primitive{
		a.settingsListenAddr,
		a.settingsSocksPort,
		a.settingsHTTPPort,
		engineActions,
		a.settingsCoreEngine,
		logActions,
		a.settingsLogLevel,
		a.settingsSkipCert,
		a.settingsTunName,
		tunActions,
		a.settingsTunMode,
		a.settingsTunMtu,
		a.settingsTunAutoRoute,
		a.settingsTunStrict,
		a.settingsDNSMode,
		a.settingsDNSList,
		proxyActions,
		a.settingsProxyMode,
		a.settingsProxyExcept,
	} {
		form.AddItem(primitive, 1, 0, false)
	}
	body := splitContent(
		a.useStackedLayout(),
		wrapPanel("Config Editor", form),
		wrapPanel("Config Summary", a.settingsSummary),
		3,
		4,
	)

	quickActions := tview.NewFlex().SetDirection(tview.FlexRow)
	controlsHeight := actionBlockHeight(a.useStackedLayout(), 3)
	quickActionsContentHeight := 1 + 1 + controlsHeight
	quickActions.AddItem(newMutedText("Quick actions (presets are available in Config Editor)"), 1, 0, false)
	quickActions.AddItem(verticalSpacer(1), 1, 0, false)
	quickActions.AddItem(controls, controlsHeight, 0, false)

	root := buildPageLayout("Quick Actions", quickActions, quickActionsContentHeight, body)
	return builtPage{
		root: root,
		focusables: joinFocusables(
			buttonsToFocusables(saveBtn, clearErrBtn, exitCleanupBtn),
			primitivesToFocusables(a.settingsListenAddr, a.settingsSocksPort, a.settingsHTTPPort),
			buttonsToFocusables(engineXray),
			primitivesToFocusables(a.settingsCoreEngine),
			buttonsToFocusables(logDebug, logInfo, logWarn, logError),
			primitivesToFocusables(a.settingsLogLevel, a.settingsSkipCert, a.settingsTunName),
			buttonsToFocusables(tunOff, tunMixed, tunSystem, tunGvisor),
			primitivesToFocusables(a.settingsTunMode, a.settingsTunMtu, a.settingsTunAutoRoute, a.settingsTunStrict, a.settingsDNSMode, a.settingsDNSList),
			buttonsToFocusables(proxyOn, proxyOff, proxyPac),
			primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept),
		),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(saveBtn, clearErrBtn, exitCleanupBtn),
			primitivesToFocusables(a.settingsListenAddr, a.settingsSocksPort, a.settingsHTTPPort),
			buttonsToFocusables(engineXray),
			primitivesToFocusables(a.settingsCoreEngine),
			buttonsToFocusables(logDebug, logInfo, logWarn, logError),
			primitivesToFocusables(a.settingsLogLevel, a.settingsSkipCert, a.settingsTunName),
			buttonsToFocusables(tunOff, tunMixed, tunSystem, tunGvisor),
			primitivesToFocusables(a.settingsTunMode, a.settingsTunMtu, a.settingsTunAutoRoute, a.settingsTunStrict, a.settingsDNSMode, a.settingsDNSList),
			buttonsToFocusables(proxyOn, proxyOff, proxyPac),
			primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept),
		},
	}
}

package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildSettingsPage() builtPage {
	saveBtn := a.actionButton(a.t("settings.btn.saveConfig"), a.saveConfigAction)
	clearErrBtn := a.actionButton(a.t("settings.btn.clearCoreError"), a.clearCoreErrorAction)
	exitCleanupBtn := a.actionButton(a.t("settings.btn.exitCleanup"), a.exitCleanupAction)
	proxyOn := a.actionButton(a.t("settings.btn.proxyOn"), a.selectProxyModeForcedChangeAction)
	proxyOff := a.actionButton(a.t("settings.btn.proxyOff"), a.selectProxyModeForcedClearAction)
	proxyPac := a.actionButton(a.t("settings.btn.proxyPac"), a.selectProxyModePacAction)
	proxyUsersDetect := a.actionButton(a.t("settings.btn.detectUsers"), a.detectSystemProxyUsersAction)
	proxyUsersDefault := a.actionButton(a.t("settings.btn.useDesktopUser"), a.useDesktopProxyUserAction)
	proxyUsersAddAll := a.actionButton(a.t("settings.btn.addNonSystemUsers"), a.addNonSystemProxyUsersAction)
	proxyUsersSelect := a.actionButton(a.t("settings.btn.selectUsers"), a.openProxyUserSelectDialogAction)
	tunOff := a.actionButton(a.t("settings.btn.tunOff"), a.selectTunModeOffAction)
	tunMixed := a.actionButton(a.t("settings.btn.tunMixed"), a.selectTunModeMixedAction)
	tunSystem := a.actionButton(a.t("settings.btn.tunSystem"), a.selectTunModeSystemAction)
	tunGvisor := a.actionButton(a.t("settings.btn.tunGvisor"), a.selectTunModeGvisorAction)
	logDebug := a.actionButton(a.t("settings.btn.logDebug"), a.selectLogLevelDebugAction)
	logInfo := a.actionButton(a.t("settings.btn.logInfo"), a.selectLogLevelInfoAction)
	logWarn := a.actionButton(a.t("settings.btn.logWarn"), a.selectLogLevelWarningAction)
	logError := a.actionButton(a.t("settings.btn.logError"), a.selectLogLevelErrorAction)
	dnsSystem := a.actionButton(a.t("settings.btn.dnsSystem"), a.selectDNSModeSystemAction)
	dnsList := a.actionButton(a.t("settings.btn.dnsList"), a.selectDNSModeListAction)
	dnsDirect := a.actionButton(a.t("settings.btn.dnsDirect"), a.selectDNSModeDirectAction)
	engineXray := a.actionButton(a.t("settings.btn.engineXray"), a.selectCoreEngineXrayAction)
	langEN := a.actionButton(a.t("settings.lang.button.en"), a.selectUILanguageEnglishAction)
	langZH := a.actionButton(a.t("settings.lang.button.zh"), a.selectUILanguageChineseAction)

	controls := buttonRow(saveBtn, clearErrBtn, exitCleanupBtn)
	proxyActions := buttonRow(proxyOn, proxyOff, proxyPac)
	proxyUserActions := buttonRow(proxyUsersDetect, proxyUsersDefault, proxyUsersAddAll, proxyUsersSelect)
	tunActions := buttonRow(tunOff, tunMixed, tunSystem, tunGvisor)
	logActions := buttonRow(logDebug, logInfo, logWarn, logError)
	dnsActions := buttonRow(dnsSystem, dnsList, dnsDirect)
	engineActions := buttonRow(engineXray)
	langActions := buttonRow(langEN, langZH)
	if a.useStackedLayout() {
		controls = buttonColumn(saveBtn, clearErrBtn, exitCleanupBtn)
		proxyActions = buttonColumn(proxyOn, proxyOff, proxyPac)
		proxyUserActions = buttonColumn(proxyUsersDetect, proxyUsersDefault, proxyUsersAddAll, proxyUsersSelect)
		tunActions = buttonColumn(tunOff, tunMixed, tunSystem, tunGvisor)
		logActions = buttonColumn(logDebug, logInfo, logWarn, logError)
		dnsActions = buttonColumn(dnsSystem, dnsList, dnsDirect)
		engineActions = buttonColumn(engineXray)
		langActions = buttonColumn(langEN, langZH)
	}

	form := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, primitive := range []tview.Primitive{
		a.settingsListenAddr,
		a.settingsSocksPort,
		a.settingsHTTPPort,
		langActions,
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
		dnsActions,
		a.settingsDNSMode,
		a.settingsDNSList,
		proxyActions,
		a.settingsProxyMode,
		a.settingsProxyExcept,
		proxyUserActions,
		a.settingsProxyUsers,
	} {
		form.AddItem(primitive, 1, 0, false)
	}
	body := splitContent(
		a.useStackedLayout(),
		wrapPanel(a.t("settings.panel.editor"), form),
		wrapPanel(a.t("settings.panel.summary"), a.settingsSummary),
		3,
		4,
	)

	quickActions := tview.NewFlex().SetDirection(tview.FlexRow)
	controlsHeight := actionBlockHeight(a.useStackedLayout(), 3)
	quickActionsContentHeight := 1 + 1 + controlsHeight
	quickActions.AddItem(newMutedText(a.t("settings.desc")), 1, 0, false)
	quickActions.AddItem(verticalSpacer(1), 1, 0, false)
	quickActions.AddItem(controls, controlsHeight, 0, false)

	root := buildPageLayout(a.t("settings.panel.quickActions"), quickActions, quickActionsContentHeight, body)
	return builtPage{
		root: root,
		focusables: joinFocusables(
			buttonsToFocusables(saveBtn, clearErrBtn, exitCleanupBtn),
			primitivesToFocusables(a.settingsListenAddr, a.settingsSocksPort, a.settingsHTTPPort),
			buttonsToFocusables(langEN, langZH),
			buttonsToFocusables(engineXray),
			primitivesToFocusables(a.settingsCoreEngine),
			buttonsToFocusables(logDebug, logInfo, logWarn, logError),
			primitivesToFocusables(a.settingsLogLevel, a.settingsSkipCert, a.settingsTunName),
			buttonsToFocusables(tunOff, tunMixed, tunSystem, tunGvisor),
			primitivesToFocusables(a.settingsTunMode, a.settingsTunMtu, a.settingsTunAutoRoute, a.settingsTunStrict),
			buttonsToFocusables(dnsSystem, dnsList, dnsDirect),
			primitivesToFocusables(a.settingsDNSMode, a.settingsDNSList),
			buttonsToFocusables(proxyOn, proxyOff, proxyPac),
			buttonsToFocusables(proxyUsersDetect, proxyUsersDefault, proxyUsersAddAll, proxyUsersSelect),
			primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept, a.settingsProxyUsers),
		),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(saveBtn, clearErrBtn, exitCleanupBtn),
			primitivesToFocusables(a.settingsListenAddr, a.settingsSocksPort, a.settingsHTTPPort),
			buttonsToFocusables(langEN, langZH),
			buttonsToFocusables(engineXray),
			primitivesToFocusables(a.settingsCoreEngine),
			buttonsToFocusables(logDebug, logInfo, logWarn, logError),
			primitivesToFocusables(a.settingsLogLevel, a.settingsSkipCert, a.settingsTunName),
			buttonsToFocusables(tunOff, tunMixed, tunSystem, tunGvisor),
			primitivesToFocusables(a.settingsTunMode, a.settingsTunMtu, a.settingsTunAutoRoute, a.settingsTunStrict),
			buttonsToFocusables(dnsSystem, dnsList, dnsDirect),
			primitivesToFocusables(a.settingsDNSMode, a.settingsDNSList),
			buttonsToFocusables(proxyOn, proxyOff, proxyPac),
			buttonsToFocusables(proxyUsersDetect, proxyUsersDefault, proxyUsersAddAll, proxyUsersSelect),
			primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept, a.settingsProxyUsers),
		},
	}
}

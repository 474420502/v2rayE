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
	// 优化按钮布局：将长按钮组拆分成更短的组
	proxyActions := buttonRow(proxyOn, proxyOff, proxyPac)
	// 代理用户按钮拆分成两行，每行2个
	proxyUserActionsRow1 := buttonRow(proxyUsersDetect, proxyUsersDefault)
	proxyUserActionsRow2 := buttonRow(proxyUsersAddAll, proxyUsersSelect)
	tunActions := buttonRow(tunOff, tunMixed, tunSystem, tunGvisor)
	logActions := buttonRow(logDebug, logInfo, logWarn, logError)
	dnsActions := buttonRow(dnsSystem, dnsList, dnsDirect)
	engineActions := buttonRow(engineXray)
	langActions := buttonRow(langEN, langZH)

	if a.useStackedLayout() {
		controls = buttonColumn(saveBtn, clearErrBtn, exitCleanupBtn)
		proxyActions = buttonColumn(proxyOn, proxyOff, proxyPac)
		proxyUserActionsRow1 = buttonColumn(proxyUsersDetect, proxyUsersDefault)
		proxyUserActionsRow2 = buttonColumn(proxyUsersAddAll, proxyUsersSelect)
		tunActions = buttonColumn(tunOff, tunMixed, tunSystem, tunGvisor)
		logActions = buttonColumn(logDebug, logInfo, logWarn, logError)
		dnsActions = buttonColumn(dnsSystem, dnsList, dnsDirect)
		engineActions = buttonColumn(engineXray)
		langActions = buttonColumn(langEN, langZH)
	}

	form := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, row := range []struct {
		primitive tview.Primitive
		height    int
	}{
		{primitive: a.settingsListenAddr, height: 1},
		{primitive: a.settingsSocksPort, height: 1},
		{primitive: a.settingsHTTPPort, height: 1},
		{primitive: langActions, height: actionBlockHeight(a.useStackedLayout(), 2)},
		{primitive: engineActions, height: actionBlockHeight(a.useStackedLayout(), 1)},
		{primitive: a.settingsCoreEngine, height: 1},
		{primitive: logActions, height: actionBlockHeight(a.useStackedLayout(), 4)},
		{primitive: a.settingsLogLevel, height: 1},
		{primitive: a.settingsSkipCert, height: 1},
		{primitive: a.settingsTunName, height: 1},
		{primitive: tunActions, height: actionBlockHeight(a.useStackedLayout(), 4)},
		{primitive: a.settingsTunMode, height: 1},
		{primitive: a.settingsTunMtu, height: 1},
		{primitive: a.settingsTunAutoRoute, height: 1},
		{primitive: a.settingsTunStrict, height: 1},
		{primitive: dnsActions, height: actionBlockHeight(a.useStackedLayout(), 3)},
		{primitive: a.settingsDNSMode, height: 1},
		{primitive: a.settingsDNSList, height: 1},
		{primitive: proxyActions, height: actionBlockHeight(a.useStackedLayout(), 3)},
		{primitive: a.settingsProxyMode, height: 1},
		{primitive: a.settingsProxyExcept, height: 1},
		{primitive: proxyUserActionsRow1, height: actionBlockHeight(a.useStackedLayout(), 2)},
		{primitive: proxyUserActionsRow2, height: actionBlockHeight(a.useStackedLayout(), 2)},
		{primitive: a.settingsProxyUsers, height: 1},
	} {
		form.AddItem(row.primitive, row.height, 0, false)
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
	// 优化焦点组：将代理用户按钮分成两组
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
			primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept),
			buttonsToFocusables(proxyUsersDetect, proxyUsersDefault),
			buttonsToFocusables(proxyUsersAddAll, proxyUsersSelect),
			primitivesToFocusables(a.settingsProxyUsers),
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
			primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept),
			buttonsToFocusables(proxyUsersDetect, proxyUsersDefault),
			buttonsToFocusables(proxyUsersAddAll, proxyUsersSelect),
			primitivesToFocusables(a.settingsProxyUsers),
		},
	}
}

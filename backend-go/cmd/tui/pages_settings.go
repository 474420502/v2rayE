package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildSettingsPage() builtPage {
	stackedRows := a.stackFormRows()
	stackedActions := a.stackActionButtons()
	buildGroupPanel := func(title string, rows ...struct {
		primitive tview.Primitive
		height    int
	}) tview.Primitive {
		content := tview.NewFlex().SetDirection(tview.FlexRow)
		for idx, row := range rows {
			content.AddItem(row.primitive, row.height, 0, false)
			if idx != len(rows)-1 {
				content.AddItem(verticalSpacer(1), 1, 0, false)
			}
		}
		return wrapPanel(title, content)
	}

	saveBtn := a.actionButton(a.t("settings.btn.saveConfig"), a.saveConfigAction)
	clearErrBtn := a.actionButton(a.t("settings.btn.clearCoreError"), a.clearCoreErrorAction)
	exitCleanupBtn := a.actionButton(a.t("settings.btn.exitCleanup"), a.exitCleanupAction)
	proxyUsersDetect := a.actionButton(a.t("settings.btn.detectUsers"), a.detectSystemProxyUsersAction)
	proxyUsersDefault := a.actionButton(a.t("settings.btn.useDesktopUser"), a.useDesktopProxyUserAction)
	proxyUsersAddAll := a.actionButton(a.t("settings.btn.addNonSystemUsers"), a.addNonSystemProxyUsersAction)
	proxyUsersSelect := a.actionButton(a.t("settings.btn.selectUsers"), a.openProxyUserSelectDialogAction)

	controls := buttonStrip(stackedActions, saveBtn, clearErrBtn, exitCleanupBtn)
	proxyUserActionsRow1 := buttonStrip(stackedActions, proxyUsersDetect, proxyUsersDefault)
	proxyUserActionsRow2 := buttonStrip(stackedActions, proxyUsersAddAll, proxyUsersSelect)

	generalPanel := buildGroupPanel(
		a.t("settings.group.general"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsListenAddr, a.settingsSocksPort, stackedRows, 3, 2), height: dualItemRowHeight(stackedRows)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsHTTPPort, a.settingsSkipCert, stackedRows, 2, 3), height: dualItemRowHeight(stackedRows)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsLanguage, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsCoreEngine, a.settingsLogLevel, stackedRows, 1, 1), height: dualItemRowHeight(stackedRows)},
	)

	tunPanel := buildGroupPanel(
		a.t("settings.group.tun"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsTunName, a.settingsTunMode, stackedRows, 2, 3), height: dualItemRowHeight(stackedRows)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsTunMtu, a.settingsTunAutoRoute, stackedRows, 2, 3), height: dualItemRowHeight(stackedRows)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsTunStrict, height: 1},
	)

	proxyPanel := buildGroupPanel(
		a.t("settings.group.proxy"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsProxyMode, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsLocalProxyMode, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsProxyExcept, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: proxyUserActionsRow1, height: actionBlockHeight(stackedActions, 2)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: proxyUserActionsRow2, height: actionBlockHeight(stackedActions, 2)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsProxyUsers, height: 1},
	)

	dnsPanel := buildGroupPanel(
		a.t("settings.group.dns"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsDNSMode, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsDNSList, height: 1},
	)

	leftColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	leftColumn.AddItem(generalPanel, 0, 6, false)
	leftColumn.AddItem(verticalSpacer(1), 1, 0, false)
	leftColumn.AddItem(tunPanel, 0, 5, false)

	rightColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	rightColumn.AddItem(proxyPanel, 0, 6, false)
	rightColumn.AddItem(verticalSpacer(1), 1, 0, false)
	rightColumn.AddItem(dnsPanel, 0, 3, false)
	rightColumn.AddItem(verticalSpacer(1), 1, 0, false)
	summaryPanel := wrapPanel(a.t("settings.panel.summary"), a.settingsSummary)
	rightColumn.AddItem(summaryPanel, 0, 4, false)
	body := wrapPanel(a.t("settings.panel.editor"), splitContent(a.stackPageColumns(), leftColumn, rightColumn, 5, 6))

	quickActions := tview.NewFlex().SetDirection(tview.FlexRow)
	controlsHeight := actionBlockHeight(stackedActions, 3)
	quickActionsContentHeight := 1 + 1 + controlsHeight
	quickActions.AddItem(newMutedText(a.t("settings.desc")), 1, 0, false)
	quickActions.AddItem(verticalSpacer(1), 1, 0, false)
	quickActions.AddItem(controls, controlsHeight, 0, false)

	root := buildPageLayout(a.t("settings.panel.quickActions"), quickActions, quickActionsContentHeight, body)

	actionGroup := buttonsToFocusables(saveBtn, clearErrBtn, exitCleanupBtn)
	generalGroup := joinFocusables(
		primitivesToFocusables(a.settingsListenAddr, a.settingsSocksPort, a.settingsHTTPPort),
		primitivesToFocusables(a.settingsLanguage, a.settingsCoreEngine, a.settingsLogLevel, a.settingsSkipCert),
	)
	proxyFieldsGroup := primitivesToFocusables(a.settingsProxyMode, a.settingsLocalProxyMode, a.settingsProxyExcept)
	proxyDetectGroup := buttonsToFocusables(proxyUsersDetect, proxyUsersDefault)
	proxySelectGroup := buttonsToFocusables(proxyUsersAddAll, proxyUsersSelect)
	proxyUsersGroup := primitivesToFocusables(a.settingsProxyUsers)
	tunGroup := primitivesToFocusables(a.settingsTunName, a.settingsTunMode, a.settingsTunMtu, a.settingsTunAutoRoute, a.settingsTunStrict)
	dnsGroup := primitivesToFocusables(a.settingsDNSMode, a.settingsDNSList)
	summaryGroup := primitivesToFocusables(a.settingsSummary)
	contentGroup := joinFocusables(
		generalGroup,
		proxyFieldsGroup,
		proxyDetectGroup,
		proxySelectGroup,
		proxyUsersGroup,
		tunGroup,
		dnsGroup,
		summaryGroup,
	)

	focusGroups := [][]tview.Primitive{
		actionGroup,
		generalGroup,
		proxyFieldsGroup,
		proxyDetectGroup,
		proxySelectGroup,
		proxyUsersGroup,
		tunGroup,
		dnsGroup,
		summaryGroup,
	}

	return builtPage{
		root:       root,
		focusables: joinFocusables(actionGroup, contentGroup),
		focusGroups: focusGroups,
	}
}

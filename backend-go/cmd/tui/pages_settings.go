package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildSettingsPage() builtPage {
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

	controls := buttonRow(saveBtn, clearErrBtn, exitCleanupBtn)
	proxyUserActionsRow1 := buttonRow(proxyUsersDetect, proxyUsersDefault)
	proxyUserActionsRow2 := buttonRow(proxyUsersAddAll, proxyUsersSelect)

	generalPanel := buildGroupPanel(
		a.t("settings.group.general"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsListenAddr, a.settingsSocksPort, false, 3, 2), height: dualItemRowHeight(false)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsHTTPPort, a.settingsSkipCert, false, 2, 3), height: dualItemRowHeight(false)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.settingsLanguage, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsCoreEngine, a.settingsLogLevel, false, 1, 1), height: dualItemRowHeight(false)},
	)

	tunPanel := buildGroupPanel(
		a.t("settings.group.tun"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsTunName, a.settingsTunMode, false, 2, 3), height: dualItemRowHeight(false)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: inputRow(a.settingsTunMtu, a.settingsTunAutoRoute, false, 2, 3), height: dualItemRowHeight(false)},
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
		}{primitive: a.settingsProxyExcept, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: proxyUserActionsRow1, height: actionBlockHeight(false, 2)},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: proxyUserActionsRow2, height: actionBlockHeight(false, 2)},
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
	rightColumn.AddItem(wrapPanel(a.t("settings.panel.summary"), a.settingsSummary), 0, 4, false)

	editorGrid := tview.NewGrid().SetBorders(false).SetGap(1, 1)
	editorGrid.SetRows(0, 0, 0).SetColumns(0, 0)
	editorGrid.AddItem(generalPanel, 0, 0, 1, 1, 0, 0, false)
	editorGrid.AddItem(proxyPanel, 0, 1, 1, 1, 0, 0, false)
	editorGrid.AddItem(tunPanel, 1, 0, 1, 1, 0, 0, false)
	editorGrid.AddItem(dnsPanel, 1, 1, 1, 1, 0, 0, false)
	editorGrid.AddItem(wrapPanel(a.t("settings.panel.summary"), a.settingsSummary), 2, 0, 1, 2, 0, 0, false)
	body := wrapPanel(a.t("settings.panel.editor"), editorGrid)

	quickActions := tview.NewFlex().SetDirection(tview.FlexRow)
	controlsHeight := actionBlockHeight(false, 3)
	quickActionsContentHeight := 1 + 1 + controlsHeight
	quickActions.AddItem(newMutedText(a.t("settings.desc")), 1, 0, false)
	quickActions.AddItem(verticalSpacer(1), 1, 0, false)
	quickActions.AddItem(controls, controlsHeight, 0, false)

	root := buildPageLayout(a.t("settings.panel.quickActions"), quickActions, quickActionsContentHeight, body)

	actionGroup := buttonsToFocusables(saveBtn, clearErrBtn, exitCleanupBtn)
	contentGroup := joinFocusables(
		primitivesToFocusables(a.settingsListenAddr, a.settingsSocksPort, a.settingsHTTPPort),
		primitivesToFocusables(a.settingsLanguage, a.settingsCoreEngine, a.settingsLogLevel, a.settingsSkipCert),
		primitivesToFocusables(a.settingsTunName, a.settingsTunMode, a.settingsTunMtu, a.settingsTunAutoRoute, a.settingsTunStrict),
		primitivesToFocusables(a.settingsDNSMode, a.settingsDNSList),
		primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept),
		buttonsToFocusables(proxyUsersDetect, proxyUsersDefault),
		buttonsToFocusables(proxyUsersAddAll, proxyUsersSelect),
		primitivesToFocusables(a.settingsProxyUsers),
	)

	focusGroups := [][]tview.Primitive{
		actionGroup,
		primitivesToFocusables(a.settingsListenAddr, a.settingsSocksPort, a.settingsHTTPPort),
		primitivesToFocusables(a.settingsLanguage, a.settingsCoreEngine, a.settingsLogLevel, a.settingsSkipCert),
		primitivesToFocusables(a.settingsTunName, a.settingsTunMode, a.settingsTunMtu, a.settingsTunAutoRoute, a.settingsTunStrict),
		primitivesToFocusables(a.settingsDNSMode, a.settingsDNSList),
		primitivesToFocusables(a.settingsProxyMode, a.settingsProxyExcept),
		buttonsToFocusables(proxyUsersDetect, proxyUsersDefault),
		buttonsToFocusables(proxyUsersAddAll, proxyUsersSelect),
		primitivesToFocusables(a.settingsProxyUsers),
	}

	return builtPage{
		root:       root,
		focusables: joinFocusables(actionGroup, contentGroup),
		focusGroups: focusGroups,
	}
}

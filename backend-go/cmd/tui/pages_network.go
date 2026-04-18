package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildNetworkPage() builtPage {
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

	checkBtn := a.actionButton(a.t("network.btn.check"), a.reloadNetworkAction)
	applyProxy := a.actionButton(a.t("network.btn.applyProxy"), a.applySystemProxyAction)
	clearProxy := a.actionButton(a.t("network.btn.clearProxy"), a.clearSystemProxyAction)
	saveRouting := a.actionButton(a.t("network.btn.saveRouting"), a.saveRoutingModeAction)
	geoUpdate := a.actionButton(a.t("network.btn.geoUpdate"), a.updateGeoDataAction)
	repairTun := a.actionButton(a.t("network.btn.repairTun"), a.repairTunAction)
	routeTest := a.actionButton(a.t("network.btn.routeTest"), a.routeTestAction)

	proxyActionsRow := buttonStrip(stackedActions, applyProxy, clearProxy)
	operationsRow1 := buttonStrip(stackedActions, saveRouting, geoUpdate)
	operationsRow2 := buttonStrip(stackedActions, repairTun, routeTest)

	testRow := inputRow(a.networkTestTarget, a.networkTestPort, stackedRows, 4, 1)
	proxyActionsHeight := actionBlockHeight(stackedActions, 2)
	operationsHeight := actionBlockHeight(stackedActions, 2)
	testRowHeight := dualItemRowHeight(stackedRows)

	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanelHeight := 1 + 1 + 1
	actionsPanel.AddItem(newMutedText(a.t("network.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(buttonStrip(stackedActions, checkBtn), actionBlockHeight(stackedActions, 1), 0, false)

	presetsPanel := buildGroupPanel(
		a.t("network.group.presets"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.networkPresetSelect, height: 1},
	)

	routingPanel := buildGroupPanel(
		a.t("network.group.routing"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.networkRoutingMode, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.networkDomainStrategy, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.networkLocalBypass, height: 1},
	)

	systemProxyPanel := buildGroupPanel(
		a.t("network.group.systemProxy"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: proxyActionsRow, height: proxyActionsHeight},
	)

	toolsPanel := buildGroupPanel(
		a.t("network.group.tools"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: operationsRow1, height: operationsHeight},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: operationsRow2, height: operationsHeight},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: testRow, height: testRowHeight},
	)

	diagnosticsPanel := wrapPanel(a.t("network.panel.diagnostics"), a.networkSummary)
	testResultPanel := wrapPanel(a.t("network.panel.testResult"), a.networkTestResult)

	leftColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	leftColumn.AddItem(presetsPanel, 0, 3, false)
	leftColumn.AddItem(verticalSpacer(1), 1, 0, false)
	leftColumn.AddItem(routingPanel, 0, 4, false)
	leftColumn.AddItem(verticalSpacer(1), 1, 0, false)
	leftColumn.AddItem(systemProxyPanel, 0, 2, false)
	leftColumn.AddItem(verticalSpacer(1), 1, 0, false)
	leftColumn.AddItem(toolsPanel, 0, 3, false)

	rightColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	rightColumn.AddItem(diagnosticsPanel, 0, 7, false)
	rightColumn.AddItem(verticalSpacer(1), 1, 0, false)
	rightColumn.AddItem(testResultPanel, 0, 3, false)

	body := splitContent(a.stackPageColumns(), leftColumn, rightColumn, 5, 6)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsPanelHeight, body)
	checkGroup := buttonsToFocusables(checkBtn)
	presetGroup := primitivesToFocusables(a.networkPresetSelect)
	routingGroup := primitivesToFocusables(a.networkRoutingMode, a.networkDomainStrategy, a.networkLocalBypass)
	systemProxyGroup := buttonsToFocusables(applyProxy, clearProxy)
	toolsPrimaryGroup := buttonsToFocusables(saveRouting, geoUpdate)
	toolsSecondaryGroup := buttonsToFocusables(repairTun, routeTest)
	testInputGroup := primitivesToFocusables(a.networkTestTarget, a.networkTestPort)
	diagnosticsGroup := primitivesToFocusables(a.networkSummary)
	testResultGroup := primitivesToFocusables(a.networkTestResult)

	return builtPage{
		root: root,
		focusables: joinFocusables(
			checkGroup,
			presetGroup,
			routingGroup,
			systemProxyGroup,
			toolsPrimaryGroup,
			toolsSecondaryGroup,
			testInputGroup,
			diagnosticsGroup,
			testResultGroup,
		),
		focusGroups: [][]tview.Primitive{
			checkGroup,
			presetGroup,
			routingGroup,
			systemProxyGroup,
			toolsPrimaryGroup,
			toolsSecondaryGroup,
			testInputGroup,
			diagnosticsGroup,
			testResultGroup,
		},
	}
}

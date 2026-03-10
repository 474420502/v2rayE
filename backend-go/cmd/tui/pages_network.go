package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildNetworkPage() builtPage {
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

	checkBtn := a.actionButton(a.t("network.btn.check"), a.reloadOverviewAction)
	applyProxy := a.actionButton(a.t("network.btn.applyProxy"), a.applySystemProxyAction)
	clearProxy := a.actionButton(a.t("network.btn.clearProxy"), a.clearSystemProxyAction)
	saveRouting := a.actionButton(a.t("network.btn.saveRouting"), a.saveRoutingModeAction)
	geoUpdate := a.actionButton(a.t("network.btn.geoUpdate"), a.updateGeoDataAction)
	repairTun := a.actionButton(a.t("network.btn.repairTun"), a.repairTunAction)
	routeTest := a.actionButton(a.t("network.btn.routeTest"), a.routeTestAction)

	proxyActionsRow := buttonRow(applyProxy, clearProxy)
	operationsRow1 := buttonRow(saveRouting, geoUpdate)
	operationsRow2 := buttonRow(repairTun, routeTest)

	testRow := inputRow(a.networkTestTarget, a.networkTestPort, false, 4, 1)
	proxyActionsHeight := actionBlockHeight(false, 2)
	operationsHeight := actionBlockHeight(false, 2)
	testRowHeight := dualItemRowHeight(false)

	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanelHeight := 1 + 1 + 1
	actionsPanel.AddItem(newMutedText(a.t("network.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(buttonRow(checkBtn), 1, 0, false)

	presetsPanel := buildGroupPanel(
		a.t("network.group.presets"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.networkPresetSelect, height: 1},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: proxyActionsRow, height: proxyActionsHeight},
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
	leftColumn.AddItem(toolsPanel, 0, 3, false)

	rightColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	rightColumn.AddItem(diagnosticsPanel, 0, 7, false)
	rightColumn.AddItem(verticalSpacer(1), 1, 0, false)
	rightColumn.AddItem(testResultPanel, 0, 3, false)

	body := splitContent(false, leftColumn, rightColumn, 5, 6)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsPanelHeight, body)

	return builtPage{
		root: root,
		focusables: joinFocusables(
			buttonsToFocusables(checkBtn),
			primitivesToFocusables(a.networkPresetSelect),
			buttonsToFocusables(applyProxy, clearProxy),
			primitivesToFocusables(a.networkRoutingMode, a.networkDomainStrategy, a.networkLocalBypass),
			buttonsToFocusables(saveRouting, geoUpdate),
			buttonsToFocusables(repairTun, routeTest),
			primitivesToFocusables(a.networkTestTarget, a.networkTestPort),
		),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(checkBtn),
			primitivesToFocusables(a.networkPresetSelect),
			buttonsToFocusables(applyProxy, clearProxy),
			primitivesToFocusables(a.networkRoutingMode, a.networkDomainStrategy, a.networkLocalBypass),
			buttonsToFocusables(saveRouting, geoUpdate),
			buttonsToFocusables(repairTun, routeTest),
			primitivesToFocusables(a.networkTestTarget, a.networkTestPort),
		},
	}
}

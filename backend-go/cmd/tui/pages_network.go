package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildNetworkPage() builtPage {
	checkBtn := a.actionButton(a.t("network.btn.check"), a.reloadOverviewAction)
	globalPreset := a.actionButton(a.t("network.btn.presetGlobal"), a.presetGlobalProxyAction)
	bypassPreset := a.actionButton(a.t("network.btn.presetBypass"), a.presetBypassCNProxyAction)
	directPreset := a.actionButton(a.t("network.btn.presetDirect"), a.presetDirectNoProxyAction)
	applyProxy := a.actionButton(a.t("network.btn.applyProxy"), a.applySystemProxyAction)
	clearProxy := a.actionButton(a.t("network.btn.clearProxy"), a.clearSystemProxyAction)
	saveRouting := a.actionButton(a.t("network.btn.saveRouting"), a.saveRoutingModeAction)
	geoUpdate := a.actionButton(a.t("network.btn.geoUpdate"), a.updateGeoDataAction)
	repairTun := a.actionButton(a.t("network.btn.repairTun"), a.repairTunAction)
	routeTest := a.actionButton(a.t("network.btn.routeTest"), a.routeTestAction)
	selectGlobal := a.actionButton(a.t("network.btn.global"), a.selectRoutingGlobalAction)
	selectBypass := a.actionButton(a.t("network.btn.bypassCN"), a.selectRoutingBypassCNAction)
	selectDirect := a.actionButton(a.t("network.btn.direct"), a.selectRoutingDirectAction)
	selectCustom := a.actionButton(a.t("network.btn.custom"), a.selectRoutingCustomAction)

	primaryActions := buttonRow(checkBtn, globalPreset, bypassPreset, directPreset, applyProxy, clearProxy)
	secondaryActions := buttonRow(saveRouting, geoUpdate, repairTun, routeTest)
	modeActions := buttonRow(selectGlobal, selectBypass, selectDirect, selectCustom)
	if a.useStackedLayout() {
		primaryActions = buttonColumn(checkBtn, globalPreset, bypassPreset, directPreset, applyProxy, clearProxy)
		secondaryActions = buttonColumn(saveRouting, geoUpdate, repairTun, routeTest)
		modeActions = buttonColumn(selectGlobal, selectBypass, selectDirect, selectCustom)
	}
	testRow := inputRow(a.networkTestTarget, a.networkTestPort, a.useStackedLayout(), 4, 1)

	controls := tview.NewFlex().SetDirection(tview.FlexRow)
	primaryActionsHeight := actionBlockHeight(a.useStackedLayout(), 6)
	modeActionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	secondaryActionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	testRowHeight := dualItemRowHeight(a.useStackedLayout())
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanelHeight := 1 + 1 + primaryActionsHeight + 1 + modeActionsHeight + 1 + 1 + 1
	actionsPanel.AddItem(newMutedText(a.t("network.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(primaryActions, primaryActionsHeight, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(newMutedText(a.t("network.targetMode")), 1, 0, false)
	actionsPanel.AddItem(a.networkRoutingMode, 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(modeActions, modeActionsHeight, 0, false)

	controls.AddItem(a.networkDomainStrategy, 1, 0, false)
	controls.AddItem(a.networkLocalBypass, 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(secondaryActions, secondaryActionsHeight, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(testRow, testRowHeight, 0, false)

	results := tview.NewFlex().SetDirection(tview.FlexRow)
	results.AddItem(wrapPanel(a.t("network.panel.diagnostics"), a.networkSummary), 0, 3, false)
	results.AddItem(verticalSpacer(1), 1, 0, false)
	results.AddItem(wrapPanel(a.t("network.panel.testResult"), a.networkTestResult), 0, 2, false)

	leftWeight := 4
	rightWeight := 7
	if !a.useStackedLayout() {
		leftWeight = 5
		rightWeight = 6
	}
	body := splitContent(
		a.useStackedLayout(),
		wrapPanel(a.t("network.panel.form"), controls),
		results,
		leftWeight,
		rightWeight,
	)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsPanelHeight, body)
	return builtPage{
		root: root,
		focusables: joinFocusables(
			buttonsToFocusables(checkBtn, globalPreset, bypassPreset, directPreset, applyProxy, clearProxy),
			primitivesToFocusables(a.networkRoutingMode),
			buttonsToFocusables(selectGlobal, selectBypass, selectDirect, selectCustom),
			primitivesToFocusables(a.networkDomainStrategy, a.networkLocalBypass),
			buttonsToFocusables(saveRouting, geoUpdate, repairTun, routeTest),
			primitivesToFocusables(a.networkTestTarget, a.networkTestPort),
		),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(checkBtn, globalPreset, bypassPreset, directPreset, applyProxy, clearProxy),
			primitivesToFocusables(a.networkRoutingMode),
			buttonsToFocusables(selectGlobal, selectBypass, selectDirect, selectCustom),
			primitivesToFocusables(a.networkDomainStrategy, a.networkLocalBypass),
			buttonsToFocusables(saveRouting, geoUpdate, repairTun, routeTest),
			primitivesToFocusables(a.networkTestTarget, a.networkTestPort),
		},
	}
}

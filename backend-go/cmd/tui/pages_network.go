package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildNetworkPage() builtPage {
	checkBtn := a.actionButton("Check Network", a.reloadOverviewAction)
	globalPreset := a.actionButton("Preset Global", a.presetGlobalProxyAction)
	bypassPreset := a.actionButton("Preset BypassCN", a.presetBypassCNProxyAction)
	directPreset := a.actionButton("Preset Direct", a.presetDirectNoProxyAction)
	applyProxy := a.actionButton("Apply Proxy", a.applySystemProxyAction)
	clearProxy := a.actionButton("Clear Proxy", a.clearSystemProxyAction)
	saveRouting := a.actionButton("Save Routing", a.saveRoutingModeAction)
	geoUpdate := a.actionButton("Geo Update", a.updateGeoDataAction)
	repairTun := a.actionButton("Repair TUN", a.repairTunAction)
	routeTest := a.actionButton("Route Test", a.routeTestAction)
	selectGlobal := a.actionButton("Global", a.selectRoutingGlobalAction)
	selectBypass := a.actionButton("Bypass CN", a.selectRoutingBypassCNAction)
	selectDirect := a.actionButton("Direct", a.selectRoutingDirectAction)
	selectCustom := a.actionButton("Custom", a.selectRoutingCustomAction)

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
	actionsPanelHeight := 1 + 1 + primaryActionsHeight + 1 + modeActionsHeight
	actionsPanel.AddItem(newMutedText("Routing/Proxy presets"), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(primaryActions, primaryActionsHeight, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(modeActions, modeActionsHeight, 0, false)

	controls.AddItem(newMutedText("Edit routing fields then save/apply test"), 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(a.networkRoutingMode, 1, 0, false)
	controls.AddItem(a.networkDomainStrategy, 1, 0, false)
	controls.AddItem(a.networkLocalBypass, 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(secondaryActions, secondaryActionsHeight, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(testRow, testRowHeight, 0, false)

	results := tview.NewFlex().SetDirection(tview.FlexRow)
	results.AddItem(wrapPanel("Routing Diagnostics", a.networkSummary), 0, 3, false)
	results.AddItem(verticalSpacer(1), 1, 0, false)
	results.AddItem(wrapPanel("Route Test Result", a.networkTestResult), 0, 2, false)

	body := splitContent(
		a.useStackedLayout(),
		wrapPanel("Control Center", controls),
		results,
		4,
		7,
	)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(wrapPanel("Actions", actionsPanel), panelHeight(actionsPanelHeight), 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(body, 0, 1, false)
	return builtPage{
		root: root,
		focusables: joinFocusables(
			buttonsToFocusables(checkBtn, globalPreset, bypassPreset, directPreset, applyProxy, clearProxy, saveRouting, geoUpdate, repairTun, routeTest, selectGlobal, selectBypass, selectDirect, selectCustom),
			primitivesToFocusables(a.networkRoutingMode, a.networkDomainStrategy, a.networkLocalBypass, a.networkTestTarget, a.networkTestPort),
		),
	}
}

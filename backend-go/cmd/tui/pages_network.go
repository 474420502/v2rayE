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
	body := splitContent(
		a.useStackedLayout(),
		wrapPanel("Routing Diagnostics", a.networkSummary),
		wrapPanel("Route Test Result", a.networkTestResult),
		3,
		2,
	)
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(newMutedText("Select or type target routing policy fields, then Save Routing"), 1, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(primaryActions, 3, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(modeActions, 3, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(a.networkRoutingMode, 1, 0, false)
	root.AddItem(a.networkDomainStrategy, 1, 0, false)
	root.AddItem(a.networkLocalBypass, 1, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(secondaryActions, 3, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(testRow, 3, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(body, 0, 1, false)
	return builtPage{
		root: root,
		focusables: joinFocusables(
			buttonsToFocusables(checkBtn, globalPreset, bypassPreset, directPreset, applyProxy, clearProxy, saveRouting, geoUpdate, repairTun, routeTest, selectGlobal, selectBypass, selectDirect, selectCustom),
			primitivesToFocusables(a.networkRoutingMode, a.networkDomainStrategy, a.networkLocalBypass, a.networkTestTarget, a.networkTestPort, a.networkSummary, a.networkTestResult),
		),
	}
}

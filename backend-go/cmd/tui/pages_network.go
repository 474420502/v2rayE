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

	// 分组布局：网络检查和预设放一行，代理设置单独一行
	networkCheckRow := buttonRow(checkBtn, globalPreset, bypassPreset, directPreset)
	proxyActionsRow := buttonRow(applyProxy, clearProxy)
	modeActions := buttonRow(selectGlobal, selectBypass, selectDirect, selectCustom)
	secondaryActions := buttonRow(saveRouting, geoUpdate, repairTun, routeTest)

	if a.useStackedLayout() {
		networkCheckRow = buttonColumn(checkBtn, globalPreset, bypassPreset, directPreset)
		proxyActionsRow = buttonColumn(applyProxy, clearProxy)
		modeActions = buttonColumn(selectGlobal, selectBypass, selectDirect, selectCustom)
		secondaryActions = buttonColumn(saveRouting, geoUpdate, repairTun, routeTest)
	}
	testRow := inputRow(a.networkTestTarget, a.networkTestPort, a.useStackedLayout(), 4, 1)

	controls := tview.NewFlex().SetDirection(tview.FlexRow)
	networkCheckHeight := actionBlockHeight(a.useStackedLayout(), 4)
	proxyActionsHeight := actionBlockHeight(a.useStackedLayout(), 2)
	modeActionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	secondaryActionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	testRowHeight := dualItemRowHeight(a.useStackedLayout())

	// 优化控制面板高度计算
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanelHeight := 1 + 1 + networkCheckHeight + 1 + proxyActionsHeight + 1 + modeActionsHeight + 1 + 1 + 1
	actionsPanel.AddItem(newMutedText(a.t("network.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(networkCheckRow, networkCheckHeight, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	// 添加代理应用按钮行
	actionsPanel.AddItem(proxyActionsRow, proxyActionsHeight, 0, false)
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

	// 优化焦点组，按功能分组
	return builtPage{
		root: root,
		focusables: joinFocusables(
			buttonsToFocusables(checkBtn, globalPreset, bypassPreset, directPreset),
			buttonsToFocusables(applyProxy, clearProxy),
			primitivesToFocusables(a.networkRoutingMode),
			buttonsToFocusables(selectGlobal, selectBypass, selectDirect, selectCustom),
			primitivesToFocusables(a.networkDomainStrategy, a.networkLocalBypass),
			buttonsToFocusables(saveRouting, geoUpdate, repairTun, routeTest),
			primitivesToFocusables(a.networkTestTarget, a.networkTestPort),
		),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(checkBtn, globalPreset, bypassPreset, directPreset),
			buttonsToFocusables(applyProxy, clearProxy),
			primitivesToFocusables(a.networkRoutingMode),
			buttonsToFocusables(selectGlobal, selectBypass, selectDirect, selectCustom),
			primitivesToFocusables(a.networkDomainStrategy, a.networkLocalBypass),
			buttonsToFocusables(saveRouting, geoUpdate, repairTun, routeTest),
			primitivesToFocusables(a.networkTestTarget, a.networkTestPort),
		},
	}
}

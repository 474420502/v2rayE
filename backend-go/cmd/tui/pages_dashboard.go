package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildDashboardPage() builtPage {
	stackedActions := a.stackActionButtons()
	start := a.actionButton(a.t("dashboard.btn.start"), a.startCoreAction)
	stop := a.actionButton(a.t("dashboard.btn.stop"), a.stopCoreAction)
	restart := a.actionButton(a.t("dashboard.btn.restart"), a.restartCoreAction)
	refresh := a.actionButton(a.t("dashboard.btn.refresh"), a.refreshAllAction)

	actions := buttonStrip(stackedActions, start, stop, restart, refresh)

	statusCard := wrapPanel(a.t("dashboard.panel.status"), a.dashboardStatus)
	telemetryCard := wrapPanel(a.t("dashboard.panel.telemetry"), a.dashboardTelemetry)
	configCard := wrapPanel(a.t("dashboard.panel.config"), a.dashboardConfig)
	eventsCard := wrapPanel(a.t("dashboard.panel.events"), a.dashboardEvents)

	var body tview.Primitive
	if a.stackPageColumns() {
		stacked := tview.NewFlex().SetDirection(tview.FlexRow)
		stacked.AddItem(statusCard, 0, 3, false)
		stacked.AddItem(verticalSpacer(1), 1, 0, false)
		stacked.AddItem(telemetryCard, 0, 4, false)
		stacked.AddItem(verticalSpacer(1), 1, 0, false)
		stacked.AddItem(configCard, 0, 3, false)
		stacked.AddItem(verticalSpacer(1), 1, 0, false)
		stacked.AddItem(eventsCard, 0, 5, false)
		body = stacked
	} else {
		leftColumn := tview.NewFlex().SetDirection(tview.FlexRow)
		leftColumn.AddItem(statusCard, 0, 4, false)
		leftColumn.AddItem(verticalSpacer(1), 1, 0, false)
		leftColumn.AddItem(configCard, 0, 3, false)

		rightColumn := tview.NewFlex().SetDirection(tview.FlexRow)
		rightColumn.AddItem(telemetryCard, 0, 4, false)
		rightColumn.AddItem(verticalSpacer(1), 1, 0, false)
		rightColumn.AddItem(eventsCard, 0, 5, false)
		body = splitContent(false, leftColumn, rightColumn, 4, 5)
	}

	actionsHeight := actionBlockHeight(stackedActions, 4)
	actionsContentHeight := 1 + 1 + actionsHeight
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText(a.t("dashboard.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsContentHeight, body)
	actionGroup := buttonsToFocusables(start, stop, restart, refresh)
	statusGroup := primitivesToFocusables(a.dashboardStatus)
	telemetryGroup := primitivesToFocusables(a.dashboardTelemetry)
	configGroup := primitivesToFocusables(a.dashboardConfig)
	eventsGroup := primitivesToFocusables(a.dashboardEvents)

	return builtPage{
		root:       root,
		focusables: joinFocusables(actionGroup, statusGroup, telemetryGroup, configGroup, eventsGroup),
		focusGroups: [][]tview.Primitive{
			actionGroup,
			statusGroup,
			telemetryGroup,
			configGroup,
			eventsGroup,
		},
	}
}

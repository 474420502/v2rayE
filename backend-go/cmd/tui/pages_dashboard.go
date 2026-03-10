package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildDashboardPage() builtPage {
	start := a.actionButton(a.t("dashboard.btn.start"), a.startCoreAction)
	stop := a.actionButton(a.t("dashboard.btn.stop"), a.stopCoreAction)
	restart := a.actionButton(a.t("dashboard.btn.restart"), a.restartCoreAction)
	refresh := a.actionButton(a.t("dashboard.btn.refresh"), a.refreshAllAction)

	actions := buttonRow(start, stop, restart, refresh)
	if a.useStackedLayout() {
		actions = buttonColumn(start, stop, restart, refresh)
	}

	statusCard := wrapPanel(a.t("dashboard.panel.status"), a.dashboardStatus)
	telemetryCard := wrapPanel(a.t("dashboard.panel.telemetry"), a.dashboardTelemetry)
	configCard := wrapPanel(a.t("dashboard.panel.config"), a.dashboardConfig)
	eventsCard := wrapPanel(a.t("dashboard.panel.events"), a.dashboardEvents)

	var mainContent tview.Primitive
	if a.useStackedLayout() {
		stack := tview.NewFlex().SetDirection(tview.FlexRow)
		stack.AddItem(statusCard, 0, 3, false)
		stack.AddItem(verticalSpacer(1), 1, 0, false)
		stack.AddItem(telemetryCard, 0, 3, false)
		stack.AddItem(verticalSpacer(1), 1, 0, false)
		stack.AddItem(configCard, 0, 2, false)
		stack.AddItem(verticalSpacer(1), 1, 0, false)
		stack.AddItem(eventsCard, 0, 4, false)
		mainContent = stack
	} else {
		grid := tview.NewGrid().SetBorders(false).SetGap(1, 1)
		grid.SetRows(0, 0).SetColumns(0, 0, 0)
		grid.AddItem(statusCard, 0, 0, 1, 1, 0, 0, false)
		grid.AddItem(telemetryCard, 0, 1, 1, 1, 0, 0, false)
		grid.AddItem(configCard, 0, 2, 1, 1, 0, 0, false)
		grid.AddItem(eventsCard, 1, 0, 1, 3, 0, 0, false)
		mainContent = grid
	}

	actionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	actionsContentHeight := 1 + 1 + actionsHeight
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText(a.t("dashboard.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsContentHeight, mainContent)

	return builtPage{
		root:       root,
		focusables: buttonsToFocusables(start, stop, restart, refresh),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(start, stop, restart, refresh),
			primitivesToFocusables(a.dashboardStatus),
			primitivesToFocusables(a.dashboardTelemetry),
			primitivesToFocusables(a.dashboardConfig),
			primitivesToFocusables(a.dashboardEvents),
		},
	}
}

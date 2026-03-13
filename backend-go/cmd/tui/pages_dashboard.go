package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildDashboardPage() builtPage {
	start := a.actionButton(a.t("dashboard.btn.start"), a.startCoreAction)
	stop := a.actionButton(a.t("dashboard.btn.stop"), a.stopCoreAction)
	restart := a.actionButton(a.t("dashboard.btn.restart"), a.restartCoreAction)
	refresh := a.actionButton(a.t("dashboard.btn.refresh"), a.refreshAllAction)

	actions := buttonRow(start, stop, restart, refresh)

	statusCard := wrapPanel(a.t("dashboard.panel.status"), a.dashboardStatus)
	telemetryCard := wrapPanel(a.t("dashboard.panel.telemetry"), a.dashboardTelemetry)
	configCard := wrapPanel(a.t("dashboard.panel.config"), a.dashboardConfig)
	eventsCard := wrapPanel(a.t("dashboard.panel.events"), a.dashboardEvents)

	grid := tview.NewGrid().SetBorders(false).SetGap(1, 1)
	grid.SetRows(0, 0).SetColumns(0, 0, 0, 0)
	grid.AddItem(statusCard, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(telemetryCard, 0, 1, 1, 2, 0, 0, false)
	grid.AddItem(configCard, 0, 3, 1, 1, 0, 0, false)
	grid.AddItem(eventsCard, 1, 0, 1, 4, 0, 0, false)

	actionsHeight := actionBlockHeight(false, 4)
	actionsContentHeight := 1 + 1 + actionsHeight
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText(a.t("dashboard.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsContentHeight, grid)

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

package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildDashboardPage() builtPage {
	start := a.actionButton("Start", a.startCoreAction)
	stop := a.actionButton("Stop", a.stopCoreAction)
	restart := a.actionButton("Restart", a.restartCoreAction)
	refresh := a.actionButton("Refresh", a.refreshAllAction)

	actions := buttonRow(start, stop, restart, refresh)
	if a.useStackedLayout() {
		actions = buttonColumn(start, stop, restart, refresh)
	}

	mainContent := splitContent(
		a.useStackedLayout(),
		wrapPanel("Runtime Summary", a.dashboardSummary),
		wrapPanel("Recent Events", a.dashboardEvents),
		3,
		2,
	)

	actionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	actionsContentHeight := 1 + 1 + actionsHeight
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText("Core lifecycle and runtime telemetry overview"), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root := buildPageLayout("Actions", actionsPanel, actionsContentHeight, mainContent)

	return builtPage{
		root:       root,
		focusables: buttonsToFocusables(start, stop, restart, refresh),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(start, stop, restart, refresh),
			primitivesToFocusables(a.dashboardSummary),
			primitivesToFocusables(a.dashboardEvents),
		},
	}
}

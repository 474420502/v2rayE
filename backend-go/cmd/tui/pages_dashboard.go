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

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText("Core lifecycle and runtime telemetry overview"), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root.AddItem(wrapPanel("Actions", actionsPanel), panelHeight(1+1+actionsHeight), 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(mainContent, 0, 1, false)

	return builtPage{
		root:       root,
		focusables: buttonsToFocusables(start, stop, restart, refresh),
	}
}

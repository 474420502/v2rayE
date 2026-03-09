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

	mainContent := splitContent(
		a.useStackedLayout(),
		wrapPanel(a.t("dashboard.panel.summary"), a.dashboardSummary),
		wrapPanel(a.t("dashboard.panel.events"), a.dashboardEvents),
		3,
		2,
	)

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
			primitivesToFocusables(a.dashboardSummary),
			primitivesToFocusables(a.dashboardEvents),
		},
	}
}

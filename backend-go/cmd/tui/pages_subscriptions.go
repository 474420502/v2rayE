package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildSubscriptionsPage() builtPage {
	updateAll := a.actionButton(a.t("subs.btn.updateAll"), a.updateAllSubscriptionsAction)
	updateSelected := a.actionButton(a.t("subs.btn.updateSelected"), a.updateSelectedSubscriptionAction)
	actions := buttonRow(updateAll, updateSelected)
	if a.useStackedLayout() {
		actions = buttonColumn(updateAll, updateSelected)
	}
	body := splitContent(
		a.useStackedLayout(),
		wrapPanel(a.t("subs.panel.list"), a.subscriptionsList),
		wrapPanel(a.t("subs.panel.selected"), a.subscriptionDetail),
		5,
		6,
	)
	actionsHeight := actionBlockHeight(a.useStackedLayout(), 2)
	actionsContentHeight := 1 + 1 + actionsHeight
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText(a.t("subs.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsContentHeight, body)
	return builtPage{
		root:       root,
		focusables: joinFocusables(buttonsToFocusables(updateAll, updateSelected), primitivesToFocusables(a.subscriptionsList)),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(updateAll, updateSelected),
			primitivesToFocusables(a.subscriptionsList),
		},
	}
}

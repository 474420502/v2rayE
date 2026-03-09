package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildSubscriptionsPage() builtPage {
	updateAll := a.actionButton("Update All", a.updateAllSubscriptionsAction)
	updateSelected := a.actionButton("Update Selected", a.updateSelectedSubscriptionAction)
	actions := buttonRow(updateAll, updateSelected)
	if a.useStackedLayout() {
		actions = buttonColumn(updateAll, updateSelected)
	}
	body := splitContent(
		a.useStackedLayout(),
		wrapPanel("Subscriptions", a.subscriptionsList),
		wrapPanel("Selected Subscription", a.subscriptionDetail),
		5,
		6,
	)
	actionsHeight := actionBlockHeight(a.useStackedLayout(), 2)
	actionsContentHeight := 1 + 1 + actionsHeight
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText("Update all or only the selected subscription source"), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root := buildPageLayout("Actions", actionsPanel, actionsContentHeight, body)
	return builtPage{
		root:       root,
		focusables: joinFocusables(buttonsToFocusables(updateAll, updateSelected), primitivesToFocusables(a.subscriptionsList)),
	}
}

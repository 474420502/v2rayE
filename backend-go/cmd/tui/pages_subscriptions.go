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
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(newMutedText("Update all or only the selected subscription source"), 1, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(actions, 3, 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(body, 0, 1, false)
	return builtPage{
		root:       root,
		focusables: joinFocusables(buttonsToFocusables(updateAll, updateSelected), primitivesToFocusables(a.subscriptionsList)),
	}
}

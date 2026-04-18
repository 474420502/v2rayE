package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildSubscriptionsPage() builtPage {
	updateAll := a.actionButton(a.t("subs.btn.updateAll"), a.updateAllSubscriptionsAction)
	updateSelected := a.actionButton(a.t("subs.btn.updateSelected"), a.updateSelectedSubscriptionAction)
	actions := buttonRow(updateAll, updateSelected)
	body := splitContent(
		false,
		wrapPanel(a.t("subs.panel.list"), a.subscriptionsList),
		wrapPanel(a.t("subs.panel.selected"), a.subscriptionDetail),
		5,
		6,
	)
	actionsHeight := actionBlockHeight(false, 2)
	actionsContentHeight := 1 + 1 + actionsHeight
	actionsPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	actionsPanel.AddItem(newMutedText(a.t("subs.desc")), 1, 0, false)
	actionsPanel.AddItem(verticalSpacer(1), 1, 0, false)
	actionsPanel.AddItem(actions, actionsHeight, 0, false)
	root := buildPageLayout(a.t("common.actions"), actionsPanel, actionsContentHeight, body)
	actionGroup := buttonsToFocusables(updateAll, updateSelected)
	listGroup := primitivesToFocusables(a.subscriptionsList)
	detailGroup := primitivesToFocusables(a.subscriptionDetail)
	return builtPage{
		root:       root,
		focusables: joinFocusables(actionGroup, listGroup, detailGroup),
		focusGroups: [][]tview.Primitive{
			actionGroup,
			listGroup,
			detailGroup,
		},
	}
}

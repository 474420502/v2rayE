package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildProfilesPage() builtPage {
	stackedActions := a.stackActionButtons()
	importURLBtn := a.actionButton(a.t("profiles.btn.importURL"), a.openImportProfileDialogAction)
	importClipboardBtn := a.actionButton(a.t("profiles.btn.importClipboard"), a.importProfileFromClipboardAction)
	batchDelayBtn := a.actionButton(a.t("profiles.btn.batchDelay"), a.batchDelayProfilesAction)

	controlsActions := buttonStrip(stackedActions, importURLBtn, importClipboardBtn, batchDelayBtn)

	workflowPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	workflowPanel.AddItem(a.profileEditStatus, 1, 0, false)
	workflowPanel.AddItem(verticalSpacer(1), 1, 0, false)
	workflowPanel.AddItem(newMutedText(a.t("profiles.editor.hint1")), 1, 0, false)
	workflowPanel.AddItem(newMutedText(a.t("profiles.editor.hint2")), 1, 0, false)
	workflowPanel.AddItem(verticalSpacer(1), 1, 0, false)
	workflowPanel.AddItem(a.profileBatchStatus, 1, 0, false)

	profilesPanel := wrapPanel(a.t("profiles.panel.list"), a.profilesList)
	selectedPanel := wrapPanel(a.t("profiles.panel.selected"), a.profileDetail)
	workflowCard := wrapPanel(a.t("profiles.panel.workflow"), workflowPanel)
	detailColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	detailColumn.AddItem(selectedPanel, 0, 5, false)
	detailColumn.AddItem(verticalSpacer(1), 1, 0, false)
	detailColumn.AddItem(workflowCard, 0, 4, false)
	body := splitContent(a.stackPageColumns(), profilesPanel, detailColumn, 5, 4)

	controls := tview.NewFlex().SetDirection(tview.FlexRow)
	controlsHeight := actionBlockHeight(stackedActions, 3)
	controlsContentHeight := 1 + 1 + controlsHeight
	controls.AddItem(newMutedText(a.t("profiles.controls.desc")), 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(controlsActions, controlsHeight, 0, false)

	root := buildPageLayout(a.t("profiles.panel.controls"), controls, controlsContentHeight, body)
	actionGroup := buttonsToFocusables(importURLBtn, importClipboardBtn, batchDelayBtn)
	listGroup := primitivesToFocusables(a.profilesList)
	detailGroup := primitivesToFocusables(a.profileDetail)
	workflowGroup := primitivesToFocusables(a.profileEditStatus, a.profileBatchStatus)

	return builtPage{
		root:       root,
		focusables: joinFocusables(actionGroup, listGroup, detailGroup, workflowGroup),
		focusGroups: [][]tview.Primitive{
			actionGroup,
			listGroup,
			detailGroup,
			workflowGroup,
		},
	}
}

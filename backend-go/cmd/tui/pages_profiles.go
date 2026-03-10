package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildProfilesPage() builtPage {
	importURLBtn := a.actionButton(a.t("profiles.btn.importURL"), a.openImportProfileDialogAction)
	importClipboardBtn := a.actionButton(a.t("profiles.btn.importClipboard"), a.importProfileFromClipboardAction)
	batchDelayBtn := a.actionButton(a.t("profiles.btn.batchDelay"), a.batchDelayProfilesAction)

	controlsActions := buttonRow(importURLBtn, importClipboardBtn, batchDelayBtn)
	if a.useStackedLayout() {
		controlsActions = buttonColumn(importURLBtn, importClipboardBtn, batchDelayBtn)
	}

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

	var workspace tview.Primitive
	if a.useStackedLayout() {
		stack := tview.NewFlex().SetDirection(tview.FlexRow)
		stack.AddItem(profilesPanel, 0, 5, false)
		stack.AddItem(verticalSpacer(1), 1, 0, false)
		stack.AddItem(selectedPanel, 0, 4, false)
		stack.AddItem(verticalSpacer(1), 1, 0, false)
		stack.AddItem(workflowCard, 0, 2, false)
		workspace = stack
	} else {
		grid := tview.NewGrid().SetBorders(false).SetGap(1, 1)
		grid.SetRows(0, 0).SetColumns(0, 0)
		grid.AddItem(profilesPanel, 0, 0, 2, 1, 0, 0, false)
		grid.AddItem(selectedPanel, 0, 1, 1, 1, 0, 0, false)
		grid.AddItem(workflowCard, 1, 1, 1, 1, 0, 0, false)
		workspace = grid
	}

	controls := tview.NewFlex().SetDirection(tview.FlexRow)
	controlsHeight := actionBlockHeight(a.useStackedLayout(), 3)
	controlsContentHeight := 1 + 1 + controlsHeight
	controls.AddItem(newMutedText(a.t("profiles.controls.desc")), 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(controlsActions, controlsHeight, 0, false)

	root := buildPageLayout(a.t("profiles.panel.controls"), controls, controlsContentHeight, workspace)

	return builtPage{
		root:       root,
		focusables: joinFocusables(buttonsToFocusables(importURLBtn, importClipboardBtn, batchDelayBtn), primitivesToFocusables(a.profilesList)),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(importURLBtn, importClipboardBtn, batchDelayBtn),
			primitivesToFocusables(a.profilesList),
		},
	}
}

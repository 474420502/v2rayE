package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildProfilesPage() builtPage {
	importBtn := a.actionButton("Import URI", a.importProfileAction)
	importLoadBtn := a.actionButton("Import+Load", a.importAndLoadProfileAction)
	activateBtn := a.actionButton("Activate", a.activateProfileAction)
	batchBtn := a.actionButton("Batch Delay", a.batchDelayProfilesAction)
	delayBtn := a.actionButton("Delay Test", a.testSelectedProfileDelayAction)
	deleteBtn := a.actionButton("Delete Selected", a.deleteSelectedProfileAction)
	loadBtn := a.actionButton("Load Selected", a.loadSelectedProfileEditorAction)
	resetBtn := a.actionButton("Reset Edit", a.resetProfileEditAction)
	saveBtn := a.actionButton("Save Edit", a.saveSelectedProfileEditAction)

	importRow := inputRow(a.profileImport, buttonRow(importBtn, importLoadBtn), a.useStackedLayout(), 5, 4)
	actions := buttonRow(activateBtn, batchBtn, delayBtn, deleteBtn)
	editActions := buttonRow(loadBtn, resetBtn, saveBtn)
	if a.useStackedLayout() {
		actions = buttonColumn(activateBtn, batchBtn, delayBtn, deleteBtn)
		editActions = buttonColumn(loadBtn, resetBtn, saveBtn)
	}

	editorForm := tview.NewFlex().SetDirection(tview.FlexRow)
	editorForm.AddItem(a.profileEditStatus, 1, 0, false)
	editorForm.AddItem(verticalSpacer(1), 1, 0, false)
	for _, field := range []*inputWidget{
		a.profileEditName,
		a.profileEditAddress,
		a.profileEditPort,
		a.profileEditNetwork,
		a.profileEditTLS,
		a.profileEditSNI,
		a.profileEditFingerprint,
		a.profileEditALPN,
		a.profileEditRealityPK,
		a.profileEditRealitySID,
		a.profileEditWSPath,
		a.profileEditGRPCSvc,
	} {
		editorForm.AddItem(field, 1, 0, false)
	}

	right := splitContent(
		a.useStackedLayout(),
		wrapPanel("Selected Profile", a.profileDetail),
		wrapPanel("Profile Editor", editorForm),
		4,
		5,
	)
	workspace := splitContent(
		a.useStackedLayout(),
		wrapPanel("Profiles", a.profilesList),
		right,
		4,
		7,
	)

	controls := tview.NewFlex().SetDirection(tview.FlexRow)
	importRowHeight := dualItemRowHeight(a.useStackedLayout())
	actionsHeight := actionBlockHeight(a.useStackedLayout(), 4)
	editActionsHeight := actionBlockHeight(a.useStackedLayout(), 3)
	controlsContentHeight := 1 + 1 + importRowHeight + 1 + 1 + 1 + actionsHeight + 1 + 1 + 1 + editActionsHeight
	controls.AddItem(newMutedText("Import / run tests / manage selected profile"), 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(importRow, importRowHeight, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(a.profileBatchStatus, 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(actions, actionsHeight, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(a.profileDeleteConfirm, 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(editActions, editActionsHeight, 0, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(wrapPanel("Controls", controls), panelHeight(controlsContentHeight), 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(workspace, 0, 1, false)

	return builtPage{
		root: root,
		focusables: joinFocusables(
			primitivesToFocusables(a.profileImport),
			buttonsToFocusables(importBtn, importLoadBtn, activateBtn, batchBtn, delayBtn, deleteBtn, loadBtn, resetBtn, saveBtn),
			primitivesToFocusables(a.profilesList, a.profileDeleteConfirm, a.profileEditName, a.profileEditAddress, a.profileEditPort, a.profileEditNetwork, a.profileEditTLS, a.profileEditSNI, a.profileEditFingerprint, a.profileEditALPN, a.profileEditRealityPK, a.profileEditRealitySID, a.profileEditWSPath, a.profileEditGRPCSvc),
		),
	}
}

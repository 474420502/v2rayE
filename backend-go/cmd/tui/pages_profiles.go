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

	editorHint := tview.NewFlex().SetDirection(tview.FlexRow)
	editorHint.AddItem(a.profileEditStatus, 1, 0, false)
	editorHint.AddItem(verticalSpacer(1), 1, 0, false)
	editorHint.AddItem(newMutedText(a.t("profiles.editor.hint1")), 1, 0, false)
	editorHint.AddItem(newMutedText(a.t("profiles.editor.hint2")), 1, 0, false)

	right := tview.NewFlex().SetDirection(tview.FlexRow)
	right.AddItem(wrapPanel(a.t("profiles.panel.selected"), a.profileDetail), 0, 4, false)
	right.AddItem(verticalSpacer(1), 1, 0, false)
	right.AddItem(wrapPanel(a.t("profiles.panel.editor"), editorHint), 0, 1, false)
	workspace := splitContent(
		a.useStackedLayout(),
		wrapPanel(a.t("profiles.panel.list"), a.profilesList),
		right,
		4,
		7,
	)

	controls := tview.NewFlex().SetDirection(tview.FlexRow)
	controlsHeight := actionBlockHeight(a.useStackedLayout(), 3)
	controlsContentHeight := 1 + 1 + controlsHeight + 1 + 1 + 1
	controls.AddItem(newMutedText(a.t("profiles.controls.desc")), 1, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(controlsActions, controlsHeight, 0, false)
	controls.AddItem(verticalSpacer(1), 1, 0, false)
	controls.AddItem(a.profileBatchStatus, 1, 0, false)

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

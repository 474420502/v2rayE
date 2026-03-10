package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const profileImportDialogPage = "profile-import-dialog"
const profileQuickEditDialogPage = "profile-quick-edit-dialog"
const profileQuickEditDiffPage = "profile-quick-edit-diff"
const profileActionsMenuPage = "profile-actions-menu"
const profileDeleteConfirmPage = "profile-delete-confirm"

func (a *tuiApp) profileQuickEditOverlay(content tview.Primitive) tview.Primitive {
	return centeredPrimitive(content, 120, 30)
}

func (a *tuiApp) openProfileDeleteConfirmDialogAction(context.Context) error {
	a.openProfileDeleteConfirmDialog()
	return nil
}

func (a *tuiApp) openProfileDeleteConfirmDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || a.profileDeleteVisible.Load() || a.commandPaletteVisible.Load() || a.profileImportVisible.Load() || a.profileEditVisible.Load() {
			return
		}
		selected := a.selectedProfile()
		if selected == nil {
			a.setFooter(a.t("footer.profile.delete.noSelection"))
			return
		}

		message := tview.NewTextView()
		message.SetTextAlign(tview.AlignCenter)
		displayName := emptyFallback(selected.Name, selected.ID)
		message.SetText(a.tf("dialog.profile.delete.body", displayName, strings.ToLower(strings.TrimSpace(selected.Protocol)), strings.TrimSpace(selected.Address), selected.Port, selected.ID))
		message.SetTextColor(tcell.ColorWhite)

		confirmBtn := tview.NewButton(a.t("dialog.common.delete"))
		cancelBtn := tview.NewButton(a.t("dialog.common.cancel"))
		for _, btn := range []*tview.Button{confirmBtn, cancelBtn} {
			btn.SetLabelColor(tcell.ColorWhite)
			btn.SetLabelColorActivated(tcell.ColorBlack)
			btn.SetBackgroundColor(tcell.ColorDarkCyan)
			btn.SetBackgroundColorActivated(tcell.ColorYellow)
		}

		confirmBtn.SetSelectedFunc(func() {
			a.closeProfileDeleteConfirmDialog()
			go a.runAction("delete profile", a.deleteSelectedProfileAction)
		})
		cancelBtn.SetSelectedFunc(func() {
			a.closeProfileDeleteConfirmDialog()
		})

		container := tview.NewFlex().SetDirection(tview.FlexRow)
		container.SetBorder(true)
		container.SetTitle(" " + a.t("dialog.profile.delete.title") + " ")
		container.AddItem(message, 0, 1, false)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(buttonRow(confirmBtn, cancelBtn), 1, 0, true)

		a.profileDeletePrev = app.GetFocus()
		a.pageHolder.AddPage(profileDeleteConfirmPage, centeredPrimitive(container, 72, 12), true, true)
		a.profileDeleteVisible.Store(true)
		app.SetFocus(confirmBtn)
		a.setFooter(a.t("footer.profile.delete.confirmHint"))
	})
}

func (a *tuiApp) closeProfileDeleteConfirmDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || !a.profileDeleteVisible.Load() {
			return
		}
		a.pageHolder.RemovePage(profileDeleteConfirmPage)
		a.profileDeleteVisible.Store(false)
		if a.profileDeletePrev != nil {
			app.SetFocus(a.profileDeletePrev)
		} else if len(a.focusables) > 0 {
			app.SetFocus(a.focusables[0])
		}
		a.profileDeletePrev = nil
		a.setFooter(a.tf("status.page", pageDisplayName(a.page)))
	})
}

func (a *tuiApp) openProfileActionsMenu() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || a.profileActionsVisible.Load() || a.profileDeleteVisible.Load() || a.commandPaletteVisible.Load() || a.profileImportVisible.Load() || a.profileEditVisible.Load() {
			return
		}
		selected := a.selectedProfile()

		a.profileActionsMenu = newListWidget()
		a.profileActionsMenu.ShowSecondaryText(true)
		a.profileActionsMenu.SetMainTextColor(editableValueColor)
		a.profileActionsMenu.SetSecondaryTextColor(tcell.ColorLightGray)
		a.profileActionsMenu.SetSelectedTextColor(tcell.ColorBlack)
		a.profileActionsMenu.SetSelectedBackgroundColor(tcell.ColorYellow)

		runAndClose := func(label string, action func(context.Context) error) func() {
			return func() {
				a.closeProfileActionsMenu()
				go a.runAction(label, action)
			}
		}

		if selected != nil {
			a.profileActionsMenu.AddItem(a.t("menu.profile.activate"), a.t("menu.profile.activate.desc"), 0, runAndClose("activate profile", a.activateProfileAction))
			a.profileActionsMenu.AddItem(a.t("menu.profile.delay"), a.t("menu.profile.delay.desc"), 0, runAndClose("delay test", a.testSelectedProfileDelayAction))
			a.profileActionsMenu.AddItem(a.t("menu.profile.edit"), a.t("menu.profile.edit.desc"), 0, func() {
				a.closeProfileActionsMenu()
				a.openProfileQuickEditDialog()
			})
			a.profileActionsMenu.AddItem(a.t("menu.profile.delete"), a.t("menu.profile.delete.desc"), 0, func() {
				a.closeProfileActionsMenu()
				a.openProfileDeleteConfirmDialog()
			})
		}
		a.profileActionsMenu.AddItem(a.t("dialog.common.close"), "Esc", 0, func() {
			a.closeProfileActionsMenu()
		})

		a.profileActionsPrev = app.GetFocus()

		container := tview.NewFlex().SetDirection(tview.FlexRow)
		container.SetBorder(true)
		container.SetTitle(" " + a.t("menu.profile.title") + " ")
		container.AddItem(newMutedText(a.t("menu.profile.hint")), 1, 0, false)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		titleText := a.t("state.profile.noSelection")
		if selected != nil {
			titleText = emptyFallback(selected.Name, selected.ID)
		}
		container.AddItem(newMutedText(titleText), 1, 0, false)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(a.profileActionsMenu, 0, 1, true)

		overlay := centeredPrimitive(container, 64, 14)
		a.pageHolder.AddPage(profileActionsMenuPage, overlay, true, true)
		a.profileActionsVisible.Store(true)
		app.SetFocus(a.profileActionsMenu)
		a.setFooter(a.t("footer.profile.actionsHint"))
	})
}

func (a *tuiApp) closeProfileActionsMenu() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || !a.profileActionsVisible.Load() {
			return
		}
		a.pageHolder.RemovePage(profileActionsMenuPage)
		a.profileActionsVisible.Store(false)
		a.profileActionsMenu = nil
		if a.profileActionsPrev != nil {
			app.SetFocus(a.profileActionsPrev)
		} else if len(a.focusables) > 0 {
			app.SetFocus(a.focusables[0])
		}
		a.profileActionsPrev = nil
		a.setFooter(a.tf("status.page", pageDisplayName(a.page)))
	})
}

func (a *tuiApp) openImportProfileDialogAction(context.Context) error {
	a.openProfileImportDialog()
	return nil
}

func (a *tuiApp) importProfileFromClipboardAction(ctx context.Context) error {
	uri, err := readClipboardText(ctx)
	if err != nil {
		return errors.New(a.t("error.profile.clipboardUnavailable"))
	}
	return a.importProfileURI(ctx, uri, false)
}

func (a *tuiApp) openProfileQuickEditDialogAction(context.Context) error {
	a.openProfileQuickEditDialog()
	return nil
}

func (a *tuiApp) openProfileQuickEditDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || a.profileEditVisible.Load() {
			return
		}
		selected := a.selectedProfile()
		if selected == nil {
			a.setFooter(a.t("footer.profile.edit.noSelection"))
			return
		}

		payload, err := json.MarshalIndent(selected, "", "  ")
		if err != nil {
			a.setFooter(a.tf("footer.profile.edit.failed", err.Error()))
			return
		}

		editor := tview.NewTextArea()
		editor.SetLabel(a.t("dialog.profile.editor.label"))
		editor.SetLabelStyle(tcell.StyleDefault.Foreground(editableLabelColor))
		editor.SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.ColorDarkGray))
		editor.SetText(string(payload), false)
		originalText := string(payload)

		status := newMutedText(a.t("dialog.profile.editor.statusHint"))

		parseEditor := func() (ProfileItem, error) {
			text := strings.TrimSpace(editor.GetText())
			if text == "" {
				return ProfileItem{}, errors.New(a.t("error.profile.emptyJSON"))
			}
			var updated ProfileItem
			if err := json.Unmarshal([]byte(text), &updated); err != nil {
				return ProfileItem{}, err
			}
			if strings.TrimSpace(updated.ID) == "" {
				updated.ID = selected.ID
			}
			if updated.ID != selected.ID {
				return ProfileItem{}, errors.New(a.t("error.profile.idImmutable"))
			}
			if strings.TrimSpace(updated.Protocol) == "" {
				return ProfileItem{}, errors.New(a.t("error.profile.protocolRequired"))
			}
			if strings.TrimSpace(updated.Address) == "" {
				return ProfileItem{}, errors.New(a.t("error.profile.addressRequired"))
			}
			if updated.Port <= 0 || updated.Port > 65535 {
				return ProfileItem{}, errors.New(a.t("error.profile.invalidPortRange"))
			}
			return updated, nil
		}

		formatJSON := func() {
			updated, parseErr := parseEditor()
			if parseErr != nil {
				status.SetText(a.tf("dialog.profile.editor.formatError", parseErr.Error()))
				return
			}
			pretty, marshalErr := json.MarshalIndent(updated, "", "  ")
			if marshalErr != nil {
				status.SetText(a.tf("dialog.profile.editor.formatError", marshalErr.Error()))
				return
			}
			editor.SetText(string(pretty), false)
			status.SetText(a.t("dialog.profile.editor.formatted"))
		}

		previewChanges := func() {
			updated, parseErr := parseEditor()
			if parseErr != nil {
				status.SetText(a.tf("dialog.profile.editor.previewError", parseErr.Error()))
				return
			}
			changes := collectChangedJSONPaths(profileToComparableMap(*selected), profileToComparableMap(updated), "")
			if len(changes) == 0 {
				status.SetText(a.t("dialog.profile.editor.noChanges"))
				return
			}
			sort.Strings(changes)
			if len(changes) > 180 {
				changes = append(changes[:180], a.tf("dialog.profile.editor.moreChanges", len(changes)-180))
			}
			a.openProfileQuickEditDiffDialog(changes)
		}

		resetEditor := func() {
			editor.SetText(originalText, false)
			status.SetText(a.t("dialog.profile.editor.resetDone"))
		}

		saveBtn := tview.NewButton(a.t("dialog.common.save"))
		previewBtn := tview.NewButton(a.t("dialog.common.preview"))
		resetBtn := tview.NewButton(a.t("dialog.common.reset"))
		formatBtn := tview.NewButton(a.t("dialog.common.format"))
		cancelBtn := tview.NewButton(a.t("dialog.common.cancel"))
		for _, btn := range []*tview.Button{saveBtn, previewBtn, resetBtn, formatBtn, cancelBtn} {
			btn.SetLabelColor(tcell.ColorWhite)
			btn.SetLabelColorActivated(tcell.ColorBlack)
			btn.SetBackgroundColor(tcell.ColorDarkCyan)
			btn.SetBackgroundColorActivated(tcell.ColorYellow)
		}

		submit := func() {
			updated, parseErr := parseEditor()
			if parseErr != nil {
				status.SetText(a.tf("dialog.profile.editor.saveError", parseErr.Error()))
				return
			}
			a.closeProfileQuickEditDialog()
			go a.runAction("save profile editor", func(ctx context.Context) error {
				result, err := a.client.UpdateProfile(ctx, selected.ID, updated)
				if err != nil {
					return err
				}
				a.storeSelectedProfileID(result.ID)
				a.clearProfileEditDirty()
				a.setProfileEditMessage("Profile editor: saved successfully.")
				a.pushEvent("profile updated: " + result.ID)
				if err := a.reloadProfiles(); err != nil {
					return err
				}
				return a.reloadOverview()
			})
		}

		saveBtn.SetSelectedFunc(submit)
		previewBtn.SetSelectedFunc(previewChanges)
		resetBtn.SetSelectedFunc(resetEditor)
		formatBtn.SetSelectedFunc(formatJSON)
		cancelBtn.SetSelectedFunc(func() { a.closeProfileQuickEditDialog() })
		editor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				a.closeProfileQuickEditDialog()
				return nil
			case tcell.KeyCtrlS:
				submit()
				return nil
			case tcell.KeyCtrlF:
				formatJSON()
				return nil
			case tcell.KeyCtrlR:
				resetEditor()
				return nil
			default:
				return event
			}
		})

		form := tview.NewFlex().SetDirection(tview.FlexRow)
		form.SetBorder(true)
		form.SetTitle(" " + a.t("dialog.profile.editor.title") + " ")
		form.AddItem(newMutedText(a.t("dialog.profile.editor.description")), 1, 0, false)
		form.AddItem(verticalSpacer(1), 1, 0, false)
		form.AddItem(editor, 0, 1, true)
		form.AddItem(verticalSpacer(1), 1, 0, false)
		form.AddItem(status, 1, 0, false)
		form.AddItem(verticalSpacer(1), 1, 0, false)
		form.AddItem(newMutedText(a.t("dialog.profile.editor.shortcuts")), 1, 0, false)
		form.AddItem(verticalSpacer(1), 1, 0, false)
		form.AddItem(buttonRow(previewBtn, resetBtn, formatBtn, saveBtn, cancelBtn), 1, 0, false)

		a.profileEditPrev = app.GetFocus()
		a.pageHolder.AddPage(profileQuickEditDialogPage, a.profileQuickEditOverlay(form), true, true)
		a.profileEditVisible.Store(true)
		app.SetFocus(editor)
		a.setFooter(a.t("footer.profile.editorHint"))
	})
}

func (a *tuiApp) openProfileQuickEditDiffDialog(changes []string) {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil {
			return
		}
		preview := readOnlyEditor(strings.Join(changes, "\n"))
		closeBtn := tview.NewButton(a.t("dialog.common.close"))
		closeBtn.SetLabelColor(tcell.ColorWhite)
		closeBtn.SetLabelColorActivated(tcell.ColorBlack)
		closeBtn.SetBackgroundColor(tcell.ColorDarkCyan)
		closeBtn.SetBackgroundColorActivated(tcell.ColorYellow)
		closeBtn.SetSelectedFunc(func() {
			a.runUI(func(app *tview.Application) {
				a.pageHolder.RemovePage(profileQuickEditDiffPage)
				if a.profileEditVisible.Load() {
					app.SetFocus(a.pageHolder)
				}
			})
		})
		box := tview.NewFlex().SetDirection(tview.FlexRow)
		box.SetBorder(true)
		box.SetTitle(" " + a.t("dialog.profile.diff.title") + " ")
		box.AddItem(newMutedText(a.t("dialog.profile.diff.description")), 1, 0, false)
		box.AddItem(verticalSpacer(1), 1, 0, false)
		box.AddItem(preview, 0, 1, false)
		box.AddItem(verticalSpacer(1), 1, 0, false)
		box.AddItem(buttonRow(closeBtn), 1, 0, true)
		a.pageHolder.RemovePage(profileQuickEditDiffPage)
		a.pageHolder.AddPage(profileQuickEditDiffPage, centeredPrimitive(box, 90, 22), true, true)
		app.SetFocus(closeBtn)
	})
}

func (a *tuiApp) closeProfileQuickEditDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || !a.profileEditVisible.Load() {
			return
		}
		a.pageHolder.RemovePage(profileQuickEditDialogPage)
		a.pageHolder.RemovePage(profileQuickEditDiffPage)
		a.profileEditVisible.Store(false)
		if a.profileEditPrev != nil {
			app.SetFocus(a.profileEditPrev)
		} else if len(a.focusables) > 0 {
			app.SetFocus(a.focusables[0])
		}
		a.profileEditPrev = nil
		a.setFooter(a.tf("status.page", pageDisplayName(a.page)))
	})
}

func profileToComparableMap(profile ProfileItem) map[string]any {
	payload, err := json.Marshal(profile)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func collectChangedJSONPaths(before, after any, path string) []string {
	if before == nil && after == nil {
		return nil
	}
	if before == nil || after == nil {
		return []string{normalizeChangePath(path)}
	}

	bm, bok := before.(map[string]any)
	am, aok := after.(map[string]any)
	if bok && aok {
		keysMap := map[string]struct{}{}
		for k := range bm {
			keysMap[k] = struct{}{}
		}
		for k := range am {
			keysMap[k] = struct{}{}
		}
		keys := make([]string, 0, len(keysMap))
		for k := range keysMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var out []string
		for _, k := range keys {
			next := k
			if path != "" {
				next = path + "." + k
			}
			out = append(out, collectChangedJSONPaths(bm[k], am[k], next)...)
		}
		return out
	}

	if !reflect.DeepEqual(before, after) {
		return []string{normalizeChangePath(path)}
	}
	return nil
}

func normalizeChangePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "<root>"
	}
	return path
}

func (a *tuiApp) openProfileImportDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || a.profileImportVisible.Load() {
			return
		}

		a.profileImportInput = tview.NewInputField()
		a.profileImportInput.SetLabel(a.t("dialog.profile.import.urlLabel"))
		a.profileImportInput.SetFieldWidth(0)
		a.profileImportInput.SetLabelColor(editableLabelColor)
		a.profileImportInput.SetFieldTextColor(editableValueColor)
		a.profileImportInput.SetFieldBackgroundColor(tcell.ColorBlack)
		a.profileImportInput.SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEsc:
				a.closeProfileImportDialog()
			case tcell.KeyEnter:
				a.submitProfileImport(false)
			}
		})

		pasteBtn := tview.NewButton(a.t("dialog.common.paste"))
		pasteBtn.SetSelectedFunc(func() {
			ctx, cancel := context.WithTimeout(a.ctx, 2*time.Second)
			defer cancel()
			uri, err := readClipboardText(ctx)
			if err != nil {
				a.setFooter(a.tf("footer.profile.paste.failed", a.t("error.profile.clipboardUnavailable")))
				return
			}
			a.runUI(func(app *tview.Application) {
				if a.profileImportInput != nil {
					a.profileImportInput.SetText(uri)
					app.SetFocus(a.profileImportInput)
				}
			})
		})

		importBtn := tview.NewButton(a.t("dialog.common.import"))
		importBtn.SetSelectedFunc(func() { a.submitProfileImport(false) })
		importLoadBtn := tview.NewButton(a.t("dialog.profile.import.importLoad"))
		importLoadBtn.SetSelectedFunc(func() { a.submitProfileImport(true) })
		cancelBtn := tview.NewButton(a.t("dialog.common.cancel"))
		cancelBtn.SetSelectedFunc(func() { a.closeProfileImportDialog() })

		navItems := []tview.Primitive{a.profileImportInput, pasteBtn, importBtn, importLoadBtn, cancelBtn}
		focusMove := func(delta int) {
			current := app.GetFocus()
			index := 0
			for i, item := range navItems {
				if item == current {
					index = i
					break
				}
			}
			index = (index + delta + len(navItems)) % len(navItems)
			app.SetFocus(navItems[index])
		}

		for _, btn := range []*tview.Button{pasteBtn, importBtn, importLoadBtn, cancelBtn} {
			btn.SetLabelColor(tcell.ColorWhite)
			btn.SetLabelColorActivated(tcell.ColorBlack)
			btn.SetBackgroundColor(tcell.ColorDarkCyan)
			btn.SetBackgroundColorActivated(tcell.ColorYellow)
			btn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEsc:
					a.closeProfileImportDialog()
					return nil
				case tcell.KeyLeft:
					focusMove(-1)
					return nil
				case tcell.KeyRight:
					focusMove(1)
					return nil
				case tcell.KeyUp:
					app.SetFocus(a.profileImportInput)
					return nil
				case tcell.KeyDown:
					app.SetFocus(a.profileImportInput)
					return nil
				case tcell.KeyTAB:
					focusMove(1)
					return nil
				case tcell.KeyBacktab:
					focusMove(-1)
					return nil
				default:
					return event
				}
			})
		}

		a.profileImportInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				a.closeProfileImportDialog()
				return nil
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyTAB:
				app.SetFocus(pasteBtn)
				return nil
			case tcell.KeyBacktab:
				app.SetFocus(cancelBtn)
				return nil
			default:
				return event
			}
		})

		buttonBar := buttonRow(pasteBtn, importBtn, importLoadBtn, cancelBtn)
		container := tview.NewFlex().SetDirection(tview.FlexRow)
		container.SetBorder(true)
		container.SetTitle(" " + a.t("dialog.profile.import.title") + " ")
		container.AddItem(newMutedText(a.t("dialog.profile.import.description")), 1, 0, false)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(a.profileImportInput, 1, 0, true)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(buttonBar, 1, 0, false)

		a.profileImportPrev = app.GetFocus()
		overlay := centeredPrimitive(container, 100, 9)
		a.pageHolder.AddPage(profileImportDialogPage, overlay, true, true)
		a.profileImportVisible.Store(true)
		app.SetFocus(a.profileImportInput)
		a.setFooter(a.t("footer.profile.import.dialogHint"))
	})
}

func (a *tuiApp) closeProfileImportDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || !a.profileImportVisible.Load() {
			return
		}
		a.pageHolder.RemovePage(profileImportDialogPage)
		a.profileImportVisible.Store(false)
		a.profileImportInput = nil
		if a.profileImportPrev != nil {
			app.SetFocus(a.profileImportPrev)
		} else if len(a.focusables) > 0 {
			app.SetFocus(a.focusables[0])
		}
		a.profileImportPrev = nil
		a.setFooter(a.tf("status.page", pageDisplayName(a.page)))
	})
}

func (a *tuiApp) submitProfileImport(loadEditor bool) {
	uri := ""
	if a.profileImportInput != nil {
		uri = strings.TrimSpace(a.profileImportInput.GetText())
	}
	if uri == "" {
		a.setFooter(a.t("footer.profile.import.emptyURL"))
		return
	}
	a.closeProfileImportDialog()
	label := "import profile"
	if loadEditor {
		label = "import+load profile"
	}
	go a.runAction(label, func(ctx context.Context) error {
		return a.importProfileURI(ctx, uri, loadEditor)
	})
}

func (a *tuiApp) importProfileURI(ctx context.Context, uri string, loadEditor bool) error {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return errors.New(a.t("error.profile.emptyURI"))
	}
	if strings.HasPrefix(strings.ToLower(uri), "http://") || strings.HasPrefix(strings.ToLower(uri), "https://") {
		return a.importSubscriptionURI(ctx, uri)
	}
	profile, err := a.client.ImportProfile(ctx, uri)
	if err != nil {
		return err
	}
	a.storeSelectedProfileID(profile.ID)
	a.clearProfileEditDirty()
	if err := a.reloadProfiles(); err != nil {
		return err
	}
	if loadEditor {
		if err := a.loadSelectedProfileEditorAction(ctx); err != nil {
			return err
		}
		a.setProfileEditMessage(a.t("profileEdit.message.importLoaded"))
		a.pushEvent("profile imported+loaded: " + profile.ID)
		a.setFooter(a.tf("footer.profile.import.loaded", profile.ID))
		return nil
	}
	a.pushEvent("profile imported: " + profile.ID)
	a.setFooter(a.tf("footer.profile.import.ok", profile.ID))
	return nil
}

func (a *tuiApp) importSubscriptionURI(ctx context.Context, uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("%s: %w", a.t("error.profile.invalidSubscriptionURL"), err)
	}
	remarks := strings.TrimSpace(parsed.Hostname())
	if remarks == "" {
		remarks = "imported-subscription"
	}
	remarks = "import-" + remarks
	sub, err := a.client.CreateSubscription(ctx, SubscriptionUpsertRequest{
		Remarks:           remarks,
		URL:               uri,
		Enabled:           true,
		UserAgent:         "v2rayN/7.x",
		AutoUpdateMinutes: 0,
	})
	if err != nil {
		return err
	}
	if err := a.client.UpdateSubscription(ctx, sub.ID); err != nil {
		return err
	}
	a.pushEvent("subscription imported: " + sub.ID)
	return a.reloadAll()
}

func readClipboardText(ctx context.Context) (string, error) {
	commands := [][]string{
		{"wl-paste", "-n"},
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
	}
	for _, cmdArgs := range commands {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(output))
		if text != "" {
			return text, nil
		}
	}
	return "", errors.New("clipboard unavailable (need wl-paste/xclip/xsel)")
}

func (a *tuiApp) importProfileAction(ctx context.Context) error {
	a.openProfileImportDialog()
	return nil
}

func (a *tuiApp) importAndLoadProfileAction(ctx context.Context) error {
	a.openProfileImportDialog()
	return nil
}

func (a *tuiApp) loadSelectedProfileEditorAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return errors.New(a.t("error.profile.noSelection"))
	}
	profile, err := a.client.GetProfile(ctx, id)
	if err != nil {
		return err
	}
	a.storeSelectedProfileID(profile.ID)
	a.clearProfileEditDirty()
	a.refreshWidgets()
	a.setProfileEditMessage(a.t("profileEdit.message.loadedBackend"))
	return nil
}

func (a *tuiApp) resetProfileEditAction(ctx context.Context) error {
	a.setProfileEditMessage(a.t("profileEdit.message.resetBackend"))
	return a.loadSelectedProfileEditorAction(ctx)
}

func (a *tuiApp) saveSelectedProfileEditAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return errors.New(a.t("error.profile.noSelection"))
	}

	profile, err := a.client.GetProfile(ctx, id)
	if err != nil {
		a.setProfileEditMessage(a.t("profileEdit.message.loadBeforeSaveFailed"))
		return err
	}

	name := strings.TrimSpace(a.profileEditName.Text())
	address := strings.TrimSpace(a.profileEditAddress.Text())
	portText := strings.TrimSpace(a.profileEditPort.Text())
	vmessUUID := strings.TrimSpace(a.profileEditVMessUUID.Text())
	vmessAlterText := strings.TrimSpace(a.profileEditVMessAlter.Text())
	vmessSec := strings.TrimSpace(a.profileEditVMessSec.Text())
	vlessUUID := strings.TrimSpace(a.profileEditVLESSUUID.Text())
	vlessFlow := strings.TrimSpace(a.profileEditVLESSFlow.Text())
	vlessEnc := strings.TrimSpace(a.profileEditVLESSEnc.Text())
	ssMethod := strings.TrimSpace(a.profileEditSSMethod.Text())
	ssPassword := strings.TrimSpace(a.profileEditSSPassword.Text())
	ssPlugin := strings.TrimSpace(a.profileEditSSPlugin.Text())
	ssPluginOpts := strings.TrimSpace(a.profileEditSSPluginOpt.Text())
	trojanPwd := strings.TrimSpace(a.profileEditTrojanPwd.Text())
	hy2Pwd := strings.TrimSpace(a.profileEditHy2Pwd.Text())
	hy2SNI := strings.TrimSpace(a.profileEditHy2SNI.Text())
	hy2Insecure := parseBoolText(a.profileEditHy2Insecure.Text())
	hy2UpText := strings.TrimSpace(a.profileEditHy2UpMbps.Text())
	hy2DownText := strings.TrimSpace(a.profileEditHy2DownMbps.Text())
	hy2Obfs := strings.TrimSpace(a.profileEditHy2Obfs.Text())
	hy2ObfsPwd := strings.TrimSpace(a.profileEditHy2ObfsPwd.Text())
	tuicUUID := strings.TrimSpace(a.profileEditTuicUUID.Text())
	tuicPwd := strings.TrimSpace(a.profileEditTuicPwd.Text())
	tuicCC := strings.TrimSpace(a.profileEditTuicCC.Text())
	tuicSNI := strings.TrimSpace(a.profileEditTuicSNI.Text())
	tuicInsecure := parseBoolText(a.profileEditTuicInsec.Text())
	tuicALPN := splitCommaStrings(a.profileEditTuicALPN.Text())
	port := profile.Port
	if portText != "" {
		parsed, convErr := strconv.Atoi(portText)
		if convErr != nil || parsed <= 0 || parsed > 65535 {
			err := errors.New(a.t("error.profile.invalidPortRange"))
			a.setProfileEditMessage(a.tf("profileEdit.message.error", err.Error()))
			return err
		}
		port = parsed
	}
	network := strings.TrimSpace(strings.ToLower(a.profileEditNetwork.Text()))
	if network != "" && network != "tcp" && network != "ws" && network != "grpc" && network != "h2" && network != "kcp" && network != "quic" && network != "xhttp" {
		err := errors.New(a.t("error.profile.invalidNetwork"))
		a.setProfileEditMessage(a.tf("profileEdit.message.error", err.Error()))
		return err
	}
	tlsEnabled := parseBoolText(a.profileEditTLS.Text())
	sni := strings.TrimSpace(a.profileEditSNI.Text())
	fingerprint := strings.TrimSpace(a.profileEditFingerprint.Text())
	alpn := splitCommaStrings(a.profileEditALPN.Text())
	skipCert := parseBoolText(a.profileEditSkipCert.Text())
	realityPublicKey := strings.TrimSpace(a.profileEditRealityPK.Text())
	realityShortID := strings.TrimSpace(a.profileEditRealitySID.Text())
	wsPath := strings.TrimSpace(a.profileEditWSPath.Text())
	h2Path := splitCommaStrings(a.profileEditH2Path.Text())
	h2Host := splitCommaStrings(a.profileEditH2Host.Text())
	grpcService := strings.TrimSpace(a.profileEditGRPCSvc.Text())
	grpcMode := strings.TrimSpace(strings.ToLower(a.profileEditGRPCMode.Text()))
	if grpcMode != "" && grpcMode != "gun" && grpcMode != "multi" {
		err := errors.New(a.t("error.profile.invalidGRPCMode"))
		a.setProfileEditMessage(a.tf("profileEdit.message.error", err.Error()))
		return err
	}

	vmessAlter := 0
	if vmessAlterText != "" {
		parsed, convErr := strconv.Atoi(vmessAlterText)
		if convErr != nil || parsed < 0 {
			err := errors.New(a.t("error.profile.invalidVMessAlterID"))
			a.setProfileEditMessage(a.tf("profileEdit.message.error", err.Error()))
			return err
		}
		vmessAlter = parsed
	}

	hy2Up := 0
	if hy2UpText != "" {
		parsed, convErr := strconv.Atoi(hy2UpText)
		if convErr != nil || parsed < 0 {
			err := errors.New(a.t("error.profile.invalidHy2Up"))
			a.setProfileEditMessage(a.tf("profileEdit.message.error", err.Error()))
			return err
		}
		hy2Up = parsed
	}
	hy2Down := 0
	if hy2DownText != "" {
		parsed, convErr := strconv.Atoi(hy2DownText)
		if convErr != nil || parsed < 0 {
			err := errors.New(a.t("error.profile.invalidHy2Down"))
			a.setProfileEditMessage(a.tf("profileEdit.message.error", err.Error()))
			return err
		}
		hy2Down = parsed
	}

	if name != "" {
		profile.Name = name
	}
	if address != "" {
		profile.Address = address
	}
	if port > 0 {
		profile.Port = port
	}
	if profile.VMess != nil {
		if vmessUUID != "" {
			profile.VMess.UUID = vmessUUID
		}
		if vmessAlterText != "" {
			profile.VMess.AlterID = vmessAlter
		}
		if vmessSec != "" {
			profile.VMess.Security = vmessSec
		}
	}
	if profile.VLESS != nil {
		if vlessUUID != "" {
			profile.VLESS.UUID = vlessUUID
		}
		if vlessFlow != "" {
			profile.VLESS.Flow = vlessFlow
		}
		if vlessEnc != "" {
			profile.VLESS.Encryption = vlessEnc
		}
	}
	if profile.Shadowsocks != nil {
		if ssMethod != "" {
			profile.Shadowsocks.Method = ssMethod
		}
		if ssPassword != "" {
			profile.Shadowsocks.Password = ssPassword
		}
		profile.Shadowsocks.Plugin = ssPlugin
		profile.Shadowsocks.PluginOpts = ssPluginOpts
	}
	if profile.Trojan != nil {
		if trojanPwd != "" {
			profile.Trojan.Password = trojanPwd
		}
	}
	if profile.Hysteria2 != nil {
		if hy2Pwd != "" {
			profile.Hysteria2.Password = hy2Pwd
		}
		profile.Hysteria2.SNI = hy2SNI
		profile.Hysteria2.Insecure = hy2Insecure
		if hy2UpText != "" {
			profile.Hysteria2.UpMbps = hy2Up
		}
		if hy2DownText != "" {
			profile.Hysteria2.DownMbps = hy2Down
		}
		profile.Hysteria2.Obfs = hy2Obfs
		profile.Hysteria2.ObfsPassword = hy2ObfsPwd
	}
	if profile.TUIC != nil {
		if tuicUUID != "" {
			profile.TUIC.UUID = tuicUUID
		}
		if tuicPwd != "" {
			profile.TUIC.Password = tuicPwd
		}
		if tuicCC != "" {
			profile.TUIC.CongestionControl = tuicCC
		}
		profile.TUIC.SNI = tuicSNI
		profile.TUIC.Insecure = tuicInsecure
		profile.TUIC.ALPN = tuicALPN
	}

	if profile.Transport == nil {
		profile.Transport = &TransportConfig{}
	}
	if network != "" {
		profile.Transport.Network = network
	}
	profile.Transport.TLS = tlsEnabled
	profile.Transport.SNI = sni
	profile.Transport.Fingerprint = fingerprint
	profile.Transport.ALPN = alpn
	profile.Transport.SkipCertVerify = skipCert
	profile.Transport.RealityPublicKey = realityPublicKey
	profile.Transport.RealityShortID = realityShortID
	profile.Transport.WSPath = wsPath
	profile.Transport.H2Path = h2Path
	profile.Transport.H2Host = h2Host
	profile.Transport.GRPCServiceName = grpcService
	profile.Transport.GRPCMode = grpcMode

	updated, err := a.client.UpdateProfile(ctx, id, profile)
	if err != nil {
		a.setProfileEditMessage(a.t("profileEdit.message.saveFailed"))
		return err
	}
	a.storeSelectedProfileID(updated.ID)
	a.clearProfileEditDirty()
	a.setProfileEditMessage(a.t("profileEdit.message.saved"))
	a.pushEvent("profile updated: " + updated.ID)
	if err := a.reloadProfiles(); err != nil {
		return err
	}
	return a.reloadOverview()
}

func (a *tuiApp) activateProfileAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return errors.New(a.t("error.profile.noSelection"))
	}
	if err := a.client.SelectProfile(ctx, id); err != nil {
		return err
	}
	return a.reloadAll()
}

func (a *tuiApp) deleteSelectedProfileAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return errors.New(a.t("error.profile.noSelection"))
	}
	if err := a.client.DeleteProfile(ctx, id); err != nil {
		a.setProfileEditMessage(a.t("profileEdit.message.deleteFailed"))
		return err
	}
	a.setProfileEditMessage(a.t("profileEdit.message.deleted"))
	a.pushEvent("profile deleted: " + id)
	a.storeSelectedProfileID("")
	a.clearProfileEditDirty()
	return a.reloadProfiles()
}

func (a *tuiApp) batchDelayProfilesAction(ctx context.Context) error {
	ids := make([]string, 0, len(a.profiles))
	a.mu.Lock()
	for _, profile := range a.profiles {
		ids = append(ids, profile.ID)
	}
	a.mu.Unlock()
	a.storeBatchDelayState(true, nil)
	a.refreshWidgets()
	if len(ids) == 0 {
		a.storeBatchDelayState(false, nil)
		a.refreshWidgets()
		return errors.New(a.t("error.profile.noProfiles"))
	}
	result, err := a.client.BatchTestProfileDelay(ctx, ids, 5000, 5)
	if err == nil {
		a.storeBatchDelayState(false, &result)
	} else {
		a.storeBatchDelayState(false, nil)
	}
	a.refreshWidgets()
	if err != nil {
		return err
	}
	a.pushEvent(fmt.Sprintf("batch delay test completed total=%d success=%d failed=%d", result.Total, result.Success, result.Failed))
	return a.reloadProfiles()
}

func (a *tuiApp) testSelectedProfileDelayAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return errors.New(a.t("error.profile.noSelection"))
	}
	result, err := a.client.TestProfileDelay(ctx, id)
	if err != nil {
		return err
	}
	a.pushEvent(fmt.Sprintf("profile.delay %s available=%t delay=%dms %s", id, result.Available, result.DelayMs, result.Message))
	return a.reloadProfiles()
}

func (a *tuiApp) updateAllSubscriptionsAction(ctx context.Context) error {
	if err := a.client.UpdateAllSubscriptions(ctx); err != nil {
		return err
	}
	return a.reloadSubscriptions()
}

func (a *tuiApp) updateSelectedSubscriptionAction(ctx context.Context) error {
	id := a.currentSubscriptionID()
	if id == "" {
		return errors.New(a.t("error.subscription.noSelection"))
	}
	if err := a.client.UpdateSubscription(ctx, id); err != nil {
		return err
	}
	return a.reloadAll()
}

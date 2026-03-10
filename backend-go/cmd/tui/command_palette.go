package tui

import (
	"context"
	"strings"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const commandPalettePage = "command-palette"

type paletteAction struct {
	main      string
	secondary string
	run       func()
}

func (a *tuiApp) openCommandPalette() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || a.commandPaletteVisible.Load() {
			return
		}

		a.palettePreviousFocus = app.GetFocus()
		a.paletteActionsCache = a.paletteActions()

		a.commandPaletteInput = tview.NewInputField()
		a.commandPaletteInput.SetLabel(a.t("palette.search"))
		a.commandPaletteInput.SetFieldWidth(0)
		a.commandPaletteInput.SetFieldTextColor(editableValueColor)
		a.commandPaletteInput.SetLabelColor(editableLabelColor)
		a.commandPaletteInput.SetChangedFunc(func(text string) {
			a.refreshCommandPaletteItems(text)
		})
		a.commandPaletteInput.SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEsc:
				a.closeCommandPalette()
			case tcell.KeyEnter, tcell.KeyTab:
				if a.commandPalette != nil {
					app.SetFocus(a.commandPalette)
				}
			}
		})

		a.commandPalette = tview.NewList()
		a.commandPalette.ShowSecondaryText(true)
		a.commandPalette.SetBorder(true)
		a.commandPalette.SetTitle(" " + a.t("palette.title.actions") + " ")
		a.commandPalette.SetMainTextColor(editableValueColor)
		a.commandPalette.SetSecondaryTextColor(tcell.ColorLightGray)
		a.commandPalette.SetSelectedTextColor(tcell.ColorBlack)
		a.commandPalette.SetSelectedBackgroundColor(tcell.ColorYellow)
		a.commandPalette.SetDoneFunc(func() {
			if a.commandPaletteInput != nil {
				app.SetFocus(a.commandPaletteInput)
			}
		})
		a.commandPalette.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
				if a.commandPaletteInput != nil {
					app.SetFocus(a.commandPaletteInput)
				}
				return nil
			}
			return event
		})

		a.refreshCommandPaletteItems("")

		container := tview.NewFlex().SetDirection(tview.FlexRow)
		container.SetBorder(true)
		container.SetTitle(" " + a.t("palette.title.command") + " ")
		container.AddItem(a.commandPaletteInput, 1, 0, true)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(a.commandPalette, 0, 1, false)

		overlay := centeredPrimitive(container, 96, 24)
		a.pageHolder.AddPage(commandPalettePage, overlay, true, true)
		a.commandPaletteVisible.Store(true)
		app.SetFocus(a.commandPaletteInput)
		a.footerStatus = a.t("palette.footer.hint")
		a.refreshFooter()
	})
}

func (a *tuiApp) closeCommandPalette() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || !a.commandPaletteVisible.Load() {
			return
		}
		a.pageHolder.RemovePage(commandPalettePage)
		a.commandPaletteVisible.Store(false)
		a.commandPalette = nil
		a.commandPaletteInput = nil
		a.paletteActionsCache = nil
		if a.palettePreviousFocus != nil {
			app.SetFocus(a.palettePreviousFocus)
		} else if len(a.focusables) > 0 {
			app.SetFocus(a.focusables[0])
		}
		a.palettePreviousFocus = nil
		a.footerStatus = a.tf("status.page", pageDisplayName(a.page))
		a.refreshFooter()
	})
}

func (a *tuiApp) refreshCommandPaletteItems(filter string) {
	if a.commandPalette == nil {
		return
	}

	query := strings.ToLower(strings.TrimSpace(filter))
	a.commandPalette.Clear()
	count := 0
	for _, item := range a.paletteActionsCache {
		if query != "" {
			target := strings.ToLower(item.main + " " + item.secondary)
			if !strings.Contains(target, query) {
				continue
			}
		}
		item := item
		a.commandPalette.AddItem(item.main, item.secondary, 0, func() {
			a.closeCommandPalette()
			if item.run != nil {
				item.run()
			}
		})
		count++
	}

	if count == 0 {
		a.commandPalette.AddItem(a.t("palette.empty.main"), a.t("palette.empty.secondary"), 0, nil)
	}
}

func (a *tuiApp) paletteActions() []paletteAction {
	runAction := func(label string, action func(context.Context) error) func() {
		return func() {
			go a.runAction(label, action)
		}
	}

	actions := []paletteAction{
		{main: a.tf("palette.main.goPage", a.t("page.dashboard")), secondary: a.tf("palette.secondary.switchPage", 1), run: func() { a.setActivePage(pageDashboard) }},
		{main: a.tf("palette.main.goPage", a.t("page.profiles")), secondary: a.tf("palette.secondary.switchPage", 2), run: func() { a.setActivePage(pageProfiles) }},
		{main: a.tf("palette.main.goPage", a.t("page.subscriptions")), secondary: a.tf("palette.secondary.switchPage", 3), run: func() { a.setActivePage(pageSubscriptions) }},
		{main: a.tf("palette.main.goPage", a.t("page.network")), secondary: a.tf("palette.secondary.switchPage", 4), run: func() { a.setActivePage(pageNetwork) }},
		{main: a.tf("palette.main.goPage", a.t("page.settings")), secondary: a.tf("palette.secondary.switchPage", 5), run: func() { a.setActivePage(pageSettings) }},
		{main: a.tf("palette.main.goPage", a.t("page.logs")), secondary: a.tf("palette.secondary.switchPage", 6), run: func() { a.setActivePage(pageLogs) }},
		{main: a.t("palette.main.refreshAll"), secondary: a.t("palette.secondary.refreshAll"), run: runAction(a.t("action.refresh"), a.refreshAllAction)},
		{main: a.t("palette.main.coreStart"), secondary: a.t("palette.secondary.coreStart"), run: runAction(a.t("dashboard.btn.start"), a.startCoreAction)},
		{main: a.t("palette.main.coreStop"), secondary: a.t("palette.secondary.coreStop"), run: runAction(a.t("dashboard.btn.stop"), a.stopCoreAction)},
		{main: a.t("palette.main.coreRestart"), secondary: a.t("palette.secondary.coreRestart"), run: runAction(a.t("dashboard.btn.restart"), a.restartCoreAction)},
	}

	switch a.page {
	case pageProfiles:
		actions = append(actions,
			paletteAction{main: a.t("palette.main.profiles.importURL"), secondary: a.t("palette.secondary.profiles.importURL"), run: runAction(a.t("profiles.btn.importURL"), a.openImportProfileDialogAction)},
			paletteAction{main: a.t("palette.main.profiles.importClipboard"), secondary: a.t("palette.secondary.profiles.importClipboard"), run: runAction(a.t("profiles.btn.importClipboard"), a.importProfileFromClipboardAction)},
			paletteAction{main: a.t("palette.main.profiles.batchDelay"), secondary: a.t("palette.secondary.profiles.batchDelay"), run: runAction(a.t("profiles.btn.batchDelay"), a.batchDelayProfilesAction)},
			paletteAction{main: a.t("palette.main.profiles.activate"), secondary: a.t("palette.secondary.profiles.activate"), run: runAction(a.t("action.activateProfile"), a.activateProfileAction)},
			paletteAction{main: a.t("palette.main.profiles.delayTest"), secondary: a.t("palette.secondary.profiles.delayTest"), run: runAction(a.t("menu.profile.delay"), a.testSelectedProfileDelayAction)},
			paletteAction{main: a.t("palette.main.profiles.saveEdit"), secondary: a.t("palette.secondary.profiles.saveEdit"), run: runAction(a.t("dialog.common.save"), a.saveSelectedProfileEditAction)},
		)
	case pageSubscriptions:
		actions = append(actions,
			paletteAction{main: a.t("palette.main.subs.updateSelected"), secondary: a.t("palette.secondary.subs.updateSelected"), run: runAction(a.t("subs.btn.updateSelected"), a.updateSelectedSubscriptionAction)},
			paletteAction{main: a.t("palette.main.subs.updateAll"), secondary: a.t("palette.secondary.subs.updateAll"), run: runAction(a.t("subs.btn.updateAll"), a.updateAllSubscriptionsAction)},
		)
	case pageNetwork:
		actions = append(actions,
			paletteAction{main: a.t("palette.main.network.saveRouting"), secondary: a.t("palette.secondary.network.saveRouting"), run: runAction(a.t("network.btn.saveRouting"), a.saveRoutingModeAction)},
			paletteAction{main: a.t("palette.main.network.routeTest"), secondary: a.t("palette.secondary.network.routeTest"), run: runAction(a.t("network.btn.routeTest"), a.routeTestAction)},
			paletteAction{main: a.t("palette.main.network.repairTun"), secondary: a.t("palette.secondary.network.repairTun"), run: runAction(a.t("network.btn.repairTun"), a.repairTunAction)},
		)
	case pageSettings:
		actions = append(actions,
			paletteAction{main: a.t("palette.main.settings.saveConfig"), secondary: a.t("palette.secondary.settings.saveConfig"), run: runAction(a.t("settings.btn.saveConfig"), a.saveConfigAction)},
			paletteAction{main: a.t("palette.main.settings.clearCoreError"), secondary: a.t("palette.secondary.settings.clearCoreError"), run: runAction(a.t("settings.btn.clearCoreError"), a.clearCoreErrorAction)},
			paletteAction{main: a.t("palette.main.settings.langZH"), secondary: a.t("palette.secondary.settings.langZH"), run: runAction(a.t("settings.lang.chinese"), a.selectUILanguageChineseAction)},
			paletteAction{main: a.t("palette.main.settings.langEN"), secondary: a.t("palette.secondary.settings.langEN"), run: runAction(a.t("settings.lang.english"), a.selectUILanguageEnglishAction)},
		)
	case pageLogs:
		actions = append(actions,
			paletteAction{main: a.t("palette.main.logs.applySearch"), secondary: a.t("palette.secondary.logs.applySearch"), run: runAction(a.t("logs.btn.applySearch"), a.applyLogSearchAction)},
			paletteAction{main: a.t("palette.main.logs.clearSearch"), secondary: a.t("palette.secondary.logs.clearSearch"), run: runAction(a.t("logs.btn.clearSearch"), a.clearLogSearchAction)},
		)
	}

	return actions
}

func centeredPrimitive(p tview.Primitive, width, height int) tview.Primitive {
	rows := tview.NewFlex().SetDirection(tview.FlexRow)
	rows.AddItem(nil, 0, 1, false)
	rows.AddItem(p, height, 0, true)
	rows.AddItem(nil, 0, 1, false)

	cols := tview.NewFlex().SetDirection(tview.FlexColumn)
	cols.AddItem(nil, 0, 1, false)
	cols.AddItem(rows, width, 0, true)
	cols.AddItem(nil, 0, 1, false)
	return cols
}

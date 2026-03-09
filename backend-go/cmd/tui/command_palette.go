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
		a.commandPaletteInput.SetLabel("Search: ")
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
		a.commandPalette.SetTitle(" Actions ")
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
		container.SetTitle(" Command Palette ")
		container.AddItem(a.commandPaletteInput, 1, 0, true)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(a.commandPalette, 0, 1, false)

		overlay := centeredPrimitive(container, 96, 24)
		a.pageHolder.AddPage(commandPalettePage, overlay, true, true)
		a.commandPaletteVisible.Store(true)
		app.SetFocus(a.commandPaletteInput)
		a.footerStatus = "Command palette: type to filter, Enter run, Esc close"
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
		a.commandPalette.AddItem("No matching actions", "Try another keyword", 0, nil)
	}
}

func (a *tuiApp) paletteActions() []paletteAction {
	runAction := func(label string, action func(context.Context) error) func() {
		return func() {
			go a.runAction(label, action)
		}
	}

	actions := []paletteAction{
		{main: "Go Dashboard", secondary: "Switch to page 1", run: func() { a.setActivePage(pageDashboard) }},
		{main: "Go Profiles", secondary: "Switch to page 2", run: func() { a.setActivePage(pageProfiles) }},
		{main: "Go Subscriptions", secondary: "Switch to page 3", run: func() { a.setActivePage(pageSubscriptions) }},
		{main: "Go Network", secondary: "Switch to page 4", run: func() { a.setActivePage(pageNetwork) }},
		{main: "Go Settings", secondary: "Switch to page 5", run: func() { a.setActivePage(pageSettings) }},
		{main: "Go Logs", secondary: "Switch to page 6", run: func() { a.setActivePage(pageLogs) }},
		{main: "Refresh All", secondary: "Reload overview/profiles/subscriptions/network", run: runAction("refresh", a.refreshAllAction)},
		{main: "Core Start", secondary: "Start service core", run: runAction("start", a.startCoreAction)},
		{main: "Core Stop", secondary: "Stop service core", run: runAction("stop", a.stopCoreAction)},
		{main: "Core Restart", secondary: "Restart service core", run: runAction("restart", a.restartCoreAction)},
	}

	switch a.page {
	case pageProfiles:
		actions = append(actions,
			paletteAction{main: "Profiles: Import URL", secondary: "Open URL import dialog", run: runAction("import url", a.openImportProfileDialogAction)},
			paletteAction{main: "Profiles: Import Clipboard", secondary: "Import profile from clipboard content", run: runAction("import clipboard", a.importProfileFromClipboardAction)},
			paletteAction{main: "Profiles: Batch Delay", secondary: "Run delay test for all profiles", run: runAction("batch delay", a.batchDelayProfilesAction)},
			paletteAction{main: "Profiles: Activate Selected", secondary: "Activate highlighted profile", run: runAction("activate", a.activateProfileAction)},
			paletteAction{main: "Profiles: Delay Test", secondary: "Test selected profile delay", run: runAction("delay test", a.testSelectedProfileDelayAction)},
			paletteAction{main: "Profiles: Save Edit", secondary: "Persist editor fields to backend", run: runAction("save edit", a.saveSelectedProfileEditAction)},
		)
	case pageSubscriptions:
		actions = append(actions,
			paletteAction{main: "Subscriptions: Update Selected", secondary: "Pull selected subscription source", run: runAction("update selected", a.updateSelectedSubscriptionAction)},
			paletteAction{main: "Subscriptions: Update All", secondary: "Pull all subscription sources", run: runAction("update all", a.updateAllSubscriptionsAction)},
		)
	case pageNetwork:
		actions = append(actions,
			paletteAction{main: "Network: Save Routing", secondary: "Apply current routing target fields", run: runAction("save routing", a.saveRoutingModeAction)},
			paletteAction{main: "Network: Route Test", secondary: "Run routing diagnosis for target", run: runAction("route test", a.routeTestAction)},
			paletteAction{main: "Network: Repair TUN", secondary: "Run TUN repair command", run: runAction("repair tun", a.repairTunAction)},
		)
	case pageSettings:
		actions = append(actions,
			paletteAction{main: "Settings: Save Config", secondary: "Apply config editor fields", run: runAction("save config", a.saveConfigAction)},
			paletteAction{main: "Settings: Clear Core Error", secondary: "Clear backend core error state", run: runAction("clear core error", a.clearCoreErrorAction)},
			paletteAction{main: "Settings: UI Language Chinese", secondary: "Switch interface to Chinese", run: runAction("lang zh", a.selectUILanguageChineseAction)},
			paletteAction{main: "Settings: UI Language English", secondary: "Switch interface to English", run: runAction("lang en", a.selectUILanguageEnglishAction)},
		)
	case pageLogs:
		actions = append(actions,
			paletteAction{main: "Logs: Apply Search", secondary: "Apply current search text", run: runAction("apply search", a.applyLogSearchAction)},
			paletteAction{main: "Logs: Clear Search", secondary: "Reset search filter", run: runAction("clear search", a.clearLogSearchAction)},
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

package tui

import (
	"context"
	"strings"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *tuiApp) attachApp(app *tview.Application) {
	a.app = app
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		cols, rows := screen.Size()
		a.viewportCols = cols
		a.viewportRows = rows
		return false
	})
}

func (a *tuiApp) handler(event *tcell.EventKey) *tcell.EventKey {
	if a.app != nil {
		// 将 DropDown 当作按钮：仅 Enter 打开覆盖层选择菜单。
		if dropdown, ok := focusedDropDownFromPrimitive(a.app.GetFocus()); ok {
			a.lastDropdownFocus = dropdown
			if event.Key() == tcell.KeyEnter {
				a.dropdownCancelValue = a.dropdownCurrentValue(dropdown)
				a.dropdownCancelArmed = true
				if widget, ok := focusedDropdownWidgetFromPrimitive(a.app.GetFocus()); ok {
					a.openDropdownSelectDialog(widget)
					return nil
				}
			}
		}

		// 检查焦点是否在 DropDown 的内部 List 上
		// tview 的 DropDown 在打开选择列表时会将焦点切换到内部的 List
		if list, ok := a.app.GetFocus().(*tview.List); ok && !a.isStandaloneListWidget(list) {
			// 这是一个下拉菜单的内部列表
			switch event.Key() {
			case tcell.KeyEsc:
				// 用户按 ESC 关闭选择列表
				if a.lastDropdownFocus != nil {
					// 如果之前按下了 Enter（armed），则恢复原始值
					if a.dropdownCancelArmed {
						a.restoreDropdownValue(a.lastDropdownFocus, a.dropdownCancelValue)
					}
					a.dropdownCancelArmed = false
				}
				// 交给 tview 继续处理 ESC，确保下拉内部状态被正确关闭。
				return event
			case tcell.KeyEnter:
				// 用户在列表中按 Enter 选择选项
				// 焦点会自动回到 DropDown，不需要额外处理
				a.dropdownCancelArmed = false
			case tcell.KeyTAB:
				// Tab 切换到下一个焦点项
				if a.lastDropdownFocus != nil {
					a.dropdownCancelArmed = false
					a.app.SetFocus(a.lastDropdownFocus)
					a.cycleFocus(false)
					return nil
				}
			case tcell.KeyBacktab:
				// Backtab 切换到上一个焦点项
				if a.lastDropdownFocus != nil {
					a.dropdownCancelArmed = false
					a.app.SetFocus(a.lastDropdownFocus)
					a.cycleFocus(true)
					return nil
				}
			case tcell.KeyLeft, tcell.KeyRight:
				// 下拉列表仅接受上下导航，忽略左右键。
				return nil
			}
		}
	}

	if a.dropdownSelectVisible.Load() {
		if event.Key() == tcell.KeyEsc {
			a.closeDropdownSelectDialog(true)
			return nil
		}
		return event
	}

	// Esc handling must be layered from top-most overlays to base layout.
	// Keep this ordering: close visible dialogs first, then fallback to sidebar back.
	if a.commandPaletteVisible.Load() {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyCtrlP:
			a.closeCommandPalette()
			return nil
		default:
			return event
		}
	}

	if a.profileActionsVisible.Load() {
		if event.Key() == tcell.KeyEsc {
			a.closeProfileActionsMenu()
			return nil
		}
		return event
	}

	if a.profileDeleteVisible.Load() {
		if event.Key() == tcell.KeyEsc {
			a.closeProfileDeleteConfirmDialog()
			return nil
		}
		return event
	}

	if a.profileImportVisible.Load() {
		if event.Key() == tcell.KeyEsc {
			a.closeProfileImportDialog()
			return nil
		}
		return event
	}

	if a.profileEditVisible.Load() {
		if event.Key() == tcell.KeyEsc {
			a.closeProfileQuickEditDialog()
			return nil
		}
		return event
	}

	if a.proxyUserSelectVisible.Load() {
		if event.Key() == tcell.KeyEsc {
			a.closeProxyUserSelectDialog()
			return nil
		}
		return event
	}

	switch event.Key() {
	case tcell.KeyEsc:
		if !a.focusIsSidebar() {
			a.focusSidebarSelected()
		}
		return nil
	case tcell.KeyCtrlC:
		a.cancel()
		if a.app != nil {
			a.app.Stop()
		}
		return nil
	case tcell.KeyCtrlP:
		a.openCommandPalette()
		return nil
	case tcell.KeyRight:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.cycleFocusItems(false)
		return nil
	case tcell.KeyLeft:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.cycleFocusItems(true)
		return nil
	case tcell.KeyDown:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.cycleFocusItems(false)
		return nil
	case tcell.KeyUp:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.cycleFocusItems(true)
		return nil
	case tcell.KeyTAB:
		a.cycleFocus(false)
		return nil
	case tcell.KeyBacktab:
		a.cycleFocus(true)
		return nil
	}

	if a.focusIsInput() {
		return event
	}

	if event.Key() == tcell.KeyRune && event.Rune() == '?' {
		a.openCommandPalette()
		return nil
	}

	if strings.EqualFold(string(event.Rune()), "q") {
		a.cancel()
		if a.app != nil {
			a.app.Stop()
		}
		return nil
	}

	if a.handleShortcut(event) {
		return nil
	}
	return event
}

func (a *tuiApp) isStandaloneListWidget(list *tview.List) bool {
	switch list {
	case a.profilesList, a.subscriptionsList, a.commandPalette, a.profileActionsMenu, a.proxyUserSelectMenu, a.dropdownSelectMenu:
		return true
	default:
		return false
	}
}

func (a *tuiApp) dropdownCurrentValue(dropdown *tview.DropDown) string {
	if dropdown == nil {
		return ""
	}
	for _, widget := range []*dropdownWidget{
		a.logsLevelSelect,
		a.logsSourceSelect,
		a.profileEditNetwork,
		a.profileEditTLS,
		a.profileEditSkipCert,
		a.profileEditGRPCMode,
		a.profileEditVMessSec,
		a.profileEditVLESSEnc,
		a.profileEditHy2Insecure,
		a.profileEditTuicCC,
		a.profileEditTuicInsec,
		a.networkPresetSelect,
		a.networkRoutingMode,
		a.networkDomainStrategy,
		a.networkLocalBypass,
		a.settingsLanguage,
		a.settingsTunMode,
		a.settingsTunAutoRoute,
		a.settingsTunStrict,
		a.settingsProxyMode,
		a.settingsCoreEngine,
		a.settingsLogLevel,
		a.settingsSkipCert,
		a.settingsDNSMode,
	} {
		if widget != nil && widget.DropDown == dropdown {
			return widget.Text()
		}
	}
	return ""
}

func (a *tuiApp) restoreDropdownValue(dropdown *tview.DropDown, value string) {
	if dropdown == nil {
		return
	}
	for _, widget := range []*dropdownWidget{
		a.logsLevelSelect,
		a.logsSourceSelect,
		a.profileEditNetwork,
		a.profileEditTLS,
		a.profileEditSkipCert,
		a.profileEditGRPCMode,
		a.profileEditVMessSec,
		a.profileEditVLESSEnc,
		a.profileEditHy2Insecure,
		a.profileEditTuicCC,
		a.profileEditTuicInsec,
		a.networkPresetSelect,
		a.networkRoutingMode,
		a.networkDomainStrategy,
		a.networkLocalBypass,
		a.settingsLanguage,
		a.settingsTunMode,
		a.settingsTunAutoRoute,
		a.settingsTunStrict,
		a.settingsProxyMode,
		a.settingsCoreEngine,
		a.settingsLogLevel,
		a.settingsSkipCert,
		a.settingsDNSMode,
	} {
		if widget != nil && widget.DropDown == dropdown {
			widget.SetText(value, a.app)
			return
		}
	}
}

func isSettingsEditKey(key *tcell.EventKey) bool {
	switch key.Key() {
	case tcell.KeyRune, tcell.KeyBackspace2, tcell.KeyDelete, tcell.KeyCtrlK, tcell.KeyCtrlU, tcell.KeyCtrlW:
		return true
	default:
		return false
	}
}

func (a *tuiApp) runAction(label string, action func(context.Context) error) {
	a.setFooter(a.tf("footer.running", label))
	err := action(a.ctx)
	if err != nil {
		a.pushEvent(label + " failed: " + err.Error())
		a.setFooter(a.tf("footer.failed", label, err.Error()))
		return
	}
	a.setFooter(a.tf("footer.completed", label))
}

func (a *tuiApp) actionButton(label string, action func(context.Context) error) *tview.Button {
	btn := tview.NewButton(label)
	btn.SetLabelColor(tcell.ColorWhite)
	btn.SetLabelColorActivated(tcell.ColorBlack)
	btn.SetBackgroundColor(tcell.ColorDarkCyan)
	btn.SetBackgroundColorActivated(tcell.ColorYellow)
	btn.SetSelectedFunc(func() {
		go a.runAction(strings.ToLower(label), action)
	})
	return btn
}

func (a *tuiApp) cycleFocus(reverse bool) {
	if a.app == nil {
		return
	}
	if len(a.focusGroups) > 1 {
		current := a.app.GetFocus()
		groupIdx := -1
		for idx, group := range a.focusGroups {
			for _, primitive := range group {
				if primitive == current {
					groupIdx = idx
					break
				}
			}
			if groupIdx >= 0 {
				break
			}
		}
		if groupIdx < 0 {
			groupIdx = a.focusGroup
		}
		if groupIdx < 0 || groupIdx >= len(a.focusGroups) {
			groupIdx = 0
		}
		if reverse {
			groupIdx = (groupIdx - 1 + len(a.focusGroups)) % len(a.focusGroups)
		} else {
			groupIdx = (groupIdx + 1) % len(a.focusGroups)
		}
		a.focusGroup = groupIdx
		if len(a.focusGroups[groupIdx]) > 0 {
			a.app.SetFocus(a.focusGroups[groupIdx][0])
		}
		return
	}
	a.cycleFocusItems(reverse)
}

func (a *tuiApp) cycleFocusItems(reverse bool) {
	if a.app == nil {
		return
	}
	if len(a.focusables) == 0 {
		return
	}
	current := a.app.GetFocus()
	index := -1
	for idx, primitive := range a.focusables {
		if primitive == current {
			index = idx
			break
		}
	}
	if index == -1 {
		index = 0
	} else if reverse {
		index = (index - 1 + len(a.focusables)) % len(a.focusables)
	} else {
		index = (index + 1) % len(a.focusables)
	}
	a.app.SetFocus(a.focusables[index])
}

func (a *tuiApp) focusHandlesDirectionalKeys() bool {
	if a.app == nil {
		return false
	}
	switch a.app.GetFocus().(type) {
	case *tview.InputField, *tview.List, *tview.TextView, *tview.Table, *tview.TextArea:
		return true
	default:
		return false
	}
}

func (a *tuiApp) focusIsSidebar() bool {
	if a.app == nil || a.sidebar == nil {
		return false
	}
	current := a.app.GetFocus()
	for _, primitive := range a.sidebar.GetFocusables() {
		if primitive == current {
			return true
		}
	}
	return false
}

func (a *tuiApp) focusSidebarSelected() bool {
	if a.app == nil || a.sidebar == nil {
		return false
	}
	buttons := a.sidebar.GetAllButtons()
	if len(buttons) == 0 {
		return false
	}
	index := a.sidebar.GetSelectedIndex()
	if index < 0 || index >= len(buttons) {
		index = 0
	}
	if buttons[index] == nil {
		return false
	}
	a.app.SetFocus(buttons[index])
	return true
}

func focusedDropDownFromPrimitive(primitive tview.Primitive) (*tview.DropDown, bool) {
	switch widget := primitive.(type) {
	case *tview.DropDown:
		return widget, true
	case *dropdownWidget:
		if widget.DropDown == nil {
			return nil, false
		}
		return widget.DropDown, true
	default:
		return nil, false
	}
}

func focusedDropdownWidgetFromPrimitive(primitive tview.Primitive) (*dropdownWidget, bool) {
	widget, ok := primitive.(*dropdownWidget)
	if !ok || widget == nil {
		return nil, false
	}
	return widget, true
}

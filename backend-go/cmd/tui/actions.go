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
		cols, _ := screen.Size()
		a.viewportCols = cols
		return false
	})
}

func (a *tuiApp) handler(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC:
		a.cancel()
		if a.app != nil {
			a.app.Stop()
		}
		return nil
	case tcell.KeyRight:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.shiftActivePage(1)
		return nil
	case tcell.KeyLeft:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.shiftActivePage(-1)
		return nil
	case tcell.KeyDown:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.cycleFocus(false)
		return nil
	case tcell.KeyUp:
		if a.focusHandlesDirectionalKeys() {
			return event
		}
		a.cycleFocus(true)
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

func isSettingsEditKey(key *tcell.EventKey) bool {
	switch key.Key() {
	case tcell.KeyRune, tcell.KeyBackspace2, tcell.KeyDelete, tcell.KeyCtrlK, tcell.KeyCtrlU, tcell.KeyCtrlW:
		return true
	default:
		return false
	}
}

func (a *tuiApp) runAction(label string, action func(context.Context) error) {
	a.setFooter("Running " + label + "...")
	err := action(a.ctx)
	if err != nil {
		a.pushEvent(label + " failed: " + err.Error())
		a.setFooter(label + " failed: " + err.Error())
		return
	}
	a.setFooter(label + " completed")
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
	if a.app == nil || len(a.focusables) == 0 {
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

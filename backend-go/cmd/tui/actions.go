package tui

import (
	"context"
	"strings"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/button"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
	tcell "github.com/gdamore/tcell/v2"
)

func (a *tuiApp) attachApp(app *gowid.App) {
	a.app = app
}

func (a *tuiApp) handler(app gowid.IApp, ev interface{}) bool {
	if gowid.HandleQuitKeys(app, ev) {
		a.cancel()
		return true
	}

	key, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	if !a.handleShortcut(app, key) {
		return false
	}
	return true
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

func (a *tuiApp) actionButton(label string, action func(context.Context) error) gowid.IWidget {
	btn := button.New(text.New(label))
	btn.OnClick(gowid.MakeWidgetCallback("btn-"+label, func(app gowid.IApp, _ gowid.IWidget) {
		go a.runAction(strings.ToLower(label), action)
	}))
	return styled.NewExt(btn, gowid.MakePaletteRef("button"), gowid.MakePaletteRef("button-focus"))
}

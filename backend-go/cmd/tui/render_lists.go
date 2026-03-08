package tui

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/button"
	"github.com/gcla/gowid/widgets/list"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
)

func (a *tuiApp) makeProfilesList() gowid.IWidget {
	if len(a.profiles) == 0 {
		return text.New("No profiles available")
	}

	profiles := a.sortedProfilesForDisplay()
	widgets := make([]gowid.IWidget, 0, len(profiles))
	for _, profile := range profiles {
		profile := profile
		label := a.profileLabel(profile)
		btn := button.NewBare(text.New(label))
		btn.OnClick(gowid.MakeWidgetCallback("profile-"+profile.ID, func(app gowid.IApp, _ gowid.IWidget) {
			a.mu.Lock()
			a.selectedProfileID = profile.ID
			a.mu.Unlock()
			a.refreshWidgets()
		}))
		style := "panel"
		if profile.ID == a.selectedProfileID {
			style = "button-selected"
		}
		widgets = append(widgets, styled.NewExt(btn, gowid.MakePaletteRef(style), gowid.MakePaletteRef("button-focus")))
	}
	return list.New(list.NewSimpleListWalker(widgets))
}

func (a *tuiApp) makeSubscriptionsList() gowid.IWidget {
	if len(a.subscriptions) == 0 {
		return text.New("No subscriptions available")
	}

	widgets := make([]gowid.IWidget, 0, len(a.subscriptions))
	for _, sub := range a.subscriptions {
		sub := sub
		label := sub.Remarks
		if label == "" {
			label = sub.URL
		}
		if !sub.Enabled {
			label = "[off] " + label
		}
		btn := button.NewBare(text.New(label))
		btn.OnClick(gowid.MakeWidgetCallback("sub-"+sub.ID, func(app gowid.IApp, _ gowid.IWidget) {
			a.mu.Lock()
			a.selectedSubID = sub.ID
			a.mu.Unlock()
			a.refreshWidgets()
		}))
		style := "panel"
		if sub.ID == a.selectedSubID {
			style = "button-selected"
		}
		widgets = append(widgets, styled.NewExt(btn, gowid.MakePaletteRef(style), gowid.MakePaletteRef("button-focus")))
	}
	return list.New(list.NewSimpleListWalker(widgets))
}

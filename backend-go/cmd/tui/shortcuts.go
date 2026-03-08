package main

import (
	"context"

	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
)

const (
	pageDashboard     = "dashboard"
	pageProfiles      = "profiles"
	pageSubscriptions = "subscriptions"
	pageNetwork       = "network"
	pageSettings      = "settings"
	pageLogs          = "logs"
)

type pageTab struct {
	key      string
	label    string
	shortcut rune
}

func tuiPageTabs() []pageTab {
	return []pageTab{
		{key: pageDashboard, label: "1 Dashboard", shortcut: '1'},
		{key: pageProfiles, label: "2 Profiles", shortcut: '2'},
		{key: pageSubscriptions, label: "3 Subs", shortcut: '3'},
		{key: pageNetwork, label: "4 Network", shortcut: '4'},
		{key: pageSettings, label: "5 Settings", shortcut: '5'},
		{key: pageLogs, label: "6 Logs", shortcut: '6'},
	}
}

func (a *tuiApp) handleShortcut(app gowid.IApp, key *tcell.EventKey) bool {
	if a.page == pageSettings && isSettingsEditKey(key) {
		a.markSettingsDirty()
	}

	if page := pageForShortcut(key.Rune()); page != "" {
		a.setActivePage(page, app)
		return true
	}

	switch key.Rune() {
	case 'r', 'R':
		go a.runAction("refresh", func(context.Context) error {
			return a.reloadAll()
		})
		return true
	default:
		return false
	}
}

func pageForShortcut(shortcut rune) string {
	for _, tab := range tuiPageTabs() {
		if tab.shortcut == shortcut {
			return tab.key
		}
	}
	return ""
}

func (a *tuiApp) setActivePage(page string, app gowid.IApp) {
	a.page = page
	a.syncPages(app)
	if app != nil && a.footer != nil {
		a.footer.SetText(footerText(page), app)
	}
}

func footerText(page string) string {
	return "Page: " + page + " | r refresh | q quit"
}

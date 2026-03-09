package tui

import (
	"context"
	"fmt"
	"strings"

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

func (a *tuiApp) handleShortcut(key *tcell.EventKey) bool {
	if page := pageForShortcut(key.Rune()); page != "" {
		a.setActivePage(page)
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

func isProfileEditKey(key *tcell.EventKey) bool {
	return isSettingsEditKey(key)
}

func pageForShortcut(shortcut rune) string {
	for _, tab := range tuiPageTabs() {
		if tab.shortcut == shortcut {
			return tab.key
		}
	}
	return ""
}

func (a *tuiApp) setActivePage(page string) {
	a.page = page
	a.syncPages()
	a.footerStatus = fmt.Sprintf("Page: %s", pageDisplayName(page))
	a.setFooter(a.footerStatus)
}

func (a *tuiApp) shiftActivePage(delta int) {
	tabs := tuiPageTabs()
	if len(tabs) == 0 {
		return
	}
	index := 0
	for i, tab := range tabs {
		if tab.key == a.page {
			index = i
			break
		}
	}
	next := (index + delta + len(tabs)) % len(tabs)
	a.setActivePage(tabs[next].key)
}

func footerText(page, status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		trimmed = "Ready"
	}
	return fmt.Sprintf("%s | %s", trimmed, pageHint(page))
}

func pageHint(page string) string {
	base := "1-6/←→ pages | Tab/↑↓ focus | r refresh | q quit"
	switch page {
	case pageProfiles:
		return "Profiles: select -> Activate/Delay | " + base
	case pageSubscriptions:
		return "Subscriptions: select -> Update Selected | " + base
	case pageNetwork:
		return "Network: set target/port -> Route Test | " + base
	case pageSettings:
		return "Settings: edit fields -> Save Config | " + base
	case pageLogs:
		return "Logs: level/source/search filters | " + base
	default:
		return "Dashboard: Start/Stop/Restart core | " + base
	}
}

func pageDisplayName(page string) string {
	switch page {
	case pageDashboard:
		return "Dashboard"
	case pageProfiles:
		return "Profiles"
	case pageSubscriptions:
		return "Subscriptions"
	case pageNetwork:
		return "Network"
	case pageSettings:
		return "Settings"
	case pageLogs:
		return "Logs"
	default:
		return "Dashboard"
	}
}

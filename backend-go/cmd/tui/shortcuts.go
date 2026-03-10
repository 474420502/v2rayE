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
	shortcut rune
}

func tuiPageTabs() []pageTab {
	return []pageTab{
		{key: pageDashboard, shortcut: '1'},
		{key: pageProfiles, shortcut: '2'},
		{key: pageSubscriptions, shortcut: '3'},
		{key: pageNetwork, shortcut: '4'},
		{key: pageSettings, shortcut: '5'},
		{key: pageLogs, shortcut: '6'},
	}
}

func (a *tuiApp) handleShortcut(key *tcell.EventKey) bool {
	if page := pageForShortcut(key.Rune()); page != "" {
		a.setActivePage(page)
		return true
	}

	switch key.Rune() {
	case 'r', 'R':
		go a.runAction(a.t("action.refresh"), func(context.Context) error {
			return a.reloadAll()
		})
		return true
	case 'l', 'L':
		go a.runAction(a.t("action.toggleLanguage"), a.toggleUILanguageAction)
		return true
	case 'a', 'A':
		if a.page == pageProfiles {
			go a.runAction(a.t("action.activateProfile"), a.activateProfileAction)
			return true
		}
		return false
	case 'e', 'E':
		if a.page == pageProfiles {
			a.openProfileQuickEditDialog()
			return true
		}
		return false
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
	a.footerStatus = fmt.Sprintf(a.t("status.page"), pageDisplayName(page))
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
		trimmed = tr(currentGlobalUILanguage(), "status.ready")
	}
	return fmt.Sprintf("%s | %s", trimmed, pageHint(page))
}

func pageHint(page string) string {
	lang := currentGlobalUILanguage()
	base := tr(lang, "hint.base")
	switch page {
	case pageProfiles:
		return fmt.Sprintf(tr(lang, "hint.profiles"), base)
	case pageSubscriptions:
		return fmt.Sprintf(tr(lang, "hint.subscriptions"), base)
	case pageNetwork:
		return fmt.Sprintf(tr(lang, "hint.network"), base)
	case pageSettings:
		return fmt.Sprintf(tr(lang, "hint.settings"), base)
	case pageLogs:
		return fmt.Sprintf(tr(lang, "hint.logs"), base)
	default:
		return fmt.Sprintf(tr(lang, "hint.dashboard"), base)
	}
}

func pageDisplayName(page string) string {
	lang := currentGlobalUILanguage()
	switch page {
	case pageDashboard:
		return tr(lang, "page.dashboard")
	case pageProfiles:
		return tr(lang, "page.profiles")
	case pageSubscriptions:
		return tr(lang, "page.subscriptions")
	case pageNetwork:
		return tr(lang, "page.network")
	case pageSettings:
		return tr(lang, "page.settings")
	case pageLogs:
		return tr(lang, "page.logs")
	default:
		return tr(lang, "page.dashboard")
	}
}

// tabBarText 生成紧凑的 Tab 导航条文本（超窄视口时替换帮助栏）。
// 格式: "[1]Dash  [2]Prof  ..." 当前页黑字青底高亮，其余灰色。
func (a *tuiApp) tabBarText() string {
	tabs := tuiPageTabs()
	parts := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		name := tabShortName(tab.key)
		if tab.key == a.page {
			parts = append(parts, fmt.Sprintf("[black:teal:b][%c]%s[-:-:-]", tab.shortcut, name))
		} else {
			parts = append(parts, fmt.Sprintf("[darkgray::-][%c]%s[-:-:-]", tab.shortcut, name))
		}
	}
	return strings.Join(parts, " ")
}

func tabShortName(page string) string {
	switch page {
	case pageDashboard:
		return "Dash"
	case pageProfiles:
		return "Prof"
	case pageSubscriptions:
		return "Subs"
	case pageNetwork:
		return "Net"
	case pageSettings:
		return "Set"
	case pageLogs:
		return "Logs"
	default:
		return page
	}
}

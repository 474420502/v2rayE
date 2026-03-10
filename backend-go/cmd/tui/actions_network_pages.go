package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const proxyUserSelectDialogPage = "proxy-user-select-dialog"

func mergeCSVUsers(base string, additions ...string) string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, 8)
	appendUser := func(value string) {
		name := strings.TrimSpace(value)
		if name == "" {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		ordered = append(ordered, name)
	}
	for _, item := range splitCommaStrings(base) {
		appendUser(item)
	}
	for _, item := range additions {
		appendUser(item)
	}
	return strings.Join(ordered, ",")
}

func (a *tuiApp) reloadOverviewAction(context.Context) error {
	return a.reloadOverview()
}

func (a *tuiApp) applySystemProxyAction(ctx context.Context) error {
	cfg := a.copyConfig()
	_, err := a.client.ApplySystemProxy(ctx, stringValue(cfg, "systemProxyMode"), stringValue(cfg, "systemProxyExceptions"))
	return err
}

func (a *tuiApp) clearSystemProxyAction(ctx context.Context) error {
	_, err := a.client.ApplySystemProxy(ctx, "forced_clear", "")
	return err
}

func (a *tuiApp) detectSystemProxyUsersAction(ctx context.Context) error {
	users, err := a.client.ListSystemProxyUsers(ctx)
	if err != nil {
		return err
	}
	a.storeSystemProxyUsers(users)
	if len(users) == 0 {
		a.setFooter(a.t("footer.proxy.detect.none"))
		a.refreshWidgets()
		return nil
	}
	current := strings.TrimSpace(a.settingsProxyUsers.Text())
	if current == "" {
		a.withSuspendedFieldTracking(func() {
			a.runUI(func(app *tview.Application) {
				a.settingsProxyUsers.SetText(users[0].Username, app)
			})
		})
		a.markSettingsDirty()
		a.setFooter(a.t("footer.proxy.detect.defaultAdded"))
	} else {
		a.setFooter(a.t("footer.proxy.detect.refreshed"))
	}
	a.refreshWidgets()
	return nil
}

func (a *tuiApp) useDesktopProxyUserAction(ctx context.Context) error {
	users := a.currentSystemProxyUsers()
	if len(users) == 0 {
		var err error
		users, err = a.client.ListSystemProxyUsers(ctx)
		if err != nil {
			return err
		}
		a.storeSystemProxyUsers(users)
	}
	if len(users) == 0 {
		return errors.New(a.t("error.proxy.noCandidates"))
	}
	updated := mergeCSVUsers(a.settingsProxyUsers.Text(), users[0].Username)
	a.withSuspendedFieldTracking(func() {
		a.runUI(func(app *tview.Application) {
			a.settingsProxyUsers.SetText(updated, app)
		})
	})
	a.markSettingsDirty()
	a.refreshWidgets()
	a.setFooter(a.tf("footer.proxy.addDesktop", users[0].Username))
	return nil
}

func (a *tuiApp) addNonSystemProxyUsersAction(ctx context.Context) error {
	users := a.currentSystemProxyUsers()
	if len(users) == 0 {
		var err error
		users, err = a.client.ListSystemProxyUsers(ctx)
		if err != nil {
			return err
		}
		a.storeSystemProxyUsers(users)
	}
	if len(users) == 0 {
		return errors.New(a.t("error.proxy.noCandidates"))
	}
	additions := make([]string, 0, len(users))
	for _, candidate := range users {
		if candidate.IsSystem {
			continue
		}
		additions = append(additions, candidate.Username)
	}
	if len(additions) == 0 {
		return errors.New(a.t("error.proxy.noNonSystemUsers"))
	}
	updated := mergeCSVUsers(a.settingsProxyUsers.Text(), additions...)
	a.withSuspendedFieldTracking(func() {
		a.runUI(func(app *tview.Application) {
			a.settingsProxyUsers.SetText(updated, app)
		})
	})
	a.markSettingsDirty()
	a.refreshWidgets()
	a.setFooter(a.tf("footer.proxy.addNonSystem", len(additions)))
	return nil
}

func (a *tuiApp) openProxyUserSelectDialogAction(ctx context.Context) error {
	users := a.currentSystemProxyUsers()
	if len(users) == 0 {
		fetched, err := a.client.ListSystemProxyUsers(ctx)
		if err != nil {
			return err
		}
		a.storeSystemProxyUsers(fetched)
	}
	a.openProxyUserSelectDialog()
	return nil
}

func (a *tuiApp) openProxyUserSelectDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || a.proxyUserSelectVisible.Load() || a.commandPaletteVisible.Load() || a.profileActionsVisible.Load() || a.profileDeleteVisible.Load() || a.profileImportVisible.Load() || a.profileEditVisible.Load() {
			return
		}

		users := a.currentSystemProxyUsers()
		if len(users) == 0 {
			a.setFooter(a.t("footer.proxy.select.noCandidates"))
			return
		}

		selected := map[string]bool{}
		for _, item := range splitCommaStrings(a.settingsProxyUsers.Text()) {
			selected[item] = true
		}

		list := newListWidget()
		list.ShowSecondaryText(true)
		list.SetBorder(true)
		list.SetTitle(" " + a.t("dialog.proxy.select.listTitle") + " ")

		var refreshRows func()
		refreshRows = func() {
			list.Clear()
			for _, user := range users {
				mark := "[ ]"
				if selected[user.Username] {
					mark = "[x]"
				}
				badge := a.t("dialog.proxy.select.badge.nonSystem")
				if user.IsSystem {
					badge = a.t("dialog.proxy.select.badge.system")
				}
				if user.HasSessionBus {
					badge += "," + a.t("dialog.proxy.select.badge.bus")
				}
				main := fmt.Sprintf("%s %s", mark, user.Username)
				secondary := fmt.Sprintf("uid=%d %s", user.UID, badge)
				username := user.Username
				list.AddItem(main, secondary, 0, func() {
					selected[username] = !selected[username]
					refreshRows()
				})
			}
		}
		refreshRows()

		applyBtn := tview.NewButton(a.t("dialog.common.apply"))
		cancelBtn := tview.NewButton(a.t("dialog.common.cancel"))
		clearBtn := tview.NewButton(a.t("dialog.common.clear"))
		for _, btn := range []*tview.Button{applyBtn, cancelBtn, clearBtn} {
			btn.SetLabelColor(tcell.ColorWhite)
			btn.SetLabelColorActivated(tcell.ColorBlack)
			btn.SetBackgroundColor(tcell.ColorDarkCyan)
			btn.SetBackgroundColorActivated(tcell.ColorYellow)
		}

		applyBtn.SetSelectedFunc(func() {
			ordered := make([]string, 0, len(users))
			for _, user := range users {
				if selected[user.Username] {
					ordered = append(ordered, user.Username)
				}
			}
			value := strings.Join(ordered, ",")
			a.withSuspendedFieldTracking(func() {
				a.settingsProxyUsers.SetText(value, app)
			})
			a.markSettingsDirty()
			a.closeProxyUserSelectDialog()
			a.refreshWidgets()
			a.setFooter(a.tf("footer.proxy.select.selectedCount", len(ordered)))
		})

		clearBtn.SetSelectedFunc(func() {
			for key := range selected {
				selected[key] = false
			}
			refreshRows()
		})

		cancelBtn.SetSelectedFunc(func() {
			a.closeProxyUserSelectDialog()
		})

		buttons := buttonRow(applyBtn, clearBtn, cancelBtn)
		container := tview.NewFlex().SetDirection(tview.FlexRow)
		container.SetBorder(true)
		container.SetTitle(" " + a.t("dialog.proxy.select.title") + " ")
		container.AddItem(newMutedText(a.t("dialog.proxy.select.description")), 1, 0, false)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(list, 0, 1, true)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(buttons, 1, 0, false)

		list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				a.closeProxyUserSelectDialog()
				return nil
			case tcell.KeyTAB, tcell.KeyRight:
				app.SetFocus(applyBtn)
				return nil
			case tcell.KeyRune:
				if event.Rune() == ' ' {
					index := list.GetCurrentItem()
					if index >= 0 && index < len(users) {
						name := users[index].Username
						selected[name] = !selected[name]
						refreshRows()
						if index < list.GetItemCount() {
							list.SetCurrentItem(index)
						}
					}
					return nil
				}
			}
			return event
		})

		a.proxyUserSelectMenu = list
		a.proxyUserSelectPrev = app.GetFocus()
		a.pageHolder.AddPage(proxyUserSelectDialogPage, centeredPrimitive(container, 90, 24), true, true)
		a.proxyUserSelectVisible.Store(true)
		app.SetFocus(list)
		a.setFooter(a.t("footer.proxy.select.hint"))
	})
}

func (a *tuiApp) closeProxyUserSelectDialog() {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || !a.proxyUserSelectVisible.Load() {
			return
		}
		a.pageHolder.RemovePage(proxyUserSelectDialogPage)
		a.proxyUserSelectVisible.Store(false)
		a.proxyUserSelectMenu = nil
		if a.proxyUserSelectPrev != nil {
			app.SetFocus(a.proxyUserSelectPrev)
		} else if len(a.focusables) > 0 {
			app.SetFocus(a.focusables[0])
		}
		a.proxyUserSelectPrev = nil
		a.setFooter(a.tf("status.page", pageDisplayName(a.page)))
	})
}

func (a *tuiApp) updateGeoDataAction(ctx context.Context) error {
	_, err := a.client.UpdateGeoData(ctx)
	if err == nil {
		return a.reloadNetwork()
	}
	return err
}

func (a *tuiApp) repairTunAction(ctx context.Context) error {
	result, err := a.client.RepairTun(ctx)
	if err == nil {
		a.setFooter(a.tf("footer.routing.repairResult", result.Running, result.TunTakeoverActive, result.TunDirectBypassRule, result.TunDirectBypassMark))
		return a.reloadNetwork()
	}
	return err
}

func (a *tuiApp) routeTestAction(ctx context.Context) error {
	target := strings.TrimSpace(a.networkTestTarget.Text())
	if target == "" {
		return errors.New(a.t("error.routing.emptyTarget"))
	}
	port := mustAtoiDefault(a.networkTestPort.Text(), 443)
	result, err := a.client.TestRouting(ctx, RoutingTestRequest{Target: target, Protocol: "tcp", Port: port})
	if err != nil {
		return err
	}
	a.storeRoutingTestResult(result)
	a.refreshWidgets()
	return nil
}

func (a *tuiApp) saveRoutingModeAction(ctx context.Context) error {
	mode := strings.ToLower(strings.TrimSpace(a.networkRoutingMode.Text()))
	if mode == "" {
		return errors.New(a.t("error.routing.emptyMode"))
	}
	switch mode {
	case "global", "bypass_cn", "direct", "custom":
	default:
		return errors.New(a.t("error.routing.invalidMode"))
	}

	routing := a.copyRouting()
	routing.Mode = mode
	strategy := strings.TrimSpace(a.networkDomainStrategy.Text())
	if strategy != "" {
		switch strategy {
		case "IPIfNonMatch", "IPOnDemand", "AsIs":
		default:
			return errors.New(a.t("error.routing.invalidDomainStrategy"))
		}
		routing.DomainStrategy = strategy
	}
	localBypassText := strings.TrimSpace(a.networkLocalBypass.Text())
	if localBypassText != "" {
		localBypass := parseBoolText(localBypassText)
		routing.LocalBypassEnabled = &localBypass
	}
	if _, err := a.client.UpdateRouting(ctx, routing); err != nil {
		return err
	}
	a.clearNetworkRoutingDirty()
	return a.reloadNetwork()
}

func (a *tuiApp) applyRoutingPresetToForm(preset string) {
	var (
		mode          string
		domainStrategy string
		localBypass    bool
		footerKey      string
	)

	switch strings.TrimSpace(preset) {
	case "global":
		mode = "global"
		domainStrategy = "IPIfNonMatch"
		localBypass = true
		footerKey = "footer.routing.preset.global"
	case "bypass_cn":
		mode = "bypass_cn"
		domainStrategy = "IPIfNonMatch"
		localBypass = true
		footerKey = "footer.routing.preset.bypassCN"
	case "direct":
		mode = "direct"
		domainStrategy = "AsIs"
		localBypass = true
		footerKey = "footer.routing.preset.direct"
	default:
		return
	}

	a.withSuspendedFieldTracking(func() {
		a.networkRoutingMode.SetText(mode, a.app)
		a.networkDomainStrategy.SetText(domainStrategy, a.app)
		a.networkLocalBypass.SetText(strconv.FormatBool(localBypass), a.app)
	})
	a.markNetworkRoutingDirty()
	a.setFooter(a.t(footerKey))
}

func (a *tuiApp) selectRoutingGlobalAction(context.Context) error {
	a.setNetworkRoutingMode("global")
	return nil
}

func (a *tuiApp) selectRoutingBypassCNAction(context.Context) error {
	a.setNetworkRoutingMode("bypass_cn")
	return nil
}

func (a *tuiApp) selectRoutingDirectAction(context.Context) error {
	a.setNetworkRoutingMode("direct")
	return nil
}

func (a *tuiApp) selectRoutingCustomAction(context.Context) error {
	a.setNetworkRoutingMode("custom")
	return nil
}

func (a *tuiApp) setNetworkRoutingMode(mode string) {
	a.markNetworkRoutingDirty()
	a.runUI(func(app *tview.Application) {
		a.networkRoutingMode.SetText(mode, app)
	})
	a.setFooter(a.tf("footer.routing.targetMode", mode))
}

func (a *tuiApp) saveConfigAction(ctx context.Context) error {
	base := a.copyConfig()
	payload := map[string]any{
		"listenAddr":            strings.TrimSpace(a.settingsListenAddr.Text()),
		"socksPort":             mustAtoiDefault(a.settingsSocksPort.Text(), intValue(base, "socksPort")),
		"httpPort":              mustAtoiDefault(a.settingsHTTPPort.Text(), intValue(base, "httpPort")),
		"uiLanguage":            currentGlobalUILanguage(),
		"coreEngine":            strings.TrimSpace(a.settingsCoreEngine.Text()),
		"logLevel":              strings.TrimSpace(a.settingsLogLevel.Text()),
		"skipCertVerify":        parseBoolText(a.settingsSkipCert.Text()),
		"tunName":               strings.TrimSpace(a.settingsTunName.Text()),
		"tunMode":               strings.TrimSpace(a.settingsTunMode.Text()),
		"tunMtu":                mustAtoiDefault(a.settingsTunMtu.Text(), intValue(base, "tunMtu")),
		"tunAutoRoute":          parseBoolText(a.settingsTunAutoRoute.Text()),
		"tunStrictRoute":        parseBoolText(a.settingsTunStrict.Text()),
		"dnsMode":               strings.TrimSpace(a.settingsDNSMode.Text()),
		"dnsList":               splitCSV(a.settingsDNSList.Text()),
		"systemProxyMode":       strings.TrimSpace(a.settingsProxyMode.Text()),
		"systemProxyExceptions": strings.TrimSpace(a.settingsProxyExcept.Text()),
		"systemProxyUsers":      splitCSV(a.settingsProxyUsers.Text()),
	}
	_, err := a.client.UpdateConfig(ctx, payload)
	if err == nil {
		a.clearSettingsDirty()
		return a.reloadAll()
	}
	return err
}

func (a *tuiApp) selectProxyModeForcedChangeAction(context.Context) error {
	a.setSettingsField(a.settingsProxyMode, "forced_change", "footer.settings.proxyMode", "forced_change")
	return nil
}

func (a *tuiApp) selectProxyModeForcedClearAction(context.Context) error {
	a.setSettingsField(a.settingsProxyMode, "forced_clear", "footer.settings.proxyMode", "forced_clear")
	return nil
}

func (a *tuiApp) selectProxyModePacAction(context.Context) error {
	a.setSettingsField(a.settingsProxyMode, "pac", "footer.settings.proxyMode", "pac")
	return nil
}

func (a *tuiApp) selectTunModeOffAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "off", "footer.settings.tunMode", "off")
	return nil
}

func (a *tuiApp) selectTunModeMixedAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "mixed", "footer.settings.tunMode", "mixed")
	return nil
}

func (a *tuiApp) selectTunModeSystemAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "system", "footer.settings.tunMode", "system")
	return nil
}

func (a *tuiApp) selectTunModeGvisorAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "gvisor", "footer.settings.tunMode", "gvisor")
	return nil
}

func (a *tuiApp) selectLogLevelDebugAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "debug", "footer.settings.logLevel", "debug")
	return nil
}

func (a *tuiApp) selectLogLevelInfoAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "info", "footer.settings.logLevel", "info")
	return nil
}

func (a *tuiApp) selectLogLevelWarningAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "warning", "footer.settings.logLevel", "warning")
	return nil
}

func (a *tuiApp) selectLogLevelErrorAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "error", "footer.settings.logLevel", "error")
	return nil
}

func (a *tuiApp) selectCoreEngineXrayAction(context.Context) error {
	a.setSettingsField(a.settingsCoreEngine, "xray-core", "footer.settings.coreEngine", "xray-core")
	return nil
}

func (a *tuiApp) selectDNSModeSystemAction(context.Context) error {
	a.setSettingsField(a.settingsDNSMode, "UseSystemDNS", "footer.settings.dnsMode", "UseSystemDNS")
	return nil
}

func (a *tuiApp) selectDNSModeListAction(context.Context) error {
	a.setSettingsField(a.settingsDNSMode, "UseDNSList", "footer.settings.dnsMode", "UseDNSList")
	return nil
}

func (a *tuiApp) selectDNSModeDirectAction(context.Context) error {
	a.setSettingsField(a.settingsDNSMode, "Direct", "footer.settings.dnsMode", "Direct")
	return nil
}

func (a *tuiApp) setSettingsField(field textSetter, value, footerKey string, footerArgs ...any) {
	a.markSettingsDirty()
	a.runUI(func(app *tview.Application) {
		field.SetText(value, app)
	})
	a.setFooter(a.tf(footerKey, footerArgs...))
}

func (a *tuiApp) clearCoreErrorAction(ctx context.Context) error {
	_, err := a.client.ClearCoreError(ctx)
	if err == nil {
		return a.reloadOverview()
	}
	return err
}

func (a *tuiApp) exitCleanupAction(ctx context.Context) error {
	_, err := a.client.ExitCleanup(ctx, false)
	if err == nil {
		return a.reloadAll()
	}
	return err
}

package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rivo/tview"
)

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

func (a *tuiApp) updateGeoDataAction(ctx context.Context) error {
	_, err := a.client.UpdateGeoData(ctx)
	if err == nil {
		return a.reloadNetwork()
	}
	return err
}

func (a *tuiApp) repairTunAction(ctx context.Context) error {
	_, err := a.client.RepairTun(ctx)
	if err == nil {
		return a.reloadNetwork()
	}
	return err
}

func (a *tuiApp) routeTestAction(ctx context.Context) error {
	target := strings.TrimSpace(a.networkTestTarget.Text())
	if target == "" {
		return fmt.Errorf("empty routing target")
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
		return fmt.Errorf("empty routing mode")
	}
	switch mode {
	case "global", "bypass_cn", "direct", "custom":
	default:
		return fmt.Errorf("invalid routing mode, use global|bypass_cn|direct|custom")
	}

	routing := a.copyRouting()
	routing.Mode = mode
	strategy := strings.TrimSpace(a.networkDomainStrategy.Text())
	if strategy != "" {
		switch strategy {
		case "IPIfNonMatch", "IPOnDemand", "AsIs":
		default:
			return fmt.Errorf("invalid domain strategy, use IPIfNonMatch|IPOnDemand|AsIs")
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

func (a *tuiApp) presetGlobalProxyAction(ctx context.Context) error {
	if err := a.applyRoutingPreset(ctx, "global", "IPIfNonMatch", true); err != nil {
		return err
	}
	_, err := a.client.ApplySystemProxy(ctx, "forced_change", stringValue(a.copyConfig(), "systemProxyExceptions"))
	if err != nil {
		return err
	}
	a.setFooter("Applied preset: Global + System Proxy ON")
	return a.reloadAll()
}

func (a *tuiApp) presetBypassCNProxyAction(ctx context.Context) error {
	if err := a.applyRoutingPreset(ctx, "bypass_cn", "IPIfNonMatch", true); err != nil {
		return err
	}
	_, err := a.client.ApplySystemProxy(ctx, "forced_change", stringValue(a.copyConfig(), "systemProxyExceptions"))
	if err != nil {
		return err
	}
	a.setFooter("Applied preset: BypassCN + System Proxy ON")
	return a.reloadAll()
}

func (a *tuiApp) presetDirectNoProxyAction(ctx context.Context) error {
	if err := a.applyRoutingPreset(ctx, "direct", "AsIs", true); err != nil {
		return err
	}
	if _, err := a.client.ApplySystemProxy(ctx, "forced_clear", ""); err != nil {
		return err
	}
	a.setFooter("Applied preset: Direct + System Proxy OFF")
	return a.reloadAll()
}

func (a *tuiApp) applyRoutingPreset(ctx context.Context, mode, domainStrategy string, localBypass bool) error {
	routing := a.copyRouting()
	routing.Mode = mode
	routing.DomainStrategy = domainStrategy
	routing.LocalBypassEnabled = &localBypass
	if _, err := a.client.UpdateRouting(ctx, routing); err != nil {
		return err
	}
	a.markNetworkRoutingDirty()
	a.runUI(func(app *tview.Application) {
		a.networkRoutingMode.SetText(mode, app)
		a.networkDomainStrategy.SetText(domainStrategy, app)
		a.networkLocalBypass.SetText(strconv.FormatBool(localBypass), app)
	})
	a.clearNetworkRoutingDirty()
	return nil
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
	a.setFooter("Routing target mode set to " + mode + ", click Save Routing to apply")
}

func (a *tuiApp) saveConfigAction(ctx context.Context) error {
	base := a.copyConfig()
	payload := map[string]any{
		"listenAddr":            strings.TrimSpace(a.settingsListenAddr.Text()),
		"socksPort":             mustAtoiDefault(a.settingsSocksPort.Text(), intValue(base, "socksPort")),
		"httpPort":              mustAtoiDefault(a.settingsHTTPPort.Text(), intValue(base, "httpPort")),
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
	}
	_, err := a.client.UpdateConfig(ctx, payload)
	if err == nil {
		a.clearSettingsDirty()
		return a.reloadAll()
	}
	return err
}

func (a *tuiApp) selectProxyModeForcedChangeAction(context.Context) error {
	a.setSettingsField(a.settingsProxyMode, "forced_change", "proxy target mode set to forced_change, click Save Config")
	return nil
}

func (a *tuiApp) selectProxyModeForcedClearAction(context.Context) error {
	a.setSettingsField(a.settingsProxyMode, "forced_clear", "proxy target mode set to forced_clear, click Save Config")
	return nil
}

func (a *tuiApp) selectProxyModePacAction(context.Context) error {
	a.setSettingsField(a.settingsProxyMode, "pac", "proxy target mode set to pac, click Save Config")
	return nil
}

func (a *tuiApp) selectTunModeOffAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "off", "tun mode set to off, click Save Config")
	return nil
}

func (a *tuiApp) selectTunModeMixedAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "mixed", "tun mode set to mixed, click Save Config")
	return nil
}

func (a *tuiApp) selectTunModeSystemAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "system", "tun mode set to system, click Save Config")
	return nil
}

func (a *tuiApp) selectTunModeGvisorAction(context.Context) error {
	a.setSettingsField(a.settingsTunMode, "gvisor", "tun mode set to gvisor, click Save Config")
	return nil
}

func (a *tuiApp) selectLogLevelDebugAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "debug", "log level set to debug, click Save Config")
	return nil
}

func (a *tuiApp) selectLogLevelInfoAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "info", "log level set to info, click Save Config")
	return nil
}

func (a *tuiApp) selectLogLevelWarningAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "warning", "log level set to warning, click Save Config")
	return nil
}

func (a *tuiApp) selectLogLevelErrorAction(context.Context) error {
	a.setSettingsField(a.settingsLogLevel, "error", "log level set to error, click Save Config")
	return nil
}

func (a *tuiApp) selectCoreEngineXrayAction(context.Context) error {
	a.setSettingsField(a.settingsCoreEngine, "xray-core", "core engine set to xray-core, click Save Config")
	return nil
}

func (a *tuiApp) setSettingsField(field textSetter, value, footer string) {
	a.markSettingsDirty()
	a.runUI(func(app *tview.Application) {
		field.SetText(value, app)
	})
	a.setFooter(footer)
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

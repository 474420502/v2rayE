package tui

import (
	"context"
	"fmt"
	"strings"
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

func (a *tuiApp) saveConfigAction(ctx context.Context) error {
	payload := map[string]any{
		"listenAddr":            strings.TrimSpace(a.settingsListenAddr.Text()),
		"socksPort":             mustAtoiDefault(a.settingsSocksPort.Text(), intValue(a.copyConfig(), "socksPort")),
		"httpPort":              mustAtoiDefault(a.settingsHTTPPort.Text(), intValue(a.copyConfig(), "httpPort")),
		"tunName":               strings.TrimSpace(a.settingsTunName.Text()),
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

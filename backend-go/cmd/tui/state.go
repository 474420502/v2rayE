package tui

import (
	"fmt"
	"sort"
	"strings"
)

func (a *tuiApp) outboundLabel(outbound string) string {
	switch strings.ToLower(strings.TrimSpace(outbound)) {
	case "total":
		return a.t("state.outbound.total")
	case "proxy":
		return a.t("state.outbound.proxy")
	case "direct":
		return a.t("state.outbound.direct")
	case "block":
		return a.t("state.outbound.block")
	default:
		return outbound
	}
}

func (a *tuiApp) formatDashboardSummary() string {
	return strings.Join([]string{
		a.formatDashboardStatus(),
		"",
		a.formatDashboardTelemetry(),
		"",
		a.formatDashboardConfig(),
	}, "\n")
}

func (a *tuiApp) formatDashboardStatus() string {
	selected := a.selectedProfileLocked()
	profileName := a.t("state.none")
	if selected != nil {
		profileName = selected.Name
	}

	return strings.Join([]string{
		fmt.Sprintf("  %s: %t", a.t("state.label.running"), a.status.Running),
		fmt.Sprintf("  %s: %s", a.t("state.label.state"), emptyFallback(a.status.State, a.t("state.unknown"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.engine"), emptyFallback(a.status.EngineResolved, emptyFallback(a.status.EngineMode, a.t("state.unknown")))),
		fmt.Sprintf("  %s: %s", a.t("state.label.currentProfile"), emptyFallback(a.status.CurrentProfileID, profileName)),
		fmt.Sprintf("  %s: %s", a.t("state.label.startedAt"), emptyFallback(a.status.StartedAt, a.t("state.na"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.uptime"), formatCoreUptime(a.status)),
		fmt.Sprintf("  %s: %s", a.t("state.label.error"), emptyFallback(a.status.Error, a.t("state.none"))),
	}, "\n")
}

func (a *tuiApp) formatDashboardTelemetry() string {
	lines := make([]string, 0, 8)
	hitsByOutbound := make(map[string]RoutingOutboundHit, len(a.hits.Items))
	total := RoutingOutboundHit{Outbound: "total"}
	for _, item := range a.hits.Items {
		hitsByOutbound[item.Outbound] = item
		total.UpBytes += item.UpBytes
		total.DownBytes += item.DownBytes
		total.UpSpeed += item.UpSpeed
		total.DownSpeed += item.DownSpeed
	}

	if len(a.hits.Items) > 0 {
		lines = append(lines, fmt.Sprintf("  %s: %s=%s | %s/s, %s=%s | %s/s",
			a.outboundLabel(total.Outbound),
			a.t("state.label.up"), humanBytes(total.UpBytes), humanBytes(total.UpSpeed),
			a.t("state.label.down"), humanBytes(total.DownBytes), humanBytes(total.DownSpeed),
		))
		ordered := []string{"proxy", "direct", "block"}
		for _, outbound := range ordered {
			item, ok := hitsByOutbound[outbound]
			if !ok {
				item = RoutingOutboundHit{Outbound: outbound}
			}
			lines = append(lines, fmt.Sprintf("  %s: %s=%s | %s/s, %s=%s | %s/s",
				a.outboundLabel(item.Outbound),
				a.t("state.label.up"), humanBytes(item.UpBytes), humanBytes(item.UpSpeed),
				a.t("state.label.down"), humanBytes(item.DownBytes), humanBytes(item.DownSpeed),
			))
			delete(hitsByOutbound, outbound)
		}
	} else {
		lines = append(lines,
			fmt.Sprintf("  %s: %s=%s | %s/s, %s=%s | %s/s",
				a.outboundLabel("proxy"),
				a.t("state.label.up"), humanBytes(a.stats.UpBytes), humanBytes(a.stats.UpSpeed),
				a.t("state.label.down"), humanBytes(a.stats.DownBytes), humanBytes(a.stats.DownSpeed),
			),
		)
	}

	if len(hitsByOutbound) > 0 {
		others := make([]string, 0, len(hitsByOutbound))
		for outbound := range hitsByOutbound {
			others = append(others, outbound)
		}
		sort.Strings(others)
		for _, outbound := range others {
			item := hitsByOutbound[outbound]
			lines = append(lines, fmt.Sprintf("  %s: %s=%s | %s/s, %s=%s | %s/s",
				a.outboundLabel(item.Outbound),
				a.t("state.label.up"), humanBytes(item.UpBytes), humanBytes(item.UpSpeed),
				a.t("state.label.down"), humanBytes(item.DownBytes), humanBytes(item.DownSpeed),
			))
		}
	}

	lines = append(lines,
		fmt.Sprintf("  %s: %t (%dms) %s", a.t("state.label.network"), a.availability.Available, a.availability.ElapsedMs, a.availability.Message),
		fmt.Sprintf("  %s: %s", a.t("state.label.logs"), emptyFallback(a.logsStreamState, a.t("state.idle"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.events"), emptyFallback(a.eventsStreamState, a.t("state.idle"))),
	)

	return strings.Join(lines, "\n")
}

func (a *tuiApp) formatDashboardConfig() string {
	return strings.Join([]string{
		fmt.Sprintf("  %s: %d", a.t("state.label.socksPort"), intValue(a.config, "socksPort")),
		fmt.Sprintf("  %s: %d", a.t("state.label.httpPort"), intValue(a.config, "httpPort")),
		fmt.Sprintf("  %s: %s", a.t("state.label.tunName"), emptyFallback(stringValue(a.config, "tunName"), a.t("state.unset"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.proxyMode"), emptyFallback(stringValue(a.config, "systemProxyMode"), a.t("state.unset"))),
		fmt.Sprintf("  %s: %d | %s: %d", a.t("state.label.profiles"), len(a.profiles), a.t("state.label.subscriptions"), len(a.subscriptions)),
	}, "\n")
}

func (a *tuiApp) formatSelectedProfile() string {
	p := a.selectedProfileLocked()
	if p == nil {
		if len(a.batchDelay.Results) == 0 {
			return a.t("state.profile.noSelection")
		}
		return strings.Join([]string{a.t("state.profile.noSelection"), "", a.formatBatchDelayResults()}, "\n")
	}

	lines := []string{
		fmt.Sprintf("%s: %s", a.t("state.label.name"), p.Name),
		fmt.Sprintf("%s: %s", a.t("state.label.id"), p.ID),
		fmt.Sprintf("%s: %s", a.t("state.label.protocol"), p.Protocol),
		fmt.Sprintf("%s: %s:%d", a.t("state.label.address"), p.Address, p.Port),
		fmt.Sprintf("%s: %dms", a.t("state.label.delay"), p.DelayMs),
		fmt.Sprintf("%s: %s (%s)", a.t("state.label.subscription"), emptyFallback(p.SubName, a.t("state.manual")), emptyFallback(p.SubID, a.t("state.none"))),
	}
	if p.Transport != nil {
		lines = append(lines,
			"",
			a.t("state.profile.transport"),
			fmt.Sprintf("  %s: %s", a.t("state.label.network"), p.Transport.Network),
			fmt.Sprintf("  %s: %t", a.t("state.label.tls"), p.Transport.TLS),
			fmt.Sprintf("  %s: %s", a.t("state.label.sni"), emptyFallback(p.Transport.SNI, a.t("state.none"))),
		)
	}
	if len(a.batchDelay.Results) > 0 || a.batchRunning {
		lines = append(lines, "", a.formatBatchDelayResults())
	}
	return strings.Join(lines, "\n")
}

func (a *tuiApp) formatBatchDelayStatus() string {
	if a.batchRunning {
		return a.t("state.batch.running")
	}
	if a.batchDelay.Total == 0 {
		return a.t("state.batch.idle")
	}
	return fmt.Sprintf(a.t("state.batch.completed"), a.batchDelay.Total, a.batchDelay.Success, a.batchDelay.Failed)
}

func (a *tuiApp) formatProfileEditStatus() string {
	if strings.TrimSpace(a.profileEditMessage) != "" {
		return a.profileEditMessage
	}
	if a.selectedProfileID == "" {
		return a.t("state.editor.selectFirst")
	}
	if !a.profileEditLoaded {
		return a.t("state.editor.loading")
	}
	if a.profileEditDirty {
		return a.t("state.editor.staged")
	}
	return a.t("state.editor.synced")
}

func (a *tuiApp) formatBatchDelayResults() string {
	if a.batchRunning {
		return a.t("state.batch.results.running")
	}
	if len(a.batchDelay.Results) == 0 {
		return a.t("state.batch.results.empty")
	}

	lines := []string{
		a.t("state.batch.title"),
		fmt.Sprintf("  %s=%d %s=%d %s=%d", a.t("state.label.total"), a.batchDelay.Total, a.t("state.label.success"), a.batchDelay.Success, a.t("state.label.failed"), a.batchDelay.Failed),
	}
	limit := len(a.batchDelay.Results)
	if limit > 8 {
		limit = 8
	}
	for _, result := range a.batchDelay.Results[:limit] {
		status := a.t("state.delay.fail")
		delayText := result.Error
		if result.Available {
			status = delayBucket(result.DelayMs)
			delayText = fmt.Sprintf("%dms", result.DelayMs)
		}
		lines = append(lines, fmt.Sprintf("  %-6s %-20s %s", status, truncateRunes(result.Name, 20), delayText))
	}
	if len(a.batchDelay.Results) > limit {
		lines = append(lines, fmt.Sprintf(a.t("state.batch.more"), len(a.batchDelay.Results)-limit))
	}
	return strings.Join(lines, "\n")
}

func (a *tuiApp) formatSelectedSubscription() string {
	sub := a.selectedSubscriptionLocked()
	if sub == nil {
		return a.t("state.sub.noSelection")
	}

	return strings.Join([]string{
		fmt.Sprintf("%s: %s", a.t("state.label.remarks"), sub.Remarks),
		fmt.Sprintf("%s: %s", a.t("state.label.id"), sub.ID),
		fmt.Sprintf("%s: %t", a.t("state.label.enabled"), sub.Enabled),
		fmt.Sprintf("%s: %d", a.t("state.label.profiles"), sub.ProfileCount),
		fmt.Sprintf("%s: %d %s", a.t("state.label.autoUpdate"), sub.AutoUpdateMinutes, a.t("state.minutes")),
		fmt.Sprintf("%s: %s", a.t("state.label.filter"), emptyFallback(sub.Filter, a.t("state.none"))),
		fmt.Sprintf("%s: %s", a.t("state.label.convertTarget"), emptyFallback(sub.ConvertTarget, a.t("state.default"))),
		fmt.Sprintf("%s: %s", a.t("state.label.updatedAt"), emptyFallback(sub.UpdatedAt, a.t("state.never"))),
		"",
		fmt.Sprintf("%s: %s", a.t("state.label.url"), sub.URL),
	}, "\n")
}

func (a *tuiApp) formatNetworkSummary() string {
	targetMode := strings.ToLower(strings.TrimSpace(a.networkRoutingMode.Text()))
	if targetMode == "" {
		targetMode = a.routing.Mode
	}
	targetStrategy := strings.TrimSpace(a.networkDomainStrategy.Text())
	if targetStrategy == "" {
		targetStrategy = a.routing.DomainStrategy
	}
	currentLocalBypass := true
	if a.routing.LocalBypassEnabled != nil {
		currentLocalBypass = *a.routing.LocalBypassEnabled
	}
	targetLocalBypass := currentLocalBypass
	localBypassText := strings.TrimSpace(a.networkLocalBypass.Text())
	if localBypassText != "" {
		targetLocalBypass = parseBoolText(localBypassText)
	}
	currentPreset := a.routingPresetLabel(routingPresetKey(a.routing))
	targetPreset := a.routingPresetLabel(a.targetRoutingPresetWithPending(strings.TrimSpace(a.networkPresetApplied)))
	lines := []string{
		a.t("state.network.availability"),
		fmt.Sprintf("  %s: %t", a.t("state.label.available"), a.availability.Available),
		fmt.Sprintf("  %s: %dms", a.t("state.label.elapsed"), a.availability.ElapsedMs),
		fmt.Sprintf("  %s: %s", a.t("state.label.message"), emptyFallback(a.availability.Message, a.t("state.none"))),
		"",
		a.t("state.network.routing"),
		fmt.Sprintf("  %s: %s", a.t("state.label.currentPreset"), currentPreset),
		fmt.Sprintf("  %s: %s", a.t("state.label.targetPreset"), targetPreset),
		fmt.Sprintf("  %s: %s", a.t("state.label.currentMode"), a.routing.Mode),
		fmt.Sprintf("  %s: %s", a.t("state.label.targetMode"), targetMode),
		fmt.Sprintf("  %s: %s", a.t("state.label.currentDomainStrategy"), a.routing.DomainStrategy),
		fmt.Sprintf("  %s: %s", a.t("state.label.targetDomainStrategy"), targetStrategy),
		fmt.Sprintf("  %s: %t", a.t("state.label.currentLocalBypass"), currentLocalBypass),
		fmt.Sprintf("  %s: %t", a.t("state.label.targetLocalBypass"), targetLocalBypass),
		fmt.Sprintf("  %s: %d", a.t("state.label.diagnosticsRuleCount"), a.diagnostics.RuleCount),
		fmt.Sprintf("  %s: %t", a.t("state.label.tunEnabled"), a.diagnostics.TunEnabled),
		fmt.Sprintf("  %s: %t", a.t("state.label.tunTakeoverActive"), a.diagnostics.TunTakeoverActive),
		fmt.Sprintf("  %s: %t", a.t("state.label.tunDirectBypassRule"), a.diagnostics.TunDirectBypassRule),
		fmt.Sprintf("  %s: %d", a.t("state.label.tunDirectBypassMark"), a.diagnostics.TunDirectBypassMark),
		fmt.Sprintf("  %s: %s", a.t("state.label.defaultRouteDevice"), emptyFallback(a.diagnostics.DefaultRouteDevice, a.t("state.unknown"))),
		fmt.Sprintf("  %s: %t", a.t("state.label.hasGeoIP"), a.diagnostics.HasGeoIP),
		fmt.Sprintf("  %s: %t", a.t("state.label.hasGeoSite"), a.diagnostics.HasGeoSite),
		fmt.Sprintf("  %s: %t", a.t("state.label.geodataAvailable"), a.diagnostics.GeoDataAvailable),
	}
	if len(a.hits.Items) > 0 {
		lines = append(lines, "", a.t("state.network.outboundHits"))
		for _, item := range a.hits.Items {
			lines = append(lines, fmt.Sprintf("  %s %s=%s/s %s=%s/s", a.outboundLabel(item.Outbound), a.t("state.label.up"), humanBytes(item.UpSpeed), a.t("state.label.down"), humanBytes(item.DownSpeed)))
		}
	}
	if a.diagnostics.Warning != "" {
		lines = append(lines, "", a.t("state.warning"), "  "+a.diagnostics.Warning)
	}
	return strings.Join(lines, "\n")
}

func routingPresetKey(routing RoutingConfig) string {
	localBypass := true
	if routing.LocalBypassEnabled != nil {
		localBypass = *routing.LocalBypassEnabled
	}
	if len(routing.Rules) > 0 {
		return ""
	}
	switch routing.Mode {
	case "global":
		if routing.DomainStrategy == "IPIfNonMatch" && localBypass {
			return "global"
		}
	case "bypass_cn":
		if routing.DomainStrategy == "IPIfNonMatch" && localBypass {
			return "bypass_cn"
		}
	case "direct":
		if routing.DomainStrategy == "AsIs" && localBypass {
			return "direct"
		}
	}
	return ""
}

func (a *tuiApp) targetRoutingPreset() string {
	return a.targetRoutingPresetWithPending(strings.TrimSpace(a.pendingNetworkPreset()))
}

func (a *tuiApp) targetRoutingPresetWithPending(preset string) string {
	if preset != "" {
		return preset
	}
	targetMode := strings.ToLower(strings.TrimSpace(a.networkRoutingMode.Text()))
	if targetMode == "" {
		targetMode = a.routing.Mode
	}
	targetStrategy := strings.TrimSpace(a.networkDomainStrategy.Text())
	if targetStrategy == "" {
		targetStrategy = a.routing.DomainStrategy
	}
	targetLocalBypass := true
	if a.routing.LocalBypassEnabled != nil {
		targetLocalBypass = *a.routing.LocalBypassEnabled
	}
	if localBypassText := strings.TrimSpace(a.networkLocalBypass.Text()); localBypassText != "" {
		targetLocalBypass = parseBoolText(localBypassText)
	}
	if len(a.routing.Rules) > 0 {
		return ""
	}
	return routingPresetKey(RoutingConfig{
		Mode:           targetMode,
		DomainStrategy: targetStrategy,
		LocalBypassEnabled: func(v bool) *bool {
			return &v
		}(targetLocalBypass),
	})
}

func (a *tuiApp) routingPresetLabel(preset string) string {
	switch preset {
	case "global":
		return a.t("dropdown.routing.preset.global")
	case "bypass_cn":
		return a.t("dropdown.routing.preset.bypassCN")
	case "direct":
		return a.t("dropdown.routing.preset.direct")
	default:
		return a.t("state.none")
	}
}

func (a *tuiApp) formatRoutingTestResult() string {
	if strings.TrimSpace(a.routingTest.Target) == "" {
		return a.t("state.routingTest.empty")
	}
	return strings.Join([]string{
		a.t("state.routingTest.title"),
		fmt.Sprintf("  %s: %s", a.t("state.label.target"), a.routingTest.Target),
		fmt.Sprintf("  %s: %s", a.t("state.label.type"), emptyFallback(a.routingTest.Type, a.t("state.unknown"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.inboundTag"), emptyFallback(a.routingTest.InboundTag, a.t("state.none"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.protocol"), emptyFallback(a.routingTest.Protocol, "tcp")),
		fmt.Sprintf("  %s: %d", a.t("state.label.port"), a.routingTest.Port),
		fmt.Sprintf("  %s: %s", a.t("state.label.matchedRule"), emptyFallback(a.routingTest.MatchedRule, a.t("state.default"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.matchedValue"), emptyFallback(a.routingTest.MatchedValue, a.t("state.none"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.resolvedIps"), emptyFallback(strings.Join(a.routingTest.ResolvedIPs, ","), a.t("state.none"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.outbound"), a.outboundLabel(emptyFallback(a.routingTest.Outbound, "proxy"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.action"), a.outboundLabel(emptyFallback(a.routingTest.Action, "proxy"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.note"), emptyFallback(a.routingTest.Note, a.t("state.none"))),
	}, "\n")
}

func (a *tuiApp) formatLogsStatus() string {
	total := len(a.logLines)
	filtered := a.filteredLogLines()
	errorCount := 0
	warnCount := 0
	for _, line := range filtered {
		switch normalizeLogLevel(line.Level) {
		case "error":
			errorCount++
		case "warning":
			warnCount++
		}
	}
	search := a.logSearchQuery
	if search == "" {
		search = a.t("state.none")
	}
	return fmt.Sprintf(a.t("state.logs.status"), total, len(filtered), a.logLevelFilter, a.logSourceFilter, search, errorCount, warnCount, emptyFallback(a.logsStreamState, a.t("state.idle")))
}

func (a *tuiApp) formatFilteredLogs() string {
	filtered := a.filteredLogLines()
	if len(filtered) == 0 {
		return a.t("state.logs.empty")
	}
	lines := make([]string, 0, len(filtered))
	for _, line := range filtered {
		lines = append(lines, formatHighlightedLogLine(line, a.logSearchQuery))
	}
	return strings.Join(lines, "\n")
}

func (a *tuiApp) filteredLogLines() []LogLine {
	filtered := make([]LogLine, 0, len(a.logLines))
	query := strings.ToLower(strings.TrimSpace(a.logSearchQuery))
	for _, line := range a.logLines {
		if !matchesLogLevelFilter(normalizeLogLevel(line.Level), a.logLevelFilter) {
			continue
		}
		if !matchesLogSourceFilter(normalizeLogSource(line.Source), a.logSourceFilter) {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(line.Timestamp + " " + line.Level + " " + line.Source + " " + line.Message)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func matchesLogLevelFilter(level, filter string) bool {
	filter = normalizeLogLevel(filter)
	if filter == "all" || filter == "" {
		return true
	}
	return level == filter
}

func matchesLogSourceFilter(source, filter string) bool {
	filter = normalizeLogSource(filter)
	if filter == "all" || filter == "" {
		return true
	}
	return source == filter
}

func normalizeLogSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "app":
		return "app"
	case "core", "xray", "xray-core":
		return "xray-core"
	default:
		return "unknown"
	}
}

func normalizeLogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "warn", "warning":
		return "warning"
	case "err", "error":
		return "error"
	case "debug":
		return "debug"
	default:
		return "info"
	}
}

func (a *tuiApp) formatSettingsSummary() string {
	return strings.Join([]string{
		a.t("state.settings.title"),
		fmt.Sprintf("  %s: %t", a.t("state.label.stagedChanges"), a.settingsDirty),
		fmt.Sprintf("  %s: %s", a.t("state.label.listenAddr"), emptyFallback(stringValue(a.config, "listenAddr"), a.t("state.defaultListenAddr"))),
		fmt.Sprintf("  %s: %d", a.t("state.label.socksPort"), intValue(a.config, "socksPort")),
		fmt.Sprintf("  %s: %d", a.t("state.label.httpPort"), intValue(a.config, "httpPort")),
		fmt.Sprintf("  %s: %s", a.t("state.label.coreEngine"), emptyFallback(stringValue(a.config, "coreEngine"), "xray-core")),
		fmt.Sprintf("  %s: %s", a.t("state.label.logLevel"), emptyFallback(stringValue(a.config, "logLevel"), "warning")),
		fmt.Sprintf("  %s: %t", a.t("state.label.skipCertVerify"), boolValue(a.config, "skipCertVerify")),
		fmt.Sprintf("  %s: %t", a.t("state.label.enableTun"), boolValue(a.config, "enableTun")),
		fmt.Sprintf("  %s: %s", a.t("state.label.tunMode"), emptyFallback(stringValue(a.config, "tunMode"), "off")),
		fmt.Sprintf("  %s: %s", a.t("state.label.tunName"), emptyFallback(stringValue(a.config, "tunName"), a.t("state.unset"))),
		fmt.Sprintf("  %s: %d", a.t("state.label.tunMtu"), intValue(a.config, "tunMtu")),
		fmt.Sprintf("  %s: %t", a.t("state.label.tunAutoRoute"), boolValue(a.config, "tunAutoRoute")),
		fmt.Sprintf("  %s: %t", a.t("state.label.tunStrictRoute"), boolValue(a.config, "tunStrictRoute")),
		fmt.Sprintf("  %s: %s", a.t("state.label.dnsMode"), emptyFallback(stringValue(a.config, "dnsMode"), "UseSystemDNS")),
		fmt.Sprintf("  %s: %s", a.t("state.label.dnsList"), emptyFallback(strings.Join(toStringSlice(a.config["dnsList"]), ","), a.t("state.none"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.systemProxyMode"), emptyFallback(stringValue(a.config, "systemProxyMode"), a.t("state.unset"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.localProxyMode"), emptyFallback(stringValue(a.config, "localProxyMode"), "follow-routing")),
		fmt.Sprintf("  %s: %s", a.t("state.label.systemProxyExceptions"), emptyFallback(stringValue(a.config, "systemProxyExceptions"), a.t("state.none"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.systemProxyUsers"), emptyFallback(strings.Join(toStringSlice(a.config["systemProxyUsers"]), ","), a.t("state.autoDetect"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.proxyUserCandidates"), emptyFallback(a.proxyUsersStatus, a.t("state.notLoaded"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.proxyUserPreview"), emptyFallback(a.proxyUserCandidatesPreviewLocked(), a.t("state.none"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.engineMode"), emptyFallback(a.status.EngineMode, a.t("state.unknown"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.coreResolved"), emptyFallback(a.status.EngineResolved, a.t("state.unknown"))),
		fmt.Sprintf("  %s: %s", a.t("state.label.coreUptime"), formatCoreUptime(a.status)),
	}, "\n")
}

func (a *tuiApp) proxyUserCandidatesPreview() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.proxyUserCandidatesPreviewLocked()
}

func (a *tuiApp) proxyUserCandidatesPreviewLocked() string {
	if len(a.systemProxyUsers) == 0 {
		return ""
	}
	parts := make([]string, 0, 4)
	limit := 4
	if len(a.systemProxyUsers) < limit {
		limit = len(a.systemProxyUsers)
	}
	for _, candidate := range a.systemProxyUsers[:limit] {
		tag := candidate.Username
		if candidate.HasSessionBus {
			tag += "(bus)"
		}
		if candidate.IsSystem {
			tag += "(sys)"
		}
		parts = append(parts, tag)
	}
	if len(a.systemProxyUsers) > limit {
		parts = append(parts, fmt.Sprintf("+%d", len(a.systemProxyUsers)-limit))
	}
	return strings.Join(parts, ",")
}

func formatCoreUptime(status CoreStatus) string {
	if !status.Running || status.UptimeSec <= 0 {
		return tr(currentGlobalUILanguage(), "state.stopped")
	}
	return humanDurationSeconds(status.UptimeSec)
}

func (a *tuiApp) selectedProfile() *ProfileItem {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.selectedProfileLocked()
}

func (a *tuiApp) selectedProfileLocked() *ProfileItem {
	for idx := range a.profiles {
		if a.profiles[idx].ID == a.selectedProfileID {
			profile := a.profiles[idx]
			return &profile
		}
	}
	return nil
}

func (a *tuiApp) selectedSubscription() *SubscriptionItem {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.selectedSubscriptionLocked()
}

func (a *tuiApp) selectedSubscriptionLocked() *SubscriptionItem {
	for idx := range a.subscriptions {
		if a.subscriptions[idx].ID == a.selectedSubID {
			subscription := a.subscriptions[idx]
			return &subscription
		}
	}
	return nil
}

func (a *tuiApp) currentProfileID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.selectedProfileID
}

func (a *tuiApp) currentSubscriptionID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.selectedSubID
}

func (a *tuiApp) subscriptionsSnapshot() []SubscriptionItem {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]SubscriptionItem(nil), a.subscriptions...)
}

func (a *tuiApp) profileIDsSnapshot() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.profiles))
	for _, profile := range a.profiles {
		ids = append(ids, profile.ID)
	}
	return ids
}

func (a *tuiApp) profileLabel(profile ProfileItem) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.profileLabelLocked(profile)
}

func (a *tuiApp) profileLabelLocked(profile ProfileItem) string {
	prefix := "  "
	if profile.ID == a.status.CurrentProfileID {
		prefix = "> "
	} else if profile.ID == a.selectedProfileID {
		prefix = "* "
	}
	return fmt.Sprintf("%s%s [%s] %s:%d %dms", prefix, profile.Name, profile.Protocol, profile.Address, profile.Port, profile.DelayMs)
}

func (a *tuiApp) sortedProfilesForDisplay() []ProfileItem {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sortedProfilesForDisplayLocked()
}

func (a *tuiApp) sortedProfilesForDisplayLocked() []ProfileItem {
	profiles := append([]ProfileItem(nil), a.profiles...)
	if len(a.batchDelay.Results) == 0 {
		return profiles
	}

	byID := make(map[string]ProfileDelayResult, len(a.batchDelay.Results))
	for _, result := range a.batchDelay.Results {
		byID[result.ProfileID] = result
	}

	sort.SliceStable(profiles, func(i, j int) bool {
		left, lok := byID[profiles[i].ID]
		right, rok := byID[profiles[j].ID]
		if lok != rok {
			return lok
		}
		if !lok {
			return strings.ToLower(profiles[i].Name) < strings.ToLower(profiles[j].Name)
		}
		if left.Available != right.Available {
			return left.Available
		}
		if left.Available && left.DelayMs != right.DelayMs {
			return left.DelayMs < right.DelayMs
		}
		return strings.ToLower(profiles[i].Name) < strings.ToLower(profiles[j].Name)
	})
	return profiles
}

func delayBucket(delayMs int) string {
	switch {
	case delayMs < 100:
		return tr(currentGlobalUILanguage(), "state.delay.fast")
	case delayMs < 300:
		return tr(currentGlobalUILanguage(), "state.delay.mid")
	default:
		return tr(currentGlobalUILanguage(), "state.delay.slow")
	}
}

func truncateRunes(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func (a *tuiApp) copyConfig() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	clone := make(map[string]any, len(a.config))
	for key, value := range a.config {
		clone[key] = value
	}
	return clone
}

func (a *tuiApp) copyRouting() RoutingConfig {
	a.mu.Lock()
	defer a.mu.Unlock()

	routing := a.routing
	rules := make([]RoutingRule, len(a.routing.Rules))
	for i, rule := range a.routing.Rules {
		rules[i] = rule
		rules[i].Values = append([]string(nil), rule.Values...)
	}
	routing.Rules = rules
	return routing
}

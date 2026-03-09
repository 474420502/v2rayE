package tui

import (
	"fmt"
	"sort"
	"strings"
)

func (a *tuiApp) formatDashboardSummary() string {
	selected := a.selectedProfile()
	profileName := "none"
	if selected != nil {
		profileName = selected.Name
	}

	return strings.Join([]string{
		"Core",
		fmt.Sprintf("  running: %t", a.status.Running),
		fmt.Sprintf("  state: %s", emptyFallback(a.status.State, "unknown")),
		fmt.Sprintf("  engine: %s", emptyFallback(a.status.EngineResolved, emptyFallback(a.status.EngineMode, "unknown"))),
		fmt.Sprintf("  currentProfile: %s", emptyFallback(a.status.CurrentProfileID, profileName)),
		fmt.Sprintf("  startedAt: %s", emptyFallback(a.status.StartedAt, "n/a")),
		fmt.Sprintf("  uptime: %s", formatCoreUptime(a.status)),
		fmt.Sprintf("  error: %s", emptyFallback(a.status.Error, "none")),
		"",
		"Telemetry",
		fmt.Sprintf("  up: %s total | %s/s", humanBytes(a.stats.UpBytes), humanBytes(a.stats.UpSpeed)),
		fmt.Sprintf("  down: %s total | %s/s", humanBytes(a.stats.DownBytes), humanBytes(a.stats.DownSpeed)),
		fmt.Sprintf("  network: %t (%dms) %s", a.availability.Available, a.availability.ElapsedMs, a.availability.Message),
		"",
		"Streams",
		fmt.Sprintf("  logs: %s", emptyFallback(a.logsStreamState, "idle")),
		fmt.Sprintf("  events: %s", emptyFallback(a.eventsStreamState, "idle")),
		"",
		"Config",
		fmt.Sprintf("  socksPort: %d", intValue(a.config, "socksPort")),
		fmt.Sprintf("  httpPort: %d", intValue(a.config, "httpPort")),
		fmt.Sprintf("  tunName: %s", emptyFallback(stringValue(a.config, "tunName"), "unset")),
		fmt.Sprintf("  proxyMode: %s", emptyFallback(stringValue(a.config, "systemProxyMode"), "unset")),
		fmt.Sprintf("  profiles: %d | subscriptions: %d", len(a.profiles), len(a.subscriptions)),
	}, "\n")
}

func (a *tuiApp) formatSelectedProfile() string {
	p := a.selectedProfile()
	if p == nil {
		if len(a.batchDelay.Results) == 0 {
			return "No profile selected."
		}
		return strings.Join([]string{"No profile selected.", "", a.formatBatchDelayResults()}, "\n")
	}

	lines := []string{
		fmt.Sprintf("Name: %s", p.Name),
		fmt.Sprintf("ID: %s", p.ID),
		fmt.Sprintf("Protocol: %s", p.Protocol),
		fmt.Sprintf("Address: %s:%d", p.Address, p.Port),
		fmt.Sprintf("Delay: %dms", p.DelayMs),
		fmt.Sprintf("Subscription: %s (%s)", emptyFallback(p.SubName, "manual"), emptyFallback(p.SubID, "none")),
	}
	if p.Transport != nil {
		lines = append(lines,
			"",
			"Transport",
			fmt.Sprintf("  network: %s", p.Transport.Network),
			fmt.Sprintf("  tls: %t", p.Transport.TLS),
			fmt.Sprintf("  sni: %s", emptyFallback(p.Transport.SNI, "none")),
		)
	}
	if len(a.batchDelay.Results) > 0 || a.batchRunning {
		lines = append(lines, "", a.formatBatchDelayResults())
	}
	return strings.Join(lines, "\n")
}

func (a *tuiApp) formatBatchDelayStatus() string {
	if a.batchRunning {
		return "Batch delay test running for all profiles..."
	}
	if a.batchDelay.Total == 0 {
		return "Batch delay test idle. Use Batch Delay to measure all profiles."
	}
	return fmt.Sprintf("Batch delay completed: total=%d success=%d failed=%d", a.batchDelay.Total, a.batchDelay.Success, a.batchDelay.Failed)
}

func (a *tuiApp) formatProfileEditStatus() string {
	if strings.TrimSpace(a.profileEditMessage) != "" {
		return a.profileEditMessage
	}
	if a.selectedProfileID == "" {
		return "Profile editor: select a profile first."
	}
	if !a.profileEditLoaded {
		return "Profile editor: loading selected profile..."
	}
	if a.profileEditDirty {
		return "Profile editor: staged changes (not saved)."
	}
	return "Profile editor: synced with selected profile."
}

func (a *tuiApp) formatBatchDelayResults() string {
	if a.batchRunning {
		return "Batch Delay\n  running..."
	}
	if len(a.batchDelay.Results) == 0 {
		return "Batch Delay\n  no results yet"
	}

	lines := []string{
		"Batch Delay",
		fmt.Sprintf("  total=%d success=%d failed=%d", a.batchDelay.Total, a.batchDelay.Success, a.batchDelay.Failed),
	}
	limit := len(a.batchDelay.Results)
	if limit > 8 {
		limit = 8
	}
	for _, result := range a.batchDelay.Results[:limit] {
		status := "fail"
		delayText := result.Error
		if result.Available {
			status = delayBucket(result.DelayMs)
			delayText = fmt.Sprintf("%dms", result.DelayMs)
		}
		lines = append(lines, fmt.Sprintf("  %-6s %-20s %s", status, truncateRunes(result.Name, 20), delayText))
	}
	if len(a.batchDelay.Results) > limit {
		lines = append(lines, fmt.Sprintf("  ... and %d more", len(a.batchDelay.Results)-limit))
	}
	return strings.Join(lines, "\n")
}

func (a *tuiApp) formatSelectedSubscription() string {
	sub := a.selectedSubscription()
	if sub == nil {
		return "No subscription selected."
	}

	return strings.Join([]string{
		fmt.Sprintf("Remarks: %s", sub.Remarks),
		fmt.Sprintf("ID: %s", sub.ID),
		fmt.Sprintf("Enabled: %t", sub.Enabled),
		fmt.Sprintf("Profiles: %d", sub.ProfileCount),
		fmt.Sprintf("Auto Update: %d minutes", sub.AutoUpdateMinutes),
		fmt.Sprintf("Filter: %s", emptyFallback(sub.Filter, "none")),
		fmt.Sprintf("Convert Target: %s", emptyFallback(sub.ConvertTarget, "default")),
		fmt.Sprintf("Updated At: %s", emptyFallback(sub.UpdatedAt, "never")),
		"",
		fmt.Sprintf("URL: %s", sub.URL),
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
	lines := []string{
		"Availability",
		fmt.Sprintf("  available: %t", a.availability.Available),
		fmt.Sprintf("  elapsed: %dms", a.availability.ElapsedMs),
		fmt.Sprintf("  message: %s", emptyFallback(a.availability.Message, "none")),
		"",
		"Routing",
		fmt.Sprintf("  currentMode: %s", a.routing.Mode),
		fmt.Sprintf("  targetMode: %s", targetMode),
		fmt.Sprintf("  currentDomainStrategy: %s", a.routing.DomainStrategy),
		fmt.Sprintf("  targetDomainStrategy: %s", targetStrategy),
		fmt.Sprintf("  currentLocalBypass: %t", currentLocalBypass),
		fmt.Sprintf("  targetLocalBypass: %t", targetLocalBypass),
		fmt.Sprintf("  diagnostics ruleCount: %d", a.diagnostics.RuleCount),
		fmt.Sprintf("  tunEnabled: %t", a.diagnostics.TunEnabled),
		fmt.Sprintf("  tunTakeoverActive: %t", a.diagnostics.TunTakeoverActive),
		fmt.Sprintf("  defaultRouteDevice: %s", emptyFallback(a.diagnostics.DefaultRouteDevice, "unknown")),
		fmt.Sprintf("  hasGeoIP: %t", a.diagnostics.HasGeoIP),
		fmt.Sprintf("  hasGeoSite: %t", a.diagnostics.HasGeoSite),
		fmt.Sprintf("  geodataAvailable: %t", a.diagnostics.GeoDataAvailable),
	}
	if len(a.hits.Items) > 0 {
		lines = append(lines, "", "Outbound Hits")
		for _, item := range a.hits.Items {
			lines = append(lines, fmt.Sprintf("  %s up=%s/s down=%s/s", item.Outbound, humanBytes(item.UpSpeed), humanBytes(item.DownSpeed)))
		}
	}
	if a.diagnostics.Warning != "" {
		lines = append(lines, "", "Warning", "  "+a.diagnostics.Warning)
	}
	return strings.Join(lines, "\n")
}

func (a *tuiApp) formatRoutingTestResult() string {
	if strings.TrimSpace(a.routingTest.Target) == "" {
		return "Routing Test\n  No routing test executed."
	}
	return strings.Join([]string{
		"Routing Test",
		fmt.Sprintf("  target: %s", a.routingTest.Target),
		fmt.Sprintf("  type: %s", emptyFallback(a.routingTest.Type, "unknown")),
		fmt.Sprintf("  protocol: %s", emptyFallback(a.routingTest.Protocol, "tcp")),
		fmt.Sprintf("  port: %d", a.routingTest.Port),
		fmt.Sprintf("  matchedRule: %s", emptyFallback(a.routingTest.MatchedRule, "default")),
		fmt.Sprintf("  matchedValue: %s", emptyFallback(a.routingTest.MatchedValue, "none")),
		fmt.Sprintf("  outbound: %s", emptyFallback(a.routingTest.Outbound, "proxy")),
		fmt.Sprintf("  action: %s", emptyFallback(a.routingTest.Action, "proxy")),
		fmt.Sprintf("  note: %s", emptyFallback(a.routingTest.Note, "none")),
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
		search = "none"
	}
	return fmt.Sprintf("Logs total=%d shown=%d level=%s source=%s search=%s errors=%d warns=%d stream=%s", total, len(filtered), a.logLevelFilter, a.logSourceFilter, search, errorCount, warnCount, emptyFallback(a.logsStreamState, "idle"))
}

func (a *tuiApp) formatFilteredLogs() string {
	filtered := a.filteredLogLines()
	if len(filtered) == 0 {
		return "No logs matched the current filter."
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
		"Current config snapshot",
		fmt.Sprintf("  stagedChanges: %t", a.settingsDirty),
		fmt.Sprintf("  listenAddr: %s", emptyFallback(stringValue(a.config, "listenAddr"), "127.0.0.1:8080")),
		fmt.Sprintf("  socksPort: %d", intValue(a.config, "socksPort")),
		fmt.Sprintf("  httpPort: %d", intValue(a.config, "httpPort")),
		fmt.Sprintf("  coreEngine: %s", emptyFallback(stringValue(a.config, "coreEngine"), "xray-core")),
		fmt.Sprintf("  logLevel: %s", emptyFallback(stringValue(a.config, "logLevel"), "warning")),
		fmt.Sprintf("  skipCertVerify: %t", boolValue(a.config, "skipCertVerify")),
		fmt.Sprintf("  enableTun: %t", boolValue(a.config, "enableTun")),
		fmt.Sprintf("  tunMode: %s", emptyFallback(stringValue(a.config, "tunMode"), "off")),
		fmt.Sprintf("  tunName: %s", emptyFallback(stringValue(a.config, "tunName"), "unset")),
		fmt.Sprintf("  tunMtu: %d", intValue(a.config, "tunMtu")),
		fmt.Sprintf("  tunAutoRoute: %t", boolValue(a.config, "tunAutoRoute")),
		fmt.Sprintf("  tunStrictRoute: %t", boolValue(a.config, "tunStrictRoute")),
		fmt.Sprintf("  dnsMode: %s", emptyFallback(stringValue(a.config, "dnsMode"), "UseSystemDNS")),
		fmt.Sprintf("  dnsList: %s", emptyFallback(strings.Join(toStringSlice(a.config["dnsList"]), ","), "none")),
		fmt.Sprintf("  systemProxyMode: %s", emptyFallback(stringValue(a.config, "systemProxyMode"), "unset")),
		fmt.Sprintf("  systemProxyExceptions: %s", emptyFallback(stringValue(a.config, "systemProxyExceptions"), "none")),
		fmt.Sprintf("  engineMode: %s", emptyFallback(a.status.EngineMode, "unknown")),
		fmt.Sprintf("  coreResolved: %s", emptyFallback(a.status.EngineResolved, "unknown")),
		fmt.Sprintf("  coreUptime: %s", formatCoreUptime(a.status)),
	}, "\n")
}

func formatCoreUptime(status CoreStatus) string {
	if !status.Running || status.UptimeSec <= 0 {
		return "stopped"
	}
	return humanDurationSeconds(status.UptimeSec)
}

func (a *tuiApp) selectedProfile() *ProfileItem {
	for idx := range a.profiles {
		if a.profiles[idx].ID == a.selectedProfileID {
			return &a.profiles[idx]
		}
	}
	return nil
}

func (a *tuiApp) selectedSubscription() *SubscriptionItem {
	for idx := range a.subscriptions {
		if a.subscriptions[idx].ID == a.selectedSubID {
			return &a.subscriptions[idx]
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

func (a *tuiApp) profileLabel(profile ProfileItem) string {
	prefix := "  "
	if profile.ID == a.status.CurrentProfileID {
		prefix = "> "
	} else if profile.ID == a.selectedProfileID {
		prefix = "* "
	}
	return fmt.Sprintf("%s%s [%s] %s:%d %dms", prefix, profile.Name, profile.Protocol, profile.Address, profile.Port, profile.DelayMs)
}

func (a *tuiApp) sortedProfilesForDisplay() []ProfileItem {
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
		return "fast"
	case delayMs < 300:
		return "mid"
	default:
		return "slow"
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

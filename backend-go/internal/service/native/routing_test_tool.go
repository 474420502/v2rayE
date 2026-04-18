package native

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"v2raye/backend-go/internal/domain"

	xrayrouter "github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/common/platform/filesystem"
	"google.golang.org/protobuf/proto"
)

var routingLookupIPAddr = func(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

var geoIPMatcherCache struct {
	mu       sync.RWMutex
	matchers map[string]xrayrouter.GeoIPMatcher
}

func (s *Service) TestRouting(req domain.RoutingTestRequest) domain.RoutingTestResult {
	target := strings.TrimSpace(req.Target)
	protocol := strings.TrimSpace(req.Protocol)
	inboundTag := strings.TrimSpace(req.InboundTag)
	if protocol == "" {
		protocol = "tcp"
	}
	routingCfg := s.loadRoutingConfig()
	strategy := routingDomainStrategy(routingCfg)
	result := domain.RoutingTestResult{
		Target:   target,
		Protocol: protocol,
		Port:     req.Port,
		InboundTag: inboundTag,
		Outbound: "proxy",
		Action:   "proxy",
		Type:     inferRoutingTargetType(target),
	}

	host, port := parseRoutingTarget(target, req.Port)
	if port > 0 {
		result.Port = port
	}

	rules := s.GetRoutingDiagnostics().Rules
	deferred := make([]deferredIPRule, 0, len(rules))
	for idx, rule := range rules {
		match := matchRoutingRule(host, port, protocol, inboundTag, strategy, rule)
		if match.deferIP {
			deferred = append(deferred, deferredIPRule{index: idx, rule: rule})
			continue
		}
		if !match.ok {
			continue
		}
		result.RuleIndex = idx + 1
		result.MatchedRule = routingRuleName(rule, idx)
		result.MatchedValue = match.matchedValue
		result.Outbound = match.outbound
		result.Action = match.outbound
		if note, _ := rule["type"].(string); note != "" {
			result.Note = note
		}
		if match.note != "" {
			result.Note = appendRoutingNote(result.Note, match.note)
		}
		return result
	}

	if len(deferred) > 0 {
		resolvedIPs, err := resolveRoutingIPs(host)
		if err != nil {
			result.Note = appendRoutingNote(result.Note, fmt.Sprintf("IP-based routing check skipped: %v", err))
		} else if len(resolvedIPs) > 0 {
			result.ResolvedIPs = resolvedIPs
			for _, entry := range deferred {
				match := matchRoutingRuleAgainstResolvedIPs(resolvedIPs, port, protocol, inboundTag, entry.rule)
				if !match.ok {
					continue
				}
				result.RuleIndex = entry.index + 1
				result.MatchedRule = routingRuleName(entry.rule, entry.index)
				result.MatchedValue = match.matchedValue
				result.Outbound = match.outbound
				result.Action = match.outbound
				if note, _ := entry.rule["type"].(string); note != "" {
					result.Note = note
				}
				result.Note = appendRoutingNote(result.Note, fmt.Sprintf("matched resolved IP using %s", strategy))
				if match.note != "" {
					result.Note = appendRoutingNote(result.Note, match.note)
				}
				return result
			}
		}
	}

	if routingCfg.Mode == "direct" {
		result.Outbound = "direct"
		result.Action = "direct"
		result.Note = "default direct mode"
		return result
	}
	result.Note = appendRoutingNote(result.Note, "no explicit rule matched; default outbound applied")
	return result
}

type deferredIPRule struct {
	index int
	rule  map[string]interface{}
}

type routingRuleMatch struct {
	matchedValue string
	outbound     string
	note         string
	ok           bool
	deferIP      bool
}

func inferRoutingTargetType(target string) string {
	host, _ := parseRoutingTarget(target, 0)
	if net.ParseIP(host) != nil {
		return "ip"
	}
	if host == "" {
		return "unknown"
	}
	return "domain"
}

func parseRoutingTarget(target string, fallbackPort int) (string, int) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fallbackPort
	}
	if host, portRaw, err := net.SplitHostPort(target); err == nil {
		if parsed, err := strconv.Atoi(portRaw); err == nil {
			return host, parsed
		}
		return host, fallbackPort
	}
	if strings.Count(target, ":") == 1 {
		parts := strings.SplitN(target, ":", 2)
		if parsed, err := strconv.Atoi(parts[1]); err == nil {
			return parts[0], parsed
		}
	}
	return target, fallbackPort
}

func matchRoutingRule(host string, port int, protocol string, inboundTag string, domainStrategy string, rule map[string]interface{}) routingRuleMatch {
	outbound := routingRuleOutbound(rule)
	matchedIndicators := make([]string, 0, 4)

	if matched, specified, ok := ruleMatchesInboundTag(inboundTag, rule["inboundTag"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}
	if matched, specified, ok := ruleMatchesProtocol(protocol, rule["network"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}
	if matched, specified, ok := ruleMatchesProtocol(protocol, rule["protocol"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}
	if matched, specified, ok := ruleMatchesPort(port, rule["port"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}

	hasDomain := hasRoutingValues(rule["domain"])
	hasIP := hasRoutingValues(rule["ip"])
	hostIP := net.ParseIP(strings.TrimSpace(host))
	strategy := strings.ToLower(strings.TrimSpace(domainStrategy))

	if hostIP != nil {
		if hasDomain {
			return routingRuleMatch{}
		}
		if hasIP {
			matched, ok := ruleMatchesIP(host, rule["ip"])
			if !ok {
				return routingRuleMatch{}
			}
			matchedIndicators = append(matchedIndicators, matched)
		}
		return buildRoutingRuleMatch(matchedIndicators, outbound, "")
	}

	if hasDomain {
		matched, ok := ruleMatchesDomain(host, rule["domain"])
		if !ok {
			if !hasIP {
				return routingRuleMatch{}
			}
			if strategy != "ipifnonmatch" && strategy != "ipondemand" {
				return routingRuleMatch{}
			}
		} else {
			matchedIndicators = append(matchedIndicators, matched)
			if !hasIP {
				return buildRoutingRuleMatch(matchedIndicators, outbound, "")
			}
		}
	}

	if hasIP {
		switch strategy {
		case "ipondemand":
			resolvedIPs, err := resolveRoutingIPs(host)
			if err != nil {
				return routingRuleMatch{}
			}
			match := matchRoutingRuleAgainstResolvedIPs(resolvedIPs, port, protocol, inboundTag, rule)
			if !match.ok {
				return routingRuleMatch{}
			}
			match.note = appendRoutingNote(match.note, "matched resolved IP using IPOnDemand")
			return match
		case "ipifnonmatch":
			return routingRuleMatch{deferIP: true}
		default:
			return routingRuleMatch{}
		}
	}

	return buildRoutingRuleMatch(matchedIndicators, outbound, "")
}


func matchRoutingRuleAgainstResolvedIPs(resolvedIPs []string, port int, protocol string, inboundTag string, rule map[string]interface{}) routingRuleMatch {
	if len(resolvedIPs) == 0 {
		return routingRuleMatch{}
	}
	outbound := routingRuleOutbound(rule)
	matchedIndicators := make([]string, 0, 4)

	if matched, specified, ok := ruleMatchesInboundTag(inboundTag, rule["inboundTag"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}
	if matched, specified, ok := ruleMatchesProtocol(protocol, rule["network"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}
	if matched, specified, ok := ruleMatchesProtocol(protocol, rule["protocol"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}
	if matched, specified, ok := ruleMatchesPort(port, rule["port"]); specified {
		if !ok {
			return routingRuleMatch{}
		}
		matchedIndicators = append(matchedIndicators, matched)
	}
	if hasRoutingValues(rule["domain"]) {
		return routingRuleMatch{}
	}
	if !hasRoutingValues(rule["ip"]) {
		return buildRoutingRuleMatch(matchedIndicators, outbound, "")
	}
	for _, ip := range resolvedIPs {
		matched, ok := ruleMatchesIP(ip, rule["ip"])
		if !ok {
			continue
		}
		return buildRoutingRuleMatch(append(matchedIndicators, matched), outbound, ip)
	}
	return routingRuleMatch{}
}

func buildRoutingRuleMatch(indicators []string, outbound string, resolvedIP string) routingRuleMatch {
	if len(indicators) == 0 {
		return routingRuleMatch{}
	}
	match := routingRuleMatch{
		matchedValue: indicators[len(indicators)-1],
		outbound:     outbound,
		ok:           true,
	}
	if resolvedIP != "" {
		match.note = "resolved IP: " + resolvedIP
	}
	return match
}

func routingRuleOutbound(rule map[string]interface{}) string {
	outbound, _ := rule["outboundTag"].(string)
	if outbound == "" {
		outbound, _ = rule["outbound"].(string)
	}
	if outbound == "" {
		outbound = "proxy"
	}
	return outbound
}

func resolveRoutingIPs(host string) ([]string, error) {
	host = strings.TrimSpace(host)
	if host == "" || net.ParseIP(host) != nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	addrs, err := routingLookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	resolved := make([]string, 0, len(addrs))
	seen := make(map[string]struct{}, len(addrs))
	for _, addr := range addrs {
		ip := addr.IP.String()
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		resolved = append(resolved, ip)
	}
	return resolved, nil
}

func appendRoutingNote(current string, next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return current
	}
	current = strings.TrimSpace(current)
	if current == "" {
		return next
	}
	if strings.Contains(current, next) {
		return current
	}
	return current + "; " + next
}

func hasRoutingValues(raw interface{}) bool {
	return len(routingToStringSlice(raw)) > 0
}

func ruleMatchesProtocol(protocol string, raw interface{}) (string, bool, bool) {
	values := routingToStringSlice(raw)
	if len(values) == 0 {
		return "", false, true
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), protocol) {
			return value, true, true
		}
	}
	return "", true, false
}

func ruleMatchesInboundTag(inboundTag string, raw interface{}) (string, bool, bool) {
	values := routingToStringSlice(raw)
	if len(values) == 0 {
		return "", false, true
	}
	if inboundTag == "" {
		return "", true, false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), inboundTag) {
			return value, true, true
		}
	}
	return "", true, false
}

func ruleMatchesPort(port int, raw interface{}) (string, bool, bool) {
	values := routingToStringSlice(raw)
	if len(values) == 0 {
		return "", false, true
	}
	if port <= 0 {
		return "", true, false
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, "-") {
			parts := strings.SplitN(value, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && port >= start && port <= end {
				return value, true, true
			}
			continue
		}
		if parsed, err := strconv.Atoi(value); err == nil && parsed == port {
			return value, true, true
		}
	}
	return "", true, false
}

func ruleMatchesDomain(host string, raw interface{}) (string, bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || net.ParseIP(host) != nil {
		return "", false
	}
	for _, value := range routingToStringSlice(raw) {
		candidate := strings.ToLower(strings.TrimSpace(value))
		switch {
		case candidate == "":
			continue
		case strings.HasPrefix(candidate, "full:"):
			matched := strings.TrimPrefix(candidate, "full:")
			if host == matched {
				return value, true
			}
		case strings.HasPrefix(candidate, "domain:"):
			matched := strings.TrimPrefix(candidate, "domain:")
			if host == matched || strings.HasSuffix(host, "."+matched) {
				return value, true
			}
		case strings.HasPrefix(candidate, "keyword:"):
			matched := strings.TrimPrefix(candidate, "keyword:")
			if strings.Contains(host, matched) {
				return value, true
			}
		case strings.HasPrefix(candidate, "regexp:"):
			// Regexp-based domain rules are intentionally not evaluated here.
			continue
		case strings.HasPrefix(candidate, "geosite:cn"):
			if strings.HasSuffix(host, ".cn") {
				return value, true
			}
		case host == candidate || strings.HasSuffix(host, "."+candidate):
			return value, true
		}
	}
	return "", false
}

func ruleMatchesIP(host string, raw interface{}) (string, bool) {
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return "", false
	}
	for _, value := range routingToStringSlice(raw) {
		candidate := strings.ToLower(strings.TrimSpace(value))
		switch {
		case candidate == "geoip:private":
			if geoIPCodeMatches(ip, "PRIVATE") || ip.IsPrivate() || ip.IsLoopback() {
				return value, true
			}
		case candidate == "geoip:cn":
			if geoIPCodeMatches(ip, "CN") || builtinCNIPContains(ip) {
				return value, true
			}
		case strings.HasPrefix(candidate, "geoip:"):
			if geoIPCodeMatches(ip, strings.TrimPrefix(candidate, "geoip:")) {
				return value, true
			}
		case strings.Contains(candidate, "/"):
			if _, network, err := net.ParseCIDR(candidate); err == nil && network.Contains(ip) {
				return value, true
			}
		case ip.String() == candidate:
			return value, true
		}
	}
	return "", false
}

func geoIPCodeMatches(ip net.IP, code string) bool {
	matcher, ok := loadGeoIPMatcher(code)
	if !ok || matcher == nil {
		return false
	}
	return matcher.Match(ip)
}

func loadGeoIPMatcher(code string) (xrayrouter.GeoIPMatcher, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, false
	}

	geoIPMatcherCache.mu.RLock()
	if matcher := geoIPMatcherCache.matchers[code]; matcher != nil {
		geoIPMatcherCache.mu.RUnlock()
		return matcher, true
	}
	geoIPMatcherCache.mu.RUnlock()

	data, err := filesystem.ReadAsset("geoip.dat")
	if err != nil {
		return nil, false
	}
	var list xrayrouter.GeoIPList
	if err := proto.Unmarshal(data, &list); err != nil {
		return nil, false
	}
	for _, entry := range list.Entry {
		if entry == nil || !strings.EqualFold(entry.CountryCode, code) {
			continue
		}
		matcher, err := xrayrouter.BuildOptimizedGeoIPMatcher(entry)
		if err != nil {
			return nil, false
		}
		geoIPMatcherCache.mu.Lock()
		if geoIPMatcherCache.matchers == nil {
			geoIPMatcherCache.matchers = make(map[string]xrayrouter.GeoIPMatcher)
		}
		geoIPMatcherCache.matchers[code] = matcher
		geoIPMatcherCache.mu.Unlock()
		return matcher, true
	}
	return nil, false
}

func builtinCNIPContains(ip net.IP) bool {
	for _, cidr := range builtinCNIPs {
		if _, network, err := net.ParseCIDR(cidr); err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func resetGeoIPMatcherCache() {
	geoIPMatcherCache.mu.Lock()
	defer geoIPMatcherCache.mu.Unlock()
	geoIPMatcherCache.matchers = nil
}

func routingToStringSlice(raw interface{}) []string {
	switch typed := raw.(type) {
	case []string:
		return typed
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

func routingRuleName(rule map[string]interface{}, index int) string {
	if id, _ := rule["id"].(string); id != "" {
		return id
	}
	if ruleType, _ := rule["type"].(string); ruleType != "" {
		return fmt.Sprintf("%s-%d", ruleType, index+1)
	}
	return fmt.Sprintf("rule-%d", index+1)
}
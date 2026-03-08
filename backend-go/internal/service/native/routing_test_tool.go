package native

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"v2raye/backend-go/internal/domain"
)

func (s *Service) TestRouting(req domain.RoutingTestRequest) domain.RoutingTestResult {
	target := strings.TrimSpace(req.Target)
	protocol := strings.TrimSpace(req.Protocol)
	if protocol == "" {
		protocol = "tcp"
	}
	result := domain.RoutingTestResult{
		Target:   target,
		Protocol: protocol,
		Port:     req.Port,
		Outbound: "proxy",
		Action:   "proxy",
		Type:     inferRoutingTargetType(target),
	}

	host, port := parseRoutingTarget(target, req.Port)
	if port > 0 {
		result.Port = port
	}

	rules := s.GetRoutingDiagnostics().Rules
	for idx, rule := range rules {
		matchedValue, outbound, ok := matchRoutingRule(host, port, protocol, rule)
		if !ok {
			continue
		}
		result.RuleIndex = idx + 1
		result.MatchedRule = routingRuleName(rule, idx)
		result.MatchedValue = matchedValue
		result.Outbound = outbound
		result.Action = outbound
		if note, _ := rule["type"].(string); note != "" {
			result.Note = note
		}
		return result
	}

	rc := s.loadRoutingConfig()
	if rc.Mode == "direct" {
		result.Outbound = "direct"
		result.Action = "direct"
		result.Note = "default direct mode"
		return result
	}
	result.Note = "no explicit rule matched; default outbound applied"
	return result
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

func matchRoutingRule(host string, port int, protocol string, rule map[string]interface{}) (string, string, bool) {
	outbound, _ := rule["outboundTag"].(string)
	if outbound == "" {
		outbound, _ = rule["outbound"].(string)
	}
	if outbound == "" {
		outbound = "proxy"
	}

	if !ruleMatchesProtocol(protocol, rule["network"]) {
		return "", "", false
	}
	if matched, ok := ruleMatchesPort(port, rule["port"]); ok {
		return matched, outbound, true
	}
	if matched, ok := ruleMatchesDomain(host, rule["domain"]); ok {
		return matched, outbound, true
	}
	if matched, ok := ruleMatchesIP(host, rule["ip"]); ok {
		return matched, outbound, true
	}
	return "", "", false
}

func ruleMatchesProtocol(protocol string, raw interface{}) bool {
	values := routingToStringSlice(raw)
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), protocol) {
			return true
		}
	}
	return false
}

func ruleMatchesPort(port int, raw interface{}) (string, bool) {
	if port <= 0 {
		return "", false
	}
	for _, value := range routingToStringSlice(raw) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, "-") {
			parts := strings.SplitN(value, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && port >= start && port <= end {
				return value, true
			}
			continue
		}
		if parsed, err := strconv.Atoi(value); err == nil && parsed == port {
			return value, true
		}
	}
	return "", false
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
			if ip.IsPrivate() || ip.IsLoopback() {
				return value, true
			}
		case candidate == "geoip:cn":
			for _, cidr := range builtinCNIPs {
				if _, network, err := net.ParseCIDR(cidr); err == nil && network.Contains(ip) {
					return value, true
				}
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
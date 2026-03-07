package native

import (
	"encoding/json"
	"testing"

	"v2raye/backend-go/internal/domain"
)

func TestTunModeFromConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  map[string]interface{}
		want string
	}{
		{
			name: "explicit off",
			cfg:  map[string]interface{}{"tunMode": "off", "enableTun": true, "tunStack": "mixed"},
			want: "off",
		},
		{
			name: "explicit gvisor",
			cfg:  map[string]interface{}{"tunMode": "gvisor"},
			want: "gvisor",
		},
		{
			name: "legacy enabled system stack",
			cfg:  map[string]interface{}{"enableTun": true, "tunStack": "system"},
			want: "system",
		},
		{
			name: "legacy enabled default mixed",
			cfg:  map[string]interface{}{"enableTun": true},
			want: "mixed",
		},
		{
			name: "legacy disabled",
			cfg:  map[string]interface{}{"enableTun": false, "tunStack": "system"},
			want: "off",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tunModeFromConfig(tc.cfg); got != tc.want {
				t.Fatalf("tunModeFromConfig() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGenerateXrayConfigIncludesTunSettings(t *testing.T) {
	profile := domain.ProfileItem{
		ID:       "p1",
		Name:     "test",
		Protocol: domain.ProtocolVLESS,
		Address:  "example.com",
		Port:     443,
		VLESS: &domain.VLESSConfig{
			UUID: "11111111-1111-1111-1111-111111111111",
		},
		Transport: &domain.TransportConfig{
			Network: "tcp",
			TLS:     true,
		},
	}

	cfg := map[string]interface{}{
		"socksPort":      10808,
		"httpPort":       10809,
		"statsPort":      10085,
		"listenAddr":     "127.0.0.1",
		"logLevel":       "warning",
		"tunMode":        "system",
		"tunName":        "xray0",
		"tunMtu":         1400,
		"tunAutoRoute":   false,
		"tunStrictRoute": true,
	}

	raw, err := generateXrayConfig(profile, cfg, domain.RoutingConfig{Mode: "global", DomainStrategy: "IPIfNonMatch"})
	if err != nil {
		t.Fatalf("generateXrayConfig() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal generated config: %v", err)
	}

	inbounds, ok := parsed["inbounds"].([]interface{})
	if !ok {
		t.Fatalf("generated config missing inbounds")
	}

	var tunInbound map[string]interface{}
	for _, inbound := range inbounds {
		item, ok := inbound.(map[string]interface{})
		if ok && item["tag"] == "tun" {
			tunInbound = item
			break
		}
	}
	if tunInbound == nil {
		t.Fatalf("generated config missing tun inbound")
	}

	settings, ok := tunInbound["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("tun inbound missing settings")
	}
	if got := settings["stack"]; got != "system" {
		t.Fatalf("tun stack = %#v, want %q", got, "system")
	}
	if got := settings["autoRoute"]; got != false {
		t.Fatalf("tun autoRoute = %#v, want false", got)
	}
	if got := settings["strictRoute"]; got != true {
		t.Fatalf("tun strictRoute = %#v, want true", got)
	}
}

func TestBuildRoutingRulesBypassCNFallbackWithoutGeoData(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{Mode: "bypass_cn"}, false, false)
	if len(rules) < 2 {
		t.Fatalf("expected at least api rule + private rule, got %d", len(rules))
	}

	privateRule, ok := rules[1].(map[string]interface{})
	if !ok {
		t.Fatalf("private rule has unexpected type: %T", rules[1])
	}
	ipList, ok := privateRule["ip"].([]string)
	if !ok {
		t.Fatalf("private rule ip list has unexpected type: %T", privateRule["ip"])
	}

	if containsString(ipList, "geoip:private") {
		t.Fatalf("fallback private rule should not use geoip:private when geodata is missing")
	}
	if !containsString(ipList, "10.0.0.0/8") || !containsString(ipList, "fc00::/7") {
		t.Fatalf("fallback private rule missing expected CIDRs: %#v", ipList)
	}
}

func TestBuildRoutingRulesBypassCNWithGeoData(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{Mode: "bypass_cn"}, true, true)
	if len(rules) < 4 {
		t.Fatalf("expected api + private + cn ip + cn domain rules, got %d", len(rules))
	}

	privateRule, ok := rules[1].(map[string]interface{})
	if !ok {
		t.Fatalf("private rule has unexpected type: %T", rules[1])
	}
	ipList, ok := privateRule["ip"].([]string)
	if !ok {
		t.Fatalf("private rule ip list has unexpected type: %T", privateRule["ip"])
	}
	if !containsString(ipList, "geoip:private") {
		t.Fatalf("expected geoip:private when geodata is available, got %#v", ipList)
	}
}

func TestBuildRoutingRulesBypassCNWithGeoSiteOnly(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{Mode: "bypass_cn"}, false, true)

	if len(rules) < 3 {
		t.Fatalf("expected api + private + cn domain rules, got %d", len(rules))
	}

	privateRule, ok := rules[1].(map[string]interface{})
	if !ok {
		t.Fatalf("private rule has unexpected type: %T", rules[1])
	}
	ipList, ok := privateRule["ip"].([]string)
	if !ok {
		t.Fatalf("private rule ip list has unexpected type: %T", privateRule["ip"])
	}
	if containsString(ipList, "geoip:private") {
		t.Fatalf("expected CIDR fallback when geoip.dat is missing")
	}

	foundGeoSiteCN := false
	for _, rule := range rules {
		item, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}
		domains, ok := item["domain"].([]string)
		if !ok {
			continue
		}
		if containsString(domains, "geosite:cn") {
			foundGeoSiteCN = true
			break
		}
	}
	if !foundGeoSiteCN {
		t.Fatalf("expected geosite:cn rule when geosite.dat is available")
	}
}

func TestGenerateXrayConfigTunBypassCNDoesNotForceAllTunToProxy(t *testing.T) {
	profile := domain.ProfileItem{
		ID:       "p1",
		Name:     "test",
		Protocol: domain.ProtocolVLESS,
		Address:  "example.com",
		Port:     443,
		VLESS: &domain.VLESSConfig{
			UUID: "11111111-1111-1111-1111-111111111111",
		},
		Transport: &domain.TransportConfig{
			Network: "tcp",
			TLS:     true,
		},
	}

	cfg := map[string]interface{}{
		"socksPort":      10808,
		"httpPort":       10809,
		"statsPort":      10085,
		"listenAddr":     "127.0.0.1",
		"logLevel":       "warning",
		"tunMode":        "system",
		"tunName":        "xray0",
		"tunMtu":         1400,
		"tunAutoRoute":   true,
		"tunStrictRoute": false,
	}

	routing := domain.RoutingConfig{
		Mode:           "bypass_cn",
		DomainStrategy: "IPIfNonMatch",
		Rules: []domain.RoutingRule{
			{
				ID:       "r1",
				Type:     "domain",
				Values:   []string{"example.internal"},
				Outbound: "direct",
			},
		},
	}

	raw, err := generateXrayConfig(profile, cfg, routing)
	if err != nil {
		t.Fatalf("generateXrayConfig() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal generated config: %v", err)
	}

	routingCfg, ok := parsed["routing"].(map[string]interface{})
	if !ok {
		t.Fatalf("generated config missing routing config")
	}
	rules, ok := routingCfg["rules"].([]interface{})
	if !ok {
		t.Fatalf("generated config missing routing rules")
	}

	hasBypassDirect := false
	hasCustomDomainRule := false
	hasTunForceProxy := false

	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}

		if inboundTags, ok := rule["inboundTag"].([]interface{}); ok {
			hasTunTag := false
			for _, tag := range inboundTags {
				if s, ok := tag.(string); ok && s == "tun" {
					hasTunTag = true
					break
				}
			}
			if hasTunTag {
				if outbound, ok := rule["outboundTag"].(string); ok && outbound == "proxy" {
					hasTunForceProxy = true
				}
			}
		}

		if outbound, ok := rule["outboundTag"].(string); ok && outbound == "direct" {
			if _, hasIP := rule["ip"]; hasIP {
				hasBypassDirect = true
			}
			if domains, ok := rule["domain"].([]interface{}); ok {
				for _, d := range domains {
					if s, ok := d.(string); ok && s == "example.internal" {
						hasCustomDomainRule = true
						break
					}
				}
			}
		}
	}

	if hasTunForceProxy {
		t.Fatalf("unexpected forced tun->proxy rule present; tun should follow routing mode rules")
	}
	if !hasBypassDirect {
		t.Fatalf("expected bypass_cn direct rule to exist under tun mode")
	}
	if !hasCustomDomainRule {
		t.Fatalf("expected custom routing rule to be preserved under tun mode")
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
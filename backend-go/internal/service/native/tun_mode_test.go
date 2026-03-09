package native

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestGenerateXrayConfigBindsProxyAndDirectOutboundsToPhysicalInterface(t *testing.T) {
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
		"listenAddr":        "127.0.0.1",
		"outboundInterface": "eth0",
		"tunMode":           "system",
	}

	raw, err := generateXrayConfig(profile, cfg, domain.RoutingConfig{Mode: "bypass_cn", DomainStrategy: "IPIfNonMatch"})
	if err != nil {
		t.Fatalf("generateXrayConfig() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal generated config: %v", err)
	}

	outbounds, ok := parsed["outbounds"].([]interface{})
	if !ok {
		t.Fatalf("generated config missing outbounds")
	}

	assertSockopt := func(tag string, wantIface string, wantMark int, expectMark bool) {
		t.Helper()
		for _, entry := range outbounds {
			outbound, ok := entry.(map[string]interface{})
			if !ok || outbound["tag"] != tag {
				continue
			}
			streamSettings, ok := outbound["streamSettings"].(map[string]interface{})
			if !ok {
				t.Fatalf("%s outbound missing streamSettings", tag)
			}
			sockopt, ok := streamSettings["sockopt"].(map[string]interface{})
			if !ok {
				t.Fatalf("%s outbound missing sockopt", tag)
			}
			if got := sockopt["interface"]; got != wantIface {
				t.Fatalf("%s outbound interface = %#v, want %q", tag, got, wantIface)
			}
			mark, hasMark := sockopt["mark"]
			if expectMark {
				if !hasMark {
					t.Fatalf("%s outbound missing mark", tag)
				}
				if got := int(mark.(float64)); got != wantMark {
					t.Fatalf("%s outbound mark = %d, want %d", tag, got, wantMark)
				}
			} else if hasMark {
				t.Fatalf("%s outbound unexpected mark = %#v", tag, mark)
			}
			return
		}
		t.Fatalf("outbound %s not found", tag)
	}

	assertSockopt("proxy", "eth0", 0, false)
	assertSockopt("direct", "eth0", tunDirectBypassMark, true)
}

func TestBuildRoutingRulesBypassCNFallbackWithoutGeoData(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{Mode: "bypass_cn"}, false, false)
	privateRule, ipList := findRuleWithIPCIDR(rules, "10.0.0.0/8")
	if privateRule == nil {
		t.Fatalf("expected fallback private CIDR rule in bypass_cn mode")
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
	privateRule, ipList := findRuleWithIPCIDR(rules, "geoip:private")
	if privateRule == nil {
		t.Fatalf("expected geoip:private rule when geodata is available")
	}
	if !containsString(ipList, "geoip:private") {
		t.Fatalf("expected geoip:private when geodata is available, got %#v", ipList)
	}
}

func TestBuildRoutingRulesBypassCNWithGeoSiteOnly(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{Mode: "bypass_cn"}, false, true)

	privateRule, ipList := findRuleWithIPCIDR(rules, "10.0.0.0/8")
	if privateRule == nil {
		t.Fatalf("expected fallback private CIDR rule when geoip.dat is missing")
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

func TestBuildRoutingRulesAlwaysBypassesLocalControlPlane(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{Mode: "global"}, false, false)

	if len(rules) < 3 {
		t.Fatalf("expected api + localhost bypass rules, got %d", len(rules))
	}

	ruleDomain, ok := rules[1].(map[string]interface{})
	if !ok {
		t.Fatalf("rules[1] should be localhost domain bypass, got %T", rules[1])
	}
	if got := ruleDomain["outboundTag"]; got != "direct" {
		t.Fatalf("localhost domain bypass outbound = %#v, want %q", got, "direct")
	}
	domains, ok := ruleDomain["domain"].([]string)
	if !ok || !containsString(domains, "full:localhost") {
		t.Fatalf("expected full:localhost bypass rule, got %#v", ruleDomain["domain"])
	}

	ruleIP, ok := rules[2].(map[string]interface{})
	if !ok {
		t.Fatalf("rules[2] should be loopback IP bypass, got %T", rules[2])
	}
	if got := ruleIP["outboundTag"]; got != "direct" {
		t.Fatalf("loopback bypass outbound = %#v, want %q", got, "direct")
	}
	ipList, ok := ruleIP["ip"].([]string)
	if !ok {
		t.Fatalf("loopback bypass ip list has unexpected type: %T", ruleIP["ip"])
	}
	if !containsString(ipList, "127.0.0.0/8") || !containsString(ipList, "::1/128") {
		t.Fatalf("expected loopback bypass CIDRs, got %#v", ipList)
	}
}

func TestBuildRoutingRulesCanDisableLocalControlBypass(t *testing.T) {
	disabled := false
	rules := buildRoutingRules(domain.RoutingConfig{Mode: "global", LocalBypassEnabled: &disabled}, false, false)

	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}
		if domains, ok := rule["domain"].([]string); ok && containsString(domains, "full:localhost") {
			t.Fatalf("localhost bypass should be disabled, got rule %#v", rule)
		}
		if ips, ok := rule["ip"].([]string); ok && (containsString(ips, "127.0.0.0/8") || containsString(ips, "::1/128")) {
			t.Fatalf("loopback bypass should be disabled, got rule %#v", rule)
		}
	}
}

func TestBuildRoutingRulesDirectModeAppliesDefaultAfterCustomRules(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{
		Mode: "direct",
		Rules: []domain.RoutingRule{
			{ID: "custom", Type: "domain", Values: []string{"full:example.com"}, Outbound: "proxy"},
		},
	}, false, false)

	customIdx := -1
	defaultIdx := -1
	for idx, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}

		if domains, ok := rule["domain"].([]string); ok && containsString(domains, "full:example.com") {
			customIdx = idx
		}
		if outbound, ok := rule["outboundTag"].(string); ok && outbound == "direct" {
			if network, ok := rule["network"].(string); ok && network == "tcp,udp" {
				defaultIdx = idx
			}
		}
	}

	if customIdx == -1 {
		t.Fatalf("expected custom rule in direct mode")
	}
	if defaultIdx == -1 {
		t.Fatalf("expected default direct catch-all rule in direct mode")
	}
	if defaultIdx < customIdx {
		t.Fatalf("default direct rule must be after custom rules, got default=%d custom=%d", defaultIdx, customIdx)
	}
}

func TestBuildRoutingRulesBypassCNKeepsLayerOrder(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{
		Mode: "bypass_cn",
		Rules: []domain.RoutingRule{
			{ID: "custom", Type: "domain", Values: []string{"full:override.example"}, Outbound: "proxy"},
		},
	}, false, false)

	indexOf := func(match func(map[string]interface{}) bool) int {
		for idx, rawRule := range rules {
			rule, ok := rawRule.(map[string]interface{})
			if !ok {
				continue
			}
			if match(rule) {
				return idx
			}
		}
		return -1
	}

	apiIdx := indexOf(func(rule map[string]interface{}) bool {
		tags, ok := rule["inboundTag"].([]string)
		return ok && containsString(tags, "api")
	})
	localhostIdx := indexOf(func(rule map[string]interface{}) bool {
		domains, ok := rule["domain"].([]string)
		return ok && containsString(domains, "full:localhost")
	})
	loopbackIdx := indexOf(func(rule map[string]interface{}) bool {
		ips, ok := rule["ip"].([]string)
		return ok && containsString(ips, "127.0.0.0/8")
	})
	privateIdx := indexOf(func(rule map[string]interface{}) bool {
		ips, ok := rule["ip"].([]string)
		return ok && containsString(ips, "10.0.0.0/8")
	})
	cnIdx := indexOf(func(rule map[string]interface{}) bool {
		ips, ok := rule["ip"].([]string)
		return ok && containsString(ips, "1.0.1.0/24")
	})
	customIdx := indexOf(func(rule map[string]interface{}) bool {
		domains, ok := rule["domain"].([]string)
		if !ok || !containsString(domains, "full:override.example") {
			return false
		}
		outbound, _ := rule["outboundTag"].(string)
		return outbound == "proxy"
	})

	if apiIdx == -1 || localhostIdx == -1 || loopbackIdx == -1 || privateIdx == -1 || cnIdx == -1 || customIdx == -1 {
		t.Fatalf("expected all routing layers to exist, got api=%d localhost=%d loopback=%d private=%d cn=%d custom=%d", apiIdx, localhostIdx, loopbackIdx, privateIdx, cnIdx, customIdx)
	}

	if !(apiIdx < localhostIdx && localhostIdx < loopbackIdx && loopbackIdx < privateIdx && privateIdx < cnIdx && cnIdx < customIdx) {
		t.Fatalf("unexpected bypass_cn layer order: api=%d localhost=%d loopback=%d private=%d cn=%d custom=%d", apiIdx, localhostIdx, loopbackIdx, privateIdx, cnIdx, customIdx)
	}

	for idx, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}
		outbound, _ := rule["outboundTag"].(string)
		network, _ := rule["network"].(string)
		if outbound == "direct" && network == "tcp,udp" {
			t.Fatalf("unexpected direct catch-all rule in bypass_cn mode at index %d", idx)
		}
	}
}

func TestBuildRoutingRulesDirectKeepsControlBypassBeforeCatchAll(t *testing.T) {
	rules := buildRoutingRules(domain.RoutingConfig{
		Mode: "direct",
		Rules: []domain.RoutingRule{
			{ID: "custom", Type: "domain", Values: []string{"full:example.com"}, Outbound: "proxy"},
		},
	}, false, false)

	indexOf := func(match func(map[string]interface{}) bool) int {
		for idx, rawRule := range rules {
			rule, ok := rawRule.(map[string]interface{})
			if !ok {
				continue
			}
			if match(rule) {
				return idx
			}
		}
		return -1
	}

	localhostIdx := indexOf(func(rule map[string]interface{}) bool {
		domains, ok := rule["domain"].([]string)
		return ok && containsString(domains, "full:localhost")
	})
	loopbackIdx := indexOf(func(rule map[string]interface{}) bool {
		ips, ok := rule["ip"].([]string)
		return ok && containsString(ips, "127.0.0.0/8")
	})
	customIdx := indexOf(func(rule map[string]interface{}) bool {
		domains, ok := rule["domain"].([]string)
		return ok && containsString(domains, "full:example.com")
	})
	defaultIdx := indexOf(func(rule map[string]interface{}) bool {
		outbound, _ := rule["outboundTag"].(string)
		network, _ := rule["network"].(string)
		return outbound == "direct" && network == "tcp,udp"
	})

	if localhostIdx == -1 || loopbackIdx == -1 || customIdx == -1 || defaultIdx == -1 {
		t.Fatalf("expected localhost/loopback/custom/default rules in direct mode, got localhost=%d loopback=%d custom=%d default=%d", localhostIdx, loopbackIdx, customIdx, defaultIdx)
	}

	if !(localhostIdx < loopbackIdx && loopbackIdx < customIdx && customIdx < defaultIdx) {
		t.Fatalf("unexpected direct mode order: localhost=%d loopback=%d custom=%d default=%d", localhostIdx, loopbackIdx, customIdx, defaultIdx)
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

func TestGenerateXrayConfigDisablesTunInboundAutoRouteOnLinux(t *testing.T) {
	profile := domain.ProfileItem{
		ID:       "p1",
		Name:     "test",
		Protocol: domain.ProtocolVLESS,
		Address:  "example.com",
		Port:     443,
		VLESS: &domain.VLESSConfig{
			UUID: "11111111-1111-1111-1111-111111111111",
		},
	}

	cfg := map[string]interface{}{
		"tunMode":      "mixed",
		"tunName":      "xraye0",
		"tunMtu":       1500,
		"tunAutoRoute": true,
	}

	raw, err := generateXrayConfig(profile, cfg, domain.RoutingConfig{Mode: "global"})
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
	for _, inbound := range inbounds {
		item, ok := inbound.(map[string]interface{})
		if !ok || item["tag"] != "tun" {
			continue
		}
		settings := item["settings"].(map[string]interface{})
		if got := settings["autoRoute"]; got != false {
			t.Fatalf("tun inbound autoRoute = %#v, want false on Linux backend-managed routing", got)
		}
		return
	}
	t.Fatalf("generated config missing tun inbound")
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func findRuleWithIPCIDR(rules []interface{}, cidr string) (map[string]interface{}, []string) {
	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}
		ipList, ok := rule["ip"].([]string)
		if !ok {
			continue
		}
		if containsString(ipList, cidr) {
			return rule, ipList
		}
	}
	return nil, nil
}

func TestHasGeoAssetChecksEnvAssetDirs(t *testing.T) {
	tmp := t.TempDir()
	assetPath := filepath.Join(tmp, "geosite.dat")
	if err := os.WriteFile(assetPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write geosite.dat: %v", err)
	}

	oldXray := os.Getenv("XRAY_LOCATION_ASSET")
	oldV2ray := os.Getenv("V2RAY_LOCATION_ASSET")
	t.Cleanup(func() {
		_ = os.Setenv("XRAY_LOCATION_ASSET", oldXray)
		_ = os.Setenv("V2RAY_LOCATION_ASSET", oldV2ray)
	})

	if err := os.Setenv("XRAY_LOCATION_ASSET", tmp); err != nil {
		t.Fatalf("set XRAY_LOCATION_ASSET: %v", err)
	}
	if err := os.Setenv("V2RAY_LOCATION_ASSET", ""); err != nil {
		t.Fatalf("clear V2RAY_LOCATION_ASSET: %v", err)
	}

	if !hasGeoSiteAsset() {
		t.Fatalf("expected hasGeoSiteAsset to detect geosite.dat in XRAY_LOCATION_ASSET")
	}
}
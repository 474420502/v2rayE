package native

import (
	"net"
	"path/filepath"
	"testing"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/storage"
)

func TestParseDefaultRouteHint(t *testing.T) {
	tests := []struct {
		name   string
		routes []string
		tun    string
		wantD  string
		wantV  string
	}{
		{
			name:   "extract via and dev",
			routes: []string{"default via 192.168.1.1 dev wlp2s0 proto dhcp metric 600"},
			tun:    "xray0",
			wantD:  "wlp2s0",
			wantV:  "192.168.1.1",
		},
		{
			name:   "skip tun route",
			routes: []string{"default dev xray0"},
			tun:    "xray0",
			wantD:  "",
			wantV:  "",
		},
		{
			name:   "extract dev without gateway",
			routes: []string{"default dev eth0 proto static"},
			tun:    "xray0",
			wantD:  "eth0",
			wantV:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotDev, gotVia := parseDefaultRouteHint(tc.routes, tc.tun)
			if gotDev != tc.wantD || gotVia != tc.wantV {
				t.Fatalf("parseDefaultRouteHint()=(%q,%q), want (%q,%q)", gotDev, gotVia, tc.wantD, tc.wantV)
			}
		})
	}
}

func TestBuildTunRestoreFallbackRoutes(t *testing.T) {
	svc := &Service{}

	withGateway := svc.buildTunRestoreFallbackRoutes(map[string]interface{}{
		"tunRestoreHintDev": "wlp2s0",
		"tunRestoreHintVia": "192.168.1.1",
	}, "xray0")
	if len(withGateway) != 1 || withGateway[0] != "default via 192.168.1.1 dev wlp2s0" {
		t.Fatalf("unexpected fallback route with gateway: %#v", withGateway)
	}

	withoutGateway := svc.buildTunRestoreFallbackRoutes(map[string]interface{}{
		"tunRestoreHintDev": "eth0",
	}, "xray0")
	if len(withoutGateway) != 1 || withoutGateway[0] != "default dev eth0" {
		t.Fatalf("unexpected fallback route without gateway: %#v", withoutGateway)
	}

	ignoreTun := svc.buildTunRestoreFallbackRoutes(map[string]interface{}{
		"tunRestoreHintDev": "xray0",
		"tunRestoreHintVia": "10.0.0.1",
	}, "xray0")
	if len(ignoreTun) != 0 {
		t.Fatalf("expected no fallback route for tun device hint, got %#v", ignoreTun)
	}
}

func TestSanitizeTunRestoreRoutes(t *testing.T) {
	routes := sanitizeTunRestoreRoutes([]string{
		"default dev xraye0",
		"default via 192.168.1.1 dev wlp2s0 proto dhcp metric 600",
		"default via 192.168.1.1 dev wlp2s0 proto dhcp metric 600",
		" ",
	}, "xraye0")

	if len(routes) != 1 {
		t.Fatalf("expected one sanitized route, got %#v", routes)
	}
	if routes[0] != "default via 192.168.1.1 dev wlp2s0 proto dhcp metric 600" {
		t.Fatalf("unexpected sanitized route: %#v", routes)
	}
}

func TestResolveTunRestoreRoutesUsesPersistedRouteWhenCurrentDefaultIsTun(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage.New(filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}

	cfg := storage.DefaultConfig()
	cfg["tunName"] = "xraye0"
	cfg["tunRestoreRoutes"] = []interface{}{"default via 192.168.1.1 dev eth0 proto dhcp metric 100"}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	svc := &Service{store: store}
	routes := svc.resolveTunRestoreRoutes(cfg, []string{"default dev xraye0"}, "xraye0")
	if len(routes) != 1 || routes[0] != "default via 192.168.1.1 dev eth0 proto dhcp metric 100" {
		t.Fatalf("resolveTunRestoreRoutes() = %#v", routes)
	}
}

func TestPersistTunRestoreRoutesKeepsExistingHintWhenRoutesAreTunOnly(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage.New(filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}

	cfg := storage.DefaultConfig()
	cfg["tunName"] = "xraye0"
	cfg["tunRestoreHintDev"] = "eth0"
	cfg["tunRestoreHintVia"] = "192.168.1.1"
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	svc := &Service{store: store}
	svc.persistTunRestoreRoutes([]string{"default dev xraye0"}, "xraye0")

	saved, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if _, ok := saved["tunRestoreRoutes"]; ok {
		t.Fatalf("expected tunRestoreRoutes to be removed, got %#v", saved["tunRestoreRoutes"])
	}
	if got := saved["tunRestoreHintDev"]; got != "eth0" {
		t.Fatalf("tunRestoreHintDev = %#v, want %q", got, "eth0")
	}
	if got := saved["tunRestoreHintVia"]; got != "192.168.1.1" {
		t.Fatalf("tunRestoreHintVia = %#v, want %q", got, "192.168.1.1")
	}

	svc.persistTunRestoreRoutes(nil, "xraye0")
	saved, err = store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() after clear error = %v", err)
	}
	if _, ok := saved["tunRestoreHintDev"]; ok {
		t.Fatalf("expected tunRestoreHintDev to be cleared on explicit reset")
	}
	if _, ok := saved["tunRestoreHintVia"]; ok {
		t.Fatalf("expected tunRestoreHintVia to be cleared on explicit reset")
	}
}

func TestShouldHijackTunDefaultRouteDisabledByDefault(t *testing.T) {
	cfg := storage.DefaultConfig()
	cfg["tunMode"] = "system"
	cfg["tunAutoRoute"] = true

	if shouldHijackTunDefaultRoute(cfg) {
		t.Fatalf("expected manual TUN default-route hijack to be disabled by default")
	}

	cfg["tunHijackDefaultRoute"] = true
	if !shouldHijackTunDefaultRoute(cfg) {
		t.Fatalf("expected manual TUN default-route hijack to be enabled when explicitly requested")
	}
}

func TestShouldManageTunTrafficFollowsTunAutoRoute(t *testing.T) {
	cfg := storage.DefaultConfig()
	cfg["tunMode"] = "mixed"
	if !shouldManageTunTraffic(cfg) {
		t.Fatalf("expected tun auto route to enable managed TUN traffic")
	}
	cfg["tunAutoRoute"] = false
	if shouldManageTunTraffic(cfg) {
		t.Fatalf("expected tun auto route false to disable managed TUN traffic")
	}
}

func TestBuildTunPolicyBypassRules(t *testing.T) {
	rules := buildTunPolicyBypassRules([]string{
		"default via 192.168.124.1 dev enp9s0 proto dhcp src 192.168.124.8 metric 100",
		"192.168.124.0/24 dev enp9s0 proto kernel scope link src 192.168.124.8 metric 100",
		"172.17.0.0/16 dev docker0 proto kernel scope link src 172.17.0.1",
	}, map[string]interface{}{
		"dnsList": []interface{}{"1.1.1.1", "https://dns.google/dns-query", "8.8.8.8"},
	}, &domain.ProfileItem{Address: "45.63.82.225"})

	want := map[string]bool{
		"192.168.124.0/24": true,
		"172.17.0.0/16":    true,
		"45.63.82.225/32":  true,
		"1.1.1.1/32":       true,
		"8.8.8.8/32":       true,
	}
	if len(rules) != len(want) {
		t.Fatalf("unexpected bypass rules: %#v", rules)
	}
	for _, rule := range rules {
		if !want[rule] {
			t.Fatalf("unexpected bypass rule %q in %#v", rule, rules)
		}
	}
}

func TestBuildTunPolicyBypassRulesResolvesProfileDomain(t *testing.T) {
	originalLookup := lookupIPForTunBypass
	lookupIPForTunBypass = func(host string) ([]net.IP, error) {
		if host != "edge.example.com" {
			t.Fatalf("unexpected lookup host: %s", host)
		}
		return []net.IP{
			net.ParseIP("203.0.113.10"),
			net.ParseIP("203.0.113.10"),
			net.ParseIP("2001:db8::10"),
		}, nil
	}
	defer func() {
		lookupIPForTunBypass = originalLookup
	}()

	rules := buildTunPolicyBypassRules([]string{
		"default via 192.168.124.1 dev enp9s0 proto dhcp src 192.168.124.8 metric 100",
		"192.168.124.0/24 dev enp9s0 proto kernel scope link src 192.168.124.8 metric 100",
	}, map[string]interface{}{}, &domain.ProfileItem{Address: "edge.example.com"})

	if !containsString(rules, "203.0.113.10/32") {
		t.Fatalf("expected resolved profile IP bypass rule, got %#v", rules)
	}
	count := 0
	for _, rule := range rules {
		if rule == "203.0.113.10/32" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected deduplicated resolved profile IP bypass rule, got %#v", rules)
	}
}

func TestBuildTunPolicyBypassRulesIPv6(t *testing.T) {
	rules := buildTunPolicyBypassRulesForFamily([]string{
		"default via fe80::1 dev enp9s0 proto ra metric 100",
		"2001:db8:1::/64 dev enp9s0 proto kernel metric 256 pref medium",
		"fd00:1234::/64 dev docker0 proto kernel metric 256 pref medium",
	}, map[string]interface{}{
		"dnsList": []interface{}{"2606:4700:4700::1111", "https://dns.google/dns-query", "2001:4860:4860::8888"},
	}, &domain.ProfileItem{Address: "2001:db8::10"}, "-6")

	want := map[string]bool{
		"2001:db8::10/128":          true,
		"2606:4700:4700::1111/128":  true,
		"2001:4860:4860::8888/128":  true,
		"2001:db8:1::/64":           true,
		"fd00:1234::/64":            true,
	}
	if len(rules) != len(want) {
		t.Fatalf("unexpected IPv6 bypass rules: %#v", rules)
	}
	for _, rule := range rules {
		if !want[rule] {
			t.Fatalf("unexpected IPv6 bypass rule %q in %#v", rule, rules)
		}
	}
}

func TestResolveProfileBypassPrefixesDirectIPv4(t *testing.T) {
	prefixes := resolveProfileBypassPrefixes(&domain.ProfileItem{Address: "45.63.82.225"})
	if len(prefixes) != 1 || prefixes[0] != "45.63.82.225/32" {
		t.Fatalf("resolveProfileBypassPrefixes() = %#v", prefixes)
	}
}

func TestResolveProfileBypassPrefixesDirectIPv6(t *testing.T) {
	prefixes := resolveProfileBypassPrefixesForFamily(&domain.ProfileItem{Address: "2001:db8::10"}, "-6")
	if len(prefixes) != 1 || prefixes[0] != "2001:db8::10/128" {
		t.Fatalf("resolveProfileBypassPrefixesForFamily() = %#v", prefixes)
	}
}
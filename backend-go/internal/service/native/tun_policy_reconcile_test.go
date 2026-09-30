package native

import (
	"testing"

	"v2raye/backend-go/internal/domain"
)

// `ip route show table main` prints point-to-point/VPN links as bare host
// addresses ("10.88.0.1 dev wg0 scope link"). They used to be dropped from the
// bypass set because netip.ParsePrefix rejects a value without "/", which sent
// that traffic into the TUN instead of the interface that owns the route.
func TestBuildTunPolicyBypassRulesKeepsBareHostRoutes(t *testing.T) {
	rules := buildTunPolicyBypassRules([]string{
		"default via 192.168.124.1 dev enp9s0 proto dhcp src 192.168.124.8 metric 100",
		"10.88.0.1 dev gcash-pera-dev scope link",
		"172.17.0.0/16 dev docker0 proto kernel scope link src 172.17.0.1 linkdown",
		"192.168.124.0/24 dev enp9s0 proto kernel scope link src 192.168.124.8 metric 100",
	}, map[string]interface{}{
		"dnsList": []interface{}{"1.1.1.1"},
	}, &domain.ProfileItem{Address: "198.51.100.10"})

	want := map[string]bool{
		"198.51.100.10/32": true,
		"1.1.1.1/32":        true,
		"10.88.0.1/32":      true,
		"172.17.0.0/16":     true,
		"192.168.124.0/24":  true,
	}
	if len(rules) != len(want) {
		t.Fatalf("unexpected bypass rules: %#v", rules)
	}
	for _, rule := range rules {
		if !want[rule] {
			t.Fatalf("unexpected bypass rule %q in %#v", rule, rules)
		}
	}
	if !containsString(rules, "10.88.0.1/32") {
		t.Fatalf("bare host route was dropped from bypass rules: %#v", rules)
	}
}

func TestBuildTunPolicyBypassRulesKeepsBareHostRoutesIPv6(t *testing.T) {
	rules := buildTunPolicyBypassRulesForFamily([]string{
		"2001:db8::1 dev wg0 proto kernel metric 256 pref medium",
		"2001:db8:1::/64 dev enp9s0 proto kernel metric 256 pref medium",
	}, map[string]interface{}{}, nil, "-6")

	if !containsString(rules, "2001:db8::1/128") {
		t.Fatalf("bare IPv6 host route was dropped from bypass rules: %#v", rules)
	}
}

// Real `ip -4 rule show` output captured from a running takeover.
const sampleIPRuleShow = `0:	from all lookup local
100:	from all to 10.88.0.1 lookup main
10000:	from all fwmark 0x2d11 lookup main
10001:	from all to 198.51.100.10 lookup main
10002:	from all to 1.1.1.1 lookup main
10003:	from all to 8.8.8.8 lookup main
10004:	from all to 172.22.0.0/16 lookup main
10005:	from all to 192.168.124.0/24 lookup main
10006:	from all lookup 20230
32766:	from all lookup main
32767:	from all lookup default
`

func TestParseInstalledTunPolicyBypassTargets(t *testing.T) {
	got := parseInstalledTunPolicyBypassTargets(sampleIPRuleShow, "-4")
	want := []string{"1.1.1.1/32", "8.8.8.8/32", "198.51.100.10/32", "172.22.0.0/16", "192.168.124.0/24"}
	if len(got) != len(want) {
		t.Fatalf("parsed targets = %#v, want %d entries", got, len(want))
	}
	for _, prefix := range want {
		if _, ok := got[prefix]; !ok {
			t.Fatalf("missing parsed target %q in %#v", prefix, got)
		}
	}
	// Rules outside the managed window, the fwmark escape rule and the
	// catch-all rule must not be mistaken for bypass targets.
	if _, ok := got["10.88.0.1/32"]; ok {
		t.Fatalf("rule outside the priority window was parsed as a bypass target: %#v", got)
	}
	if _, ok := got["8.8.8.8/32"]; !ok {
		t.Fatal("bypass target missing")
	}
}

func TestParseInstalledTunPolicyBypassTargetsIPv6(t *testing.T) {
	output := `10000:	from all fwmark 0x2d11 lookup main
10001:	from all to 2001:db8::1 lookup main
10002:	from all to 2001:db8:1::/64 lookup main
10003:	from all lookup 20230
`
	got := parseInstalledTunPolicyBypassTargets(output, "-6")
	if _, ok := got["2001:db8::1/128"]; !ok {
		t.Fatalf("bare IPv6 target not normalized: %#v", got)
	}
	if _, ok := got["2001:db8:1::/64"]; !ok {
		t.Fatalf("IPv6 prefix target missing: %#v", got)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected targets: %#v", got)
	}
}

func TestIsManagedTunPolicyRuleLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"fwmark escape rule", "10000:\tfrom all fwmark 0x2d11 lookup main", true},
		{"fwmark escape rule decimal", "10000:\tfrom all fwmark 11537 lookup main", true},
		{"bypass rule", "10005:\tfrom all to 192.168.124.0/24 lookup main", true},
		{"catch-all rule", "10006:\tfrom all lookup 20230", true},
		{"outside priority window", "100:\tfrom all to 10.88.0.1 lookup main", false},
		{"local table", "0:\tfrom all lookup local", false},
		{"default rule", "32766:\tfrom all lookup main", false},
		{"other tool source rule", "10500:\tfrom 10.0.0.0/8 lookup main", false},
		{"other tool table", "10501:\tfrom all to 10.0.0.0/8 lookup 19999", false},
		{"other tool mark in window", "10502:\tfrom all fwmark 0x1234 lookup main", false},
	}
	for _, tc := range tests {
		if got := isManagedTunPolicyRuleLine(tc.line); got != tc.want {
			t.Errorf("%s: isManagedTunPolicyRuleLine(%q) = %t, want %t", tc.name, tc.line, got, tc.want)
		}
	}
}

func TestDiffTunPolicyPrefixSets(t *testing.T) {
	desired := map[string]struct{}{
		"192.168.124.0/24": {},
		"10.88.0.1/32":     {},
		"1.1.1.1/32":       {},
	}
	installed := map[string]struct{}{
		"192.168.124.0/24": {},
		"1.1.1.1/32":       {},
		"172.17.0.0/16":    {},
	}
	drift := diffTunPolicyPrefixSets(desired, installed)
	if drift.Empty() {
		t.Fatal("expected drift")
	}
	if len(drift.Missing) != 1 || drift.Missing[0] != "10.88.0.1/32" {
		t.Fatalf("missing = %#v", drift.Missing)
	}
	if len(drift.Stale) != 1 || drift.Stale[0] != "172.17.0.0/16" {
		t.Fatalf("stale = %#v", drift.Stale)
	}

	if !diffTunPolicyPrefixSets(desired, desired).Empty() {
		t.Fatal("identical sets must not report drift")
	}
}

func TestResolveOutboundInterfaceForSockopt(t *testing.T) {
	original := networkInterfaceExists
	defer func() { networkInterfaceExists = original }()
	networkInterfaceExists = func(name string) bool { return name == "wg-test0" }

	t.Run("tun off drops a missing device", func(t *testing.T) {
		cfg := map[string]interface{}{"tunMode": "off", "outboundInterface": "enp7s0"}
		resolveOutboundInterfaceForSockopt(cfg)
		if _, ok := cfg["outboundInterface"]; ok {
			t.Fatalf("stale interface kept: %#v", cfg["outboundInterface"])
		}
	})

	t.Run("tun off keeps an existing device", func(t *testing.T) {
		cfg := map[string]interface{}{"tunMode": "off", "outboundInterface": "wg-test0"}
		resolveOutboundInterfaceForSockopt(cfg)
		if got := cfg["outboundInterface"]; got != "wg-test0" {
			t.Fatalf("existing interface dropped: %#v", got)
		}
	})

	t.Run("tun on never keeps a missing device", func(t *testing.T) {
		cfg := map[string]interface{}{"tunMode": "mixed", "outboundInterface": "enp7s0"}
		resolveOutboundInterfaceForSockopt(cfg)
		if got := cfg["outboundInterface"]; got == "enp7s0" {
			t.Fatalf("missing device survived TUN-mode resolution: %#v", got)
		}
	})
}

func TestTunPolicyBypassDriftString(t *testing.T) {
	drift := tunPolicyBypassDrift{Missing: []string{"10.0.0.0/8"}, Stale: []string{"172.16.0.0/12"}}
	if got, want := drift.String(), "missing=10.0.0.0/8 stale=172.16.0.0/12"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if !(tunPolicyBypassDrift{}).Empty() {
		t.Fatal("zero drift must be empty")
	}
}

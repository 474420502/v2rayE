package native

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/storage"

	xrayrouter "github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

func TestRoutingTestResolvesDomainForIPIfNonMatch(t *testing.T) {
	svc := newRoutingTestService(t)
	svc.UpdateRoutingConfig(domain.RoutingConfig{
		Mode:           "custom",
		DomainStrategy: "IPIfNonMatch",
		Rules: []domain.RoutingRule{
			{ID: "cn-cidr", Type: "ip", Values: []string{"101.132.0.0/16"}, Outbound: "direct"},
		},
	})

	originalLookup := routingLookupIPAddr
	t.Cleanup(func() { routingLookupIPAddr = originalLookup })
	routingLookupIPAddr = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		_ = ctx
		if host != "cip.cc" {
			return nil, nil
		}
		return []net.IPAddr{{IP: net.ParseIP("101.132.60.229")}}, nil
	}

	result := svc.TestRouting(domain.RoutingTestRequest{Target: "cip.cc", Protocol: "tcp", Port: 443})
	if result.Outbound != "direct" {
		t.Fatalf("expected outbound=direct for resolved CN IP, got %q (rule=%q value=%q resolved=%#v note=%q)", result.Outbound, result.MatchedRule, result.MatchedValue, result.ResolvedIPs, result.Note)
	}
	if result.MatchedValue != "101.132.0.0/16" {
		t.Fatalf("expected matchedValue=101.132.0.0/16, got %q", result.MatchedValue)
	}
	if len(result.ResolvedIPs) == 0 || result.ResolvedIPs[0] != "101.132.60.229" {
		t.Fatalf("expected resolvedIps to include stubbed CN IP, got %#v", result.ResolvedIPs)
	}
	if result.Note == "" {
		t.Fatalf("expected note to mention resolved IP matching")
	}
}

func TestRuleMatchesGeoIPCNFromAsset(t *testing.T) {
	assetDir := t.TempDir()
	t.Setenv("XRAY_LOCATION_ASSET", assetDir)
	t.Setenv("V2RAY_LOCATION_ASSET", assetDir)
	resetGeoIPMatcherCache()
	t.Cleanup(resetGeoIPMatcherCache)

	if err := writeTestGeoIPAsset(assetDir, &xrayrouter.GeoIP{
		CountryCode: "CN",
		Cidr: []*xrayrouter.CIDR{{
			Ip:     net.ParseIP("101.132.0.0").To4(),
			Prefix: 16,
		}},
	}); err != nil {
		t.Fatalf("write test geoip.dat: %v", err)
	}

	matched, ok := ruleMatchesIP("101.132.60.229", []string{"geoip:cn"})
	if !ok {
		t.Fatalf("expected geoip:cn rule to match IP from test asset")
	}
	if matched != "geoip:cn" {
		t.Fatalf("expected matched value geoip:cn, got %q", matched)
	}
}

func TestBuildRoutingRulesAddsForceProxyForLocalProxyInbound(t *testing.T) {
	t.Parallel()

	rules := buildRoutingRulesWithConfig(map[string]interface{}{"localProxyMode": "force-proxy"}, domain.RoutingConfig{Mode: "bypass_cn"}, false, false)
	found := false
	for _, raw := range rules {
		rule, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		tags, ok := rule["inboundTag"].([]string)
		if !ok {
			continue
		}
		if containsString(tags, "http") && containsString(tags, "socks") {
			if outbound, _ := rule["outboundTag"].(string); outbound != "proxy" {
				t.Fatalf("expected force-proxy inbound rule to use proxy outbound, got %q", outbound)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected force-proxy inboundTag rule for local http/socks traffic")
	}
}

func newRoutingTestService(t *testing.T) *Service {
	t.Helper()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return New(t.TempDir(), "xray", store)
}

func writeTestGeoIPAsset(dir string, entries ...*xrayrouter.GeoIP) error {
	data, err := proto.Marshal(&xrayrouter.GeoIPList{Entry: entries})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "geoip.dat"), data, 0o600)
}
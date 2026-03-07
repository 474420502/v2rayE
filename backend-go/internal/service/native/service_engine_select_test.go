package native

import (
	"testing"

	"v2raye/backend-go/internal/domain"
)

func TestSelectCoreEngine(t *testing.T) {
	vmess := &domain.ProfileItem{Protocol: domain.ProtocolVMess}
	vless := &domain.ProfileItem{
		Protocol: domain.ProtocolVLESS,
		VLESS:    &domain.VLESSConfig{UUID: "11111111-1111-1111-1111-111111111111", Encryption: "none"},
	}
	vlessReality := &domain.ProfileItem{
		Protocol: domain.ProtocolVLESS,
		VLESS:    &domain.VLESSConfig{UUID: "11111111-1111-1111-1111-111111111111", Encryption: "none"},
		Transport: &domain.TransportConfig{
			Network:          "tcp",
			TLS:              true,
			SNI:              "example.com",
			Fingerprint:      "chrome",
			RealityPublicKey: "pubkey",
			RealityShortID:   "shortid",
		},
	}
	globalRouting := domain.RoutingConfig{Mode: "global"}
	bypassRouting := domain.RoutingConfig{Mode: "bypass_cn"}

	cases := []struct {
		name       string
		cfg        map[string]interface{}
		routing    domain.RoutingConfig
		profile    *domain.ProfileItem
		embedded   bool
		expectMode string
		resolved   string
	}{
		{
			name:       "default mode uses xray core",
			cfg:        map[string]interface{}{},
			routing:    globalRouting,
			profile:    vless,
			embedded:   false,
			expectMode: "xray-core",
			resolved:   "xray-core",
		},
		{
			name:       "legacy auto still maps to xray core",
			cfg:        map[string]interface{}{"coreEngine": "auto"},
			routing:    globalRouting,
			profile:    vmess,
			embedded:   false,
			expectMode: "xray-core",
			resolved:   "xray-core",
		},
		{
			name:       "auto vless reality uses xray core",
			cfg:        map[string]interface{}{"coreEngine": "auto"},
			routing:    globalRouting,
			profile:    vlessReality,
			embedded:   false,
			expectMode: "xray-core",
			resolved:   "xray-core",
		},
		{
			name:       "legacy embedded still maps to xray core",
			cfg:        map[string]interface{}{},
			routing:    bypassRouting,
			profile:    vless,
			embedded:   false,
			expectMode: "xray-core",
			resolved:   "xray-core",
		},
		{
			name:       "explicit xray uses xray core",
			cfg:        map[string]interface{}{"coreEngine": "xray"},
			routing:    globalRouting,
			profile:    vless,
			embedded:   false,
			expectMode: "xray-core",
			resolved:   "xray-core",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			embedded, mode, resolved := selectCoreEngine(tc.cfg, tc.routing, tc.profile)
			if embedded != tc.embedded {
				t.Fatalf("embedded=%v, want %v", embedded, tc.embedded)
			}
			if mode != tc.expectMode {
				t.Fatalf("mode=%s, want %s", mode, tc.expectMode)
			}
			if resolved != tc.resolved {
				t.Fatalf("resolved=%s, want %s", resolved, tc.resolved)
			}
		})
	}
}

package native

import "testing"

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
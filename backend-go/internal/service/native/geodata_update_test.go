package native

import (
	"testing"

	"v2raye/backend-go/internal/domain"
)

func TestRoutingGeoDataRequirements(t *testing.T) {
	tests := []struct {
		name            string
		rc              domain.RoutingConfig
		wantNeedGeoSite bool
		wantNeedGeoIP   bool
	}{
		{
			name:            "bypass cn mode requires geosite and geoip",
			rc:              domain.RoutingConfig{Mode: "bypass_cn"},
			wantNeedGeoSite: true,
			wantNeedGeoIP:   true,
		},
		{
			name: "custom geosite rule requires geosite only",
			rc: domain.RoutingConfig{
				Mode: "custom",
				Rules: []domain.RoutingRule{
					{Type: "geosite", Values: []string{"cn"}, Outbound: "direct"},
				},
			},
			wantNeedGeoSite: true,
			wantNeedGeoIP:   false,
		},
		{
			name: "custom geoip rule requires geoip only",
			rc: domain.RoutingConfig{
				Mode: "custom",
				Rules: []domain.RoutingRule{
					{Type: "geoip", Values: []string{"cn"}, Outbound: "direct"},
				},
			},
			wantNeedGeoSite: false,
			wantNeedGeoIP:   true,
		},
		{
			name: "custom non geodata rules require nothing",
			rc: domain.RoutingConfig{
				Mode: "custom",
				Rules: []domain.RoutingRule{
					{Type: "domain", Values: []string{"example.com"}, Outbound: "proxy"},
				},
			},
			wantNeedGeoSite: false,
			wantNeedGeoIP:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotNeedGeoSite, gotNeedGeoIP := routingGeoDataRequirements(tc.rc)
			if gotNeedGeoSite != tc.wantNeedGeoSite || gotNeedGeoIP != tc.wantNeedGeoIP {
				t.Fatalf("routingGeoDataRequirements()=(%v,%v), want (%v,%v)", gotNeedGeoSite, gotNeedGeoIP, tc.wantNeedGeoSite, tc.wantNeedGeoIP)
			}
		})
	}
}

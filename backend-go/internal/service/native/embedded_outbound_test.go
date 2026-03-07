package native

import (
	"testing"

	"v2raye/backend-go/internal/domain"
)

func TestBuildEmbeddedDialerSupportedProtocols(t *testing.T) {
	cases := []domain.ProfileItem{
		{
			Protocol: domain.ProtocolShadowsocks,
			Address:  "127.0.0.1",
			Port:     8388,
			Shadowsocks: &domain.ShadowsocksConfig{
				Method:   "aes-128-gcm",
				Password: "pass",
			},
		},
		{
			Protocol: domain.ProtocolTrojan,
			Address:  "example.com",
			Port:     443,
			Trojan: &domain.TrojanConfig{
				Password: "secret",
			},
			Transport: &domain.TransportConfig{Network: "tcp", TLS: true, SNI: "example.com"},
		},
		{
			Protocol: domain.ProtocolVLESS,
			Address:  "example.com",
			Port:     443,
			VLESS: &domain.VLESSConfig{
				UUID:       "11111111-1111-1111-1111-111111111111",
				Encryption: "none",
			},
			Transport: &domain.TransportConfig{Network: "tcp", TLS: true, SNI: "example.com"},
		},
		{
			Protocol: domain.ProtocolVLESS,
			Address:  "example.com",
			Port:     443,
			VLESS: &domain.VLESSConfig{
				UUID:       "11111111-1111-1111-1111-111111111111",
				Encryption: "none",
			},
			Transport: &domain.TransportConfig{
				Network: "ws",
				TLS:     true,
				SNI:     "example.com",
				WSPath:  "/ray",
				WSHeaders: map[string]string{
					"Host": "edge.example.com",
				},
			},
		},
	}

	for _, profile := range cases {
		profile := profile
		t.Run(profile.Protocol, func(t *testing.T) {
			t.Parallel()
			d, err := buildEmbeddedDialer(&profile)
			if err != nil {
				t.Fatalf("expected dialer for protocol %s, got error: %v", profile.Protocol, err)
			}
			if d == nil {
				t.Fatalf("expected non-nil dialer for protocol %s", profile.Protocol)
			}
		})
	}
}

func TestBuildEmbeddedDialerVMessUnsupported(t *testing.T) {
	profile := domain.ProfileItem{
		Protocol: domain.ProtocolVMess,
		Address:  "example.com",
		Port:     443,
		VMess: &domain.VMessConfig{
			UUID:     "11111111-1111-1111-1111-111111111111",
			AlterID:  0,
			Security: "auto",
		},
	}

	_, err := buildEmbeddedDialer(&profile)
	if err == nil {
		t.Fatalf("expected vmess to be unsupported in embedded dialer")
	}
}

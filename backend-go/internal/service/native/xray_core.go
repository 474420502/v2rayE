package native

import (
	xraycore "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
	_ "github.com/xtls/xray-core/main/json"

	"v2raye/backend-go/internal/domain"
)

type managedXrayCore struct {
	instance *xraycore.Instance
}

func startManagedXrayCore(configJSON []byte) (*managedXrayCore, error) {
	instance, err := xraycore.StartInstance("json", configJSON)
	if err != nil {
		return nil, err
	}
	return &managedXrayCore{instance: instance}, nil
}

func (c *managedXrayCore) Close() error {
	if c == nil || c.instance == nil {
		return nil
	}
	return c.instance.Close()
}

func (c *managedXrayCore) IsRunning() bool {
	if c == nil || c.instance == nil {
		return false
	}
	return c.instance.IsRunning()
}

func supportsLightweightEmbedded(cfg map[string]interface{}, routing domain.RoutingConfig, profile *domain.ProfileItem) bool {
	if tunModeFromConfig(cfg) != "off" {
		return false
	}
	if routing.Mode != "" && routing.Mode != "global" {
		return false
	}
	if len(routing.Rules) > 0 {
		return false
	}
	if profile == nil {
		return true
	}

	switch profile.Protocol {
	case domain.ProtocolShadowsocks:
		if profile.Shadowsocks == nil {
			return false
		}
		if profile.Shadowsocks.Plugin != "" || profile.Shadowsocks.PluginOpts != "" {
			return false
		}
		method := profile.Shadowsocks.Method
		return method == "aes-128-gcm" || method == "aes-256-gcm"
	case domain.ProtocolTrojan:
		if profile.Trojan == nil {
			return false
		}
		return transportSupportedByLightweightEmbedded(profile.Transport, false)
	case domain.ProtocolVLESS:
		if profile.VLESS == nil {
			return false
		}
		if profile.VLESS.Flow != "" {
			return false
		}
		return transportSupportedByLightweightEmbedded(profile.Transport, true)
	default:
		return false
	}
}

func transportSupportedByLightweightEmbedded(transport *domain.TransportConfig, allowWS bool) bool {
	if transport == nil {
		return true
	}
	if transport.RealityPublicKey != "" || transport.RealityShortID != "" {
		return false
	}
	if transport.Fingerprint != "" {
		return false
	}
	if len(transport.ALPN) > 0 {
		return false
	}
	network := transport.Network
	if network == "" {
		network = "tcp"
	}
	switch network {
	case "tcp":
		return true
	case "ws":
		return allowWS
	default:
		return false
	}
}
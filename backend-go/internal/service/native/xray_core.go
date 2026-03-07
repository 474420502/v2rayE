package native

import (
	xraycore "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
	_ "github.com/xtls/xray-core/main/json"

	"v2raye/backend-go/internal/domain"
)

type managedXrayCore struct {
	instance       *xraycore.Instance
	restoreLogFunc func() // restores the previous xray-core log handler on close
}

func startManagedXrayCore(configJSON []byte, broker *logBroker) (*managedXrayCore, error) {
	// Register our log handler BEFORE starting the instance so we capture
	// all startup output in embedded mode.
	restore := RegisterXrayLogHandler(broker)

	instance, err := xraycore.StartInstance("json", configJSON)
	if err != nil {
		restore() // revert handler if start failed
		return nil, err
	}
	return &managedXrayCore{instance: instance, restoreLogFunc: restore}, nil
}

func (c *managedXrayCore) Close() error {
	if c == nil || c.instance == nil {
		return nil
	}
	err := c.instance.Close()
	if c.restoreLogFunc != nil {
		c.restoreLogFunc()
	}
	return err
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
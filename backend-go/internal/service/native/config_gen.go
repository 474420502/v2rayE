package native

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/storage"
)

var builtinCNIPs = []string{
	"1.0.1.0/24",
	"1.2.4.0/24",
	"14.102.0.0/21",
	"14.196.0.0/16",
	"27.0.0.0/10",
	"27.36.0.0/14",
	"27.116.0.0/14",
	"27.176.0.0/12",
	"36.32.0.0/12",
	"36.152.0.0/12",
	"42.0.0.0/8",
	"42.176.0.0/12",
	"58.14.0.0/15",
	"58.16.0.0/13",
	"60.0.0.0/10",
	"101.0.0.0/10",
	"103.0.0.0/10",
	"106.0.0.0/8",
	"110.0.0.0/7",
	"112.0.0.0/6",
	"116.0.0.0/6",
	"120.0.0.0/6",
	"123.0.0.0/8",
	"124.0.0.0/7",
	"140.64.0.0/11",
	"175.0.0.0/8",
	"180.64.0.0/10",
	"182.0.0.0/8",
	"202.0.0.0/9",
	"210.0.0.0/10",
	"218.0.0.0/8",
	"221.0.0.0/9",
	"222.0.0.0/8",
	"223.0.0.0/8",
}

// generateXrayConfig produces a complete Xray config.json for the given profile.
func generateXrayConfig(
	profile domain.ProfileItem,
	cfg map[string]interface{},
	routing domain.RoutingConfig,
) ([]byte, error) {
	socksPort := intCfg(cfg, "socksPort", 10808)
	httpPort := intCfg(cfg, "httpPort", 10809)
	statsPort := intCfg(cfg, "statsPort", 10085)
	listenAddr := strCfg(cfg, "listenAddr", "127.0.0.1")
	allowLAN := boolCfg(cfg, "allowLan", false)
	logLevel := strCfg(cfg, "logLevel", "warning")
	skipCertVerify := boolCfg(cfg, "skipCertVerify", false)
	tunMode := tunModeFromConfig(cfg)
	enableTun := tunMode != "off"
	if allowLAN {
		listenAddr = "0.0.0.0"
	}

	inbounds := []interface{}{
		map[string]interface{}{
			"tag":      "socks",
			"listen":   listenAddr,
			"port":     socksPort,
			"protocol": "socks",
			"settings": map[string]interface{}{
				"auth": "noauth",
				"udp":  true,
			},
			"sniffing": map[string]interface{}{
				"enabled":      true,
				"destOverride": []string{"http", "tls"},
				"routeOnly":    false,
			},
		},
		map[string]interface{}{
			"tag":      "http",
			"listen":   listenAddr,
			"port":     httpPort,
			"protocol": "http",
			"settings": map[string]interface{}{
				"allowTransparent": false,
			},
		},
		map[string]interface{}{
			"tag":      "api",
			"listen":   "127.0.0.1",
			"port":     statsPort,
			"protocol": "dokodemo-door",
			"settings": map[string]interface{}{
				"address": "127.0.0.1",
			},
		},
	}

	if enableTun {
		inbounds = append(inbounds, map[string]interface{}{
			"tag":      "tun",
			"protocol": "tun",
			"settings": map[string]interface{}{
				"name":        strCfg(cfg, "tunName", "xraye0"),
				"mtu":         intCfg(cfg, "tunMtu", 1500),
				"userLevel":   0,
				"stack":       tunMode,
				"autoRoute":   effectiveTunInboundAutoRoute(cfg),
				"strictRoute": boolCfg(cfg, "tunStrictRoute", false),
			},
			"sniffing": map[string]interface{}{
				"enabled":      true,
				"destOverride": []string{"http", "tls", "quic"},
				"routeOnly":    false,
			},
		})
	}

	outbound, err := buildOutbound(profile, cfg, skipCertVerify)
	if err != nil {
		return nil, fmt.Errorf("build outbound: %w", err)
	}
	directOutbound := map[string]interface{}{
		"tag":      "direct",
		"protocol": "freedom",
		"settings": map[string]interface{}{},
	}
	attachOutboundSockopt(directOutbound, cfg, directOutboundMark(cfg))

	outbounds := []interface{}{
		outbound,
		directOutbound,
		map[string]interface{}{
			"tag":      "block",
			"protocol": "blackhole",
			"settings": map[string]interface{}{"response": map[string]interface{}{"type": "http"}},
		},
	}

	hasGeoIP := hasGeoIPAsset()
	hasGeoSite := hasGeoSiteAsset()

	xrayCfg := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": logLevel,
		},
		"stats": map[string]interface{}{},
		"api": map[string]interface{}{
			"tag":      "api",
			"services": []string{"StatsService"},
		},
		"policy": map[string]interface{}{
			"levels": map[string]interface{}{
				"0": map[string]interface{}{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]interface{}{
				"statsInboundUplink":    true,
				"statsInboundDownlink":  true,
				"statsOutboundUplink":   true,
				"statsOutboundDownlink": true,
			},
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"routing": map[string]interface{}{
			"domainStrategy": routingDomainStrategy(routing),
			"rules":          buildRoutingRulesWithConfig(cfg, routing, hasGeoIP, hasGeoSite),
		},
	}
	if dnsCfg := buildDNSConfig(cfg); dnsCfg != nil {
		xrayCfg["dns"] = dnsCfg
	}

	return json.MarshalIndent(xrayCfg, "", "  ")
}

func localProxyTrafficMode(cfg map[string]interface{}) string {
	mode := strings.ToLower(strings.TrimSpace(strCfg(cfg, "localProxyMode", "follow-routing")))
	switch mode {
	case "force-proxy", "force_proxy", "proxy", "always-proxy":
		return "force-proxy"
	default:
		return "follow-routing"
	}
}

func effectiveTunInboundAutoRoute(cfg map[string]interface{}) bool {
	if !boolCfg(cfg, "tunAutoRoute", true) {
		return false
	}
	if runtime.GOOS == "linux" {
		return false
	}
	return true
}

func tunModeFromConfig(cfg map[string]interface{}) string {
	mode := strings.ToLower(strings.TrimSpace(strCfg(cfg, "tunMode", "")))
	switch mode {
	case "system", "mixed", "gvisor":
		return mode
	case "off", "none", "disabled":
		return "off"
	case "":
		if boolCfg(cfg, "enableTun", false) {
			stack := strings.ToLower(strings.TrimSpace(strCfg(cfg, "tunStack", "mixed")))
			switch stack {
			case "system", "gvisor":
				return stack
			default:
				return "mixed"
			}
		}
		return "off"
	default:
		return "mixed"
	}
}

func buildDNSConfig(cfg map[string]interface{}) map[string]interface{} {
	servers := toStringSlice(cfg["dnsList"])
	if len(servers) == 0 {
		servers = toStringSlice(cfg["dnsServers"])
	}
	if len(servers) == 0 {
		return nil
	}

	items := make([]interface{}, 0, len(servers))
	for _, server := range servers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		items = append(items, server)
	}
	if len(items) == 0 {
		return nil
	}

	return map[string]interface{}{
		"servers": items,
	}
}

func buildOutbound(profile domain.ProfileItem, cfg map[string]interface{}, skipCertVerify bool) (map[string]interface{}, error) {
	outbound := map[string]interface{}{"tag": "proxy"}

	switch profile.Protocol {
	case domain.ProtocolVMess:
		if profile.VMess == nil {
			return nil, fmt.Errorf("missing vmess config")
		}
		security := profile.VMess.Security
		if security == "" {
			security = "auto"
		}
		outbound["protocol"] = "vmess"
		outbound["settings"] = map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": profile.Address,
					"port":    profile.Port,
					"users": []interface{}{
						map[string]interface{}{
							"id":       profile.VMess.UUID,
							"alterId":  profile.VMess.AlterID,
							"security": security,
						},
					},
				},
			},
		}
		outbound["streamSettings"] = buildStreamSettings(profile.Transport, skipCertVerify)

	case domain.ProtocolVLESS:
		if profile.VLESS == nil {
			return nil, fmt.Errorf("missing vless config")
		}
		enc := profile.VLESS.Encryption
		if enc == "" {
			enc = "none"
		}
		u := map[string]interface{}{
			"id":         profile.VLESS.UUID,
			"encryption": enc,
		}
		if profile.VLESS.Flow != "" {
			u["flow"] = profile.VLESS.Flow
		}
		outbound["protocol"] = "vless"
		outbound["settings"] = map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": profile.Address,
					"port":    profile.Port,
					"users":   []interface{}{u},
				},
			},
		}
		outbound["streamSettings"] = buildStreamSettings(profile.Transport, skipCertVerify)

	case domain.ProtocolShadowsocks:
		if profile.Shadowsocks == nil {
			return nil, fmt.Errorf("missing shadowsocks config")
		}
		srv := map[string]interface{}{
			"address":  profile.Address,
			"port":     profile.Port,
			"method":   profile.Shadowsocks.Method,
			"password": profile.Shadowsocks.Password,
		}
		if profile.Shadowsocks.Plugin != "" {
			srv["plugin"] = profile.Shadowsocks.Plugin
			srv["pluginOpts"] = profile.Shadowsocks.PluginOpts
		}
		outbound["protocol"] = "shadowsocks"
		outbound["settings"] = map[string]interface{}{
			"servers": []interface{}{srv},
		}

	case domain.ProtocolTrojan:
		if profile.Trojan == nil {
			return nil, fmt.Errorf("missing trojan config")
		}
		outbound["protocol"] = "trojan"
		outbound["settings"] = map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"address":  profile.Address,
					"port":     profile.Port,
					"password": profile.Trojan.Password,
				},
			},
		}
		outbound["streamSettings"] = buildStreamSettings(profile.Transport, skipCertVerify)

	case domain.ProtocolHysteria2:
		if profile.Hysteria2 == nil {
			return nil, fmt.Errorf("missing hysteria2 config")
		}
		srv := map[string]interface{}{
			"address":  fmt.Sprintf("%s:%d", profile.Address, profile.Port),
			"password": profile.Hysteria2.Password,
		}
		tlsCfg := map[string]interface{}{}
		if profile.Hysteria2.SNI != "" {
			tlsCfg["serverName"] = profile.Hysteria2.SNI
		}
		if profile.Hysteria2.Insecure {
			tlsCfg["insecure"] = true
		}
		if len(tlsCfg) > 0 {
			srv["tls"] = tlsCfg
		}
		if profile.Hysteria2.Obfs != "" {
			srv["obfs"] = map[string]interface{}{
				"type":     profile.Hysteria2.Obfs,
				"password": profile.Hysteria2.ObfsPassword,
			}
		}
		if profile.Hysteria2.UpMbps > 0 {
			srv["up"] = fmt.Sprintf("%d mbps", profile.Hysteria2.UpMbps)
		}
		if profile.Hysteria2.DownMbps > 0 {
			srv["down"] = fmt.Sprintf("%d mbps", profile.Hysteria2.DownMbps)
		}
		outbound["protocol"] = "hysteria2"
		outbound["settings"] = map[string]interface{}{
			"servers": []interface{}{srv},
		}

	case domain.ProtocolTUIC:
		if profile.TUIC == nil {
			return nil, fmt.Errorf("missing tuic config")
		}
		srv := map[string]interface{}{
			"address":  fmt.Sprintf("%s:%d", profile.Address, profile.Port),
			"uuid":     profile.TUIC.UUID,
			"password": profile.TUIC.Password,
		}
		if profile.TUIC.CongestionControl != "" {
			srv["congestionController"] = profile.TUIC.CongestionControl
		}
		tlsCfg := map[string]interface{}{}
		if profile.TUIC.SNI != "" {
			tlsCfg["serverName"] = profile.TUIC.SNI
		}
		if profile.TUIC.Insecure {
			tlsCfg["insecure"] = true
		}
		if len(profile.TUIC.ALPN) > 0 {
			tlsCfg["alpn"] = profile.TUIC.ALPN
		}
		if len(tlsCfg) > 0 {
			srv["tls"] = tlsCfg
		}
		outbound["protocol"] = "tuic"
		outbound["settings"] = map[string]interface{}{
			"servers": []interface{}{srv},
		}

	default:
		return nil, fmt.Errorf("unsupported protocol: %q", profile.Protocol)
	}

	attachOutboundSockopt(outbound, cfg, 0)

	return outbound, nil
}

func attachOutboundSockopt(outbound map[string]interface{}, cfg map[string]interface{}, mark int) {
	iface := strings.TrimSpace(strCfg(cfg, "outboundInterface", ""))
	if iface == "" && mark <= 0 {
		return
	}
	streamSettings, _ := outbound["streamSettings"].(map[string]interface{})
	if streamSettings == nil {
		streamSettings = map[string]interface{}{}
		outbound["streamSettings"] = streamSettings
	}
	sockopt, _ := streamSettings["sockopt"].(map[string]interface{})
	if sockopt == nil {
		sockopt = map[string]interface{}{}
		streamSettings["sockopt"] = sockopt
	}
	if iface != "" {
		sockopt["interface"] = iface
	}
	if mark > 0 {
		sockopt["mark"] = mark
	}
}

func directOutboundMark(cfg map[string]interface{}) int {
	if runtime.GOOS != "linux" {
		return 0
	}
	if tunModeFromConfig(cfg) == "off" || !boolCfg(cfg, "tunAutoRoute", true) {
		return 0
	}
	return tunDirectBypassMark
}

func buildStreamSettings(transport *domain.TransportConfig, skipCertVerify bool) map[string]interface{} {
	if transport == nil {
		return map[string]interface{}{"network": "tcp", "security": "none"}
	}

	ss := map[string]interface{}{}
	network := transport.Network
	if network == "" {
		network = "tcp"
	}
	ss["network"] = network

	// TLS / Reality
	if transport.TLS || transport.RealityPublicKey != "" {
		security := "tls"
		if transport.RealityPublicKey != "" {
			security = "reality"
		}
		ss["security"] = security

		tlsSettings := map[string]interface{}{}
		if transport.SNI != "" {
			tlsSettings["serverName"] = transport.SNI
		}
		if skipCertVerify || transport.SkipCertVerify {
			tlsSettings["allowInsecure"] = true
		}
		fp := transport.Fingerprint
		if fp == "" && transport.RealityPublicKey != "" {
			fp = "chrome" // Reality requires a TLS fingerprint
		}
		if fp != "" {
			tlsSettings["fingerprint"] = fp
		}
		if len(transport.ALPN) > 0 {
			tlsSettings["alpn"] = transport.ALPN
		}
		if transport.RealityPublicKey != "" {
			tlsSettings["publicKey"] = transport.RealityPublicKey
			if transport.RealityShortID != "" {
				tlsSettings["shortId"] = transport.RealityShortID
			}
			ss["realitySettings"] = tlsSettings
		} else {
			ss["tlsSettings"] = tlsSettings
		}
	} else {
		ss["security"] = "none"
	}

	// Network-specific settings
	switch network {
	case "ws":
		wsCfg := map[string]interface{}{"path": transport.WSPath}
		if len(transport.WSHeaders) > 0 {
			wsCfg["headers"] = transport.WSHeaders
		}
		ss["wsSettings"] = wsCfg
	case "grpc":
		grpcCfg := map[string]interface{}{"serviceName": transport.GRPCServiceName}
		if transport.GRPCMode == "multi" {
			grpcCfg["multiMode"] = true
		}
		ss["grpcSettings"] = grpcCfg
	case "h2":
		h2Cfg := map[string]interface{}{}
		if len(transport.H2Path) > 0 {
			h2Cfg["path"] = transport.H2Path[0]
		}
		if len(transport.H2Host) > 0 {
			h2Cfg["host"] = transport.H2Host
		}
		ss["httpSettings"] = h2Cfg
	case "xhttp", "splithttp":
		xhttpCfg := map[string]interface{}{"path": "/"}
		if transport.WSPath != "" {
			xhttpCfg["path"] = transport.WSPath
		}
		if len(transport.WSHeaders) > 0 {
			if host, ok := transport.WSHeaders["Host"]; ok {
				xhttpCfg["host"] = host
			}
		}
		ss["xhttpSettings"] = xhttpCfg
	}

	return ss
}

func buildRoutingRules(routing domain.RoutingConfig, hasGeoIP, hasGeoSite bool) []interface{} {
	return buildRoutingRulesWithConfig(nil, routing, hasGeoIP, hasGeoSite)
}

func buildRoutingRulesWithConfig(cfg map[string]interface{}, routing domain.RoutingConfig, hasGeoIP, hasGeoSite bool) []interface{} {
	var rules []interface{}
	var defaultRules []interface{}

	// Internal API traffic always routes to the api inbound tag.
	rules = append(rules, map[string]interface{}{
		"type":        "field",
		"inboundTag":  []string{"api"},
		"outboundTag": "api",
	})

	if routingLocalBypassEnabled(routing) {
		// Keep local control-plane traffic out of the proxy chain.
		rules = append(rules,
			map[string]interface{}{
				"type":        "field",
				"domain":      []string{"full:localhost"},
				"outboundTag": "direct",
			},
			map[string]interface{}{
				"type":        "field",
				"ip":          []string{"127.0.0.0/8", "::1/128"},
				"outboundTag": "direct",
			},
		)
	}

	if localProxyTrafficMode(cfg) == "force-proxy" {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"inboundTag":  []string{"http", "socks"},
			"outboundTag": "proxy",
		})
	}

	switch routing.Mode {
	case "bypass_cn":
		privateIPs := []string{"geoip:private"}
		if !hasGeoIP {
			privateIPs = []string{
				"10.0.0.0/8",
				"172.16.0.0/12",
				"192.168.0.0/16",
				"127.0.0.0/8",
				"169.254.0.0/16",
				"::1/128",
				"fe80::/10",
				"fc00::/7",
			}
		}
		rules = append(rules,
			map[string]interface{}{
				"type":        "field",
				"ip":          privateIPs,
				"outboundTag": "direct",
			})
		if hasGeoIP {
			rules = append(rules,
				map[string]interface{}{
					"type":        "field",
					"ip":          []string{"geoip:cn"},
					"outboundTag": "direct",
				})
		} else {
			rules = append(rules,
				map[string]interface{}{
					"type":        "field",
					"ip":          builtinCNIPs,
					"outboundTag": "direct",
				})
		}
		if hasGeoSite {
			rules = append(rules,
				map[string]interface{}{
					"type":        "field",
					"domain":      []string{"geosite:cn"},
					"outboundTag": "direct",
				})
		}
	case "direct":
		defaultRules = append(defaultRules, map[string]interface{}{
			"type":        "field",
			"network":     "tcp,udp",
			"outboundTag": "direct",
		})
		// "global" needs no extra rules — everything goes through proxy by default.
	}

	// User-defined custom rules.
	for _, rule := range routing.Rules {
		r := map[string]interface{}{
			"type":        "field",
			"outboundTag": rule.Outbound,
		}
		switch rule.Type {
		case "domain":
			r["domain"] = rule.Values
		case "ip":
			r["ip"] = rule.Values
		case "geoip":
			if !hasGeoIP {
				continue
			}
			geo := make([]string, 0, len(rule.Values))
			for _, v := range rule.Values {
				geo = append(geo, "geoip:"+v)
			}
			r["ip"] = geo
		case "geosite":
			if !hasGeoSite {
				continue
			}
			geo := make([]string, 0, len(rule.Values))
			for _, v := range rule.Values {
				geo = append(geo, "geosite:"+v)
			}
			r["domain"] = geo
		case "port":
			r["port"] = strings.Join(rule.Values, ",")
		case "protocol":
			r["protocol"] = rule.Values
		}
		rules = append(rules, r)
	}

	rules = append(rules, defaultRules...)

	return rules
}

func routingLocalBypassEnabled(routing domain.RoutingConfig) bool {
	if routing.LocalBypassEnabled == nil {
		return true
	}
	return *routing.LocalBypassEnabled
}

func hasGeoDataAssets() bool {
	return hasGeoIPAsset() && hasGeoSiteAsset()
}

func hasGeoIPAsset() bool {
	return hasGeoAsset("geoip.dat")
}

func hasGeoSiteAsset() bool {
	return hasGeoAsset("geosite.dat")
}

func hasGeoAsset(fileName string) bool {
	for _, dir := range geoDataSearchDirs() {
		if dir == "" {
			continue
		}
		assetPath := filepath.Join(dir, fileName)
		if fileExists(assetPath) {
			return true
		}
	}
	return false
}

func geoDataSearchDirs() []string {
	execPath, _ := os.Executable()
	searchDirs := []string{
		preferredGeoDataDir(),
		strings.TrimSpace(os.Getenv("XRAY_LOCATION_ASSET")),
		strings.TrimSpace(os.Getenv("V2RAY_LOCATION_ASSET")),
		".",
		"/usr/local/share/xray",
		"/usr/share/xray",
		"/usr/local/share/v2ray",
		"/usr/share/v2ray",
		filepath.Dir(execPath),
	}
	seen := make(map[string]struct{}, len(searchDirs))
	result := make([]string, 0, len(searchDirs))
	for _, dir := range searchDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		result = append(result, dir)
	}
	return result
}

func preferredGeoDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("XRAY_LOCATION_ASSET")); dir != "" {
		return filepath.Clean(dir)
	}
	if dir := strings.TrimSpace(os.Getenv("V2RAY_LOCATION_ASSET")); dir != "" {
		return filepath.Clean(dir)
	}
	return storage.DefaultDataDir
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

func routingDomainStrategy(routing domain.RoutingConfig) string {
	if routing.DomainStrategy != "" {
		return routing.DomainStrategy
	}
	return "IPIfNonMatch"
}

func toStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// writeConfigToFile writes xray config bytes to data/runtime-config.json and
// returns the path.
func writeConfigToFile(data []byte, dataDir string) (string, error) {
	path := filepath.Join(dataDir, "runtime-config.json")
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return "", err
	}
	return path, nil
}

// ─── Config extraction helpers ────────────────────────────────────────────────

func intCfg(cfg map[string]interface{}, key string, def int) int {
	if v, ok := cfg[key]; ok {
		switch vt := v.(type) {
		case int:
			return vt
		case float64:
			return int(vt)
		case json.Number:
			if n, err := vt.Int64(); err == nil {
				return int(n)
			}
		}
	}
	return def
}

func strCfg(cfg map[string]interface{}, key string, def string) string {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}

func boolCfg(cfg map[string]interface{}, key string, def bool) bool {
	if v, ok := cfg[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

package native

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"v2raye/backend-go/internal/domain"
)

// ParseSubscriptionURL fetches a subscription URL and returns all parsed profiles.
func ParseSubscriptionURL(subURL, userAgent, subID, subName string) ([]domain.ProfileItem, error) {
	if userAgent == "" {
		userAgent = "v2rayN/7.x"
	}

	// Validate URL scheme to prevent SSRF against internal services.
	parsed, err := url.Parse(subURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid subscription URL scheme (must be http/https)")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return parseSubscriptionContent(string(body), subID, subName)
}

// parseSubscriptionContent parses raw subscription content (may be base64 encoded).
func parseSubscriptionContent(content, subID, subName string) ([]domain.ProfileItem, error) {
	content = strings.TrimSpace(content)

	// Try base64 decode first — subscription services often base64-encode the list.
	if decoded, err := decodeBase64Loose(content); err == nil && looksLikeURIList(decoded) {
		content = decoded
	}

	var profiles []domain.ProfileItem
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		p, err := ParseProfileURI(line, subID, subName)
		if err != nil {
			continue
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// ParseProfileURI parses a single URI string into a ProfileItem.
func ParseProfileURI(uri, subID, subName string) (domain.ProfileItem, error) {
	switch {
	case strings.HasPrefix(uri, "vmess://"):
		return parseVMess(uri, subID, subName)
	case strings.HasPrefix(uri, "vless://"):
		return parseVLESS(uri, subID, subName)
	case strings.HasPrefix(uri, "ss://"):
		return parseShadowsocks(uri, subID, subName)
	case strings.HasPrefix(uri, "trojan://"):
		return parseTrojan(uri, subID, subName)
	case strings.HasPrefix(uri, "hysteria2://"),
		strings.HasPrefix(uri, "hy2://"):
		return parseHysteria2(uri, subID, subName)
	case strings.HasPrefix(uri, "tuic://"):
		return parseTUIC(uri, subID, subName)
	default:
		preview := uri
		if len(preview) > 30 {
			preview = preview[:30] + "..."
		}
		return domain.ProfileItem{}, fmt.Errorf("unsupported protocol in URI: %s", preview)
	}
}

// ─── VMess ────────────────────────────────────────────────────────────────────

func parseVMess(uri, subID, subName string) (domain.ProfileItem, error) {
	encoded := strings.TrimPrefix(uri, "vmess://")
	decoded, err := decodeBase64Loose(encoded)
	if err != nil {
		return domain.ProfileItem{}, fmt.Errorf("vmess base64: %w", err)
	}

	var v struct {
		PS   string      `json:"ps"`
		Add  string      `json:"add"`
		Port interface{} `json:"port"`
		ID   string      `json:"id"`
		Aid  interface{} `json:"aid"`
		SCY  string      `json:"scy"`
		Net  string      `json:"net"`
		Host string      `json:"host"`
		Path string      `json:"path"`
		TLS  string      `json:"tls"`
		SNI  string      `json:"sni"`
		ALPN string      `json:"alpn"`
		FP   string      `json:"fp"`
	}
	if err := json.Unmarshal([]byte(decoded), &v); err != nil {
		return domain.ProfileItem{}, fmt.Errorf("vmess json: %w", err)
	}

	network := v.Net
	if network == "" {
		network = "tcp"
	}
	transport := &domain.TransportConfig{Network: network}
	if v.TLS == "tls" {
		transport.TLS = true
		transport.SNI = v.SNI
		transport.Fingerprint = v.FP
		transport.ALPN = splitComma(v.ALPN)
	}
	switch network {
	case "ws":
		transport.WSPath = v.Path
		if v.Host != "" {
			transport.WSHeaders = map[string]string{"Host": v.Host}
		}
	case "grpc":
		transport.GRPCServiceName = v.Path
	case "h2":
		transport.H2Path = []string{v.Path}
		if v.Host != "" {
			transport.H2Host = strings.Split(v.Host, ",")
		}
	}

	return domain.ProfileItem{
		ID:       newProfileID(),
		Name:     v.PS,
		Protocol: domain.ProtocolVMess,
		Address:  v.Add,
		Port:     parseIntField(v.Port),
		SubID:    subID,
		SubName:  subName,
		VMess: &domain.VMessConfig{
			UUID:     v.ID,
			AlterID:  parseIntField(v.Aid),
			Security: v.SCY,
		},
		Transport: transport,
	}, nil
}

// ─── VLESS ────────────────────────────────────────────────────────────────────

func parseVLESS(uri, subID, subName string) (domain.ProfileItem, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return domain.ProfileItem{}, fmt.Errorf("vless url: %w", err)
	}

	uuid := u.User.Username()
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	name, _ := url.QueryUnescape(u.Fragment)
	if name == "" {
		name = host
	}

	q := u.Query()
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	transport := &domain.TransportConfig{Network: network}
	security := q.Get("security")
	switch security {
	case "tls":
		transport.TLS = true
		transport.SNI = q.Get("sni")
		transport.Fingerprint = q.Get("fp")
		transport.ALPN = splitComma(q.Get("alpn"))
	case "reality":
		transport.TLS = true
		transport.SNI = q.Get("sni")
		transport.Fingerprint = q.Get("fp")
		transport.RealityPublicKey = q.Get("pbk")
		transport.RealityShortID = q.Get("sid")
	}
	switch network {
	case "ws":
		transport.WSPath = q.Get("path")
		if h := q.Get("host"); h != "" {
			transport.WSHeaders = map[string]string{"Host": h}
		}
	case "grpc":
		transport.GRPCServiceName = q.Get("serviceName")
		if q.Get("mode") == "multi" {
			transport.GRPCMode = "multi"
		}
	case "xhttp", "splithttp":
		// normalize legacy "splithttp" to "xhttp"
		transport.Network = "xhttp"
		transport.WSPath = q.Get("path")
		if h := q.Get("host"); h != "" {
			transport.WSHeaders = map[string]string{"Host": h}
		}
	}

	return domain.ProfileItem{
		ID:       newProfileID(),
		Name:     name,
		Protocol: domain.ProtocolVLESS,
		Address:  host,
		Port:     port,
		SubID:    subID,
		SubName:  subName,
		VLESS: &domain.VLESSConfig{
			UUID:       uuid,
			Flow:       q.Get("flow"),
			Encryption: q.Get("encryption"),
		},
		Transport: transport,
	}, nil
}

// ─── Shadowsocks ──────────────────────────────────────────────────────────────

func parseShadowsocks(uri, subID, subName string) (domain.ProfileItem, error) {
	s := strings.TrimPrefix(uri, "ss://")

	// Split fragment (name).
	name := ""
	if idx := strings.LastIndex(s, "#"); idx != -1 {
		name, _ = url.QueryUnescape(s[idx+1:])
		s = s[:idx]
	}

	// Try SIP002 format: base64(method:password)@host:port
	u, err := url.Parse("ss://" + s)
	if err == nil && u.User != nil && u.Host != "" {
		userInfo := u.User.String()
		decoded, decErr := decodeBase64Loose(userInfo)
		if decErr != nil {
			decoded = userInfo
		}
		if parts := strings.SplitN(decoded, ":", 2); len(parts) == 2 {
			host := u.Hostname()
			port, _ := strconv.Atoi(u.Port())
			if name == "" {
				name = host
			}
			return domain.ProfileItem{
				ID:       newProfileID(),
				Name:     name,
				Protocol: domain.ProtocolShadowsocks,
				Address:  host,
				Port:     port,
				SubID:    subID,
				SubName:  subName,
				Shadowsocks: &domain.ShadowsocksConfig{
					Method:   parts[0],
					Password: parts[1],
				},
			}, nil
		}
	}

	// Legacy: base64(method:password@host:port)
	decoded, err := decodeBase64Loose(s)
	if err != nil {
		return domain.ProfileItem{}, fmt.Errorf("ss parse: %w", err)
	}
	atIdx := strings.LastIndex(decoded, "@")
	if atIdx == -1 {
		return domain.ProfileItem{}, fmt.Errorf("ss: missing @ in legacy format")
	}
	credPart := decoded[:atIdx]
	hostPart := decoded[atIdx+1:]
	ci := strings.Index(credPart, ":")
	if ci == -1 {
		return domain.ProfileItem{}, fmt.Errorf("ss: missing : in credentials")
	}
	method := credPart[:ci]
	password := credPart[ci+1:]
	host, portStr, _ := strings.Cut(hostPart, ":")
	port, _ := strconv.Atoi(portStr)
	if name == "" {
		name = host
	}

	return domain.ProfileItem{
		ID:       newProfileID(),
		Name:     name,
		Protocol: domain.ProtocolShadowsocks,
		Address:  host,
		Port:     port,
		SubID:    subID,
		SubName:  subName,
		Shadowsocks: &domain.ShadowsocksConfig{
			Method:   method,
			Password: password,
		},
	}, nil
}

// ─── Trojan ───────────────────────────────────────────────────────────────────

func parseTrojan(uri, subID, subName string) (domain.ProfileItem, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return domain.ProfileItem{}, fmt.Errorf("trojan url: %w", err)
	}

	password := u.User.Username()
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	name, _ := url.QueryUnescape(u.Fragment)
	if name == "" {
		name = host
	}

	q := u.Query()
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	transport := &domain.TransportConfig{
		Network:        network,
		TLS:            true,
		SNI:            q.Get("sni"),
		Fingerprint:    q.Get("fp"),
		SkipCertVerify: q.Get("allowInsecure") == "1" || q.Get("allowInsecure") == "true",
	}
	if transport.SNI == "" {
		transport.SNI = host
	}
	switch network {
	case "ws":
		transport.WSPath = q.Get("path")
		if h := q.Get("host"); h != "" {
			transport.WSHeaders = map[string]string{"Host": h}
		}
	case "grpc":
		transport.GRPCServiceName = q.Get("serviceName")
	}

	return domain.ProfileItem{
		ID:        newProfileID(),
		Name:      name,
		Protocol:  domain.ProtocolTrojan,
		Address:   host,
		Port:      port,
		SubID:     subID,
		SubName:   subName,
		Trojan:    &domain.TrojanConfig{Password: password},
		Transport: transport,
	}, nil
}

// ─── Hysteria2 ────────────────────────────────────────────────────────────────

func parseHysteria2(uri, subID, subName string) (domain.ProfileItem, error) {
	// Normalise hy2:// → hysteria2://
	uri = strings.Replace(uri, "hy2://", "hysteria2://", 1)
	u, err := url.Parse(uri)
	if err != nil {
		return domain.ProfileItem{}, fmt.Errorf("hysteria2 url: %w", err)
	}

	password := u.User.Username()
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	name, _ := url.QueryUnescape(u.Fragment)
	if name == "" {
		name = host
	}

	q := u.Query()
	insecure := q.Get("insecure") == "1" || q.Get("insecure") == "true"

	return domain.ProfileItem{
		ID:       newProfileID(),
		Name:     name,
		Protocol: domain.ProtocolHysteria2,
		Address:  host,
		Port:     port,
		SubID:    subID,
		SubName:  subName,
		Hysteria2: &domain.Hysteria2Config{
			Password:     password,
			SNI:          q.Get("sni"),
			Insecure:     insecure,
			Obfs:         q.Get("obfs"),
			ObfsPassword: q.Get("obfs-password"),
		},
	}, nil
}

// ─── TUIC ─────────────────────────────────────────────────────────────────────

func parseTUIC(uri, subID, subName string) (domain.ProfileItem, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return domain.ProfileItem{}, fmt.Errorf("tuic url: %w", err)
	}

	uuid := u.User.Username()
	password, _ := u.User.Password()
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	name, _ := url.QueryUnescape(u.Fragment)
	if name == "" {
		name = host
	}

	q := u.Query()
	insecure := q.Get("allow_insecure") == "1" || q.Get("insecure") == "1"

	return domain.ProfileItem{
		ID:       newProfileID(),
		Name:     name,
		Protocol: domain.ProtocolTUIC,
		Address:  host,
		Port:     port,
		SubID:    subID,
		SubName:  subName,
		TUIC: &domain.TUICConfig{
			UUID:              uuid,
			Password:          password,
			SNI:               q.Get("sni"),
			Insecure:          insecure,
			CongestionControl: q.Get("congestion_control"),
			ALPN:              splitComma(q.Get("alpn")),
		},
	}, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// decodeBase64Loose tries all four base64 variants to decode the string.
func decodeBase64Loose(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("base64 decode failed")
}

func looksLikeURIList(s string) bool {
	for _, proto := range []string{
		"vmess://", "vless://", "ss://", "trojan://",
		"hysteria2://", "hy2://", "tuic://",
	} {
		if strings.Contains(s, proto) {
			return true
		}
	}
	return false
}

func newProfileID() string {
	return fmt.Sprintf("p%d", time.Now().UnixNano())
}

func parseIntField(v interface{}) int {
	switch vt := v.(type) {
	case int:
		return vt
	case float64:
		return int(vt)
	case string:
		n, _ := strconv.Atoi(vt)
		return n
	case json.Number:
		n, _ := vt.Int64()
		return int(n)
	}
	return 0
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

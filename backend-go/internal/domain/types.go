package domain

// APIEnvelope is the standard API response wrapper.
type APIEnvelope struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// CoreStatus represents the current state of the proxy core.
type CoreStatus struct {
	Running          bool   `json:"running"`
	CoreType         string `json:"coreType,omitempty"`
	EngineMode       string `json:"engineMode,omitempty"`
	EngineResolved   string `json:"engineResolved,omitempty"`
	CurrentProfileID string `json:"currentProfileId,omitempty"`
	State            string `json:"state,omitempty"` // stopped|starting|running|stopping
	Error            string `json:"error,omitempty"`
	ErrorAt          string `json:"errorAt,omitempty"`
}

// Protocol constants.
const (
	ProtocolVMess       = "vmess"
	ProtocolVLESS       = "vless"
	ProtocolShadowsocks = "shadowsocks"
	ProtocolTrojan      = "trojan"
	ProtocolHysteria2   = "hysteria2"
	ProtocolTUIC        = "tuic"
)

// ProfileItem is a complete proxy server configuration.
type ProfileItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Protocol  string `json:"protocol"` // vmess|vless|shadowsocks|trojan|hysteria2|tuic
	Address   string `json:"address"`
	Port      int    `json:"port"`
	DelayMs   int    `json:"delayMs,omitempty"`
	SubID     string `json:"subId,omitempty"`
	SubName   string `json:"subName,omitempty"`
	SortOrder int    `json:"sortOrder,omitempty"`

	// Protocol-specific configs (only one non-nil per profile).
	VMess       *VMessConfig       `json:"vmess,omitempty"`
	VLESS       *VLESSConfig       `json:"vless,omitempty"`
	Shadowsocks *ShadowsocksConfig `json:"shadowsocks,omitempty"`
	Trojan      *TrojanConfig      `json:"trojan,omitempty"`
	Hysteria2   *Hysteria2Config   `json:"hysteria2,omitempty"`
	TUIC        *TUICConfig        `json:"tuic,omitempty"`

	// Transport layer settings (applies to vmess/vless/trojan).
	Transport *TransportConfig `json:"transport,omitempty"`
}

// VMessConfig holds VMess-specific settings.
type VMessConfig struct {
	UUID     string `json:"uuid"`
	AlterID  int    `json:"alterId,omitempty"`
	Security string `json:"security,omitempty"` // none|auto|aes-128-gcm|chacha20-poly1305
}

// VLESSConfig holds VLESS-specific settings.
type VLESSConfig struct {
	UUID       string `json:"uuid"`
	Flow       string `json:"flow,omitempty"`       // xtls-rprx-vision etc.
	Encryption string `json:"encryption,omitempty"` // usually "none"
}

// ShadowsocksConfig holds Shadowsocks-specific settings.
type ShadowsocksConfig struct {
	Method     string `json:"method"`
	Password   string `json:"password"`
	Plugin     string `json:"plugin,omitempty"`
	PluginOpts string `json:"pluginOpts,omitempty"`
}

// TrojanConfig holds Trojan-specific settings.
type TrojanConfig struct {
	Password string `json:"password"`
}

// Hysteria2Config holds Hysteria2-specific settings.
type Hysteria2Config struct {
	Password     string `json:"password"`
	SNI          string `json:"sni,omitempty"`
	Insecure     bool   `json:"insecure,omitempty"`
	UpMbps       int    `json:"upMbps,omitempty"`
	DownMbps     int    `json:"downMbps,omitempty"`
	Obfs         string `json:"obfs,omitempty"`
	ObfsPassword string `json:"obfsPassword,omitempty"`
}

// TUICConfig holds TUIC-specific settings.
type TUICConfig struct {
	UUID              string   `json:"uuid"`
	Password          string   `json:"password"`
	CongestionControl string   `json:"congestionControl,omitempty"` // bbr|cubic|new_reno
	SNI               string   `json:"sni,omitempty"`
	Insecure          bool     `json:"insecure,omitempty"`
	ALPN              []string `json:"alpn,omitempty"`
}

// TransportConfig holds transport/stream settings for vmess/vless/trojan.
type TransportConfig struct {
	Network string `json:"network"` // tcp|ws|grpc|h2|kcp|quic

	// WebSocket
	WSPath    string            `json:"wsPath,omitempty"`
	WSHeaders map[string]string `json:"wsHeaders,omitempty"`

	// gRPC
	GRPCServiceName string `json:"grpcServiceName,omitempty"`
	GRPCMode        string `json:"grpcMode,omitempty"` // gun|multi

	// HTTP/2
	H2Path []string `json:"h2Path,omitempty"`
	H2Host []string `json:"h2Host,omitempty"`

	// TLS
	TLS            bool     `json:"tls,omitempty"`
	SNI            string   `json:"sni,omitempty"`
	Fingerprint    string   `json:"fingerprint,omitempty"`
	ALPN           []string `json:"alpn,omitempty"`
	SkipCertVerify bool     `json:"skipCertVerify,omitempty"`

	// Reality
	RealityPublicKey string `json:"realityPublicKey,omitempty"`
	RealityShortID   string `json:"realityShortId,omitempty"`
}

// DelayTestResult is the result of a TCP delay test.
type DelayTestResult struct {
	Available bool   `json:"available"`
	DelayMs   int    `json:"delayMs,omitempty"`
	Message   string `json:"message,omitempty"`
}

// SubscriptionItem represents a subscription source.
type SubscriptionItem struct {
	ID                string `json:"id"`
	Remarks           string `json:"remarks"`
	URL               string `json:"url"`
	Enabled           bool   `json:"enabled"`
	UserAgent         string `json:"userAgent,omitempty"`
	Filter            string `json:"filter,omitempty"`
	ConvertTarget     string `json:"convertTarget,omitempty"`
	AutoUpdateMinutes int    `json:"autoUpdateMinutes,omitempty"`
	UpdatedAt         string `json:"updatedAt,omitempty"`
	ProfileCount      int    `json:"profileCount,omitempty"`
}

// SubscriptionUpsertRequest is used to create or update a subscription.
type SubscriptionUpsertRequest struct {
	Remarks           string `json:"remarks"`
	URL               string `json:"url"`
	Enabled           bool   `json:"enabled"`
	UserAgent         string `json:"userAgent,omitempty"`
	Filter            string `json:"filter,omitempty"`
	ConvertTarget     string `json:"convertTarget,omitempty"`
	AutoUpdateMinutes int    `json:"autoUpdateMinutes,omitempty"`
}

// AvailabilityResult is the result of a network availability check.
type AvailabilityResult struct {
	Available bool   `json:"available"`
	ElapsedMs int    `json:"elapsedMs,omitempty"`
	Message   string `json:"message,omitempty"`
}

// SystemProxyApplyRequest is used to apply/clear system proxy.
type SystemProxyApplyRequest struct {
	Mode       string `json:"mode"`
	Exceptions string `json:"exceptions"`
}

// RoutingConfig represents the routing configuration.
type RoutingConfig struct {
	Mode           string        `json:"mode"`           // global|bypass_cn|direct|custom
	DomainStrategy string        `json:"domainStrategy"` // IPIfNonMatch|IPOnDemand|AsIs
	Rules          []RoutingRule `json:"rules,omitempty"`
}

// RoutingRule is a single routing rule.
type RoutingRule struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // domain|ip|geoip|geosite|port|protocol
	Values   []string `json:"values"`
	Outbound string   `json:"outbound"` // proxy|direct|block
}

// RoutingDiagnostics summarizes the runtime routing state and generated rules.
type RoutingDiagnostics struct {
	Mode               string                   `json:"mode"`
	DomainStrategy     string                   `json:"domainStrategy"`
	TunMode            string                   `json:"tunMode"`
	TunEnabled         bool                     `json:"tunEnabled"`
	TunTakeoverActive  bool                     `json:"tunTakeoverActive"`
	DefaultRouteDevice string                   `json:"defaultRouteDevice,omitempty"`
	HasGeoIP           bool                     `json:"hasGeoIP"`
	HasGeoSite         bool                     `json:"hasGeoSite"`
	GeoDataAvailable   bool                     `json:"geoDataAvailable"`
	CurrentProfileID   string                   `json:"currentProfileId,omitempty"`
	CurrentProfileName string                   `json:"currentProfileName,omitempty"`
	RuleCount          int                      `json:"ruleCount"`
	Rules              []map[string]interface{} `json:"rules"`
	GeneratedAt        string                   `json:"generatedAt"`
	Warning            string                   `json:"warning,omitempty"`
}

// RoutingOutboundHit is traffic hit statistics for a specific outbound tag.
type RoutingOutboundHit struct {
	Outbound string `json:"outbound"`
	UpBytes  int64  `json:"upBytes"`
	DownBytes int64 `json:"downBytes"`
	UpSpeed  int64  `json:"upSpeed"`
	DownSpeed int64 `json:"downSpeed"`
}

// RoutingHitStats summarizes runtime hit statistics by outbound tag.
type RoutingHitStats struct {
	UpdatedAt string               `json:"updatedAt"`
	Items     []RoutingOutboundHit `json:"items"`
	Note      string               `json:"note,omitempty"`
}

// TunRepairResult is the result of one-click TUN repair workflow.
type TunRepairResult struct {
	TriggeredAt        string `json:"triggeredAt"`
	WasRunning         bool   `json:"wasRunning"`
	Started            bool   `json:"started"`
	Running            bool   `json:"running"`
	TunEnabled         bool   `json:"tunEnabled"`
	TunTakeoverActive  bool   `json:"tunTakeoverActive"`
	DefaultRouteDevice string `json:"defaultRouteDevice,omitempty"`
	Message            string `json:"message,omitempty"`
	Error              string `json:"error,omitempty"`
}

// StatsResult holds bandwidth statistics.
type StatsResult struct {
	UpBytes   int64 `json:"upBytes"`
	DownBytes int64 `json:"downBytes"`
	UpSpeed   int64 `json:"upSpeed"`   // bytes per second
	DownSpeed int64 `json:"downSpeed"` // bytes per second
}

// LogLine is a single log line from the core process.
type LogLine struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// PersistentState holds stateful runtime info across restarts.
type PersistentState struct {
	CurrentProfileID string `json:"currentProfileId,omitempty"`
	CoreType         string `json:"coreType,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
}

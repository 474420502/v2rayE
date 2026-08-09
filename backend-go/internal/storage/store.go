// Package storage provides JSON-file-backed persistence for v2rayE data.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"v2raye/backend-go/internal/domain"
)

const DefaultDataDir = "/opt/v2rayE"

var legacyDefaultDataDirs = map[string]struct{}{
	"/var/lib/v2raye": {},
	"/var/lib/v2rayE": {},
	"/var/opt/v2rayE": {},
}

// Store manages JSON data files under a single data directory.
// Successful reads are cached in memory and invalidated on save, so hot paths
// (status polls, list views) do not hit disk on every call.
type Store struct {
	dataDir string
	mu      sync.RWMutex

	profiles      []domain.ProfileItem
	profilesOK    bool
	subs          []domain.SubscriptionItem
	subsOK        bool
	config        map[string]interface{}
	configOK      bool
	routing       domain.RoutingConfig
	routingOK     bool
	state         domain.PersistentState
	stateOK       bool
}

// ResolveDataDir normalizes the configured data directory and preserves
// compatibility with historical default locations.
func ResolveDataDir(dataDir string) string {
	trimmed := strings.TrimSpace(dataDir)
	if trimmed == "" {
		return DefaultDataDir
	}
	cleaned := filepath.Clean(trimmed)
	if _, ok := legacyDefaultDataDirs[cleaned]; ok {
		return DefaultDataDir
	}
	return cleaned
}

// New creates a Store and ensures the data directory exists.
func New(dataDir string) (*Store, error) {
	dataDir = ResolveDataDir(dataDir)
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("create data dir %q: %w", dataDir, err)
	}
	return &Store{dataDir: dataDir}, nil
}

// ─── Profiles ────────────────────────────────────────────────────────────────

func (s *Store) profilesPath() string {
	return filepath.Join(s.dataDir, "profiles.json")
}

// LoadProfiles reads all profiles from disk (cached in memory after first read).
func (s *Store) LoadProfiles() ([]domain.ProfileItem, error) {
	s.mu.RLock()
	if s.profilesOK {
		out := append([]domain.ProfileItem(nil), s.profiles...)
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profilesOK {
		return append([]domain.ProfileItem(nil), s.profiles...), nil
	}
	v, err := loadJSON[[]domain.ProfileItem](s.profilesPath(), []domain.ProfileItem{})
	if err != nil {
		return v, err
	}
	s.profiles = append([]domain.ProfileItem(nil), v...)
	s.profilesOK = true
	return append([]domain.ProfileItem(nil), s.profiles...), nil
}

// SaveProfiles writes profiles to disk atomically and refreshes the cache.
func (s *Store) SaveProfiles(profiles []domain.ProfileItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := saveJSON(s.profilesPath(), profiles); err != nil {
		return err
	}
	s.profiles = append([]domain.ProfileItem(nil), profiles...)
	s.profilesOK = true
	return nil
}

// UpdateProfileDelay atomically loads profiles, sets one profile's DelayMs,
// and saves under a single store lock so delay-test writes never revert a
// concurrent subscription update (load-modify-save at the service layer
// across separate Load/Save calls would race).
func (s *Store) UpdateProfileDelay(id string, delayMs int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.profilesUnlocked()
	if err != nil {
		return err
	}
	updated := false
	for i := range profiles {
		if profiles[i].ID == id {
			profiles[i].DelayMs = delayMs
			updated = true
			break
		}
	}
	if !updated {
		return nil
	}
	if err := saveJSON(s.profilesPath(), profiles); err != nil {
		return err
	}
	s.profiles = append([]domain.ProfileItem(nil), profiles...)
	s.profilesOK = true
	return nil
}

// UpdateProfileDelays atomically persists delay results for multiple profiles
// in a single load-modify-save (used by batch delay testing).
func (s *Store) UpdateProfileDelays(delays map[string]int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.profilesUnlocked()
	if err != nil {
		return err
	}
	updated := false
	for i := range profiles {
		if d, ok := delays[profiles[i].ID]; ok {
			profiles[i].DelayMs = d
			updated = true
		}
	}
	if !updated {
		return nil
	}
	if err := saveJSON(s.profilesPath(), profiles); err != nil {
		return err
	}
	s.profiles = append([]domain.ProfileItem(nil), profiles...)
	s.profilesOK = true
	return nil
}

// profilesUnlocked returns a copy of the current profiles, loading from disk
// first if not cached. Caller must hold s.mu.
func (s *Store) profilesUnlocked() ([]domain.ProfileItem, error) {
	if s.profilesOK {
		return append([]domain.ProfileItem(nil), s.profiles...), nil
	}
	v, err := loadJSON[[]domain.ProfileItem](s.profilesPath(), []domain.ProfileItem{})
	if err != nil {
		return nil, err
	}
	return append([]domain.ProfileItem(nil), v...), nil
}

// ─── Subscriptions ───────────────────────────────────────────────────────────

func (s *Store) subscriptionsPath() string {
	return filepath.Join(s.dataDir, "subscriptions.json")
}

// LoadSubscriptions reads all subscriptions from disk (cached in memory after first read).
func (s *Store) LoadSubscriptions() ([]domain.SubscriptionItem, error) {
	s.mu.RLock()
	if s.subsOK {
		out := append([]domain.SubscriptionItem(nil), s.subs...)
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subsOK {
		return append([]domain.SubscriptionItem(nil), s.subs...), nil
	}
	v, err := loadJSON[[]domain.SubscriptionItem](s.subscriptionsPath(), []domain.SubscriptionItem{})
	if err != nil {
		return v, err
	}
	s.subs = append([]domain.SubscriptionItem(nil), v...)
	s.subsOK = true
	return append([]domain.SubscriptionItem(nil), s.subs...), nil
}

// SaveSubscriptions writes subscriptions to disk atomically and refreshes the cache.
func (s *Store) SaveSubscriptions(subs []domain.SubscriptionItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := saveJSON(s.subscriptionsPath(), subs); err != nil {
		return err
	}
	s.subs = append([]domain.SubscriptionItem(nil), subs...)
	s.subsOK = true
	return nil
}

// ─── App Config ──────────────────────────────────────────────────────────────

func (s *Store) configPath() string {
	return filepath.Join(s.dataDir, "config.json")
}

// DefaultConfig returns the default application configuration.
func DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"socksPort":                     10808,
		"httpPort":                      10809,
		"listenAddr":                    "127.0.0.1",
		"allowLan":                      false,
		"logLevel":                      "warning",
		"statsPort":                     10085,
		"autoRun":                       false,
		"skipCertVerify":                false,
		"enableTun":                     false,
		"tunMode":                       "off",
		"tunName":                       "xraye0",
		"tunStack":                      "mixed",
		"tunMtu":                        1500,
		"tunAutoRoute":                  true,
		"tunHijackDefaultRoute":         false,
		"tunHijackDefaultRouteExplicit": false,
		"tunStrictRoute":                false,
		"systemProxyMode":               "forced_clear",
		"localProxyMode":                "follow-routing",
		"systemProxyExceptions":         "",
		"systemProxyUsers":              []interface{}{},
		"coreAutoRestart":               true,
		"coreAutoRestartMaxRetries":     5,
		"coreAutoRestartBackoffMs":      500,
		"coreEngine":                    "xray-core",
		"dnsMode":                       "UseSystemDNS",
		"dnsList":                       []interface{}{"1.1.1.1", "8.8.8.8"},
	}
}

// LoadConfig reads config from disk, filling missing keys with defaults
// (cached in memory after first read; a deep copy is returned so callers can
// mutate the map freely without corrupting the cache).
func (s *Store) LoadConfig() (map[string]interface{}, error) {
	s.mu.RLock()
	if s.configOK {
		out := cloneConfigMap(s.config)
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.configOK {
		return cloneConfigMap(s.config), nil
	}
	cfg, err := loadJSON[map[string]interface{}](s.configPath(), nil)
	if err != nil || cfg == nil {
		// Fresh install (no file yet): cache the defaults so status polls do
		// not hit disk on every read. Other read errors are left uncached so
		// a transient failure retries on the next read.
		if err != nil && !os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		s.config = cloneConfigMap(DefaultConfig())
		s.configOK = true
		return cloneConfigMap(s.config), nil
	}
	cfg = normalizeConfigMap(cfg)
	defaults := DefaultConfig()
	for k, v := range defaults {
		if _, ok := cfg[k]; !ok {
			cfg[k] = v
		}
	}
	s.config = cloneConfigMap(cfg)
	s.configOK = true
	return cloneConfigMap(s.config), nil
}

// SaveConfig writes config to disk atomically and refreshes the cache.
func (s *Store) SaveConfig(cfg map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalized := normalizeConfigMap(cfg)
	if err := saveJSON(s.configPath(), normalized); err != nil {
		return err
	}
	s.config = cloneConfigMap(normalized)
	s.configOK = true
	return nil
}

// cloneConfigMap returns a deep copy of a config map so cached values are
// never mutated by callers.
func cloneConfigMap(cfg map[string]interface{}) map[string]interface{} {
	if cfg == nil {
		return nil
	}
	out := make(map[string]interface{}, len(cfg))
	for k, v := range cfg {
		switch t := v.(type) {
		case map[string]interface{}:
			out[k] = cloneConfigMap(t)
		case []interface{}:
			cp := make([]interface{}, len(t))
			copy(cp, t)
			out[k] = cp
		case []string:
			out[k] = append([]string(nil), t...)
		default:
			out[k] = v
		}
	}
	return out
}

func normalizeConfigMap(cfg map[string]interface{}) map[string]interface{} {
	if cfg == nil {
		return DefaultConfig()
	}

	normalized := make(map[string]interface{}, len(cfg)+1)
	for k, v := range cfg {
		normalized[k] = v
	}

	if dnsList, ok := normalized["dnsServers"]; ok {
		if _, exists := normalized["dnsList"]; !exists {
			normalized["dnsList"] = dnsList
		}
		delete(normalized, "dnsServers")
	}

	// coreEngine 只能是 xray-core
	normalized["coreEngine"] = "xray-core"

	tunMode, hasTunMode := normalized["tunMode"].(string)
	tunMode = normalizeTunMode(tunMode)
	if !hasTunMode || tunMode == "" {
		enabled := false
		switch v := normalized["enableTun"].(type) {
		case bool:
			enabled = v
		case string:
			enabled = v == "true"
		}
		if enabled {
			tunMode = normalizeTunMode(asString(normalized["tunStack"]))
			if tunMode == "off" {
				tunMode = "mixed"
			}
		} else {
			tunMode = "off"
		}
	}
	normalized["tunMode"] = tunMode
	normalized["enableTun"] = tunMode != "off"
	if explicit, ok := normalized["tunHijackDefaultRouteExplicit"].(bool); !ok || !explicit {
		normalized["tunHijackDefaultRoute"] = false
		normalized["tunHijackDefaultRouteExplicit"] = false
	}
	normalized["systemProxyMode"] = normalizeSystemProxyMode(asString(normalized["systemProxyMode"]))
	normalized["localProxyMode"] = normalizeLocalProxyMode(asString(normalized["localProxyMode"]))
	normalized["systemProxyUsers"] = normalizeStringSliceValue(normalized["systemProxyUsers"])
	if tunMode == "off" {
		if asString(normalized["tunStack"]) == "" {
			normalized["tunStack"] = "mixed"
		}
	} else {
		normalized["tunStack"] = tunMode
	}

	return normalized
}

func normalizeStringSliceValue(value interface{}) []interface{} {
	items := make([]interface{}, 0)
	seen := map[string]struct{}{}

	appendToken := func(token string) {
		token = strings.TrimSpace(token)
		if token == "" {
			return
		}
		if _, ok := seen[token]; ok {
			return
		}
		seen[token] = struct{}{}
		items = append(items, token)
	}

	switch v := value.(type) {
	case string:
		for _, part := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
			appendToken(part)
		}
	case []string:
		for _, part := range v {
			appendToken(part)
		}
	case []interface{}:
		for _, part := range v {
			appendToken(asString(part))
		}
	}

	return items
}

func normalizeTunMode(value string) string {
	switch value {
	case "mixed", "system", "gvisor":
		return value
	case "", "off", "disabled", "none":
		return "off"
	default:
		return "mixed"
	}
}

func asString(value interface{}) string {
	s, _ := value.(string)
	return s
}

func normalizeSystemProxyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "forced_change", "change", "global", "manual", "on", "enable", "enabled":
		return "forced_change"
	case "forced_clear", "clear", "off", "disable", "disabled", "direct", "none":
		return "forced_clear"
	case "pac":
		return "pac"
	default:
		return "forced_clear"
	}
}

func normalizeLocalProxyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "follow-routing", "follow_routing", "routing", "default":
		return "follow-routing"
	case "force-proxy", "force_proxy", "proxy", "always-proxy":
		return "force-proxy"
	default:
		return "follow-routing"
	}
}

// ─── Routing ─────────────────────────────────────────────────────────────────

func (s *Store) routingPath() string {
	return filepath.Join(s.dataDir, "routing.json")
}

// DefaultRoutingConfig returns sensible default routing (bypass China).
func DefaultRoutingConfig() domain.RoutingConfig {
	return domain.RoutingConfig{
		Mode:               "bypass_cn",
		DomainStrategy:     "IPIfNonMatch",
		LocalBypassEnabled: boolPtr(true),
		Rules:              []domain.RoutingRule{},
	}
}

// boolPtr is intentionally local to this file and currently only used for
// DefaultRoutingConfig; promote it to a shared helper if reused elsewhere.
func boolPtr(v bool) *bool {
	return &v
}

// LoadRoutingConfig reads routing config from disk (cached in memory after first read).
func (s *Store) LoadRoutingConfig() (domain.RoutingConfig, error) {
	s.mu.RLock()
	if s.routingOK {
		out := cloneRoutingConfig(s.routing)
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.routingOK {
		return cloneRoutingConfig(s.routing), nil
	}
	rc, err := loadJSON[domain.RoutingConfig](s.routingPath(), domain.RoutingConfig{})
	if err != nil || rc.Mode == "" {
		// Fresh install (no file yet): cache the defaults so status polls do
		// not hit disk on every read. Other read errors are left uncached so
		// a transient failure retries on the next read.
		if err != nil && !os.IsNotExist(err) {
			return DefaultRoutingConfig(), nil
		}
		s.routing = cloneRoutingConfig(DefaultRoutingConfig())
		s.routingOK = true
		return cloneRoutingConfig(s.routing), nil
	}
	s.routing = cloneRoutingConfig(rc)
	s.routingOK = true
	return cloneRoutingConfig(s.routing), nil
}

// SaveRoutingConfig writes routing config to disk atomically and refreshes the cache.
func (s *Store) SaveRoutingConfig(rc domain.RoutingConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := saveJSON(s.routingPath(), rc); err != nil {
		return err
	}
	s.routing = cloneRoutingConfig(rc)
	s.routingOK = true
	return nil
}

// cloneRoutingConfig deep-copies rule value slices so cached data is never
// mutated by callers.
func cloneRoutingConfig(rc domain.RoutingConfig) domain.RoutingConfig {
	out := rc
	out.Rules = make([]domain.RoutingRule, len(rc.Rules))
	for i, rule := range rc.Rules {
		out.Rules[i] = rule
		out.Rules[i].Values = append([]string(nil), rule.Values...)
	}
	return out
}

// ─── Runtime State ───────────────────────────────────────────────────────────

func (s *Store) statePath() string {
	return filepath.Join(s.dataDir, "state.json")
}

// LoadState reads persisted runtime state from disk (cached in memory after first read).
func (s *Store) LoadState() (domain.PersistentState, error) {
	s.mu.RLock()
	if s.stateOK {
		out := s.state
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stateOK {
		return s.state, nil
	}
	state, err := loadJSON[domain.PersistentState](s.statePath(), domain.PersistentState{})
	if err != nil {
		return domain.PersistentState{}, nil
	}
	s.state = state
	s.stateOK = true
	return s.state, nil
}

// SaveState writes runtime state to disk atomically and refreshes the cache.
func (s *Store) SaveState(state domain.PersistentState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := saveJSON(s.statePath(), state); err != nil {
		return err
	}
	s.state = state
	s.stateOK = true
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func loadJSON[T any](path string, zero T) (T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return zero, nil
		}
		return zero, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}

func saveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// Package storage provides JSON-file-backed persistence for v2rayE data.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"v2raye/backend-go/internal/domain"
)

// Store manages JSON data files under a single data directory.
type Store struct {
	dataDir string
	mu      sync.RWMutex
}

// New creates a Store and ensures the data directory exists.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("create data dir %q: %w", dataDir, err)
	}
	return &Store{dataDir: dataDir}, nil
}

// ─── Profiles ────────────────────────────────────────────────────────────────

func (s *Store) profilesPath() string {
	return filepath.Join(s.dataDir, "profiles.json")
}

// LoadProfiles reads all profiles from disk.
func (s *Store) LoadProfiles() ([]domain.ProfileItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return loadJSON[[]domain.ProfileItem](s.profilesPath(), []domain.ProfileItem{})
}

// SaveProfiles writes profiles to disk atomically.
func (s *Store) SaveProfiles(profiles []domain.ProfileItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return saveJSON(s.profilesPath(), profiles)
}

// ─── Subscriptions ───────────────────────────────────────────────────────────

func (s *Store) subscriptionsPath() string {
	return filepath.Join(s.dataDir, "subscriptions.json")
}

// LoadSubscriptions reads all subscriptions from disk.
func (s *Store) LoadSubscriptions() ([]domain.SubscriptionItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return loadJSON[[]domain.SubscriptionItem](s.subscriptionsPath(), []domain.SubscriptionItem{})
}

// SaveSubscriptions writes subscriptions to disk atomically.
func (s *Store) SaveSubscriptions(subs []domain.SubscriptionItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return saveJSON(s.subscriptionsPath(), subs)
}

// ─── App Config ──────────────────────────────────────────────────────────────

func (s *Store) configPath() string {
	return filepath.Join(s.dataDir, "config.json")
}

// DefaultConfig returns the default application configuration.
func DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"socksPort":             10808,
		"httpPort":              10809,
		"listenAddr":            "127.0.0.1",
		"allowLan":              false,
		"logLevel":              "warning",
		"statsPort":             10085,
		"autoRun":               false,
		"skipCertVerify":        false,
		"enableTun":             false,
		"tunMode":               "off",
		"tunName":               "xraye0",
		"tunStack":              "mixed",
		"tunMtu":                1500,
		"tunAutoRoute":          true,
		"tunHijackDefaultRoute": false,
		"tunHijackDefaultRouteExplicit": false,
		"tunStrictRoute":        false,
		"systemProxyMode":       "forced_clear",
		"systemProxyExceptions": "",
		"coreAutoRestart":       true,
		"coreAutoRestartMaxRetries": 5,
		"coreAutoRestartBackoffMs":  500,
		"coreEngine":            "xray-core",
		"xrayCmd":               "xray",
		"dnsMode":               "UseSystemDNS",
		"dnsList":               []interface{}{"1.1.1.1", "8.8.8.8"},
	}
}

// LoadConfig reads config from disk, filling missing keys with defaults.
func (s *Store) LoadConfig() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, err := loadJSON[map[string]interface{}](s.configPath(), nil)
	if err != nil || cfg == nil {
		return DefaultConfig(), nil
	}
	cfg = normalizeConfigMap(cfg)
	defaults := DefaultConfig()
	for k, v := range defaults {
		if _, ok := cfg[k]; !ok {
			cfg[k] = v
		}
	}
	return cfg, nil
}

// SaveConfig writes config to disk atomically.
func (s *Store) SaveConfig(cfg map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return saveJSON(s.configPath(), normalizeConfigMap(cfg))
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

	switch asString(normalized["coreEngine"]) {
	case "", "embedded", "auto", "xray", "builtin", "internal":
		normalized["coreEngine"] = "xray-core"
	}

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
	if tunMode == "off" {
		if asString(normalized["tunStack"]) == "" {
			normalized["tunStack"] = "mixed"
		}
	} else {
		normalized["tunStack"] = tunMode
	}

	return normalized
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

// ─── Routing ─────────────────────────────────────────────────────────────────

func (s *Store) routingPath() string {
	return filepath.Join(s.dataDir, "routing.json")
}

// DefaultRoutingConfig returns sensible default routing (bypass China).
func DefaultRoutingConfig() domain.RoutingConfig {
	return domain.RoutingConfig{
		Mode:           "bypass_cn",
		DomainStrategy: "IPIfNonMatch",
		Rules:          []domain.RoutingRule{},
	}
}

// LoadRoutingConfig reads routing config from disk.
func (s *Store) LoadRoutingConfig() (domain.RoutingConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rc, err := loadJSON[domain.RoutingConfig](s.routingPath(), domain.RoutingConfig{})
	if err != nil || rc.Mode == "" {
		return DefaultRoutingConfig(), nil
	}
	return rc, nil
}

// SaveRoutingConfig writes routing config to disk atomically.
func (s *Store) SaveRoutingConfig(rc domain.RoutingConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return saveJSON(s.routingPath(), rc)
}

// ─── Runtime State ───────────────────────────────────────────────────────────

func (s *Store) statePath() string {
	return filepath.Join(s.dataDir, "state.json")
}

// LoadState reads persisted runtime state from disk.
func (s *Store) LoadState() (domain.PersistentState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, err := loadJSON[domain.PersistentState](s.statePath(), domain.PersistentState{})
	if err != nil {
		return domain.PersistentState{}, nil
	}
	return state, nil
}

// SaveState writes runtime state to disk atomically.
func (s *Store) SaveState(state domain.PersistentState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return saveJSON(s.statePath(), state)
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
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

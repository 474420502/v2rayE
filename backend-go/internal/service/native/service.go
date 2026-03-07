package native

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/service"
	"v2raye/backend-go/internal/storage"
)

// Service implements service.BackendService by managing a real Xray process.
type Service struct {
	store    *storage.Store
	dataDir  string
	xrayCmd  string
	xrayCore *managedXrayCore

	mu          sync.Mutex
	proc        *exec.Cmd
	running     bool
	trackedPID  int
	lastError   string
	lastErrorAt string
	lastEngine  string

	logs  *logBroker
	stats *statsTracker

	tunRestoreRoutes []string
}

// New creates a native Service using the given storage and xray binary path.
func New(dataDir, xrayCmd string, store *storage.Store) *Service {
	return &Service{
		store:   store,
		dataDir: dataDir,
		xrayCmd: xrayCmd,
		logs:    newLogBroker(),
	}
}

// ─── Core lifecycle ───────────────────────────────────────────────────────────

func (s *Service) CoreStatus() domain.CoreStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkProcExited()
	return s.buildStatus()
}

func (s *Service) StartCore() domain.CoreStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.checkProcExited()
		if s.running {
			return s.buildStatus()
		}
	}

	cfg, _ := s.store.LoadConfig()
	routing, _ := s.store.LoadRoutingConfig()
	state, _ := s.store.LoadState()
	selectedProfile := s.pickSelectedProfile(state.CurrentProfileID)
	if selectedProfile != nil && state.CurrentProfileID != selectedProfile.ID {
		state.CurrentProfileID = selectedProfile.ID
		if err := s.store.SaveState(state); err != nil {
			log.Printf("[native] StartCore: persist selected profile failed: %v", err)
		}
	}
	useEmbedded, mode, resolved := selectCoreEngine(cfg, routing, selectedProfile)
	_ = useEmbedded

	if mode == "auto" && selectedProfile != nil {
		log.Printf("[native] StartCore: auto mode selected xray-core engine for profile protocol=%s", selectedProfile.Protocol)
	}

	profile, err := s.findProfile(state.CurrentProfileID)
	if err != nil {
		s.setCoreError(err.Error())
		log.Printf("[native] StartCore: no selected profile (%v)", err)
		return s.buildStatus()
	}
	if state.CurrentProfileID != profile.ID {
		state.CurrentProfileID = profile.ID
		if err := s.store.SaveState(state); err != nil {
			log.Printf("[native] StartCore: persist selected profile failed: %v", err)
		}
	}

	if tunModeFromConfig(cfg) != "off" {
		if err := s.cleanupStaleTunInterface(cfg); err != nil {
			s.setCoreError(err.Error())
			log.Printf("[native] StartCore: stale TUN cleanup failed: %v", err)
			return s.buildStatus()
		}
		iface, err := detectDefaultRouteInterface()
		if err != nil {
			log.Printf("[native] StartCore: detect default interface failed: %v", err)
		} else if iface != "" {
			cfg["outboundInterface"] = iface
		}
	}

	data, err := generateXrayConfig(profile, cfg, routing)
	if err != nil {
		log.Printf("[native] StartCore: config gen failed: %v", err)
		return s.buildStatus()
	}
	configPath, err := writeConfigToFile(data, s.dataDir)
	if err != nil {
		log.Printf("[native] StartCore: write config failed: %v", err)
		return s.buildStatus()
	}

	xrayCore, err := startManagedXrayCore(data, s.logs)
	if err != nil {
		s.setCoreError(annotateTunStartError(err, cfg))
		log.Printf("[native] StartCore: xray-core start failed: %v", err)
		return s.buildStatus()
	}

	s.proc = nil
	s.xrayCore = xrayCore
	s.trackedPID = 0
	s.running = true
	s.clearCoreErrorLocked()
	s.lastEngine = resolved

	s.logs.clear()

	statsPort := intCfg(cfg, "statsPort", 10085)
	s.stats = newStatsTracker(statsPort)
	s.stats.reset()
	s.stats.start()

	if tunModeFromConfig(cfg) != "off" && boolCfg(cfg, "tunAutoRoute", true) {
		if err := s.setupTunRouting(cfg); err != nil {
			log.Printf("[native] StartCore: TUN route setup failed (first attempt): %v", err)
			_ = s.cleanupStaleTunInterface(cfg)
			if retryErr := s.setupTunRouting(cfg); retryErr != nil {
				log.Printf("[native] StartCore: TUN route setup failed (retry): %v", retryErr)
				s.setCoreError(fmt.Sprintf("TUN 接管失败，核心已降级为仅本地代理可用: %s", retryErr.Error()))
				s.logs.AppLog("warning", fmt.Sprintf("tun route takeover failed; core kept running: %v", retryErr))
			} else {
				s.logs.AppLog("info", "tun route takeover recovered after stale cleanup retry")
			}
		}
	}

	if err := s.applyConfiguredSystemProxyOnCoreStart(cfg); err != nil {
		log.Printf("[native] StartCore: apply system proxy failed: %v", err)
		s.setCoreError(err.Error())
	}

	log.Printf("[native] core started in managed xray-core mode with config=%s", configPath)
	s.logs.AppLog("info", fmt.Sprintf("core started (engine=%s, profile=%s)", resolved, profile.Name))
	_ = s.saveState()
	return s.buildStatus()
}

func (s *Service) StopCore() domain.CoreStatus {
	s.mu.Lock()
	proc := s.proc
	xrayCore := s.xrayCore
	s.proc = nil
	s.xrayCore = nil
	s.running = false
	s.trackedPID = 0
	s.clearCoreErrorLocked()
	if s.stats != nil {
		s.stats.shutdown()
		s.stats = nil
	}
	s.mu.Unlock()

	if proc != nil && proc.Process != nil {
		killGraceful(proc)
	}
	if xrayCore != nil {
		_ = xrayCore.Close()
	}
	s.clearTunRouting()
	if cfg, _ := s.store.LoadConfig(); tunModeFromConfig(cfg) != "off" {
		if err := s.cleanupStaleTunInterface(cfg); err != nil {
			log.Printf("[native] StopCore: stale TUN cleanup failed: %v", err)
		}
	}
	s.clearSystemProxyOnCoreStop()
	s.logs.AppLog("info", "core stopped")
	_ = s.saveState()
	return s.CoreStatus()
}

func (s *Service) RestartCore() domain.CoreStatus {
	s.StopCore()
	time.Sleep(200 * time.Millisecond)
	return s.StartCore()
}

func (s *Service) ClearCoreError() domain.CoreStatus {
	s.mu.Lock()
	s.clearCoreErrorLocked()
	st := s.buildStatus()
	s.mu.Unlock()
	return st
}

// ─── Profiles ─────────────────────────────────────────────────────────────────

func (s *Service) ListProfiles() []domain.ProfileItem {
	profiles, _ := s.store.LoadProfiles()
	subs, _ := s.store.LoadSubscriptions()
	subMap := make(map[string]string, len(subs))
	for _, sub := range subs {
		subMap[sub.ID] = sub.Remarks
	}
	for i := range profiles {
		if profiles[i].SubName == "" && profiles[i].SubID != "" {
			profiles[i].SubName = subMap[profiles[i].SubID]
		}
	}
	return profiles
}

func (s *Service) GetProfile(id string) (domain.ProfileItem, error) {
	profiles, _ := s.store.LoadProfiles()
	for _, p := range profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.ProfileItem{}, service.ErrNotFound
}

func (s *Service) CreateProfile(input domain.ProfileItem) (domain.ProfileItem, error) {
	if err := validateProfile(input); err != nil {
		return domain.ProfileItem{}, err
	}
	input.ID = newProfileID()
	profiles, _ := s.store.LoadProfiles()
	profiles = append(profiles, input)
	if err := s.store.SaveProfiles(profiles); err != nil {
		return domain.ProfileItem{}, fmt.Errorf("save profiles: %w", err)
	}
	return input, nil
}

func (s *Service) UpdateProfile(id string, input domain.ProfileItem) (domain.ProfileItem, error) {
	if err := validateProfile(input); err != nil {
		return domain.ProfileItem{}, err
	}
	profiles, _ := s.store.LoadProfiles()
	for i, p := range profiles {
		if p.ID == id {
			input.ID = id
			profiles[i] = input
			if err := s.store.SaveProfiles(profiles); err != nil {
				return domain.ProfileItem{}, fmt.Errorf("save profiles: %w", err)
			}
			return input, nil
		}
	}
	return domain.ProfileItem{}, service.ErrNotFound
}

func (s *Service) DeleteProfile(id string) error {
	return s.DeleteProfiles([]string{id})
}

func (s *Service) DeleteProfiles(ids []string) error {
	if len(ids) == 0 {
		return service.ErrInvalidMode
	}

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		idSet[trimmed] = struct{}{}
	}
	if len(idSet) == 0 {
		return service.ErrInvalidMode
	}

	profiles, _ := s.store.LoadProfiles()
	newList := profiles[:0]
	removedIDs := make(map[string]struct{}, len(idSet))
	for _, p := range profiles {
		if _, ok := idSet[p.ID]; ok {
			removedIDs[p.ID] = struct{}{}
			continue
		}
		newList = append(newList, p)
	}
	if len(removedIDs) == 0 {
		return service.ErrNotFound
	}
	if err := s.store.SaveProfiles(newList); err != nil {
		return err
	}

	state, _ := s.store.LoadState()
	selectedRemoved := false
	nextSelectedID := state.CurrentProfileID
	if _, ok := removedIDs[state.CurrentProfileID]; ok {
		selectedRemoved = true
		if len(newList) > 0 {
			nextSelectedID = newList[0].ID
		} else {
			nextSelectedID = ""
		}
		state.CurrentProfileID = nextSelectedID
		if err := s.store.SaveState(state); err != nil {
			return err
		}
	}

	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running && selectedRemoved {
		go func() {
			time.Sleep(100 * time.Millisecond)
			if nextSelectedID == "" {
				s.StopCore()
				return
			}
			s.RestartCore()
		}()
	}

	return nil
}

func (s *Service) SelectProfile(id string) error {
	if _, err := s.GetProfile(id); err != nil {
		return service.ErrNotFound
	}
	state, _ := s.store.LoadState()
	state.CurrentProfileID = id
	if err := s.store.SaveState(state); err != nil {
		return err
	}
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.RestartCore()
		}()
	}
	return nil
}

func (s *Service) TestProfileDelay(id string) domain.DelayTestResult {
	p, err := s.GetProfile(id)
	if err != nil {
		return domain.DelayTestResult{Message: "profile not found"}
	}
	if p.Address == "" || p.Port <= 0 {
		return domain.DelayTestResult{Message: "invalid address/port"}
	}
	addr := net.JoinHostPort(p.Address, fmt.Sprintf("%d", p.Port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	elapsed := int(time.Since(start).Milliseconds())
	if err != nil {
		return domain.DelayTestResult{Available: false, Message: err.Error()}
	}
	conn.Close()

	profiles, _ := s.store.LoadProfiles()
	for i := range profiles {
		if profiles[i].ID == id {
			profiles[i].DelayMs = elapsed
		}
	}
	_ = s.store.SaveProfiles(profiles)

	return domain.DelayTestResult{Available: true, DelayMs: elapsed}
}

func (s *Service) ImportProfileFromURI(uri string) (domain.ProfileItem, error) {
	p, err := ParseProfileURI(uri, "", "")
	if err != nil {
		return domain.ProfileItem{}, err
	}
	return s.CreateProfile(p)
}

// ─── Subscriptions ────────────────────────────────────────────────────────────

func (s *Service) ListSubscriptions() []domain.SubscriptionItem {
	subs, _ := s.store.LoadSubscriptions()
	profiles, _ := s.store.LoadProfiles()
	countMap := make(map[string]int)
	for _, p := range profiles {
		if p.SubID != "" {
			countMap[p.SubID]++
		}
	}
	for i := range subs {
		subs[i].ProfileCount = countMap[subs[i].ID]
	}
	return subs
}

func (s *Service) CreateSubscription(input domain.SubscriptionUpsertRequest) (domain.SubscriptionItem, error) {
	if strings.TrimSpace(input.Remarks) == "" || strings.TrimSpace(input.URL) == "" {
		return domain.SubscriptionItem{}, service.ErrInvalidMode
	}
	item := domain.SubscriptionItem{
		ID:                fmt.Sprintf("s%d", time.Now().UnixNano()),
		Remarks:           strings.TrimSpace(input.Remarks),
		URL:               strings.TrimSpace(input.URL),
		Enabled:           input.Enabled,
		UserAgent:         strings.TrimSpace(input.UserAgent),
		Filter:            strings.TrimSpace(input.Filter),
		ConvertTarget:     strings.TrimSpace(input.ConvertTarget),
		AutoUpdateMinutes: input.AutoUpdateMinutes,
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	subs, _ := s.store.LoadSubscriptions()
	subs = append(subs, item)
	if err := s.store.SaveSubscriptions(subs); err != nil {
		return domain.SubscriptionItem{}, fmt.Errorf("save: %w", err)
	}
	return item, nil
}

func (s *Service) UpdateSubscription(id string, input domain.SubscriptionUpsertRequest) (domain.SubscriptionItem, error) {
	if strings.TrimSpace(input.Remarks) == "" || strings.TrimSpace(input.URL) == "" {
		return domain.SubscriptionItem{}, service.ErrInvalidMode
	}
	subs, _ := s.store.LoadSubscriptions()
	for i, sub := range subs {
		if sub.ID == id {
			subs[i].Remarks = strings.TrimSpace(input.Remarks)
			subs[i].URL = strings.TrimSpace(input.URL)
			subs[i].Enabled = input.Enabled
			subs[i].UserAgent = strings.TrimSpace(input.UserAgent)
			subs[i].Filter = strings.TrimSpace(input.Filter)
			subs[i].ConvertTarget = strings.TrimSpace(input.ConvertTarget)
			subs[i].AutoUpdateMinutes = input.AutoUpdateMinutes
			subs[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := s.store.SaveSubscriptions(subs); err != nil {
				return domain.SubscriptionItem{}, fmt.Errorf("save: %w", err)
			}
			return subs[i], nil
		}
	}
	return domain.SubscriptionItem{}, service.ErrNotFound
}

func (s *Service) DeleteSubscription(id string) error {
	subs, _ := s.store.LoadSubscriptions()
	newSubs := subs[:0]
	found := false
	for _, sub := range subs {
		if sub.ID == id {
			found = true
			continue
		}
		newSubs = append(newSubs, sub)
	}
	if !found {
		return service.ErrNotFound
	}
	if err := s.store.SaveSubscriptions(newSubs); err != nil {
		return err
	}
	profiles, _ := s.store.LoadProfiles()
	kept := profiles[:0]
	for _, p := range profiles {
		if p.SubID != id {
			kept = append(kept, p)
		}
	}
	return s.store.SaveProfiles(kept)
}

func (s *Service) UpdateSubscriptions() int {
	subs, _ := s.store.LoadSubscriptions()
	updated := 0
	for _, sub := range subs {
		if !sub.Enabled {
			continue
		}
		if err := s.UpdateSubscriptionByID(sub.ID); err == nil {
			updated++
		}
	}
	return updated
}

func (s *Service) UpdateSubscriptionByID(id string) error {
	subs, _ := s.store.LoadSubscriptions()
	var sub domain.SubscriptionItem
	found := false
	for _, item := range subs {
		if item.ID == id {
			sub = item
			found = true
			break
		}
	}
	if !found {
		return service.ErrNotFound
	}

	log.Printf("[native] updating subscription %s (%s)", sub.Remarks, sub.URL)
	profiles, err := ParseSubscriptionURL(sub.URL, sub.UserAgent, sub.ID, sub.Remarks)
	if err != nil {
		return fmt.Errorf("fetch subscription: %w", err)
	}

	existing, _ := s.store.LoadProfiles()
	kept := existing[:0]
	for _, p := range existing {
		if p.SubID != id {
			kept = append(kept, p)
		}
	}
	kept = append(kept, profiles...)
	if err := s.store.SaveProfiles(kept); err != nil {
		return err
	}

	for i := range subs {
		if subs[i].ID == id {
			subs[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		}
	}
	return s.store.SaveSubscriptions(subs)
}

// ─── Network & proxy ──────────────────────────────────────────────────────────

func (s *Service) NetworkAvailability() domain.AvailabilityResult {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", "1.1.1.1:80", 5*time.Second)
	elapsed := int(time.Since(start).Milliseconds())
	if err != nil {
		return domain.AvailabilityResult{Available: false, ElapsedMs: elapsed, Message: err.Error()}
	}
	conn.Close()
	return domain.AvailabilityResult{Available: true, ElapsedMs: elapsed}
}

func (s *Service) ApplySystemProxy(mode, exceptions string) (map[string]interface{}, error) {
	switch mode {
	case "forced_change", "forced_clear", "pac":
	default:
		return nil, service.ErrInvalidMode
	}
	cfg, _ := s.store.LoadConfig()
	backend, err := s.applyDesktopSystemProxy(cfg, mode, exceptions)
	if err != nil {
		return nil, err
	}
	cfg["systemProxyMode"] = mode
	cfg["systemProxyExceptions"] = exceptions
	if backend != "" {
		cfg["systemProxyBackend"] = backend
	}
	_ = s.store.SaveConfig(cfg)
	return map[string]interface{}{
		"mode":       mode,
		"exceptions": exceptions,
		"backend":    backend,
		"http": map[string]interface{}{
			"host": portHostForDesktopProxy(cfg),
			"port": intCfg(cfg, "httpPort", 10809),
		},
		"socks": map[string]interface{}{
			"host": portHostForDesktopProxy(cfg),
			"port": intCfg(cfg, "socksPort", 10808),
		},
	}, nil
}

func (s *Service) applyDesktopSystemProxy(cfg map[string]interface{}, mode, exceptions string) (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("%w: linux desktop proxy integration is the only implemented backend", service.ErrSystemProxyUnsupported)
	}

	hasGSettings := hasCommand("gsettings")
	hasKDE6 := hasCommand("kwriteconfig6")
	hasKDE5 := hasCommand("kwriteconfig5")

	if !hasGSettings && !hasKDE6 && !hasKDE5 {
		return "", fmt.Errorf("%w: no supported desktop proxy backend found; install gsettings for GNOME or kwriteconfig5/kwriteconfig6 for KDE", service.ErrSystemProxyUnsupported)
	}

	switch mode {
	case "forced_change":
		if hasGSettings {
			if err := applyGSettingsProxy(cfg, exceptions); err != nil {
				return "", err
			}
			return "gsettings", nil
		}
		if hasKDE6 {
			if err := applyKDEProxy(cfg, exceptions, "kwriteconfig6"); err != nil {
				return "", err
			}
			return "kwriteconfig6", nil
		}
		if err := applyKDEProxy(cfg, exceptions, "kwriteconfig5"); err != nil {
			return "", err
		}
		return "kwriteconfig5", nil
	case "forced_clear":
		if hasGSettings {
			if err := clearGSettingsProxy(); err != nil {
				return "", err
			}
			return "gsettings", nil
		}
		if hasKDE6 {
			if err := clearKDEProxy("kwriteconfig6"); err != nil {
				return "", err
			}
			return "kwriteconfig6", nil
		}
		if err := clearKDEProxy("kwriteconfig5"); err != nil {
			return "", err
		}
		return "kwriteconfig5", nil
	case "pac":
		return "", fmt.Errorf("%w: pac mode is not implemented for linux desktop integration", service.ErrSystemProxyUnsupported)
	default:
		return "", nil
	}
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func applyGSettingsProxy(cfg map[string]interface{}, exceptions string) error {
	listenAddr := portHostForDesktopProxy(cfg)
	httpPort := intCfg(cfg, "httpPort", 10809)
	socksPort := intCfg(cfg, "socksPort", 10808)
	hosts := parseProxyExceptions(exceptions)

	commands := [][]string{
		{"set", "org.gnome.system.proxy", "mode", "manual"},
		{"set", "org.gnome.system.proxy.http", "host", listenAddr},
		{"set", "org.gnome.system.proxy.http", "port", fmt.Sprintf("%d", httpPort)},
		{"set", "org.gnome.system.proxy.https", "host", listenAddr},
		{"set", "org.gnome.system.proxy.https", "port", fmt.Sprintf("%d", httpPort)},
		{"set", "org.gnome.system.proxy.socks", "host", listenAddr},
		{"set", "org.gnome.system.proxy.socks", "port", fmt.Sprintf("%d", socksPort)},
		{"set", "org.gnome.system.proxy", "ignore-hosts", formatGSettingsStringArray(hosts)},
	}

	for _, args := range commands {
		if out, err := runGSettings(args...); err != nil {
			return fmt.Errorf("apply system proxy via gsettings failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
	}

	proxyURL := (&url.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", listenAddr, httpPort)}).String()
	_ = os.Setenv("http_proxy", proxyURL)
	_ = os.Setenv("https_proxy", proxyURL)
	_ = os.Setenv("HTTP_PROXY", proxyURL)
	_ = os.Setenv("HTTPS_PROXY", proxyURL)
	return nil
}

func applyKDEProxy(cfg map[string]interface{}, exceptions, binary string) error {
	listenAddr := portHostForDesktopProxy(cfg)
	httpPort := intCfg(cfg, "httpPort", 10809)
	socksPort := intCfg(cfg, "socksPort", 10808)
	noProxy := strings.Join(parseProxyExceptions(exceptions), ",")

	commands := [][]string{
		{binary, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "1"},
		{binary, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpProxy", fmt.Sprintf("http://%s:%d", listenAddr, httpPort)},
		{binary, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpsProxy", fmt.Sprintf("http://%s:%d", listenAddr, httpPort)},
		{binary, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "socksProxy", fmt.Sprintf("socks://%s:%d", listenAddr, socksPort)},
	}
	if noProxy != "" {
		commands = append(commands, []string{binary, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "NoProxyFor", noProxy})
	}

	for _, cmdArgs := range commands {
		if out, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput(); err != nil { //nolint:gosec
			return fmt.Errorf("apply system proxy via %s failed: %w (%s)", binary, err, strings.TrimSpace(string(out)))
		}
	}
	if err := notifyKDEProxyReload(); err != nil {
		log.Printf("[native] ApplySystemProxy: KDE reload hint failed: %v", err)
	}
	return nil
}

func clearGSettingsProxy() error {
	commands := [][]string{
		{"set", "org.gnome.system.proxy", "mode", "none"},
		{"set", "org.gnome.system.proxy", "ignore-hosts", "[]"},
	}

	for _, args := range commands {
		if out, err := runGSettings(args...); err != nil {
			return fmt.Errorf("clear system proxy via gsettings failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
	}

	for _, key := range []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY"} {
		_ = os.Unsetenv(key)
	}
	return nil
}

func clearKDEProxy(binary string) error {
	commands := [][]string{
		{binary, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "0"},
	}

	for _, cmdArgs := range commands {
		if out, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput(); err != nil { //nolint:gosec
			return fmt.Errorf("clear system proxy via %s failed: %w (%s)", binary, err, strings.TrimSpace(string(out)))
		}
	}
	if err := notifyKDEProxyReload(); err != nil {
		log.Printf("[native] ApplySystemProxy: KDE reload hint failed: %v", err)
	}
	for _, key := range []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY"} {
		_ = os.Unsetenv(key)
	}
	return nil
}

func notifyKDEProxyReload() error {
	if !hasCommand("dbus-send") {
		return nil
	}
	_, err := exec.Command("dbus-send", "--type=signal", "/KIO/Scheduler", "org.kde.KIO.Scheduler.reparseSlaveConfiguration", "string:").CombinedOutput() //nolint:gosec
	return err
}

func runGSettings(args ...string) ([]byte, error) {
	if os.Geteuid() == 0 {
		sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER"))
		if sudoUser != "" {
			if usr, err := user.Lookup(sudoUser); err == nil && strings.TrimSpace(usr.Uid) != "" {
				runtimeDir := fmt.Sprintf("/run/user/%s", usr.Uid)
				busAddr := fmt.Sprintf("unix:path=%s/bus", runtimeDir)
				cmdArgs := []string{"-u", sudoUser, "env", "XDG_RUNTIME_DIR=" + runtimeDir, "DBUS_SESSION_BUS_ADDRESS=" + busAddr, "gsettings"}
				cmdArgs = append(cmdArgs, args...)
				if out, err := exec.Command("sudo", cmdArgs...).CombinedOutput(); err == nil { //nolint:gosec
					return out, nil
				}
			}
		}
	}
	return exec.Command("gsettings", args...).CombinedOutput() //nolint:gosec
}

func portHostForDesktopProxy(cfg map[string]interface{}) string {
	listenAddr := strCfg(cfg, "listenAddr", "127.0.0.1")
	if boolCfg(cfg, "allowLan", false) && (listenAddr == "" || listenAddr == "127.0.0.1") {
		listenAddr = "0.0.0.0"
	}
	if listenAddr == "0.0.0.0" || listenAddr == "" {
		listenAddr = "127.0.0.1"
	}
	return listenAddr
}

func parseProxyExceptions(exceptions string) []string {
	parts := strings.FieldsFunc(exceptions, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})

	hosts := make([]string, 0, len(parts)+2)
	hosts = append(hosts, "localhost", "127.0.0.1")
	seen := map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
	}

	for _, part := range parts {
		host := strings.TrimSpace(part)
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}

	return hosts
}

func formatGSettingsStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}

	quoted := make([]string, 0, len(values))
	for _, value := range values {
		escaped := strings.ReplaceAll(value, "'", "\\'")
		quoted = append(quoted, fmt.Sprintf("'%s'", escaped))
	}
	return fmt.Sprintf("[%s]", strings.Join(quoted, ", "))
}

// ─── Config ───────────────────────────────────────────────────────────────────

func (s *Service) GetConfig() map[string]interface{} {
	cfg, _ := s.store.LoadConfig()
	return cfg
}

func (s *Service) UpdateConfig(next map[string]interface{}) map[string]interface{} {
	cfg, _ := s.store.LoadConfig()
	previousTunMode := tunModeFromConfig(cfg)
	for k, v := range next {
		cfg[k] = v
	}
	cfg = normalizeRuntimeConfig(cfg)
	nextTunMode := tunModeFromConfig(cfg)
	if err := s.store.SaveConfig(cfg); err != nil {
		log.Printf("[native] UpdateConfig: %v", err)
	}
	if previousTunMode != "off" && nextTunMode == "off" {
		s.clearTunRouting()
	}
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.RestartCore()
		}()
	}
	return cfg
}

// ─── Routing ──────────────────────────────────────────────────────────────────

func (s *Service) GetRoutingConfig() domain.RoutingConfig {
	rc, _ := s.store.LoadRoutingConfig()
	return rc
}

func (s *Service) UpdateRoutingConfig(rc domain.RoutingConfig) domain.RoutingConfig {
	needGeoSite, needGeoIP := routingGeoDataRequirements(rc)
	missingNeeded := (needGeoSite && !hasGeoSiteAsset()) || (needGeoIP && !hasGeoIPAsset())
	if missingNeeded {
		if result, err := s.ensureGeoSiteData(); err != nil {
			log.Printf("[native] UpdateRoutingConfig: auto download geodata failed: %v", err)
		} else {
			log.Printf("[native] UpdateRoutingConfig: auto geodata update applied: geositeUpdated=%v geoipUpdated=%v", result["geositeUpdated"], result["geoipUpdated"])
		}
	}

	if err := s.store.SaveRoutingConfig(rc); err != nil {
		log.Printf("[native] UpdateRoutingConfig: %v", err)
	}
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.RestartCore()
		}()
	}
	return rc
}

func (s *Service) GetRoutingDiagnostics() domain.RoutingDiagnostics {
	cfg, _ := s.store.LoadConfig()
	rc, _ := s.store.LoadRoutingConfig()
	state, _ := s.store.LoadState()
	profile := s.pickSelectedProfile(state.CurrentProfileID)

	hasGeoIP := hasGeoIPAsset()
	hasGeoSite := hasGeoSiteAsset()

	diag := domain.RoutingDiagnostics{
		Mode:             rc.Mode,
		DomainStrategy:   routingDomainStrategy(rc),
		TunMode:          tunModeFromConfig(cfg),
		TunEnabled:       tunModeFromConfig(cfg) != "off",
		HasGeoIP:         hasGeoIP,
		HasGeoSite:       hasGeoSite,
		GeoDataAvailable: hasGeoIP && hasGeoSite,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		Rules:            make([]map[string]interface{}, 0),
	}
	if dev, err := getDefaultRouteDevice(); err == nil {
		diag.DefaultRouteDevice = dev
		tunName := strCfg(cfg, "tunName", "xraye0")
		diag.TunTakeoverActive = diag.TunEnabled && dev == tunName
	}

	if profile != nil {
		diag.CurrentProfileID = profile.ID
		diag.CurrentProfileName = profile.Name

		raw, err := generateXrayConfig(*profile, cfg, rc)
		if err == nil {
			var parsed map[string]interface{}
			if err = json.Unmarshal(raw, &parsed); err == nil {
				if routingCfg, ok := parsed["routing"].(map[string]interface{}); ok {
					diag.Rules = normalizeRoutingRuleMaps(routingCfg["rules"])
				}
			}
		}
		if err != nil {
			diag.Warning = "failed to build runtime config for diagnostics: " + err.Error()
		}
	}

	if len(diag.Rules) == 0 {
		for _, r := range buildRoutingRules(rc, hasGeoIP, hasGeoSite) {
			if m, ok := r.(map[string]interface{}); ok {
				diag.Rules = append(diag.Rules, m)
			}
		}
		if profile == nil {
			diag.Warning = "no profile selected; diagnostics use routing template rules"
		}
	}

	diag.RuleCount = len(diag.Rules)
	return diag
}

func routingGeoDataRequirements(rc domain.RoutingConfig) (needGeoSite bool, needGeoIP bool) {
	if rc.Mode == "bypass_cn" {
		return true, true
	}
	for _, rule := range rc.Rules {
		switch rule.Type {
		case "geosite":
			needGeoSite = true
		case "geoip":
			needGeoIP = true
		}
	}
	return needGeoSite, needGeoIP
}

func normalizeRoutingRuleMaps(raw interface{}) []map[string]interface{} {
	entries, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	rules := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		if m, ok := entry.(map[string]interface{}); ok {
			rules = append(rules, m)
		}
	}
	return rules
}

// ─── Stats ────────────────────────────────────────────────────────────────────

func (s *Service) GetStats() domain.StatsResult {
	s.mu.Lock()
	st := s.stats
	s.mu.Unlock()
	if st == nil {
		return domain.StatsResult{}
	}
	return st.get()
}

func (s *Service) GetRoutingHitStats() domain.RoutingHitStats {
	s.mu.Lock()
	st := s.stats
	s.mu.Unlock()
	if st == nil {
		return domain.RoutingHitStats{
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			Items:     []domain.RoutingOutboundHit{},
			Note:      "核心未运行或统计未初始化。",
		}
	}
	return st.getRoutingHitStats()
}

func (s *Service) RepairTunAndRestart() domain.TunRepairResult {
	result := domain.TunRepairResult{
		TriggeredAt: time.Now().UTC().Format(time.RFC3339),
	}

	cfg, _ := s.store.LoadConfig()
	tunMode := tunModeFromConfig(cfg)
	result.TunEnabled = tunMode != "off"
	if !result.TunEnabled {
		result.Message = "TUN 未启用，已跳过自动修复"
		st := s.CoreStatus()
		result.Running = st.Running
		return result
	}

	result.WasRunning = s.CoreStatus().Running

	// Ensure stale routes/devices are cleaned before relaunch.
	s.clearTunRouting()
	if err := s.cleanupStaleTunInterface(cfg); err != nil {
		result.Error = err.Error()
		result.Message = "TUN 残留清理失败"
		return result
	}

	if result.WasRunning {
		st := s.RestartCore()
		result.Running = st.Running
		result.Started = st.Running
		if !st.Running {
			result.Error = st.Error
		}
	} else {
		st := s.StartCore()
		result.Running = st.Running
		result.Started = st.Running
		if !st.Running {
			result.Error = st.Error
		}
	}

	if dev, err := getDefaultRouteDevice(); err == nil {
		result.DefaultRouteDevice = dev
		tunName := strCfg(cfg, "tunName", "xraye0")
		result.TunTakeoverActive = result.TunEnabled && dev == tunName
	}

	if result.Error != "" {
		result.Message = "自动修复执行完成，但核心或 TUN 仍异常"
		return result
	}
	if result.TunTakeoverActive {
		result.Message = "自动修复完成，TUN 已接管默认路由"
	} else {
		result.Message = "自动修复完成，核心已运行，但 TUN 尚未接管默认路由"
	}
	return result
}

// ─── Logs ─────────────────────────────────────────────────────────────────────

func (s *Service) SubscribeCoreLogs() (<-chan domain.LogLine, func()) {
	return s.logs.subscribe()
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (s *Service) checkProcExited() {
	if s.xrayCore != nil && !s.xrayCore.IsRunning() {
		s.xrayCore = nil
		s.running = false
		s.trackedPID = 0
		s.setCoreError("xray-core instance exited")
		s.clearTunRouting()
		s.clearSystemProxyOnCoreStop()
		if s.stats != nil {
			s.stats.shutdown()
			s.stats = nil
		}
	}
	if s.proc != nil && s.proc.ProcessState != nil && s.proc.ProcessState.Exited() {
		s.proc = nil
		s.running = false
		s.trackedPID = 0
		s.setCoreError("core process exited")
		s.clearTunRouting()
		s.clearSystemProxyOnCoreStop()
		if s.stats != nil {
			s.stats.shutdown()
			s.stats = nil
		}
	}
}

func (s *Service) buildStatus() domain.CoreStatus {
	state, _ := s.store.LoadState()
	cfg, _ := s.store.LoadConfig()
	routing, _ := s.store.LoadRoutingConfig()
	profile := s.pickSelectedProfile(state.CurrentProfileID)
	_, mode, policyResolved := selectCoreEngine(cfg, routing, profile)
	st := "stopped"
	if s.running {
		st = "running"
	}
	resolved := s.lastEngine
	if resolved == "" {
		resolved = policyResolved
	}
	coreType := resolved
	if coreType == "" {
		coreType = "xray"
	}
	return domain.CoreStatus{
		Running:          s.running,
		CoreType:         coreType,
		EngineMode:       mode,
		EngineResolved:   resolved,
		CurrentProfileID: state.CurrentProfileID,
		State:            st,
		Error:            s.lastError,
		ErrorAt:          s.lastErrorAt,
	}
}

func (s *Service) saveState() error {
	state, _ := s.store.LoadState()
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return s.store.SaveState(state)
}

func (s *Service) findProfile(id string) (domain.ProfileItem, error) {
	if id != "" {
		profiles, _ := s.store.LoadProfiles()
		for _, p := range profiles {
			if p.ID == id {
				return p, nil
			}
		}
	}
	profiles, _ := s.store.LoadProfiles()
	if len(profiles) == 0 {
		return domain.ProfileItem{}, fmt.Errorf("no profiles configured")
	}
	return profiles[0], nil
}

func (s *Service) resolveXrayCmd(cfg map[string]interface{}) string {
	if s.xrayCmd != "" {
		return s.xrayCmd
	}
	return strCfg(cfg, "xrayCmd", "xray")
}

func useEmbeddedCore(cfg map[string]interface{}) bool {
	mode := strings.ToLower(strings.TrimSpace(strCfg(cfg, "coreEngine", "embedded")))
	if mode == "" {
		mode = "embedded"
	}
	return mode == "embedded" || mode == "builtin" || mode == "internal"
}

func selectCoreEngine(cfg map[string]interface{}, routing domain.RoutingConfig, profile *domain.ProfileItem) (bool, string, string) {
	mode := strings.ToLower(strings.TrimSpace(strCfg(cfg, "coreEngine", "xray-core")))
	switch mode {
	case "", "embedded", "auto", "xray", "builtin", "internal":
		mode = "xray-core"
	}
	_ = routing
	_ = profile
	return false, mode, "xray-core"
}

func (s *Service) pickSelectedProfile(currentID string) *domain.ProfileItem {
	profiles, _ := s.store.LoadProfiles()
	if len(profiles) == 0 {
		return nil
	}
	if currentID != "" {
		for i := range profiles {
			if profiles[i].ID == currentID {
				p := profiles[i]
				return &p
			}
		}
	}
	p := profiles[0]
	return &p
}

func (s *Service) waitProc(cmd *exec.Cmd) {
	_ = cmd.Wait()
	s.mu.Lock()
	if s.proc == cmd {
		s.proc = nil
		s.running = false
		s.trackedPID = 0
		s.setCoreError("core process exited")
		s.clearTunRouting()
		s.clearSystemProxyOnCoreStop()
		if s.stats != nil {
			s.stats.shutdown()
			s.stats = nil
		}
	}
	s.mu.Unlock()
	log.Printf("[native] core process exited")
	_ = s.saveState()
}

func (s *Service) setCoreError(message string) {
	s.lastError = strings.TrimSpace(message)
	if s.lastError == "" {
		s.lastErrorAt = ""
		return
	}
	s.lastErrorAt = time.Now().UTC().Format(time.RFC3339)
}

func (s *Service) clearCoreErrorLocked() {
	s.lastError = ""
	s.lastErrorAt = ""
}

func (s *Service) clearSystemProxyOnCoreStop() {
	cfg, _ := s.store.LoadConfig()
	mode := strCfg(cfg, "systemProxyMode", "forced_clear")
	if mode != "forced_change" && mode != "pac" {
		return
	}

	if _, err := s.applyDesktopSystemProxy(cfg, "forced_clear", ""); err != nil {
		log.Printf("[native] clear system proxy on core stop failed: %v", err)
		return
	}
	for _, key := range []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY"} {
		_ = os.Unsetenv(key)
	}
	log.Printf("[native] desktop system proxy cleared on core stop")
}

func normalizeRuntimeConfig(cfg map[string]interface{}) map[string]interface{} {
	if cfg == nil {
		return cfg
	}
	mode := strings.ToLower(strings.TrimSpace(strCfg(cfg, "coreEngine", "xray-core")))
	switch mode {
	case "", "embedded", "auto", "xray", "builtin", "internal":
		cfg["coreEngine"] = "xray-core"
	default:
		cfg["coreEngine"] = mode
	}
	tunMode := tunModeFromConfig(cfg)
	cfg["tunMode"] = tunMode
	cfg["enableTun"] = tunMode != "off"
	if tunMode != "off" {
		cfg["tunStack"] = tunMode
	} else if strCfg(cfg, "tunStack", "") == "" {
		cfg["tunStack"] = "mixed"
	}
	return cfg
}

func (s *Service) applyConfiguredSystemProxyOnCoreStart(cfg map[string]interface{}) error {
	mode := strCfg(cfg, "systemProxyMode", "forced_clear")
	if mode != "forced_change" && mode != "pac" {
		return nil
	}
	exceptions := strCfg(cfg, "systemProxyExceptions", "")
	backend, err := s.applyDesktopSystemProxy(cfg, mode, exceptions)
	if err != nil {
		return err
	}
	if backend != "" {
		cfg["systemProxyBackend"] = backend
	}
	if err := s.store.SaveConfig(cfg); err != nil {
		log.Printf("[native] apply system proxy on core start save config failed: %v", err)
	}
	return nil
}

func detectDefaultRouteInterface() (string, error) {
	lines, err := getDefaultRouteLines()
	if err != nil {
		return "", err
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "dev" {
				return fields[i+1], nil
			}
		}
	}
	return "", nil
}

func getDefaultRouteLines() ([]string, error) {
	out, err := exec.Command("ip", "-4", "route", "show", "default").CombinedOutput() //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("read default route: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil, fmt.Errorf("no IPv4 default route found")
	}
	lines := strings.Split(text, "\n")
	res := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no IPv4 default route found")
	}
	return res, nil
}

func getDefaultRouteDevice() (string, error) {
	lines, err := getDefaultRouteLines()
	if err != nil {
		return "", err
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "dev" {
				return fields[i+1], nil
			}
		}
	}
	return "", fmt.Errorf("default route device not found")
}

func (s *Service) setupTunRouting(cfg map[string]interface{}) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("TUN auto route is only implemented on Linux")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("TUN auto route requires root privileges")
	}
	tunName := strCfg(cfg, "tunName", "xraye0")
	if !hasCommand("ip") {
		return fmt.Errorf("TUN auto route requires ip command")
	}
	routes, err := getDefaultRouteLines()
	if err != nil {
		return err
	}
	if err := waitForNetworkInterface(tunName, 5*time.Second); err != nil {
		return err
	}
	if out, err := exec.Command("ip", "route", "replace", "default", "dev", tunName).CombinedOutput(); err != nil { //nolint:gosec
		return fmt.Errorf("replace default route with %s failed: %w (%s)", tunName, err, strings.TrimSpace(string(out)))
	}
	s.tunRestoreRoutes = append([]string(nil), routes...)
	s.persistTunRestoreRoutes(s.tunRestoreRoutes, tunName)
	return nil
}

func (s *Service) clearTunRouting() {
	if runtime.GOOS != "linux" || !hasCommand("ip") {
		s.tunRestoreRoutes = nil
		s.persistTunRestoreRoutes(nil, "")
		return
	}
	cfg, _ := s.store.LoadConfig()
	tunName := strCfg(cfg, "tunName", "xraye0")
	routes := append([]string(nil), s.tunRestoreRoutes...)
	if len(routes) == 0 {
		routes = s.loadPersistedTunRestoreRoutes()
	}
	if len(routes) == 0 {
		routes = s.buildTunRestoreFallbackRoutes(cfg, tunName)
	}
	if len(routes) == 0 {
		log.Printf("[native] clearTunRouting: no restore routes found; proceeding with best-effort stale route/device cleanup")
	} else {
		for _, route := range routes {
			fields := strings.Fields(route)
			if len(fields) == 0 {
				continue
			}
			args := append([]string{"route", "replace"}, fields...)
			if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil { //nolint:gosec
				log.Printf("[native] restore default route failed: %v (%s)", err, strings.TrimSpace(string(out)))
			}
		}
	}
	// Remove stale tun default route if it still exists after restore attempts.
	if out, err := exec.Command("ip", "route", "del", "default", "dev", tunName).CombinedOutput(); err == nil {
		_ = out
	}
	// Best-effort remove stale tun interface to avoid next-start busy error.
	if out, err := exec.Command("ip", "link", "del", "dev", tunName).CombinedOutput(); err == nil {
		_ = out
	}
	s.tunRestoreRoutes = nil
	s.persistTunRestoreRoutes(nil, tunName)
}

func (s *Service) persistTunRestoreRoutes(routes []string, tunName string) {
	cfg, _ := s.store.LoadConfig()
	if len(routes) == 0 {
		delete(cfg, "tunRestoreRoutes")
		delete(cfg, "tunRestoreHintDev")
		delete(cfg, "tunRestoreHintVia")
	} else {
		cfg["tunRestoreRoutes"] = routes
		dev, via := parseDefaultRouteHint(routes, tunName)
		if dev != "" {
			cfg["tunRestoreHintDev"] = dev
		} else {
			delete(cfg, "tunRestoreHintDev")
		}
		if via != "" {
			cfg["tunRestoreHintVia"] = via
		} else {
			delete(cfg, "tunRestoreHintVia")
		}
	}
	if err := s.store.SaveConfig(cfg); err != nil {
		log.Printf("[native] persist tun restore routes failed: %v", err)
	}
}

func (s *Service) buildTunRestoreFallbackRoutes(cfg map[string]interface{}, tunName string) []string {
	hintDev := strings.TrimSpace(strCfg(cfg, "tunRestoreHintDev", ""))
	hintVia := strings.TrimSpace(strCfg(cfg, "tunRestoreHintVia", ""))
	if hintDev != "" && hintDev != tunName {
		if hintVia != "" {
			return []string{fmt.Sprintf("default via %s dev %s", hintVia, hintDev)}
		}
		return []string{fmt.Sprintf("default dev %s", hintDev)}
	}
	return nil
}

func parseDefaultRouteHint(routes []string, tunName string) (dev string, via string) {
	for _, route := range routes {
		fields := strings.Fields(strings.TrimSpace(route))
		if len(fields) == 0 || fields[0] != "default" {
			continue
		}
		var routeDev, routeVia string
		for i := 0; i < len(fields)-1; i++ {
			switch fields[i] {
			case "dev":
				routeDev = fields[i+1]
			case "via":
				routeVia = fields[i+1]
			}
		}
		if routeDev == "" || routeDev == tunName {
			continue
		}
		return routeDev, routeVia
	}
	return "", ""
}

func (s *Service) loadPersistedTunRestoreRoutes() []string {
	cfg, _ := s.store.LoadConfig()
	raw, ok := cfg["tunRestoreRoutes"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		routes := make([]string, 0, len(v))
		for _, item := range v {
			text, ok := item.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text != "" {
				routes = append(routes, text)
			}
		}
		return routes
	default:
		return nil
	}
}

func waitForNetworkInterface(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join("/sys/class/net", name)); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for TUN interface %s", name)
}

func killGraceful(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = cmd.Process.Wait()
	}()
	select {
	case <-done:
	case <-time.After(800 * time.Millisecond):
		_ = cmd.Process.Signal(syscall.SIGKILL)
	}
}

func validateProfile(p domain.ProfileItem) error {
	if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Address) == "" || p.Port <= 0 {
		return service.ErrInvalidMode
	}
	switch p.Protocol {
	case domain.ProtocolVMess, domain.ProtocolVLESS,
		domain.ProtocolShadowsocks, domain.ProtocolTrojan,
		domain.ProtocolHysteria2, domain.ProtocolTUIC:
	default:
		return service.ErrInvalidMode
	}
	return nil
}

func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func annotateTunStartError(err error, cfg map[string]interface{}) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if tunModeFromConfig(cfg) == "off" {
		return msg
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "device or resource busy") {
		tunName := strCfg(cfg, "tunName", "xraye0")
		return fmt.Sprintf("TUN 启动失败: 设备 %s 被占用（device or resource busy）。已尝试自动清理残留设备；如仍失败请执行: sudo ip link del %s", tunName, tunName)
	}
	return msg
}

func (s *Service) cleanupStaleTunInterface(cfg map[string]interface{}) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if !hasCommand("ip") {
		return nil
	}
	tunName := strCfg(cfg, "tunName", "xraye0")
	tunName = strings.TrimSpace(tunName)
	if tunName == "" {
		return nil
	}

	if out, err := exec.Command("ip", "link", "show", "dev", tunName).CombinedOutput(); err != nil { //nolint:gosec
		_ = out
		// Device not found is expected and means no stale interface.
		return nil
	}

	if out, err := exec.Command("ip", "link", "del", "dev", tunName).CombinedOutput(); err != nil { //nolint:gosec
		return fmt.Errorf("remove stale tun interface %s failed: %w (%s)", tunName, err, strings.TrimSpace(string(out)))
	}
	log.Printf("[native] removed stale tun interface: %s", tunName)
	return nil
}

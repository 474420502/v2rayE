package launcher

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/httpapi"
	"v2raye/backend-go/internal/service/native"
	"v2raye/backend-go/internal/storage"
)

type bootRestoreService interface {
	NetworkAvailability() domain.AvailabilityResult
	StartCore() domain.CoreStatus
}

type bootRestoreOptions struct {
	initialNetworkWait time.Duration
	networkPollInterval time.Duration
	maxStartAttempts int
	retryBackoffBase time.Duration
	maxRetryBackoff time.Duration
}

func defaultBootRestoreOptions() bootRestoreOptions {
	return bootRestoreOptions{
		initialNetworkWait: 30 * time.Second,
		networkPollInterval: 3 * time.Second,
		maxStartAttempts: 6,
		retryBackoffBase: 2 * time.Second,
		maxRetryBackoff: 30 * time.Second,
	}
}

// ServerOptions configures the embedded HTTP API server runtime.
type ServerOptions struct {
	Addr           string
	Token          string
	DataDir        string
	XrayCmd        string
	AllowPublic    bool
	RestoreOnBoot  bool
	LogStartupInfo bool
}

// RunServer starts backend services and blocks until ctx is cancelled or the server exits.
func RunServer(ctx context.Context, opts ServerOptions) error {
	opts.DataDir = storage.ResolveDataDir(opts.DataDir)
	store, err := storage.New(opts.DataDir)
	if err != nil {
		return err
	}

	svc := native.New(opts.DataDir, opts.XrayCmd, store)
	httpOpts := make([]httpapi.Option, 0, 1)
	if opts.AllowPublic {
		httpOpts = append(httpOpts, httpapi.WithPublicAccessAllowed())
	}
	server := httpapi.New(opts.Addr, opts.Token, svc, httpOpts...)

	restoreOnBoot := opts.RestoreOnBoot
	if !restoreOnBoot {
		cfg, _ := store.LoadConfig()
		state, _ := store.LoadState()
		restoreOnBoot = state.CoreShouldRestore
		if !restoreOnBoot {
			if autoRun, ok := cfg["autoRun"].(bool); ok && autoRun {
				restoreOnBoot = true
			}
		}
	}

	errCh := make(chan error, 1)
	go func() {
		if opts.LogStartupInfo {
			log.Printf("[go-api] listening on http://%s  (xray=%s, data=%s)", opts.Addr, opts.XrayCmd, opts.DataDir)
			if opts.AllowPublic {
				log.Printf("[go-api] client scope: public access allowed")
			} else {
				log.Printf("[go-api] client scope: loopback + LAN only (set allow public to disable)")
			}
			if strings.TrimSpace(opts.Token) != "" {
				log.Printf("[go-api] token auth: enabled")
			} else {
				log.Printf("[go-api] token auth: disabled (set V2RAYN_API_TOKEN to enable)")
			}
		}
		errCh <- server.Run(ctx)
	}()

	if restoreOnBoot {
		go func() {
			restoreCoreOnBoot(ctx, svc, defaultBootRestoreOptions())
		}()
	}

	err = <-errCh
	wasRunning := svc.CoreStatus().Running
	svc.StopCore()
	if wasRunning {
		state, _ := store.LoadState()
		state.CoreShouldRestore = true
		_ = store.SaveState(state)
	}
	return err
}

func restoreCoreOnBoot(ctx context.Context, svc bootRestoreService, opts bootRestoreOptions) {
	if err := waitForBootNetworkReady(ctx, svc, opts.initialNetworkWait, opts.networkPollInterval); err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Printf("[main] boot restore network probe not ready yet: %v; continue with guarded retries", err)
	}

	maxAttempts := opts.maxStartAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return
		}

		st := svc.StartCore()
		if st.Running {
			log.Printf("[main] restored core on boot (profile=%s, attempt=%d/%d)", st.CurrentProfileID, attempt, maxAttempts)
			return
		}

		errMsg := strings.TrimSpace(st.Error)
		if errMsg == "" {
			errMsg = "unknown error"
		}
		log.Printf("[main] restore core on boot attempt=%d/%d failed: %s", attempt, maxAttempts, errMsg)
		if attempt == maxAttempts {
			return
		}

		delay := bootRestoreRetryDelay(attempt, opts.retryBackoffBase, opts.maxRetryBackoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func waitForBootNetworkReady(ctx context.Context, svc bootRestoreService, maxWait, pollInterval time.Duration) error {
	if maxWait <= 0 {
		return nil
	}
	if pollInterval <= 0 {
		pollInterval = time.Second
	}

	deadline := time.Now().Add(maxWait)
	last := domain.AvailabilityResult{}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		last = svc.NetworkAvailability()
		if last.Available {
			return nil
		}
		if time.Now().After(deadline) {
			msg := strings.TrimSpace(last.Message)
			if msg == "" {
				msg = "no successful external connectivity probe"
			}
			return fmt.Errorf("timed out after %s: %s", maxWait, msg)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func bootRestoreRetryDelay(attempt int, base, max time.Duration) time.Duration {
	if base <= 0 {
		base = time.Second
	}
	if max <= 0 {
		max = 30 * time.Second
	}
	delay := base
	for step := 1; step < attempt; step++ {
		delay *= 2
		if delay >= max {
			return max
		}
	}
	if delay > max {
		return max
	}
	return delay
}

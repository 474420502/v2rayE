package launcher

import (
	"context"
	"log"
	"strings"

	"v2raye/backend-go/internal/httpapi"
	"v2raye/backend-go/internal/service/native"
	"v2raye/backend-go/internal/storage"
)

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
			st := svc.StartCore()
			if st.Running {
				log.Printf("[main] restored core on boot (profile=%s)", st.CurrentProfileID)
				return
			}
			if strings.TrimSpace(st.Error) != "" {
				log.Printf("[main] restore core on boot failed: %s", st.Error)
				return
			}
			log.Printf("[main] restore core on boot failed: unknown error")
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

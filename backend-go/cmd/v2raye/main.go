package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"v2raye/backend-go/cmd/tui"
	"v2raye/backend-go/internal/launcher"
)

func main() {
	defaultAddr := envOrDefault("V2RAYN_API_ADDR", "127.0.0.1:18000")
	defaultBaseURL := envOrDefault("V2RAYN_TUI_BASE_URL", "http://"+defaultAddr)

	serverMode := flag.Bool("server", false, "run backend API server mode")
	apiAddr := flag.String("api-addr", defaultAddr, "backend API listen address")
	baseURL := flag.String("base-url", defaultBaseURL, "backend API base URL for TUI mode")
	dataDir := flag.String("data-dir", envOrDefault("V2RAYN_DATA_DIR", "/opt/v2rayE"), "backend data directory")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *serverMode {
		err := launcher.RunServer(ctx, launcher.ServerOptions{
			Addr:           *apiAddr,
			DataDir:        strings.TrimSpace(*dataDir),
			LogStartupInfo: true,
		})
		if err != nil {
			log.Fatalf("server error: %v", err)
		}
		return
	}

	if err := tui.Run(ctx, strings.TrimSpace(*baseURL), ""); err != nil {
		log.Fatalf("tui error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

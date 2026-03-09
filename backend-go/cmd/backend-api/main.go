package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"v2raye/backend-go/internal/launcher"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err := launcher.RunServer(ctx, launcher.ServerOptions{
		Addr:           envOrDefault("V2RAYN_API_ADDR", "127.0.0.1:18000"),
		DataDir:        envOrDefault("V2RAYN_DATA_DIR", "/opt/v2rayE"),
		LogStartupInfo: true,
	})
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

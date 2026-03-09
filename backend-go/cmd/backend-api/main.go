package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"v2raye/backend-go/internal/launcher"
	"v2raye/backend-go/internal/storage"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err := launcher.RunServer(ctx, launcher.ServerOptions{
		Addr:           envOrDefault("V2RAYN_API_ADDR", "0.0.0.0:18000"),
		DataDir:        storage.ResolveDataDir(envOrDefault("V2RAYN_DATA_DIR", storage.DefaultDataDir)),
		AllowPublic:    envBool("V2RAYN_API_ALLOW_PUBLIC"),
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

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

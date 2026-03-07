package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"v2raye/backend-go/internal/httpapi"
	"v2raye/backend-go/internal/service/native"
	"v2raye/backend-go/internal/storage"
)

func main() {
addr := envOrDefault("V2RAYN_API_ADDR", "127.0.0.1:18000")
token := strings.TrimSpace(os.Getenv("V2RAYN_API_TOKEN"))
dataDir := envOrDefault("V2RAYN_DATA_DIR", "/tmp/v2raye")
xrayCmd := envOrDefault("V2RAYN_XRAY_CMD", "xray")

store, err := storage.New(dataDir)
if err != nil {
log.Fatalf("[main] failed to init storage: %v", err)
}

svc := native.New(dataDir, xrayCmd, store)
server := httpapi.New(addr, token, svc)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

errCh := make(chan error, 1)
go func() {
log.Printf("[go-api] listening on http://%s  (xray=%s, data=%s)", addr, xrayCmd, dataDir)
if token != "" {
log.Printf("[go-api] token auth: enabled")
} else {
log.Printf("[go-api] token auth: disabled (set V2RAYN_API_TOKEN to enable)")
}
errCh <- server.Run(ctx)
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
select {
case <-quit:
		svc.StopCore()
cancel()
if err := <-errCh; err != nil {
log.Fatalf("server error: %v", err)
}
case err := <-errCh:
		svc.StopCore()
if err != nil {
log.Fatalf("server error: %v", err)
}
}
}

func envOrDefault(key, fallback string) string {
if v := strings.TrimSpace(os.Getenv(key)); v != "" {
return v
}
return fallback
}

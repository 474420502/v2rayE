package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gcla/gowid"
)

func main() {
	baseURL := flag.String("base-url", envOrDefault("V2RAYN_TUI_BASE_URL", "http://127.0.0.1:18000"), "backend API base URL")
	token := flag.String("token", envOrDefault("V2RAYN_TUI_TOKEN", ""), "bearer token for API access")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := newAPIClient(*baseURL, *token)
	tui := newTUI(ctx, client)

	palette := gowid.Palette{
		"body":            gowid.MakePaletteEntry(gowid.ColorWhite, gowid.ColorBlack),
		"title":           gowid.MakeStyledPaletteEntry(gowid.ColorBlack, gowid.ColorCyan, gowid.StyleBold),
		"footer":          gowid.MakePaletteEntry(gowid.ColorLightGray, gowid.ColorDarkBlue),
		"button":          gowid.MakePaletteEntry(gowid.ColorWhite, gowid.ColorDarkBlue),
		"button-focus":    gowid.MakePaletteEntry(gowid.ColorBlack, gowid.ColorYellow),
		"button-selected": gowid.MakePaletteEntry(gowid.ColorBlack, gowid.ColorLightGreen),
		"panel":           gowid.MakePaletteEntry(gowid.ColorWhite, gowid.ColorBlack),
		"muted":           gowid.MakePaletteEntry(gowid.ColorDarkGray, gowid.ColorBlack),
	}

	app, err := gowid.NewApp(gowid.AppArgs{
		View:    tui.build(),
		Palette: &palette,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start tui: %v\n", err)
		os.Exit(1)
	}

	tui.attachApp(app)
	tui.startBackgroundWork()
	app.MainLoop(gowid.UnhandledInputFunc(tui.handler))
	tui.cancel()
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

package tui

import (
	"context"
	"fmt"

	"github.com/gcla/gowid"
)

// Run starts the terminal UI and blocks until the UI exits or ctx is cancelled.
func Run(ctx context.Context, baseURL, token string) error {
	client := newAPIClient(baseURL, token)
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
		return fmt.Errorf("failed to start tui: %w", err)
	}

	tui.attachApp(app)
	tui.startBackgroundWork()
	app.MainLoop(gowid.UnhandledInputFunc(tui.handler))
	tui.cancel()
	return nil
}

package tui

import (
	"context"
	"fmt"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Run starts the terminal UI and blocks until the UI exits or ctx is cancelled.
func Run(ctx context.Context, baseURL, token string) error {
	client := newAPIClient(baseURL, token)
	tui := newTUI(ctx, client)
	app := tview.NewApplication()
	root := tui.build()

	tui.attachApp(app)
	app.SetRoot(root, true)
	app.SetInputCapture(tui.handler)
	app.SetAfterDrawFunc(func(screen tcell.Screen) {
		_ = screen
		tui.startBackgroundWork()
	})
	app.EnableMouse(true)
	go func() {
		<-ctx.Done()
		app.Stop()
	}()
	if err := app.Run(); err != nil {
		return fmt.Errorf("failed to start tui: %w", err)
	}
	tui.cancel()
	return nil
}

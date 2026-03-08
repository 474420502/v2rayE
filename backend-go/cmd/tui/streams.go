package tui

import (
	"context"
	"fmt"
	"time"
)

func (a *tuiApp) startBackgroundWork() {
	go a.runAction("initial load", func(context.Context) error {
		return a.reloadAll()
	})
	go a.pollOverview()
	go a.streamLogs()
	go a.streamEvents()
}

func (a *tuiApp) streamLogs() {
	delay := time.Second
	for {
		if a.ctx.Err() != nil {
			return
		}
		a.setLogsStreamState("connecting")
		err := a.client.StreamLogs(a.ctx, func() {
			delay = time.Second
			a.setLogsStreamState("connected")
		}, func(line LogLine) error {
			a.storeIncomingLogLine(line)
			a.refreshLogsWidget()
			return nil
		})
		if a.ctx.Err() != nil {
			return
		}
		reason := errorString(err)
		if reason == "<nil>" {
			reason = "stream closed"
		}
		a.pushEvent("log stream disconnected: " + reason)
		a.setLogsStreamState(fmt.Sprintf("reconnecting in %s (%s)", delay, reason))
		if !sleepContext(a.ctx, delay) {
			return
		}
		delay = nextBackoffDelay(delay)
	}
}

func (a *tuiApp) streamEvents() {
	delay := time.Second
	for {
		if a.ctx.Err() != nil {
			return
		}
		a.setEventsStreamState("connecting")
		err := a.client.StreamEvents(a.ctx, func() {
			delay = time.Second
			a.setEventsStreamState("connected")
		}, func(msg EventMessage) error {
			a.pushEvent(formatEvent(msg))
			return nil
		})
		if a.ctx.Err() != nil {
			return
		}
		reason := errorString(err)
		if reason == "<nil>" {
			reason = "stream closed"
		}
		a.pushEvent("event stream disconnected: " + reason)
		a.setEventsStreamState(fmt.Sprintf("reconnecting in %s (%s)", delay, reason))
		if !sleepContext(a.ctx, delay) {
			return
		}
		delay = nextBackoffDelay(delay)
	}
}

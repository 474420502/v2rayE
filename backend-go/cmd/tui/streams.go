package tui

import (
	"context"
	"time"
)

func (a *tuiApp) startBackgroundWork() {
	if !a.markBackgroundWorkStarted() {
		return
	}
	go a.runAction(a.t("action.initialLoad"), func(context.Context) error {
		return a.reloadAll()
	})
	go a.pollOverview()
	go a.streamLogs()
	go a.streamEvents()
}

func (a *tuiApp) markBackgroundWorkStarted() bool {
	return a.backgroundWorkStarted.CompareAndSwap(false, true)
}

func (a *tuiApp) streamLogs() {
	delay := time.Second
	for {
		if a.ctx.Err() != nil {
			return
		}
		a.setLogsStreamState(a.t("stream.state.connecting"))
		err := a.client.StreamLogs(a.ctx, func() {
			delay = time.Second
			a.setLogsStreamState(a.t("stream.state.connected"))
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
			reason = a.t("stream.reason.closed")
		}
		a.pushEvent(a.tf("stream.event.logDisconnected", reason))
		a.setLogsStreamState(a.tf("stream.state.reconnecting", delay, reason))
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
		a.setEventsStreamState(a.t("stream.state.connecting"))
		err := a.client.StreamEvents(a.ctx, func() {
			delay = time.Second
			a.setEventsStreamState(a.t("stream.state.connected"))
		}, func(msg EventMessage) error {
			a.pushEvent(formatEvent(msg))
			return nil
		})
		if a.ctx.Err() != nil {
			return
		}
		reason := errorString(err)
		if reason == "<nil>" {
			reason = a.t("stream.reason.closed")
		}
		a.pushEvent(a.tf("stream.event.eventDisconnected", reason))
		a.setEventsStreamState(a.tf("stream.state.reconnecting", delay, reason))
		if !sleepContext(a.ctx, delay) {
			return
		}
		delay = nextBackoffDelay(delay)
	}
}

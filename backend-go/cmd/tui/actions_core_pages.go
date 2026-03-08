package main

import "context"

func (a *tuiApp) startCoreAction(ctx context.Context) error {
	status, err := a.client.StartCore(ctx)
	if err == nil {
		a.storeCoreStatus(status)
	}
	return err
}

func (a *tuiApp) stopCoreAction(ctx context.Context) error {
	status, err := a.client.StopCore(ctx)
	if err == nil {
		a.storeCoreStatus(status)
	}
	return err
}

func (a *tuiApp) restartCoreAction(ctx context.Context) error {
	status, err := a.client.RestartCore(ctx)
	if err == nil {
		a.storeCoreStatus(status)
	}
	return err
}

func (a *tuiApp) refreshAllAction(context.Context) error {
	return a.reloadAll()
}

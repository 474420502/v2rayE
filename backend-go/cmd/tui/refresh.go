package tui

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

func (a *tuiApp) pollOverview() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			_ = a.reloadOverview()
		}
	}
}

func (a *tuiApp) reloadAll() error {
	var errs []error
	if err := a.reloadOverview(); err != nil {
		errs = append(errs, err)
	}
	if err := a.reloadProfiles(); err != nil {
		errs = append(errs, err)
	}
	if err := a.reloadSubscriptions(); err != nil {
		errs = append(errs, err)
	}
	if err := a.reloadNetwork(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func (a *tuiApp) reloadOverview() error {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	status, err := a.client.GetCoreStatus(ctx)
	if err != nil {
		return err
	}
	config, err := a.client.GetConfig(ctx)
	if err != nil {
		return err
	}
	a.applyConfiguredUILanguage(config)
	stats, err := a.client.GetStats(ctx)
	if err != nil {
		return err
	}
	availability, err := a.client.GetAvailability(ctx)
	if err != nil {
		return err
	}
	if users, usersErr := a.client.ListSystemProxyUsers(ctx); usersErr == nil {
		a.storeSystemProxyUsers(users)
	}

	a.storeOverview(status, config, stats, availability)
	a.refreshWidgets()
	return nil
}

func (a *tuiApp) reloadProfiles() error {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	profiles, err := a.client.GetProfiles(ctx)
	if err != nil {
		return err
	}

	a.storeProfiles(profiles)
	a.refreshWidgetsWithLists(true, false)
	return nil
}

func (a *tuiApp) reloadSubscriptions() error {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	subs, err := a.client.GetSubscriptions(ctx)
	if err != nil {
		return err
	}
	sort.Slice(subs, func(i, j int) bool {
		return strings.ToLower(subs[i].Remarks) < strings.ToLower(subs[j].Remarks)
	})

	a.storeSubscriptions(subs)
	a.refreshWidgetsWithLists(false, true)
	return nil
}

func (a *tuiApp) reloadNetwork() error {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	routing, err := a.client.GetRouting(ctx)
	if err != nil {
		return err
	}
	diagnostics, err := a.client.GetRoutingDiagnostics(ctx)
	if err != nil {
		return err
	}
	hits, err := a.client.GetRoutingHits(ctx)
	if err != nil {
		return err
	}

	a.storeNetwork(routing, diagnostics, hits)
	a.refreshWidgets()
	return nil
}

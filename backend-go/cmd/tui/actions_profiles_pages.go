package tui

import (
	"context"
	"fmt"
	"strings"
)

func (a *tuiApp) importProfileAction(ctx context.Context) error {
	uri := strings.TrimSpace(a.profileImport.Text())
	if uri == "" {
		return fmt.Errorf("empty profile uri")
	}
	profile, err := a.client.ImportProfile(ctx, uri)
	if err != nil {
		return err
	}
	a.storeSelectedProfileID(profile.ID)
	return a.reloadProfiles()
}

func (a *tuiApp) activateProfileAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return fmt.Errorf("no profile selected")
	}
	if err := a.client.SelectProfile(ctx, id); err != nil {
		return err
	}
	return a.reloadAll()
}

func (a *tuiApp) batchDelayProfilesAction(ctx context.Context) error {
	ids := make([]string, 0, len(a.profiles))
	a.mu.Lock()
	for _, profile := range a.profiles {
		ids = append(ids, profile.ID)
	}
	a.mu.Unlock()
	a.storeBatchDelayState(true, nil)
	a.refreshWidgets()
	if len(ids) == 0 {
		a.storeBatchDelayState(false, nil)
		a.refreshWidgets()
		return fmt.Errorf("no profiles available")
	}
	result, err := a.client.BatchTestProfileDelay(ctx, ids, 5000, 5)
	if err == nil {
		a.storeBatchDelayState(false, &result)
	} else {
		a.storeBatchDelayState(false, nil)
	}
	a.refreshWidgets()
	if err != nil {
		return err
	}
	a.pushEvent(fmt.Sprintf("batch delay test completed total=%d success=%d failed=%d", result.Total, result.Success, result.Failed))
	return a.reloadProfiles()
}

func (a *tuiApp) testSelectedProfileDelayAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return fmt.Errorf("no profile selected")
	}
	result, err := a.client.TestProfileDelay(ctx, id)
	if err != nil {
		return err
	}
	a.pushEvent(fmt.Sprintf("profile.delay %s available=%t delay=%dms %s", id, result.Available, result.DelayMs, result.Message))
	return a.reloadProfiles()
}

func (a *tuiApp) updateAllSubscriptionsAction(ctx context.Context) error {
	if err := a.client.UpdateAllSubscriptions(ctx); err != nil {
		return err
	}
	return a.reloadSubscriptions()
}

func (a *tuiApp) updateSelectedSubscriptionAction(ctx context.Context) error {
	id := a.currentSubscriptionID()
	if id == "" {
		return fmt.Errorf("no subscription selected")
	}
	if err := a.client.UpdateSubscription(ctx, id); err != nil {
		return err
	}
	return a.reloadAll()
}

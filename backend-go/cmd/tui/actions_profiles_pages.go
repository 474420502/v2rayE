package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rivo/tview"
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
	a.clearProfileEditDirty()
	a.pushEvent("profile imported: " + profile.ID)
	return a.reloadProfiles()
}

func (a *tuiApp) importAndLoadProfileAction(ctx context.Context) error {
	uri := strings.TrimSpace(a.profileImport.Text())
	if uri == "" {
		return fmt.Errorf("empty profile uri")
	}
	profile, err := a.client.ImportProfile(ctx, uri)
	if err != nil {
		return err
	}
	a.storeSelectedProfileID(profile.ID)
	a.clearProfileEditDirty()
	if err := a.reloadProfiles(); err != nil {
		return err
	}
	if err := a.loadSelectedProfileEditorAction(ctx); err != nil {
		return err
	}
	a.runUI(func(app *tview.Application) {
		a.profileImport.SetText("", app)
	})
	a.setProfileEditMessage("Profile editor: imported and loaded. You can edit and save now.")
	a.setFooter("Imported and loaded profile for editing: " + profile.ID)
	a.pushEvent("profile imported+loaded: " + profile.ID)
	return nil
}

func (a *tuiApp) loadSelectedProfileEditorAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return fmt.Errorf("no profile selected")
	}
	profile, err := a.client.GetProfile(ctx, id)
	if err != nil {
		return err
	}
	a.storeSelectedProfileID(profile.ID)
	a.clearProfileEditDirty()
	a.refreshWidgets()
	a.setProfileEditMessage("Profile editor: loaded latest values from backend.")
	return nil
}

func (a *tuiApp) resetProfileEditAction(ctx context.Context) error {
	a.setProfileEditMessage("Profile editor: reset to backend values.")
	return a.loadSelectedProfileEditorAction(ctx)
}

func (a *tuiApp) saveSelectedProfileEditAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return fmt.Errorf("no profile selected")
	}

	profile, err := a.client.GetProfile(ctx, id)
	if err != nil {
		a.setProfileEditMessage("Profile editor: failed to load current profile before save.")
		return err
	}

	name := strings.TrimSpace(a.profileEditName.Text())
	address := strings.TrimSpace(a.profileEditAddress.Text())
	portText := strings.TrimSpace(a.profileEditPort.Text())
	port := profile.Port
	if portText != "" {
		parsed, convErr := strconv.Atoi(portText)
		if convErr != nil || parsed <= 0 || parsed > 65535 {
			err := fmt.Errorf("invalid port, expected 1-65535")
			a.setProfileEditMessage("Profile editor error: " + err.Error())
			return err
		}
		port = parsed
	}
	network := strings.TrimSpace(strings.ToLower(a.profileEditNetwork.Text()))
	if network != "" && network != "tcp" && network != "ws" && network != "grpc" {
		err := fmt.Errorf("invalid network, use tcp|ws|grpc")
		a.setProfileEditMessage("Profile editor error: " + err.Error())
		return err
	}
	tlsEnabled := parseBoolText(a.profileEditTLS.Text())
	sni := strings.TrimSpace(a.profileEditSNI.Text())
	fingerprint := strings.TrimSpace(a.profileEditFingerprint.Text())
	alpn := splitCommaStrings(a.profileEditALPN.Text())
	realityPublicKey := strings.TrimSpace(a.profileEditRealityPK.Text())
	realityShortID := strings.TrimSpace(a.profileEditRealitySID.Text())
	wsPath := strings.TrimSpace(a.profileEditWSPath.Text())
	grpcService := strings.TrimSpace(a.profileEditGRPCSvc.Text())

	if name != "" {
		profile.Name = name
	}
	if address != "" {
		profile.Address = address
	}
	if port > 0 {
		profile.Port = port
	}

	if profile.Transport == nil {
		profile.Transport = &TransportConfig{}
	}
	if network != "" {
		profile.Transport.Network = network
	}
	profile.Transport.TLS = tlsEnabled
	profile.Transport.SNI = sni
	profile.Transport.Fingerprint = fingerprint
	profile.Transport.ALPN = alpn
	profile.Transport.RealityPublicKey = realityPublicKey
	profile.Transport.RealityShortID = realityShortID
	profile.Transport.WSPath = wsPath
	profile.Transport.GRPCServiceName = grpcService

	updated, err := a.client.UpdateProfile(ctx, id, profile)
	if err != nil {
		a.setProfileEditMessage("Profile editor: save failed, check parameters and try again.")
		return err
	}
	a.storeSelectedProfileID(updated.ID)
	a.clearProfileEditDirty()
	a.setProfileEditMessage("Profile editor: saved successfully.")
	a.pushEvent("profile updated: " + updated.ID)
	if err := a.reloadProfiles(); err != nil {
		return err
	}
	return a.reloadOverview()
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

func (a *tuiApp) deleteSelectedProfileAction(ctx context.Context) error {
	id := a.currentProfileID()
	if id == "" {
		return fmt.Errorf("no profile selected")
	}
	confirm := strings.TrimSpace(strings.ToUpper(a.profileDeleteConfirm.Text()))
	if confirm != "DELETE" {
		err := fmt.Errorf("delete confirm required, type DELETE")
		a.setProfileEditMessage("Profile editor error: " + err.Error())
		return err
	}
	if err := a.client.DeleteProfile(ctx, id); err != nil {
		a.setProfileEditMessage("Profile editor: delete failed.")
		return err
	}
	a.runUI(func(app *tview.Application) {
		a.profileDeleteConfirm.SetText("", app)
	})
	a.setProfileEditMessage("Profile editor: deleted selected profile.")
	a.pushEvent("profile deleted: " + id)
	a.storeSelectedProfileID("")
	a.clearProfileEditDirty()
	return a.reloadProfiles()
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

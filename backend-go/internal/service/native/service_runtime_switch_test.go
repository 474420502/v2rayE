package native

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/storage"
)

func newRuntimeSwitchTestService(t *testing.T) (*Service, *storage.Store) {
	t.Helper()
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	profiles := []domain.ProfileItem{{ID: "p1", Name: "one"}, {ID: "p2", Name: "two"}}
	if err := store.SaveProfiles(profiles); err != nil {
		t.Fatalf("SaveProfiles() error = %v", err)
	}
	if err := store.SaveState(domain.PersistentState{CurrentProfileID: "p1"}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	return &Service{store: store, running: true}, store
}

func TestSelectProfileRunningRestartsSynchronously(t *testing.T) {
	t.Parallel()

	svc, store := newRuntimeSwitchTestService(t)
	restarts := 0
	svc.restartCoreHook = func() domain.CoreStatus {
		restarts++
		return domain.CoreStatus{Running: true}
	}

	if err := svc.SelectProfile("p2"); err != nil {
		t.Fatalf("SelectProfile() error = %v", err)
	}
	if restarts != 1 {
		t.Fatalf("SelectProfile() restart count = %d, want 1", restarts)
	}
	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.CurrentProfileID != "p2" {
		t.Fatalf("CurrentProfileID = %q, want p2", state.CurrentProfileID)
	}
}

func TestUpdateRoutingConfigRunningRestartsSynchronously(t *testing.T) {
	t.Parallel()

	svc, store := newRuntimeSwitchTestService(t)
	restarts := 0
	svc.restartCoreHook = func() domain.CoreStatus {
		restarts++
		return domain.CoreStatus{Running: true}
	}

	routing := domain.RoutingConfig{Mode: "bypass_cn", DomainStrategy: "IPIfNonMatch"}
	if got := svc.UpdateRoutingConfig(routing); got.Mode != routing.Mode || got.DomainStrategy != routing.DomainStrategy {
		t.Fatalf("UpdateRoutingConfig() = %+v, want %+v", got, routing)
	}
	if restarts != 1 {
		t.Fatalf("UpdateRoutingConfig() restart count = %d, want 1", restarts)
	}
	stored, err := store.LoadRoutingConfig()
	if err != nil {
		t.Fatalf("LoadRoutingConfig() error = %v", err)
	}
	if stored.Mode != routing.Mode || stored.DomainStrategy != routing.DomainStrategy {
		t.Fatalf("stored routing = %+v, want %+v", stored, routing)
	}
}

func TestRuntimeSwitchesAreSerialized(t *testing.T) {
	t.Parallel()

	svc, _ := newRuntimeSwitchTestService(t)
	var active int32
	var maxActive int32
	svc.restartCoreHook = func() domain.CoreStatus {
		current := atomic.AddInt32(&active, 1)
		defer atomic.AddInt32(&active, -1)
		for {
			seen := atomic.LoadInt32(&maxActive)
			if current <= seen || atomic.CompareAndSwapInt32(&maxActive, seen, current) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
		return domain.CoreStatus{Running: true}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := svc.SelectProfile("p2"); err != nil {
			t.Errorf("SelectProfile() error = %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		svc.UpdateRoutingConfig(domain.RoutingConfig{Mode: "direct", DomainStrategy: "AsIs"})
	}()
	wg.Wait()

	if got := atomic.LoadInt32(&maxActive); got != 1 {
		t.Fatalf("max concurrent runtime switches = %d, want 1", got)
	}
}
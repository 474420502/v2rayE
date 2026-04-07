package native

import (
	"path/filepath"
	"testing"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/storage"
)

func TestShutdownCorePreservesRestoreIntent(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	if err := store.SaveState(domain.PersistentState{CoreShouldRestore: true}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	svc := New(dataDir, "xray", store)
	svc.ShutdownCore()

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if !state.CoreShouldRestore {
		t.Fatalf("ShutdownCore() cleared CoreShouldRestore, want preserved true")
	}
	if status := svc.CoreStatus(); status.Running {
		t.Fatalf("ShutdownCore() left service running")
	}
}

func TestStopCoreClearsRestoreIntent(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	if err := store.SaveState(domain.PersistentState{CoreShouldRestore: true}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	svc := New(dataDir, "xray", store)
	svc.StopCore()

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.CoreShouldRestore {
		t.Fatalf("StopCore() preserved CoreShouldRestore, want cleared false")
	}
}

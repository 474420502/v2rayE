//go:build unix

package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAcquireInstanceLockRejectsSecondInstance(t *testing.T) {
	dataDir := t.TempDir()

	release, err := AcquireInstanceLock(dataDir)
	if err != nil {
		t.Fatalf("first AcquireInstanceLock() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, instanceLockFileName)); err != nil {
		t.Fatalf("lock file was not created: %v", err)
	}

	_, err = AcquireInstanceLock(dataDir)
	if !errors.Is(err, ErrInstanceLocked) {
		t.Fatalf("second AcquireInstanceLock() error = %v, want ErrInstanceLocked", err)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(os.Getpid())) {
		t.Fatalf("lock error should name the holder pid, got %q", err.Error())
	}

	release()

	releaseAgain, err := AcquireInstanceLock(dataDir)
	if err != nil {
		t.Fatalf("AcquireInstanceLock() after release error = %v", err)
	}
	releaseAgain()
}

func TestInstanceLockHolderPID(t *testing.T) {
	dataDir := t.TempDir()

	if pid, ok := InstanceLockHolderPID(dataDir); ok {
		t.Fatalf("InstanceLockHolderPID() = %d before any lock was taken", pid)
	}

	release, err := AcquireInstanceLock(dataDir)
	if err != nil {
		t.Fatalf("AcquireInstanceLock() error = %v", err)
	}
	defer release()

	pid, ok := InstanceLockHolderPID(dataDir)
	if !ok {
		t.Fatal("InstanceLockHolderPID() reported no holder")
	}
	if pid != os.Getpid() {
		t.Fatalf("InstanceLockHolderPID() = %d, want %d", pid, os.Getpid())
	}
}

func TestAcquireInstanceLockIgnoresStaleContentBeforeLock(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, instanceLockFileName), []byte("999999\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	// A leftover pid from a crashed process must not block a fresh start: the
	// kernel lock, not the file content, decides ownership.
	release, err := AcquireInstanceLock(dataDir)
	if err != nil {
		t.Fatalf("AcquireInstanceLock() with stale content error = %v", err)
	}
	defer release()

	pid, ok := InstanceLockHolderPID(dataDir)
	if !ok || pid != os.Getpid() {
		t.Fatalf("holder pid = %d (%v), want %d", pid, ok, os.Getpid())
	}
}

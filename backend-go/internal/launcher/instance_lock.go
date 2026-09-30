package launcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"v2raye/backend-go/internal/storage"
)

// instanceLockFileName is the advisory lock file placed inside the data
// directory. It guards the TUN device, the policy-routing rules and the JSON
// state files, all of which are shared by every server process using that data
// directory.
const instanceLockFileName = "v2raye.lock"

// ErrInstanceLocked is returned when another server instance already manages
// the same data directory.
var ErrInstanceLocked = errors.New("another v2rayE server instance is already managing this data directory")

// AcquireInstanceLock takes an exclusive advisory lock on the data directory.
//
// Two concurrent servers sharing a data directory fight over the same TUN
// device and policy-routing rules: teardown deletes the interface by name, so a
// dying instance can tear the network away from a running one (and each
// repeated failure removes the rules again). Failing fast here keeps the
// second process from touching any of that state.
//
// The lock is held for the lifetime of the process; the kernel releases it on
// exit, so a crash never leaves a stale lock behind. The returned function
// releases the lock explicitly.
func AcquireInstanceLock(dataDir string) (func(), error) {
	path := filepath.Join(storage.ResolveDataDir(dataDir), instanceLockFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create data directory for instance lock: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o640)
	if err != nil {
		return nil, fmt.Errorf("open instance lock %s: %w", path, err)
	}
	if err := lockFileExclusive(file); err != nil {
		holder := readInstanceLockHolder(file)
		_ = file.Close()
		if errors.Is(err, errFileLocked) {
			if holder != "" {
				return nil, fmt.Errorf("%w (pid %s holds %s)", ErrInstanceLocked, holder, path)
			}
			return nil, fmt.Errorf("%w (%s)", ErrInstanceLocked, path)
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}

	// Record the holder pid for diagnostics: it lets a teardown path tell
	// whether the device it is about to delete belongs to somebody else.
	_ = file.Truncate(0)
	if _, err := file.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0); err != nil {
		// The lock itself is what enforces single-instance; the pid is a hint.
		_ = err
	}

	return func() {
		unlockFile(file)
		_ = file.Close()
	}, nil
}

// InstanceLockHolderPID returns the pid recorded inside the data directory lock
// file, if the file exists and holds a plausible pid.
func InstanceLockHolderPID(dataDir string) (int, bool) {
	raw, err := os.ReadFile(filepath.Join(storage.ResolveDataDir(dataDir), instanceLockFileName))
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func readInstanceLockHolder(file *os.File) string {
	buf := make([]byte, 32)
	n, err := file.ReadAt(buf, 0)
	if n <= 0 {
		_ = err
		return ""
	}
	return strings.TrimSpace(string(buf[:n]))
}

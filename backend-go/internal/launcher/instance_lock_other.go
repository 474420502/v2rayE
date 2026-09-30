//go:build !unix

package launcher

import (
	"errors"
	"os"
)

// errFileLocked is never returned on platforms without flock support.
var errFileLocked = errors.New("lock file is held by another process")

// lockFileExclusive is a no-op where flock is unavailable: single-instance
// enforcement is best-effort on those platforms.
func lockFileExclusive(*os.File) error { return nil }

func unlockFile(*os.File) {}

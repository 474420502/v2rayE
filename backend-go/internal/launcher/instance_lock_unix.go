//go:build unix

package launcher

import (
	"errors"
	"os"
	"syscall"
)

// errFileLocked reports that the lock file is held by another process.
var errFileLocked = errors.New("lock file is held by another process")

func lockFileExclusive(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return errFileLocked
	}
	return err
}

func unlockFile(file *os.File) {
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

//go:build unix

package processenv

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func executable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	err = unix.Access(path, unix.X_OK)
	if err == nil {
		return true
	}
	// Match os/exec's fallback when the access check is unavailable or blocked
	// by a sandbox that filters the syscall.
	if errors.Is(err, syscall.ENOSYS) || errors.Is(err, syscall.EPERM) {
		return info.Mode()&0o111 != 0
	}
	return false
}

//go:build !windows

package crosscompile

import (
	"os"
	"syscall"
)

func lockFileHandle(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

func unlockFileHandle(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

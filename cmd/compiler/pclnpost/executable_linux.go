package pclnpost

import "syscall"

// A concurrent fork can inherit a writable executable descriptor until exec,
// even with O_CLOEXEC. Closing our copy and renaming the file cannot release
// that inherited writer, so executing the published inode can fail with
// ETXTBSY (https://go.dev/issue/22315).
//
// Exclude Go's forks from before opening the file until after closing it. A
// read lock permits concurrent executable writers; existing child processes
// continue running. No subprocess may be started while this lock is held.
func lockExecutableWrite() func() {
	syscall.ForkLock.RLock()
	return syscall.ForkLock.RUnlock
}

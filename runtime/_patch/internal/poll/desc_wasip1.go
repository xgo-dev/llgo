//go:build llgo && wasip1

package poll

import "syscall"

//llgo:skip (*pollDesc).init

// The WASI runtime has a no-op poll server and returns ENOTSUP on open.
func (*pollDesc) init(*FD) error { return errnoErr(syscall.ENOTSUP) }

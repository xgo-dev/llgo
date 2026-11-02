//go:build llgo && linux && !wasm && !baremetal

package abicall

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

//go:linkname cgocaller syscall.cgocaller
func cgocaller(unsafe.Pointer, ...uintptr) uintptr

func TestLibcCredentialWrappers(t *testing.T) {
	uid, gid := os.Getuid(), os.Getgid()
	for _, err := range []error{
		syscall.Setuid(uid), syscall.Seteuid(os.Geteuid()),
		syscall.Setgid(gid), syscall.Setegid(os.Getegid()),
		syscall.Setreuid(-1, -1), syscall.Setregid(-1, -1),
		syscall.Setresuid(-1, -1, -1), syscall.Setresgid(-1, -1, -1),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := syscall.Setuid(-1); err != syscall.EINVAL {
		t.Fatalf("Setuid(-1)=%v", err)
	}
	groups, err := syscall.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Setgroups(groups); err != nil && err != syscall.EPERM {
		t.Fatal(err)
	}
	if got := cgocaller(nil); got != uintptr(syscall.ENOTSUP) {
		t.Fatalf("nil cgocaller=%d", got)
	}
}

func TestAllThreadsBuildModeBoundary(t *testing.T) {
	for _, call := range []func() (uintptr, uintptr, syscall.Errno){
		func() (uintptr, uintptr, syscall.Errno) {
			return syscall.AllThreadsSyscall(syscall.SYS_GETPID, 0, 0, 0)
		},
		func() (uintptr, uintptr, syscall.Errno) {
			return syscall.AllThreadsSyscall6(syscall.SYS_SETRESUID, ^uintptr(0), ^uintptr(0), ^uintptr(0), 0, 0, 0)
		},
	} {
		r1, r2, err := call()
		if err != syscall.ENOTSUP || r1 != ^uintptr(0) || r2 != ^uintptr(0) {
			t.Fatalf("libc/cgo all-thread boundary=%d,%d,%v", r1, r2, err)
		}
	}
}

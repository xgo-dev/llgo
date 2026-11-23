package main

import "syscall"

func init() {
	// A piped Linux core dump can bypass RLIMIT_CORE. Disable dumpability
	// for this fixture while keeping the crash signal observable.
	_, _, err := syscall.RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_DUMPABLE, 0, 0, 0, 0, 0)
	if err != 0 {
		panic(err)
	}
}

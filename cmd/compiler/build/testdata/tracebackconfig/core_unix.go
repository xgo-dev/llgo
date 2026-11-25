//go:build linux || darwin

package main

import "syscall"

func init() {
	// Crash-mode tests require abnormal termination but no core file.
	_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{})
}

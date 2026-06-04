// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package unix

// SyscallNoError may be used instead of Syscall for syscalls that don't fail.
func SyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr) {
	r1, r2, _ = Syscall(trap, a1, a2, a3)
	return
}

// RawSyscallNoError may be used instead of RawSyscall for syscalls that don't
// fail.
func RawSyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr) {
	r1, r2, _ = RawSyscall(trap, a1, a2, a3)
	return
}

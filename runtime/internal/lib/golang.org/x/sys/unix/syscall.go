// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unix

import (
	"syscall"
	_ "unsafe"
)

type Errno = syscall.Errno

//go:linkname Syscall syscall.Syscall
func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname Syscall6 syscall.Syscall6
func Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname RawSyscall syscall.RawSyscall
func RawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname RawSyscall6 syscall.RawSyscall6
func RawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname syscall_syscall syscall.syscall
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname syscall_syscall6 syscall.syscall6
func syscall_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname syscall_syscall6X syscall.syscall6X
func syscall_syscall6X(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname syscall_syscall9 syscall.syscall9
func syscall_syscall9(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2 uintptr, err Errno) // 32-bit only

//go:linkname syscall_rawSyscall syscall.rawSyscall
func syscall_rawSyscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname syscall_rawSyscall6 syscall.rawSyscall6
func syscall_rawSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno)

//go:linkname syscall_syscallPtr syscall.syscallPtr
func syscall_syscallPtr(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno)

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/Go-BSD-3-Clause.txt
// file at this module root for license terms.

//go:build arm && baremetal

package linux

import "unsafe"

//llgo:skip Syscall6

// Baremetal images masquerade as linux/arm, so os.File's console IO
// (fmt -> os.Stdout -> syscall.Write) reaches this package even though there
// is no kernel to syscall. The C library does provide write(), which lands in
// the target device's _write hook - the same fd-1/2 console that C stdio
// (builtin println, fputc) drains - so route the write syscall through it.
// Every other request still reports ENOSYS.

//go:linkname llgoSysWrite C.write
func llgoSysWrite(fd int32, buf unsafe.Pointer, count uint32) int32

// Masqueraded zsysnum_linux_arm.go numbers.
const sysWrite = 4

// The device _write hook reports failures through the C errno, which
// baremetal images do not model (see runtime/internal/clite/os/os_baremetal.go),
// so every write failure maps to EBADF regardless of the errno the hook set.
const sysEBADF = 9

func Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr) {
	_, _, _ = a4, a5, a6
	if num == sysWrite {
		n := llgoSysWrite(int32(a1), unsafe.Pointer(a2), uint32(a3))
		if n < 0 {
			return 0, 0, sysEBADF
		}
		return uintptr(n), 0, 0
	}
	_, _, _ = a1, a2, a3
	return 0, 0, 38 // ENOSYS
}

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/Go-BSD-3-Clause.txt
// file at this module root for license terms.

//go:build arm && !baremetal

package linux

//llgo:skip Syscall6

// LLGo cannot translate the arm syscall assembly, so this package's Syscall6
// gets a portable replacement. Hosted arm images answer every request with
// ENOSYS; baremetal arm images select syscall6_baremetal_arm.go instead.
func Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr) {
	_ = num
	_, _ = a1, a2
	_, _ = a3, a4
	_, _ = a5, a6
	return 0, 0, 38 // ENOSYS
}

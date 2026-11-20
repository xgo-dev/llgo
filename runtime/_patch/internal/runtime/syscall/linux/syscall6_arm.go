// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/Go-BSD-3-Clause.txt
// file at this module root for license terms.

//go:build arm

package linux

//llgo:skip Syscall6

// LLGo cannot translate the arm syscall assembly. Baremetal images have no
// syscall surface (and no LLGo caller reaches these netpoll primitives), so
// every request reports ENOSYS.
func Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr) {
	_ = num
	_, _ = a1, a2
	_, _ = a3, a4
	_, _ = a5, a6
	return 0, 0, 38 // ENOSYS
}

//go:build windows

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import _ "unsafe"

// windowsSyscall matches the argument block used by Go's Windows stdcall
// bridge. Keeping the variable-width call in one native thunk avoids inventing
// fixed-arity variants of syscall.SyscallN.
type windowsSyscall struct {
	fn   uintptr
	n    uintptr
	args *uintptr
	r1   uintptr
	r2   uintptr
	err  uintptr
}

//go:linkname c_windowsSyscall C.llgo_windows_syscall
func c_windowsSyscall(call *windowsSyscall)

// syscall_syscalln calls fn with args[:n] using the native Windows ABI.
// Like the Go runtime implementation, it clears last-error before the call and
// returns the primary and secondary result words. The secondary word is the
// floating-point result on amd64, the high integer word on 386, and zero on
// arm64.
//
//go:linkname syscall_syscalln syscall.syscalln
//go:nosplit
func syscall_syscalln(fn, n uintptr, args ...uintptr) (r1, r2, err uintptr) {
	if n > uintptr(len(args)) {
		panic("syscall: n > len(args)")
	}
	if windowsSyscallMaxArgs != 0 && n > windowsSyscallMaxArgs {
		panic("runtime: SyscallN has too many arguments")
	}

	var argp *uintptr
	if n != 0 {
		argp = &args[0]
	}
	call := windowsSyscall{fn: fn, n: n, args: argp}
	c_windowsSyscall(&call)
	return call.r1, call.r2, call.err
}

//go:build !wasm && !baremetal

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/signal"
)

const (
	// SIGSEGV is signal number 11 on all Unix-like systems (Linux, Darwin, BSD, etc.)
	// Using a hardcoded constant avoids importing the syscall package, which would
	// introduce dependencies on errors and internal/reflectlite packages that cause
	// linking issues in c-shared and c-archive build modes.
	SIGSEGV = c.Int(0xb)
)

// This file contains platform-specific runtime initialization for non-wasm targets.
// The SIGSEGV signal handler enables Go-style panic recovery for nil pointer dereferences
// instead of immediate process termination.
//
// For wasm platform compatibility, signal handling is excluded via build tags.
// See PR #1059 for wasm platform requirements.
func init() {
	handleFault := func(v c.Int) {
		if v == SIGSEGV || v == SIGBUS {
			panic(errorString("invalid memory address or nil pointer dereference"))
		}
		var buf [20]byte
		panic(errorString("unexpected signal value: " + string(itoa(buf[:], uint64(v)))))
	}
	signal.Signal(SIGSEGV, handleFault)
	if SIGBUS != 0 {
		signal.Signal(SIGBUS, handleFault)
	}
}

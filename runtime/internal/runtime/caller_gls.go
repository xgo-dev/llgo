//go:build llgo && !baremetal

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

import "github.com/xgo-dev/llgo/runtime/internal/sync/atomic"

// callerLocationStoreCurrent follows the logical goroutine: its shadow stack
// and synthetic PCs must move with that goroutine when the backend eventually
// permits migration between OS threads.
//
//llgointernal:gls
var callerLocationStoreCurrent *callerLocationStore

// callerPCSequence makes synthetic PCs unique across goroutine-local stores.
// The public runtime caches FuncForPC results process-wide, so reusing a
// compact local sequence would let one goroutine resolve another's frame.
var callerPCSequence uintptr

func nextCallerPCBase(*callerLocationStore) uintptr {
	seq := atomic.Add(&callerPCSequence, uintptr(1)) + 1
	if seq == 0 || seq > ^uintptr(0)>>2 {
		fatal("runtime: caller PC sequence exhausted")
	}
	return seq << 2
}

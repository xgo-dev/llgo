//go:build llgo && wasm && llgo.wasm.gc.linear && !nogc && !baremetal

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

package tls

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc"
)

type slot[T any] struct {
	value      T
	destructor func(*T)
}

func allocSlot(size uintptr) c.Pointer {
	// Host TLS is outside the GC's scanned roots. Keep the slot and every Go
	// pointer it contains reachable until the thread-local destructor runs.
	return c.Pointer(tinygogc.AllocRoot(size))
}

func freeSlot(ptr c.Pointer) {
	tinygogc.FreeRoot(unsafe.Pointer(ptr))
}

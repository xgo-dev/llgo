//go:build llgo && baremetal && tinygo.riscv32 && !nogc

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

// Fiber-backed sync.Pool locality for the baremetal RV32 fiber scheduler.
// The hosted implementation keys pool locals on pthread TLS and initializes
// its table through psync.Once, whose pthread_once is a libnosys stub on
// baremetal; here every G shares one hart, so one global table replaces both
// mechanisms and needs no one-shot initialization.
//
// Entries live until exit: there is no thread teardown to collect them, and
// the wordmap values stay pointer-typed so the conservative collector retains
// every parked pool local.

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
)

var poolLocals wordmap.Map[unsafe.Pointer, unsafe.Pointer]

//go:linkname syncRuntimePoolLocalAlloc sync.runtime_poolLocalAlloc
func syncRuntimePoolLocalAlloc(victim *unsafe.Pointer) unsafe.Pointer {
	// The victim field identifies the pool; on this backend there is no
	// thread exit that would publish locals back through it.
	return unsafe.Pointer(victim)
}

//go:linkname syncRuntimePoolLocalGet sync.runtime_poolLocalGet
func syncRuntimePoolLocalGet(handle unsafe.Pointer) unsafe.Pointer {
	return poolLocals.Get(handle)
}

//go:linkname syncRuntimePoolLocalSet sync.runtime_poolLocalSet
func syncRuntimePoolLocalSet(handle, local unsafe.Pointer) {
	poolLocals.Set(handle, local)
}

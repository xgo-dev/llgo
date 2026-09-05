//go:build (baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)

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
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc"
)

func AllocU(size uintptr) unsafe.Pointer {
	ret := tinygogc.Alloc(size)
	if ret == nil && size != 0 {
		panic("out of memory")
	}
	recordMemProfileAlloc(size)
	return ret
}

func AllocZ(size uintptr) unsafe.Pointer {
	ret := tinygogc.Alloc(size)
	recordMemProfileAlloc(size)
	return ret
}

func AllocRoot(size uintptr) unsafe.Pointer {
	return tinygogc.Alloc(size)
}

func FreeRoot(ptr unsafe.Pointer) {
	tinygogc.Free(ptr)
}

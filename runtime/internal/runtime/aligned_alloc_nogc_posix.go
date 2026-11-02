//go:build nogc && !baremetal && (!wasm || !llgo.wasm.gc.linear) && (aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || wasip1)

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

import "unsafe"

//go:linkname posixMemalign C.posix_memalign
func posixMemalign(result *unsafe.Pointer, alignment, size uintptr) int32

func allocUAligned(size, alignment uintptr) unsafe.Pointer {
	var ret unsafe.Pointer
	if posixMemalign(&ret, alignment, size) != 0 || ret == nil {
		panic("out of memory")
	}
	recordMemProfileAlloc(size)
	return ret
}

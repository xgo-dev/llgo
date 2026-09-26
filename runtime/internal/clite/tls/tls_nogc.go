//go:build llgo && (nogc || baremetal || (wasm && !llgo.wasm.gc.linear))

/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

type slot[T any] struct {
	value      T
	destructor func(*T)
}

func allocSlot(size uintptr) c.Pointer {
	return c.Calloc(1, size)
}

func freeSlot(ptr c.Pointer) {
	c.Free(ptr)
}

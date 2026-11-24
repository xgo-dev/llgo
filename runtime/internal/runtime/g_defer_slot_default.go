//go:build !(llgo && js && wasm && llgo.wasm.workers)

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

// ThreadDeferSlot returns the current goroutine's defer chain slot.
// On all builds covered by this file (g_tls.go, g_wasm.go, g_global.go),
// getg() always initializes and returns a non-nil *g, so unsafe.Add avoids a
// redundant nil-dereference branch on every defer activation.
func ThreadDeferSlot() **Defer {
	return (**Defer)(unsafe.Add(unsafe.Pointer(getg()), unsafe.Offsetof(g{}.defer_)))
}

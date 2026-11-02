//go:build !baremetal

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

// runtimeNano mirrors Go's runtime.nanotime: the monotonic clock in
// nanoseconds, implemented per OS by nanotime1.
func runtimeNano() int64 {
	return nanotime1()
}

// internal/poll and external packages use this hook through go:linkname,
// including on Windows and Wasm where the Unix poll adapter is not selected.
func nanotime() int64 { return runtimeNano() }

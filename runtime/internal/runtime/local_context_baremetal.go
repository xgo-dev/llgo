//go:build llgo && (baremetal || (wasm && !llgo.wasm.workers && !(wasip1 && llgo.wasi_threads)))

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

// Bare-metal and single-worker WebAssembly runtimes have one physical execution
// context. Keep this bootstrap cache outside generated locality storage:
// EnterLocalContext must be able to install the first context before that
// storage can be accessed. Multi-worker WASI and Emscripten select native TLS.
var currentLocalContext uintptr

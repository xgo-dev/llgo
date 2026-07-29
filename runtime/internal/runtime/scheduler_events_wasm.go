//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads) && !llgo.wasm_workers

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

import "github.com/goplus/llgo/runtime/internal/wasmevent"

func popWasmRunq() *g {
	wasmevent.Poll()
	return wasmSched.runq.Pop()
}

func waitWasmRunq() *g {
	for {
		if gp := popWasmRunq(); gp != nil {
			return gp
		}
		if !wasmevent.Wait() {
			return nil
		}
	}
}

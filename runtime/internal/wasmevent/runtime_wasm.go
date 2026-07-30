//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads)

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

package wasmevent

import _ "unsafe"

const LLGoFiles = "_wrap/event_wasm.c"

var runtimeQueue queue

func Reset(timer *Timer, when, period int64, callback Callback, arg any) bool {
	if timer == nil {
		return false
	}
	installEventLoop(pollRuntimeQueue, waitRuntimeQueue)
	return runtimeQueue.reset(timer, when, period, callback, arg)
}

func Stop(timer *Timer) bool {
	return runtimeQueue.stop(timer)
}

func pollRuntimeQueue() int {
	return runtimeQueue.runDue(Now())
}

func waitRuntimeQueue() bool {
	return waitForEvent(&runtimeQueue, Now, hostWait)
}

func Now() int64 {
	return hostNow()
}

//go:linkname hostNow C.llgo_wasm_event_now
func hostNow() int64

//go:linkname hostWait C.llgo_wasm_event_wait
func hostWait(nanoseconds uint64)

//go:build llgo && baremetal && tinygo.riscv32 && llgo.idf

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

import _ "unsafe" // for go:linkname

// LockCore pins the FreeRTOS task that hosts the llgo fiber scheduler (in
// llgo.idf mode that is the IDF main task every goroutine runs on) to the HP
// core with the given id.
//
// The pin goes through the weak C hook llgoLockCore (_wrap/lockcore_idf.c):
// a firmware ships its own strong definition when it wants the real pin, and
// a firmware without one still links - LockCore then reports an explicit
// error. The IDF kernel that S31 firmware links pins a task once, at
// creation (xTaskCreatePinnedToCore writes TCB.xCoreID; vTaskCoreAffinitySet
// belongs to the Amazon SMP kernel whose Kconfig entry is
// `depends on IDF_TARGET_ESP32`, so an S31 build never links it), which is
// why the hook exists at all; the reference strong implementation with its
// self-check discipline is _demo/esp-idf-lockcore/main/lockcore_glue.c.
//
// Hook return contract: 0 = pinned, 1 = not provided by this firmware,
// anything else = the implementing shim refused.
//
// Pinning moves the scheduler task between HP cores; it does not give Go
// more parallelism (the fiber scheduler stays single-hart) - it is the
// isolation knob: keep Go away from (or on the same core as) ISR-heavy tasks
// such as WiFi. Migration materializes at the next scheduling point; the pin
// verified here is the scheduler's affinity, not the core the task last ran
// on (probe that with a FreeRTOS getter after a yield, as the demo does).

// lockCoreCount is the number of HP cores the scheduler task can be pinned
// to: ESP32-S31 has two (SOC_CPU_CORES_NUM = 2; the LP core is not a
// FreeRTOS scheduler target).
const lockCoreCount = 2

// XTaskGetCurrentTaskHandle returns the FreeRTOS handle of the calling task.
//
//go:linkname XTaskGetCurrentTaskHandle C.xTaskGetCurrentTaskHandle
func XTaskGetCurrentTaskHandle() uintptr

// XTaskGetCoreID returns the core the given task is pinned to, or
// tskNO_AFFINITY for an unpinned task.
//
//go:linkname XTaskGetCoreID C.xTaskGetCoreID
func XTaskGetCoreID(handle uintptr) int32

// CLlgoLockCore applies the pin through the weak hook; see the return
// contract above.
//
//go:linkname CLlgoLockCore C.llgoLockCore
func CLlgoLockCore(core int32) int32

// LockCore pins the llgo scheduler task to HP core id. Out-of-range ids fail
// without touching the kernel; a pin is reported success only after the
// kernel's own getter reads it back.
func LockCore(id int) error {
	if id < 0 || id >= lockCoreCount {
		return errorString("runtime.LockCore: core id out of range")
	}
	handle := XTaskGetCurrentTaskHandle()
	switch rc := CLlgoLockCore(int32(id)); rc {
	case 0:
	case 1:
		return errorString("runtime.LockCore: core pinning not provided by this firmware")
	default:
		return errorString("runtime.LockCore: core pin shim refused the pin")
	}
	if XTaskGetCoreID(handle) != int32(id) {
		return errorString("runtime.LockCore: core pin did not take effect")
	}
	return nil
}

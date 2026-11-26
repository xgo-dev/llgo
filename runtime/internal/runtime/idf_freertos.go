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

// The llgo.idf build tag selects the host-FreeRTOS mode for ESP32-S31: llgo
// compiles the program into a relocatable object (the target's -r link) that
// an ESP-IDF component links into the IDF firmware image. FreeRTOS owns the
// platform, and this backend deliberately installs nothing of its own:
//
//   - Startup: the IDF app_main calls the generated C entry (main), which
//     hands package initialization and main.main to RunMain. There is no
//     llgo crt0, linker script, trap vector, watchdog or clock-init code in
//     this mode - FreeRTOS already owns all of it.
//   - Stacks: RunMain multiplexes every G onto cooperative fibers allocated
//     from the GC heap, exactly like the baremetal fiber backend. The
//     bootstrap scheduler context runs on the FreeRTOS main-task stack; the
//     guard/scan window the runtime reads through the _stack_end/_stack_top
//     symbols is provided by the IDF component (see the component's symbol
//     definition file).
//   - Clock: read-only cycle CSR through clock_riscv32.c (the esp32s31 chip
//     gate). No counter is started or configured; keep the IDF CPU frequency
//     at 240 MHz so cycle-domain deadlines match wall time.
//   - Panic/exit: a panic unwinds through setjmp/longjmp inside the fiber,
//     identical to baremetal; an unrecovered panic reaches exit(2), which the
//     IDF component should map to parking the task instead of a chip restart
//     so diagnostics survive.
//   - Scheduling: goroutines never block FreeRTOS resources directly; time
//     deadlines busy-poll the cycle clock from the main task. A FreeRTOS
//     call that blocks (vTaskDelay, queues, mutexes) blocks the whole llgo
//     scheduler and must only be used from main with that in mind.

// VTaskDelay blocks the calling task for the given number of FreeRTOS ticks.
// It is the canonical example of an llgo.idf C binding: llgo emits an
// external reference to the C symbol and the ESP-IDF link environment
// resolves it. User packages declare their own copy of this pattern (the
// runtime package is internal), or copy these declarations.
//
//go:linkname VTaskDelay C.vTaskDelay
func VTaskDelay(ticks uint32)

// XTaskGetTickCount returns the current FreeRTOS tick count.
//
//go:linkname XTaskGetTickCount C.xTaskGetTickCount
func XTaskGetTickCount() uint32

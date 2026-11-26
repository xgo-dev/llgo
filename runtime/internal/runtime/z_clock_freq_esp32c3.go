//go:build llgo && baremetal && tinygo.riscv32 && !nogc && esp32c3

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

// Nominal ESP32-C3 core frequency. Real C3 hardware implements the cycle
// CSR, but esp-emu's esp32c3 model provides no clock source at all, so the
// mutex starvation feed stays off to keep the contention fixtures runnable
// under emulation (BaremetalCycleClock); time.Sleep itself still reads the
// counter and runs on real hardware — the time fixtures are simply excluded
// from the C3 emulation suite.
const (
	baremetalCpuMHz     = 160
	BaremetalCycleClock = false
)

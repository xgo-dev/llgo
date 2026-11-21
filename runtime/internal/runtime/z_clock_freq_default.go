//go:build llgo && baremetal && tinygo.riscv32 && !nogc && !esp32c3 && !esp32s31

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

// Generic RV32 fiber target: nominal 160 MHz (the ESP32-C3 frequency) as a
// reasonable default. No emulator model is pinned for these targets, so the
// mutex starvation feed stays off until one proves the CSR read.
const (
	baremetalCpuMHz     = 160
	BaremetalCycleClock = false
)

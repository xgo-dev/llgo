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

// FPU enable for the esp32s31-f (ilp32f) hard-float target.
//
// The mask ROM hands control over with mstatus.FS = Off, and every F
// instruction raises an illegal-instruction exception (mcause = 2) until the
// FS field leaves Off — the standard RISC-V gating, confirmed under esp-emu
// (FS=on executes F instructions correctly; FS=off traps with cause 2).
// This constructor runs from crt1's __libc_init_array, before main and
// before the Go runtime starts, which is the earliest hook a linked-in C
// file can install: no Go code — including the fiber scheduler's first
// setjmp/longjmp, whose ilp32f jmp_buf carries F-register saves — executes
// before the FPU is available.
//
// Only FS = Initial (0b01) is needed: it permits F-register access, and the
// first executed F instruction advances the state to Dirty. The SRAM-only
// esp32s31-f layout never deep-sleeps the core, so no save/restore of FS
// around power modes is required here.

static void esp32s31_f_fpu_enable(void) {
  __asm__ volatile("li t0, 0x2000\n\t" /* mstatus.FS = Initial (0b01) */
                   "csrs mstatus, t0"
                   :
                   :
                   : "t0");
}

__attribute__((constructor)) static void esp32s31_f_boot_fpu(void) {
  esp32s31_f_fpu_enable();
}

// C-ABI boundary helper for the ilp32f convention: float arguments and the
// return value travel in fa0/fa1 (the cabi ILP32F classification, exercised
// from Go through a C.<symbol> linkname). Compiled with -march=rv32imafc,
// so the multiply itself also runs on the FPU.
float esp32s31_f_fmul(float a, float b) { return a * b; }

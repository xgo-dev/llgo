// Copyright (c) 2018-2026 The TinyGo Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice,
//   this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of the copyright holder nor the names of its contributors
//   may be used to endorse or promote products derived from this software
//   without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.
//
// This file was adapted for LLGo's wasmcontext ABI and wasm32 WASI scheduler.

.globaltype __stack_pointer, i32

.functype start_unwind (i32) -> ()
.import_module start_unwind, asyncify
.import_name start_unwind, start_unwind
.functype stop_unwind () -> ()
.import_module stop_unwind, asyncify
.import_name stop_unwind, stop_unwind
.functype start_rewind (i32) -> ()
.import_module start_rewind, asyncify
.import_name start_rewind, start_rewind
.functype stop_rewind () -> ()
.import_module stop_rewind, asyncify
.import_name stop_rewind, stop_rewind

.global __llgo_wasm_context_unwind
.hidden __llgo_wasm_context_unwind
.type __llgo_wasm_context_unwind,@function
__llgo_wasm_context_unwind:
    .functype __llgo_wasm_context_unwind (i32) -> ()
    i32.const 0
    i32.load8_u __llgo_wasm_context_rewinding
    if
        call stop_rewind
        i32.const 0
        i32.const 0
        i32.store8 __llgo_wasm_context_rewinding
    else
        local.get 0
        global.get __stack_pointer
        i32.store 16
        local.get 0
        i32.const 8
        i32.add
        call start_unwind
    end_if
    return
    end_function

.global __llgo_wasm_context_launch
.hidden __llgo_wasm_context_launch
.type __llgo_wasm_context_launch,@function
__llgo_wasm_context_launch:
    .functype __llgo_wasm_context_launch (i32) -> ()
    global.get __stack_pointer
    local.get 0
    i32.load 16
    global.set __stack_pointer
    local.get 0
    i32.load 4
    local.get 0
    i32.load 0
    call_indirect (i32) -> (i32)
    drop
    call stop_unwind
    global.set __stack_pointer
    return
    end_function

.global __llgo_wasm_context_rewind
.hidden __llgo_wasm_context_rewind
.type __llgo_wasm_context_rewind,@function
__llgo_wasm_context_rewind:
    .functype __llgo_wasm_context_rewind (i32) -> ()
    global.get __stack_pointer
    local.get 0
    i32.load 16
    global.set __stack_pointer
    local.get 0
    i32.load 4
    local.get 0
    i32.load 0
    i32.const 0
    i32.const 1
    i32.store8 __llgo_wasm_context_rewinding
    local.get 0
    i32.const 8
    i32.add
    call start_rewind
    call_indirect (i32) -> (i32)
    drop
    call stop_unwind
    global.set __stack_pointer
    return
    end_function

.hidden __llgo_wasm_context_rewinding
.type __llgo_wasm_context_rewinding,@object
.section .bss.__llgo_wasm_context_rewinding,"",@
.globl __llgo_wasm_context_rewinding
__llgo_wasm_context_rewinding:
    .int8 0
    .size __llgo_wasm_context_rewinding, 1

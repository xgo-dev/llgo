/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Use of this source code is governed by the Apache License, Version 2.0.
 */

/*
 * LLGo lowers aggregate WebAssembly returns to an explicit leading sret
 * pointer. A capturing closure then receives its environment before the
 * source-level argument. Keep this physical ABI detail at the runtime edge;
 * ordinary finalizer calls remain Go func-value calls.
 */
#include <stdint.h>

typedef void (*llgo_wasm_sret_func)(void *, void *);
typedef void (*llgo_wasm_sret_closure)(void *, void *, void *);
typedef struct {
    uint64_t type_or_itab;
    uint64_t data;
} llgo_wasm_interface;
typedef void (*llgo_wasm_interface_sret_func)(void *, llgo_wasm_interface);
typedef void (*llgo_wasm_interface_sret_closure)(void *, void *,
                                                 llgo_wasm_interface);

void llgo_wasm_call_sret(void *function, void *result, void *environment,
                         void *argument) {
    if (environment != 0) {
        ((llgo_wasm_sret_closure)function)(result, environment, argument);
    } else {
        ((llgo_wasm_sret_func)function)(result, argument);
    }
}

void llgo_wasm_call_interface_sret(void *function, void *result,
                                   void *environment, void *type_or_itab,
                                   void *data) {
    llgo_wasm_interface argument = {(uint64_t)(uintptr_t)type_or_itab,
                                    (uint64_t)(uintptr_t)data};
    if (environment != 0) {
        ((llgo_wasm_interface_sret_closure)function)(result, environment,
                                                     argument);
    } else {
        ((llgo_wasm_interface_sret_func)function)(result, argument);
    }
}

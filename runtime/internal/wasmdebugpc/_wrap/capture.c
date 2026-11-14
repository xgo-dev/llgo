// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

#include <emscripten.h>
#include <stdint.h>

// Error.stack is a qualified Chrome transport, not a portable Wasm ABI.
// Unknown formats, anonymous modules, user formatters and truncation all leave
// the optional snapshot unavailable. Never infer module identity from a name.
EM_JS(uint32_t, llgo_wasm_debug_capture_pcs_js, (uint64_t *output, uint32_t capacity), {
    try {
        // Pthreads instantiate an already-compiled module and leave
        // wasmBinaryFile unset. Use this generated host's own URL resolver;
        // do not rerun a user-supplied locateFile callback during suspension.
        const moduleURL = wasmBinaryFile || (!Module['locateFile'] && findWasmBinary());
        if (!capacity || capacity > 128 || typeof moduleURL !== 'string' ||
            !(moduleURL.startsWith('http:') || moduleURL.startsWith('https:')) ||
            'prepareStackTrace' in Error) return 0;
        const descriptor = Object.getOwnPropertyDescriptor(Error, 'stackTraceLimit');
        const limit = capacity * 2 + 32;
        let stack;
        try {
            Object.defineProperty(Error, 'stackTraceLimit', {value: limit, configurable: true});
            stack = new Error().stack;
        } finally {
            if (descriptor) Object.defineProperty(Error, 'stackTraceLimit', descriptor);
            else delete Error.stackTraceLimit;
        }
        if (typeof stack !== 'string') return 0;
        const lines = stack.split(String.fromCharCode(10));
        if (lines.length >= limit + 1) return 0;
        const pcs = [], marker = ':wasm-function[';
        for (const raw of lines) {
            const line = raw.trim(), at = line.lastIndexOf(marker);
            if (at < 0) continue;
            if (!line.slice(0, at).endsWith('(' + moduleURL) ||
                !line.endsWith(')') || pcs.length === capacity) return 0;
            const tail = line.slice(at + marker.length, -1), separator = tail.indexOf(']:');
            const index = tail.slice(0, separator), pc = tail.slice(separator + 2);
            if (separator < 1 || !/^[0-9]+$/.test(index) || !/^0x[0-9a-f]+$/.test(pc)) return 0;
            const offset = Number(pc);
            if (!Number.isSafeInteger(offset) || offset <= 0) return 0;
            pcs.push(offset);
        }
        const view = new DataView(HEAPU8.buffer), address = Number(output);
        for (let i = 0; i < pcs.length; ++i) view.setBigUint64(address + i * 8, BigInt(pcs[i]), true);
        return pcs.length;
    } catch (_) {
        // Debugger metadata must never change the application's control flow.
        return 0;
    }
});

uint32_t llgo_wasm_debug_capture_pcs(uint64_t *output, uint32_t capacity) {
    return llgo_wasm_debug_capture_pcs_js(output, capacity);
}

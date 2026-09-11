// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.
//
// Value encoding and reference accounting follow Go's lib/wasm/wasm_exec.js.
// Only the call frame and scheduler entry are specific to LLGo/Emscripten.

#include <stdint.h>
#include <stddef.h>
#include <emscripten.h>

EM_JS_DEPS(llgo_js_host, "$getWasmTableEntry,$Asyncify,$Fibers,$ExitStatus");

EM_JS(void, llgo_js_random_data, (uint8_t *data, size_t length), {
    globalThis.crypto.getRandomValues(HEAPU8.subarray(Number(data), Number(data) + Number(length)));
});

EM_JS(void, llgo_js_host, (int32_t op, uint64_t *frame), {
    const address = Number(frame);
    let state = Module['llgoGoJS'];
    if (!state) {
        state = {
            _pendingEvent: null,
            pending: [],
            decoder: new TextDecoder('utf-8'),
            encoder: new TextEncoder(),
        };
        state._values = [NaN, 0, null, true, false, globalThis, state];
        state._goRefCounts = new Array(7).fill(Infinity);
        state._ids = new Map([[0, 1], [null, 2], [true, 3], [false, 4], [globalThis, 5], [state, 6]]);
        state._idPool = [];
        state._makeFuncWrapper = function(id) {
            return function() {
                const event = { id: id, this: this, args: arguments };
                if (Asyncify.exportCallStack.length) {
                    state._pendingEvent = event;
                    // Delimit the reentrant Go segment. Fiber suspension must
                    // stop here, leaving the caller's JS frames alive; replay
                    // of Reflect.apply would repeat arbitrary JS side effects.
                    // The runtime event stack drains runnable Gs and polls Go
                    // timers synchronously, so the nested trampoline returns
                    // only after this handler finishes on its original G.
                    const callStack = Asyncify.exportCallStack;
                    const trampolineRunning = Fibers.trampolineRunning;
                    Asyncify.exportCallStack = [];
                    Fibers.trampolineRunning = false;
                    try {
                        state.handler();
                    } finally {
                        Asyncify.exportCallStack = callStack;
                        Fibers.trampolineRunning = trampolineRunning;
                    }
                } else {
                    state.pending.push(event);
                    HEAPU32[state.pendingFlag >>> 2] = 1;
                    const wait = Module['llgoWasmHostWait'];
                    if (wait && wait.wake) {
                        const wake = wait.wake;
                        delete wait.wake;
                        // No compiled frames are active here. Resume the
                        // scheduler before returning the callback result.
                        wake();
                    }
                }
                return event.result;
            };
        };
        Module['llgoGoJS'] = state;
    }
    // Reacquire the memory view after every operation that can call Go: a
    // callback may grow memory and detach the previous ArrayBuffer.
    const memory = () => new DataView(HEAPU8.buffer);
    const integer = (slot) => Number(memory().getBigUint64(address + slot * 8, true));
    const signed = (slot) => Number(memory().getBigInt64(address + slot * 8, true));
    // Match wasm_exec.js setInt64, including NaN-to-zero conversion for an
    // object's missing/non-numeric length property.
    const putInteger = (slot, value) => {
        const view = memory();
        view.setUint32(address + slot * 8, value, true);
        view.setUint32(address + slot * 8 + 4, Math.floor(value / 4294967296), true);
    };
    const load = (ptr) => {
        const view = memory();
        const f = view.getFloat64(ptr, true);
        if (f === 0) return undefined;
        if (!isNaN(f)) return f;
        return state._values[view.getUint32(ptr, true)];
    };
    const store = (ptr, value) => {
        const view = memory();
        const nanHead = 0x7FF80000;
        if (typeof value === 'number' && value !== 0) {
            if (isNaN(value)) {
                view.setUint32(ptr, 0, true);
                view.setUint32(ptr + 4, nanHead, true);
            } else {
                view.setFloat64(ptr, value, true);
            }
            return;
        }
        if (value === undefined) {
            view.setFloat64(ptr, 0, true);
            return;
        }
        let id = state._ids.get(value);
        if (id === undefined) {
            id = state._idPool.pop();
            if (id === undefined) id = state._values.length;
            state._values[id] = value;
            state._goRefCounts[id] = 0;
            state._ids.set(value, id);
        }
        state._goRefCounts[id]++;
        let flag = 0;
        switch (typeof value) {
            case 'object': if (value !== null) flag = 1; break;
            case 'string': flag = 2; break;
            case 'symbol': flag = 3; break;
            case 'function': flag = 4; break;
        }
        view.setUint32(ptr, id, true);
        view.setUint32(ptr + 4, nanHead | flag, true);
    };
    const text = () => state.decoder.decode(HEAPU8.subarray(integer(2), integer(2) + integer(3)));
    const bytes = () => HEAPU8.subarray(integer(2), integer(2) + integer(3));
    const args = () => {
        const values = new Array(integer(5));
        const ptr = integer(4);
        for (let i = 0; i < values.length; i++) values[i] = load(ptr + i * 8);
        return values;
    };
    switch (op) {
        case 0: {
            const id = memory().getUint32(address, true);
            state._goRefCounts[id]--;
            if (state._goRefCounts[id] === 0) {
                const value = state._values[id];
                state._values[id] = null;
                state._ids.delete(value);
                state._idPool.push(id);
            }
            break;
        }
        case 1: store(address, text()); break;
        case 2: store(address, Reflect.get(load(address), text())); break;
        case 3: Reflect.set(load(address), text(), load(address + 8)); break;
        case 4: Reflect.deleteProperty(load(address), text()); break;
        case 5: store(address, Reflect.get(load(address), signed(1))); break;
        case 6: Reflect.set(load(address), signed(1), load(address + 16)); break;
        case 7: putInteger(0, parseInt(load(address).length)); break;
        case 8:
        case 9:
        case 10: {
            try {
                const value = load(address);
                let result;
                if (op === 8) result = Reflect.apply(Reflect.get(value, text()), value, args());
                if (op === 9) result = Reflect.apply(value, undefined, args());
                if (op === 10) result = Reflect.construct(value, args());
                store(address, result);
                putInteger(1, 1);
            } catch (error) {
                // Emscripten exit/suspension and native Wasm exceptions carry
                // runtime control flow, not syscall/js errors.
                if (error === 'unwind' || error instanceof ExitStatus ||
                    (typeof WebAssembly.Exception === 'function' && error instanceof WebAssembly.Exception)) {
                    throw error;
                }
                store(address, error);
                putInteger(1, 0);
            }
            break;
        }
        case 11: {
            const value = state.encoder.encode(String(load(address)));
            store(address, value);
            putInteger(1, value.length);
            break;
        }
        case 12: bytes().set(load(address)); break;
        case 13: putInteger(0, load(address) instanceof load(address + 8) ? 1 : 0); break;
        case 14:
        case 15: {
            const value = load(address);
            if (!(value instanceof Uint8Array || value instanceof Uint8ClampedArray)) {
                putInteger(0, 0);
                putInteger(1, 0);
                break;
            }
            const buffer = bytes();
            const n = Math.min(value.length, buffer.length);
            if (op === 14) buffer.set(value.subarray(0, n));
            else value.set(buffer.subarray(0, n));
            putInteger(0, n);
            putInteger(1, 1);
            break;
        }
        case 16:
            // A raw table entry has no Asyncify unwind/rewind boundary. This
            // wrapper gives callbacks their own saved continuation entry.
            state.handler = Asyncify.instrumentFunction(getWasmTableEntry(integer(0)));
            state.pendingFlag = integer(1);
            break;
        case 17:
            state._pendingEvent = state.pending.shift() || null;
            HEAPU32[state.pendingFlag >>> 2] = state.pending.length ? 1 : 0;
            break;
        default: throw new Error('unknown LLGo syscall/js host operation: ' + op);
    }
});

void llgo_js_install(void (*handler)(void), uint32_t *pending) {
    uint64_t frame[6] = {(uintptr_t)handler, (uintptr_t)pending};
    llgo_js_host(16, frame);
}

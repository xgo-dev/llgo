#include <emscripten.h>
#include <stdint.h>
#include <stdlib.h>

// Return to Go immediately. The Go timer/scheduler owns suspension; a raw
// Asyncify Promise wait inside a Go fiber corrupts multi-worker rewind state.
EM_JS(void, llgo_pcln_load_start_js, (void *request), {
    (async () => {
    let timer, controller, pointer;
    const address = Number(request), size = address + 4, out = address + 8;
    try {
        const moduleURL = wasmBinaryFile || findWasmBinary();
        if (typeof moduleURL !== 'string') return;
        const resolved = new URL(moduleURL, import.meta.url);
        resolved.pathname += '.pclntab';
        const url = resolved.href, limit = 64 * 1024 * 1024;
        let bytes;
        if (resolved.protocol === 'file:') {
            const fs = await import('node:fs/promises');
            const {constants} = await import('node:fs');
            const fd = await fs.open(new URL(url), constants.O_RDONLY | constants.O_NONBLOCK);
            try {
                const st = await fd.stat();
                if (!st.isFile() || st.size < 256 || st.size > limit) return;
                bytes = new Uint8Array(st.size);
                let offset = 0;
                while (offset < bytes.length) {
                    const result = await fd.read(bytes, offset, bytes.length-offset, offset);
                    if (!result.bytesRead) return;
                    offset += result.bytesRead;
                }
            } finally { await fd.close(); }
        } else {
            controller = new AbortController();
            timer = setTimeout(() => controller.abort(), 5000);
            const response = await fetch(url, {signal: controller.signal});
            if (!response.ok || !response.body || Number(response.headers.get('Content-Length')) > limit) return;
            const reader = response.body.getReader(), chunks = [];
            let length = 0;
            for (;;) {
                const chunk = await reader.read();
                if (chunk.done) break;
                length += chunk.value.length;
                if (length > limit) { await reader.cancel(); return; }
                chunks.push(chunk.value);
            }
            bytes = new Uint8Array(length);
            let offset = 0;
            for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length; }
        }
        if (!bytes.length || bytes.length > limit) return;
        pointer = _malloc(typeof request === 'bigint' ? BigInt(bytes.length) : bytes.length);
        if (!pointer) return;
        HEAPU8.set(bytes, Number(pointer));
        HEAPU32[size >>> 2] = bytes.length;
        if (typeof request === 'bigint') new DataView(HEAPU8.buffer).setBigUint64(out, BigInt(pointer), true);
        else HEAPU32[out >>> 2] = Number(pointer);
        pointer = 0;
    } catch (_) {
        return;
    } finally {
        if (timer) clearTimeout(timer);
        if (controller) controller.abort();
        if (pointer) _free(pointer);
        if (typeof SharedArrayBuffer !== 'undefined' && HEAP32.buffer instanceof SharedArrayBuffer)
            Atomics.store(HEAP32, address >>> 2, 1);
        else HEAPU32[address >>> 2] = 1;
    }
    })();
});

void llgo_pcln_load_start(void *request) {
    llgo_pcln_load_start_js(request);
}

uint32_t llgo_pcln_pointer_size(void) { return sizeof(void *); }

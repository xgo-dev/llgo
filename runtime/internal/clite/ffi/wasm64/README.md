# Vendored wasm64 libffi

Prebuilt static archive for `-target emscripten-memory64` (Emscripten
Memory64 / LP64). WASI keeps the C stub in `_wrap/libffi_unavailable.c`.

- Upstream: [libffi](https://sourceware.org/libffi/) 3.8.0 (`include/ffi.h`)
- Target: `wasm64-unknown-emscripten` with `-sMEMORY64=1`
- Archive SHA-256: `a35a5f687ae4fc865550f50ffa40c2df4a3aaed2beb09839a9e59737e3a801f1`
- License: MIT, as in `include/ffi.h`

Rebuild from `~/goplus/libffi` with the Emscripten SDK. `CFLAGS` must include
`-sMEMORY64=1` **before** configure's `sizeof(size_t)` probe, otherwise the
headers record 32-bit pointers:

```sh
emconfigure ./configure --host=wasm64-unknown-emscripten \
  --enable-static --disable-shared --disable-docs \
  --disable-multi-os-directory --disable-raw-api \
  CFLAGS="-O2 -fPIC -sMEMORY64=1" LDFLAGS="-sMEMORY64=1"
emmake make
cp include/ffi.h include/ffitarget.h "$LLGO_ROOT/runtime/internal/clite/ffi/wasm64/include/"
cp .libs/libffi.a "$LLGO_ROOT/runtime/internal/clite/ffi/wasm64/libffi.a"
```

Change the generated `ffi.h` include of `ffitarget.h` from `<>` to quotes so the wrap
can compile without an extra `-I`.

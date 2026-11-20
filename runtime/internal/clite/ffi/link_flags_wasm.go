//go:build wasm && !js

package ffi

// libffi's WebAssembly backend currently supports the Emscripten ABI but not
// the WASI C ABI. Keep the low-level boundary linkable for WASI test binaries
// that do not use dynamic reflection; unsupported calls fail explicitly in the
// stub instead of accidentally linking the host machine's libffi archive.
const LLGoFiles = "_wrap/libffi_unavailable.c"

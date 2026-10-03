# SIMD behavior tests

Add executable SIMD regressions as ordinary `Test*` functions in this package.
Use `goexperiment.simd` and architecture build tags to select applicable APIs.
Shared behavior tests run under official Go and LLGo. LLGo-specific behavior
uses an additional `llgo` build tag; LLVM IR/instruction assertions belong in
compiler tests instead.

From the repository root:

```sh
GOEXPERIMENT=simd go test -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O0 -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -count=1 ./test/simd/...

GOEXPERIMENT=simd GOOS=wasip1 GOARCH=wasm go test -exec=wasmtime -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -target wasi -c -o simd.test.wasm ./test/simd/...
EMCC_CFLAGS='-fwasm-exceptions -sSUPPORT_LONGJMP=wasm -sWASM_LEGACY_EXCEPTIONS=1' \
  GOEXPERIMENT=simd llgo test -O2 -target emscripten -emulator -count=1 -timeout=2m ./test/simd/...
```

CI runs the native suite on amd64/arm64 at O0 and O2. The wasm test-command job
runs the official Go WASI oracle with Wasmtime, compiles LLGo's WASI test binary,
and executes the complete LLGo suite at O2 through Emscripten and Node (J32),
with both cold and cached packages. That execution requires Emscripten, Node,
and LLGo's supported Binaryen (`WASMOPT`).

The Node test explicitly selects native Wasm EH. Emscripten's default JS SjLj
wrappers cannot carry `v128` values across panic/recover boundaries, so the
default J32 configuration does not yet support every SIMD test. This native EH
configuration validates SIMD together with goroutines, bounds panics, deferred
calls, and unsupported-intrinsic panics; it does not change the default EH mode
or qualify all Emscripten/Asyncify combinations. `EMCC_CFLAGS` participates in
LLGo's package cache key so the two EH modes cannot reuse each other's objects.

[WAMR 2.4.5](https://github.com/bytecodealliance/wasm-micro-runtime/blob/WAMR-2.4.5/doc/build_wamr.md#enable-128-bit-simd-feature)
supports SIMD in AOT, JIT, and fast-interpreter modes, while its exception
handling requires the classic interpreter. LLGo's WASI SIMD check therefore
validates compilation only until a compatible runtime is available.

The shared suite covers implemented SIMD128 operations.
`unimplemented_llgo_test.go` checks that remaining intrinsic declarations panic
with their symbol name, including indirect, deferred, and linkname calls.
SIMD reflection is outside this stage's scope, matching the Go 1.27 support
boundary. As operations are implemented, move their behavior cases into the
shared suite and replace the corresponding fallback assertions.

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
GOEXPERIMENT=simd EM_LLVM_ROOT="$(llvm-config --bindir)" \
  EMCC_CFLAGS="-fwasm-exceptions -sSUPPORT_LONGJMP=wasm" LLGO_BUILD_CACHE=off \
  llgo test -O2 -target emscripten -emulator -count=1 -timeout=2m ./test/simd/...
```

WebAssembly execution requires Wasmtime for Go and Emscripten with Node for LLGo.
LLGo uses Emscripten because WAMR's classic interpreter supports the legacy EH
required by its WASI pthread runtime but cannot execute SIMD instructions.
Native Wasm exceptions keep vector arguments out of JavaScript invoke wrappers;
LLVM 22 consumes LLGo's IR for Wasm SjLj lowering. Disable LLGo's package cache
when changing these external compiler settings.
CI runs the native suite on amd64/arm64 and the WebAssembly suite in the existing
wasm test-command job. Native LLGo runs at O0 and O2; WebAssembly LLGo runs at O2.
The shared suite covers implemented SIMD128 operations.
`unimplemented_llgo_test.go` checks that remaining intrinsic declarations panic
with their symbol name, including indirect, deferred, and linkname calls.
SIMD reflection is outside this stage's scope, matching the Go 1.27 support
boundary. As operations are implemented, move their behavior cases into the
shared suite and replace the corresponding fallback assertions.

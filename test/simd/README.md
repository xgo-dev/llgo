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
GOEXPERIMENT=simd llgo test -O2 -target wasi -emulator -count=1 -timeout=2m ./test/simd/...
```

WASI execution requires Wasmtime and LLGo's supported Binaryen (`WASMOPT`).
CI runs the native suite on amd64/arm64 and the WASI suite in the existing wasm
test-command job. Native LLGo runs at O0 and O2; WASI runs at O2 because the unoptimized
standard testing framework exceeds Wasmtime's local-variable limit. The shared suite covers implemented SIMD128 operations, array/slice memory access,
and broadcast, including element-aligned addresses, short slices, and nil arrays.
`unimplemented_llgo_test.go` checks that remaining intrinsic declarations panic
with their symbol name, including indirect, deferred, and linkname calls.
SIMD reflection is outside this stage's scope, matching the Go 1.27 support
boundary. As operations are implemented, move their behavior cases into the
shared suite and replace the corresponding fallback assertions.

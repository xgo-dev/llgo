# Portable SIMD compatibility tests

These tests come from Go 1.27.0 `src/simd`, including the runnable
`testdata/{iface,sizeof,compiles,tostring}_test.go` fixtures. The original BSD
license is in `LICENSE`. The tests contain 27 Test functions and 6 examples.

Local adaptations are limited to Go/experiment/architecture build constraints
and fixture import paths under `internal`. Assertions and test vectors are
preserved, including the inner-product tests in emulated, 128-bit and 256-bit
modes.

The upstream subprocess driver `testdata_test.go` is intentionally not copied:
it runs `go test`, which would test the Go compiler even from an LLGo binary.
Its runnable cases execute directly here with each compiler. Its negative
compiler-diagnostic fixture `errors_test.go` is not part of this runtime suite.

Run with a Go 1.27 toolchain:

```sh
GOEXPERIMENT=simd go test ./test/std/simd
GOEXPERIMENT=simd llgo test -O2 ./test/std/simd
GODEBUG=simd=0 GOEXPERIMENT=simd llgo test -O2 ./test/std/simd
```

`dev/test_native_simd.sh` runs this package alongside the portable regressions,
including CPU overrides, emulation, 128/256-bit selection, and LTO profiles.
The WebAssembly SIMD CI step and `dev/test_wasm_simd.sh` also include it.
The separate [upstream archsimd gate](../../../dev/simd/README.md) exercises the
qualified instruction tests directly from GOROOT without copying generated code.

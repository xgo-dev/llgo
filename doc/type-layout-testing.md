# Testing type storage layouts

`unsafe.Sizeof`, `Alignof`, and `Offsetof` may already be constants when gossa
reaches LLGo. Layout regressions must also inspect the emitted LLVM types.

The compiler tests cover three layers:

- `ssa/TestGoBasicTypeLLVMStorage`: 19 Go basic types and aliases on 21 targets,
  including hosted wasm32/wasm64. Check emitted global types, LLVM allocation
  sizes, two-element arrays, ABI alignment, and string field offsets against
  explicit expectations. `go/types.Sizes` supplies frontend configuration only.
  Extended backends absent from the official Windows LLVM build explicitly skip;
  core hosted architectures remain mandatory.
- `internal/build/TestCgoTypesLLVMLayout`: run real `go tool cgo -godefs` for
  native target ABIs, lower the resulting Go declarations, and compare their
  LLVM types, sizes, and record offsets with separately compiled C. The native
  `TestCgoPackageLLVMLayout` additionally checks the complete cgo/package-loading/
  gossa path. Linux tests GCC preprocessing against LLGo's Clang; a negative
  `-fshort-enums` case proves that incompatible compiler options are detected.
- `internal/build/TestCTypeLLVMLayout` and `TestRuntimeCTypeLLVMLayout`: compare
  C globals and arrays with C `sizeof`/`_Alignof` constants, then compare the
  runtime's selected C aliases with that oracle using native C storage. The
  freestanding fixture needs target support in Clang, without target SDKs.

Run with the normal LLVM development dependencies installed:

```sh
go test ./ssa -run '^TestGoBasicTypeLLVMStorage$' -count=1
go test ./internal/build -run '^(TestCTypeLLVMLayout|TestCgoTypesLLVMLayout|TestCgoPackageLLVMLayout|TestCgoCompilerLayoutMismatch|TestCgoGCCAndLLGoClangLayout|TestRuntimeCTypeLLVMLayout)$' -count=1
```

These tests belong to packages already run by compiler CI. They run on the host
and inspect cross-target IR; they do not imply execution on every target.
`cmd/cgo` has no wasm pointer-size entry, so its wasm cases explicitly skip.
The C oracle, runtime aliases, and Go storage tests still cover wasm profiles.
Go wasm32 storage uses an eight-byte language word while native C pointers and
`size_t` use four bytes. Raw LLVM scalar ABI alignment also does not replace
Go's field-alignment rules, particularly on Windows i686.

# Upstream archsimd compatibility gate

`bash dev/test_upstream_archsimd.sh` builds and runs selected Go 1.27.0
`GOROOT/src/simd/archsimd/internal/simd_test` cases with both Go and LLGo O2.
The generated upstream tests, helpers and assertions remain unchanged in GOROOT.
`LLGO` selects the compiler and `LLGO_SIMD_ARTIFACT_DIR` retains the test binary
and the expected/actual case lists. The native SIMD CI step invokes this gate.
A removed or renamed selected case fails the list comparison instead of silently
reducing coverage. An amd64 host without AVX reports the upstream package's skip.

The AMD64 gate uses GOAMD64=v1 and the caller's effective CPU features.
Go 1.27.0 provides 184 AMD64 instruction tests. On an AVX2-only host some
tests skip or omit AVX512-only assertions. Run the complete instruction
profile on Linux/amd64 with Intel SDE:

```sh
LLGO=/path/to/llgo SDE=/path/to/sde64 bash dev/test_simd_amd64_sde.sh
```

The Sapphire Rapids profile exercises all 184 cases and the portable 512-bit
tests. The ARM64 list contains 87 cases. The compiler declaration tests
independently check the bodyless intrinsics for AMD64, ARM64 and Wasm.

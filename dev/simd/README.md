# Upstream archsimd compatibility gate

`bash dev/test_upstream_archsimd.sh` builds and runs selected Go 1.27.0
`GOROOT/src/simd/archsimd/internal/simd_test` cases with both Go and LLGo O2.
The generated upstream tests, helpers and assertions remain unchanged in GOROOT.
`LLGO` selects the compiler and `LLGO_SIMD_ARTIFACT_DIR` retains the test binary
and the expected/actual case lists. The native SIMD CI step invokes this gate.
A removed or renamed selected case fails the list comparison instead of silently
reducing coverage. An amd64 host without AVX reports the upstream package's skip.

The amd64 profile uses GOAMD64=v1 and `GODEBUG=cpu.avx512f=off` for **both**
compilers: it qualifies 128/256-bit instructions, not SIMD512. Several upstream
cases mix these widths in one function, so their 512-bit branches remain gated.
The runtime derives its aggregate AVX512 flag after this feature override.

Initial per-test audit (Go 1.27.0, each test in a separate process):

| Profile | Passed and enabled | Failed | Hardware-skipped | Empty hardware-gated bodies |
| --- | ---: | ---: | ---: | ---: |
| macOS ARM64, native | 72 | 15 | 0 | 0 |
| Linux AMD64, Docker emulation on ARM64 | 125 | 35 | 20 | 4 |

These are top-level test outcomes, not counts of intrinsic implementations.
A test can contain hardware-gated subcases. `TestFMA`, `TestTruncate`,
`TestSaturate`, and `TestOnesCountAMD64` return PASS with no assertions when
their AVX512 feature gates are false; they are excluded from the allowlist
and counted separately from executed tests. Docker evidence does not qualify
native x86 execution; the native CI jobs run the same selected tests.
The allowlists contain only tests that actually passed; hardware skips are not
promoted to passing coverage. Failure and skip inventories below are retained
so later instruction work can expand the lists deliberately.

The interleave/concatenation follow-up enables six ARM64 cases and three AMD64
cases (`TestInterleaveLo`, `TestInterleaveHi`, `TestFlattenedTranspose`). The
initial totals above remain the historical audit, not a recount after these
additions.

The conversion/resize follow-up enables `TestConvertARM64` and
`TestTruncateARM64`, plus the AMD64 ceil/floor/trunc/round, conversion,
extension and concatenating-saturation cases listed in the allowlist. The
initial audit totals above are historical. AVX512-only tests with empty gated
bodies remain excluded; LLGo's additional scalar-reference cases exercise
128/256-bit narrowing without requiring AVX512 hardware.

The pair-shuffle follow-up enables thirteen more AMD64 cases: 128/256-bit
selection from two vectors, 128-bit-half selection, and the four 4x4/8x8
transpose cases. `concatSelectedConstant`/`concatSelectedConstantGrouped`
and `ConcatPermute128Scalars` cover both constant and variable selectors.
The scalar-permute follow-up enables the six `TestPermuteScalars*` cases for
128/256-bit dword and high/low word shuffles. Dynamic selectors use fixed
shuffles and vector masks, retaining the untouched word half in each 128-bit
group. The current allowlists contain 156 AMD64 cases and 80 ARM64 cases; these
are enabled top-level tests, not counts of implemented intrinsics.

## arm64 failures

| Test | First reported unsupported intrinsic |
| --- | --- |
| `TestReduceSum` | `simd/archsimd.Int8x16.reduceSum` |
| `TestReduceMax` | `simd/archsimd.Int8x16.reduceMax` |
| `TestReduceMin` | `simd/archsimd.Int8x16.reduceMin` |
| `TestShiftSaturated` | `simd/archsimd.Int8x16.ShiftSaturated` |
| `TestConcatShiftBytesRight` | `simd/archsimd.Uint8x16.ConcatShiftBytesRight` |
| `TestMulAdd` | `simd/archsimd.Int8x16.MulAdd` |
| `TestOnesCount` | `simd/archsimd.Int8x16.OnesCount` |
## amd64 failures

Horizontal pairwise addition and subtraction are now covered for the applicable
128/256-bit shapes by `test/simd`; `TestAddSubPairs` is enabled in the amd64
allowlist. The initial audit totals above are retained as historical results;
the table below lists the remaining failures after subsequent fixes.

| Test | First reported unsupported intrinsic |
| --- | --- |
| `TestIsNaN` | `simd/archsimd.Float32x4.IsNaN` |
| `TestConcatShiftBytesRight` | `simd/archsimd.Uint8x16.ConcatShiftBytesRight` |
| `TestClearAVXUpperBits` | `simd/archsimd.ClearAVXUpperBits` |
| `TestIsZero` | `simd/archsimd.Uint64x2.IsZero` |

Hardware-skipped: `TestShiftAllConcat`, `TestShiftConcat`, `TestPermute`, `TestConcatPermute`, `TestCompress`, `TestExpand`, `TestBitMaskFromBits`, `TestBitMaskFromBitsLoad`, `TestBitMaskToBitsStore`, `TestMergeFloat512`, `TestIfElseFloat512`, `TestMaskOpt512`, `TestLeadingZeros`, `TestConcatPermuteScalarsConstGroupedUint32x16`, `TestSelect2FromPairConstGroupedInt512`, `TestSelectTernOptInt32x16`, `TestMaskOr`, `TestPartInt64`, `TestOnesCount`, `TestCeilScaledResidue`.

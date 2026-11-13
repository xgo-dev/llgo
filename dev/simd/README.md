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

## arm64 failures

| Test | First reported unsupported intrinsic |
| --- | --- |
| `TestConcatEven` | `simd/archsimd.Int8x16.ConcatEven` |
| `TestConcatOdd` | `simd/archsimd.Int8x16.ConcatOdd` |
| `TestInterleaveEven` | `simd/archsimd.Int8x16.InterleaveEven` |
| `TestInterleaveOdd` | `simd/archsimd.Int8x16.InterleaveOdd` |
| `TestInterleaveLoARM64` | `simd/archsimd.Int8x16.InterleaveLo` |
| `TestInterleaveHiARM64` | `simd/archsimd.Int8x16.InterleaveHi` |
| `TestConvertARM64` | `simd/archsimd.Float32x4.ConvertLo2ToFloat64` |
| `TestTruncateARM64` | `simd/archsimd.Int16x8.TruncToInt8` |
| `TestReduceSum` | `simd/archsimd.Int8x16.reduceSum` |
| `TestReduceMax` | `simd/archsimd.Int8x16.reduceMax` |
| `TestReduceMin` | `simd/archsimd.Int8x16.reduceMin` |
| `TestShiftSaturated` | `simd/archsimd.Int8x16.ShiftSaturated` |
| `TestConcatShiftBytesRight` | `simd/archsimd.Uint8x16.ConcatShiftBytesRight` |
| `TestMulAdd` | `simd/archsimd.Int8x16.MulAdd` |
| `TestOnesCount` | `simd/archsimd.Int8x16.OnesCount` |
## amd64 failures

| Test | First reported unsupported intrinsic |
| --- | --- |
| `TestInterleaveLo` | `simd/archsimd.Int16x8.InterleaveLo` |
| `TestInterleaveHi` | `simd/archsimd.Int16x8.InterleaveHi` |
| `TestIsNaN` | `simd/archsimd.Float32x4.IsNaN` |
| `TestConcatShiftBytesRight` | `simd/archsimd.Uint8x16.ConcatShiftBytesRight` |
| `TestFlattenedTranspose` | `simd/archsimd.Int32x4.InterleaveLo` |
| `TestClearAVXUpperBits` | `simd/archsimd.ClearAVXUpperBits` |
| `TestIsZero` | `simd/archsimd.Uint64x2.IsZero` |
| `TestSelect4FromPairConst` | `simd/archsimd.Int32x4.concatSelectedConstant` |
| `TestSelect4FromPairVar` | `simd/archsimd.Int32x4.concatSelectedConstant` |
| `TestSelect4FromPairConstGrouped` | `simd/archsimd.Float32x8.concatSelectedConstantGrouped` |
| `TestConcatPermute128Scalars` | `simd/archsimd.Uint64x4.ConcatPermute128Scalars` |
| `TestConcatPermute128ScalarsVar` | `simd/archsimd.Uint64x4.ConcatPermute128Scalars` |
| `TestSelect2FromPairConst` | `simd/archsimd.Uint64x2.concatSelectedConstant` |
| `TestSelect2FromPairConstGroupedUint` | `simd/archsimd.Uint64x4.concatSelectedConstantGrouped` |
| `TestSelect2FromPairConstGroupedFloat` | `simd/archsimd.Float64x4.concatSelectedConstantGrouped` |
| `TestSelect2FromPairConstGroupedInt` | `simd/archsimd.Int64x4.concatSelectedConstantGrouped` |
| `TestPermuteScalars` | `simd/archsimd.Int32x4.permuteScalars` |
| `TestPermuteScalarsGrouped` | `simd/archsimd.Int32x8.permuteScalarsGrouped` |
| `TestPermuteScalarsHi` | `simd/archsimd.Int16x8.permuteScalarsHi` |
| `TestPermuteScalarsLo` | `simd/archsimd.Int16x8.permuteScalarsLo` |
| `TestPermuteScalarsHiGrouped` | `simd/archsimd.Int16x16.permuteScalarsHiGrouped` |
| `TestPermuteScalarsLoGrouped` | `simd/archsimd.Int16x16.permuteScalarsLoGrouped` |
| `TestAddSubPairs` | `simd/archsimd.Int16x8.ConcatAddPairs` |
| `TestSaturateConcat` | `simd/archsimd.Int32x8.SaturateToInt16ConcatGrouped` |
| `TestTranspose4` | `simd/archsimd.Int32x4.InterleaveLo` |
| `TestTranspose8` | `simd/archsimd.Int32x8.InterleaveLoGrouped` |
| `TestTransposeTiled4` | `simd/archsimd.Int32x4.InterleaveLo` |
| `TestTransposeTiled8` | `simd/archsimd.Int32x8.InterleaveLoGrouped` |
| `TestCeilAMD64` | `simd/archsimd.Float32x8.Ceil` |
| `TestFloorAMD64` | `simd/archsimd.Float32x8.Floor` |
| `TestTruncAMD64` | `simd/archsimd.Float32x8.Trunc` |
| `TestRoundAMD64` | `simd/archsimd.Float32x8.Round` |
| `TestConvert` | `simd/archsimd.Float64x4.ConvertToFloat32` |
| `TestExtend` | `simd/archsimd.Int8x16.ExtendToInt16` |
| `TestExtendLo` | `simd/archsimd.Int8x16.ExtendLo4ToInt64` |

Hardware-skipped: `TestShiftAllConcat`, `TestShiftConcat`, `TestPermute`, `TestConcatPermute`, `TestCompress`, `TestExpand`, `TestBitMaskFromBits`, `TestBitMaskFromBitsLoad`, `TestBitMaskToBitsStore`, `TestMergeFloat512`, `TestIfElseFloat512`, `TestMaskOpt512`, `TestLeadingZeros`, `TestConcatPermuteScalarsConstGroupedUint32x16`, `TestSelect2FromPairConstGroupedInt512`, `TestSelectTernOptInt32x16`, `TestMaskOr`, `TestPartInt64`, `TestOnesCount`, `TestCeilScaledResidue`.

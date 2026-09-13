# SIMD P0 findings

The design is tracked in [proposal #2568](https://github.com/xgo-dev/llgo/issues/2568).
This groundwork establishes configuration, fixed-width type/API identity, and
reproducible compiler probes. It does not enable SIMD operations.

The source baseline is official Go `2ee6421c51553e7164590445f96b123a858c1f4a`.
The inventory stores package source hashes separately from the reference revision.
The initial probe run used Go 1.27.0 and LLVM 22 on macOS arm64; all three target
native Go compilation controls succeeded, and all three LLGo modules verified.
These controls do not establish LLGo runtime execution or CPU guard safety.

| Probe | amd64 | arm64 | wasm32 |
| --- | --- | --- | --- |
| Float32x4 Go storage size/alignment | 16/8 | 16/8 | **16/4; required 16/8** |
| Struct byte/vector/byte offsets | 0,8,24 | 0,8,24 | **0,4,20; required 0,8,24** |
| Representative Add | unresolved method call | unresolved method call | unresolved method call |
| Natural vector arithmetic | absent | absent | absent |

The ordinary `[4]float32` control has alignment 4. Separately constructed LLVM
vectors have alignment 16/32/64 depending on width. Neither the array alignment
nor natural vector alignment supplies the required Go SIMD storage contract.
On wasm32, the discrepancy already appears in frontend sizes, before LLVM:
`effectiveTypeSizes` uses a 32-bit `StdSizes`, which lacks Go's SIMD alignment rule.
Both source sizes and LLVM storage must be corrected together.

Current ABI transforms treat these values as aggregates: amd64 Vec128 splits
into two small floating vector carriers while Vec256/512 use memory; arm64
Vec128 remains a struct; wasm32 uses memory. These are observations, not a
requirement to copy the native Go ABI or proof of SIMD ABI support.

Package initialization extends the minimum operation set. On arm64/wasm,
`archsimd.init -> new64x2 -> Uint64x2.SetElem` initializes the emulated carryless
multiply masks. An actual CPU-query-only LLGo executable failed to link on
Darwin/arm64 because `SetElem` was unresolved. Native Go returned PMULL=true
at global/init/main, and false at all three points with `GODEBUG=cpu.pmull=off`.
There is no LLGo CPU-query execution result yet.

The current LLGo startup sequence in `internal/build/main_module.go` does not
call `internal/cpu.Initialize`; the alternate runtime replaces the official
scheduler bootstrap. Future CPU initialization must precede ordinary package
initialization and cover executable, C archive/shared, and wasm scheduler entry
paths. Preserve the official detection -> GODEBUG options -> derived-feature
order. AMD64 assembly feature-baseline propagation and ARM64/Linux HWCAP input
also need verification before enabling feature dispatch.

Reproduce with the [source inventory](../internal/simd/README.md) and
[compiler/CPU probes](../internal/build/testdata/simd-probe/README.md).
The compiler probe deliberately fails for unmet requirements; it is opt-in and
is not counted as a passing SIMD support test. It records LLVM IR before/after
ABI conversion and only a bounded initialization-call inspection, not a complete
reachable operation closure.

Next implementation gates are matching Go/LLVM storage, the package-init
operation closure, minimal Vec128 operations, safe calls and CPU preconditions.
Portable API/dependency inventory, full FMV, Midway, and wider execution remain
later work under the proposal's stage gates.

## 中文

本次基础建立了实验配置、三端固定宽度类型/API 清单及可复现探针，尚未启用 SIMD。
实测 wasm32 的向量对齐为 4、嵌套字段偏移为 4，均应为 8；三端 Add 仍是未实现的
方法调用。arm64/wasm 包初始化还依赖 `Uint64x2.SetElem`，CPU 查询程序因而尚不能链接。
原生 Go 的 PMULL 初始化及 GODEBUG 对照已运行，不能据此推断 LLGo 的运行结果。
后续首先修复 Go/LLVM 存储合同，并覆盖包初始化依赖和 Vec128 最小操作；CPU 初始化
必须早于普通包初始化。探针的预期失败明确作为未满足的实现要求，不计入支持率。

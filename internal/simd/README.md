# SIMD foundation

The bilingual design and staged acceptance plan are tracked in
[LLGo proposal #2568](https://github.com/xgo-dev/llgo/issues/2568).

This package provides part of P0: recognition of Go 1.27 fixed-width SIMD
representations on amd64, arm64, and wasm. `Classifier` describes lane shape,
canonical vector/mask identity, and Go storage size/alignment. It does not
select an LLVM computation type or ABI carrier.

The caller must verify the selected toolchain's standard-library package
provenance, target, and experiment state before constructing a classifier.
Package path strings alone are insufficient. Defined types retain physical
representation, but their original mask/vector declaration cannot always be
recovered from `go/types`; these receive `Kind == Derived`.

`internal/build` resolves the selected Go toolchain's effective experiments and
tool tags once per invocation. Package loading, source overlays, child commands,
and cache fingerprints use that configuration. The source Go version is recorded
separately from the Go version that built LLGo.

Run `go test ./internal/simd` to check the actual toolchain's generated type
declarations for all three architectures, plus aliases, defined types, false
matches, and Go storage layout. These are type-model tests; they do not establish
LLVM layout, vector instructions, ABI, or runtime support. The classifier is not
yet connected to the compiler. The source inventory and opt-in compiler probes below establish the next
implementation requirements. They do not enable SIMD execution.

## 中文

完整中英文设计及分阶段验收见 [提案 #2568](https://github.com/xgo-dev/llgo/issues/2568)。
本包交付 P0 的类型基础：识别 Go 1.27 三端固定宽度 SIMD 类型，描述 lane、
规范 vector/mask 身份与 Go 存储布局。调用者必须先验证所选标准库的来源、目标及实验配置。
新定义类型保留物理表示，但 `go/types` 不一定保留原始 mask/vector 声明，因此标为 `Derived`。

构建层已统一有效实验配置、工具标签、源码选择、子命令和缓存身份。
当前测试覆盖三端真实类型声明及类型模型，分类器尚未接入编译器，不能据此认定
LLVM 布局、指令、ABI 或运行时支持已完成。下述源码清单及编译器探针记录后续实现要求。

## Source inventory

Generate the complete fixed-width API inventory and verify the reviewed source:

```sh
go run ./internal/simd/cmd/simdinventory -out /tmp/simd-inventory.json
go run ./internal/simd/cmd/simdinventory \
  -check-summary internal/simd/testdata/go1.27-inventory-summary.json
```

Use `-goroot /path/to/go` to select another Go 1.27 source tree. `-check` compares
against a complete earlier inventory and reports added/removed keys, signatures,
source classifications, and source hashes separately. `-summary -out <file>`
regenerates the compact baseline after source review. The reference revision is
recorded separately from the selected GOROOT's actual content hashes.

The pinned inventory has 2353/619/492 public declarations on amd64/arm64/wasm,
including types, variables, constants, methods, and functions. These are not
counts of implemented operations. Go bodies can depend on missing intrinsics;
bodyless declarations are candidates, and the inventory carries explicit
unimplemented/unverified statuses. Portable `simd` and its specialization
closure require a separate inventory before Midway implementation.

See [compiler probes](../build/testdata/simd-probe/README.md) for reproducible
layout, ABI, and initialization observations, and [P0 findings](../../doc/simd-p0.md)
for the resulting implementation requirements.

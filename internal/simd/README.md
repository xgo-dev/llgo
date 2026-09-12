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
yet connected to the compiler. Remaining P0 work includes the operation inventory
and compiler layout, ABI, and CPU-initialization probes.

## 中文

完整中英文设计及分阶段验收见 [提案 #2568](https://github.com/xgo-dev/llgo/issues/2568)。
本包交付 P0 的类型基础：识别 Go 1.27 三端固定宽度 SIMD 类型，描述 lane、
规范 vector/mask 身份与 Go 存储布局。调用者必须先验证所选标准库的来源、目标及实验配置。
新定义类型保留物理表示，但 `go/types` 不一定保留原始 mask/vector 声明，因此标为 `Derived`。

构建层已统一有效实验配置、工具标签、源码选择、子命令和缓存身份。
当前测试覆盖三端真实类型声明及类型模型，分类器尚未接入编译器，不能据此认定
LLVM 布局、指令、ABI 或运行时支持已完成。P0 后续工作包括操作清单和编译器布局、
ABI、CPU 初始化探针。

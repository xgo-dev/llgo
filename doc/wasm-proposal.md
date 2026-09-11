# LLGo WebAssembly Proposal

Status: draft. Tracking issue: [xgo-dev/llgo#2152](https://github.com/xgo-dev/llgo/issues/2152).

R1 through R3, including R2.1, are merged and remain the delivered foundation of this proposal. W1-W3 build on that foundation to complete the supported WebAssembly profiles before advanced engine features are added.

## Delivered milestones: R1-R3 (merged)

| Milestone | Delivered scope | Merged PRs |
| --- | --- | --- |
| R1: single-worker runtime | Emscripten Fiber and WASI Asyncify scheduling; blocking channels/select/sync, timers and Sleep, host wakeups, and representative public `llgo test` execution. | [#2472](https://github.com/xgo-dev/llgo/pull/2472) |
| R2: linear-memory GC | Non-moving collection, compiler and suspended-goroutine roots, cooperative safepoints, and root-chain recovery across panic/recover and Asyncify replay. | [#2487](https://github.com/xgo-dev/llgo/pull/2487) |
| R2.1: object lifecycles | Finalizers, cleanup callbacks, weak-reference expiration, callback ordering and cancellation, and executable lifecycle regressions. | [#2492](https://github.com/xgo-dev/llgo/pull/2492) |
| R3: toolchain workflows | Compile-only test artifacts and named-target routing; explicit ownership of the Asyncify/optimization pipeline; executable scheduler, GC, lifecycle, callback, and test-command integration checks. | [#2511](https://github.com/xgo-dev/llgo/pull/2511), [#2488](https://github.com/xgo-dev/llgo/pull/2488) |

These merged milestones establish the runtime/toolchain baseline. The revised Go data model, independent host providers, standard-library completeness, and full compatibility and size/performance acceptance remain the W1-W3 work defined below.

## Profile model

A hosted profile is the product of a memory ABI and a host ABI. Source compatibility, C interoperability, runtime providers, and optional engine features are capabilities of that profile, not additional profiles.

| Profile | Memory ABI | Host ABI | W1-W3 scope |
| --- | --- | --- | --- |
| J32 | Memory32 | JavaScript | required |
| J64 | Memory64 | JavaScript | required |
| W32 | Memory32 | WASI Preview 1 | required |
| W64 | Memory64 | WASI | deferred |

Memory32 uses Core Wasm `i32` addresses and the ILP32 C ABI. Memory64 uses `i64` addresses and the LP64 C ABI. Go's language model is not a separate profile: `int`, `uint`, `uintptr`, and pointer storage follow the official Go WebAssembly 64-bit word model; a valid Memory32 address has zero high bits. C types use `github.com/goplus/lib/c`, and every Go/C or host boundary performs the required checked width conversion. Go and C `int` do not need to have the same width.

The supported public entries are:

| Entry | Profile | Initial provider |
| --- | --- | --- |
| `GOOS=js GOARCH=wasm` | J32 | Go-compatible JavaScript host shim |
| `llgo build -target emscripten` | J32 | Emscripten |
| `llgo build -target emscripten-memory64` | J64 | Emscripten Memory64 |
| `GOOS=wasip1 GOARCH=wasm` or `llgo build -target wasi` | W32 | WASI Preview 1, optionally with wasi-libc |

`-target wasm` remains an alias of `emscripten`, and `-target wasip1` remains an alias of `wasi`. Existing target JSON names and build tags such as `llgo.wasm.emscripten` remain stable; J32/J64/W32 are profile metadata, not replacements for those identifiers. The unsupported `-target wasm-unknown` and `-target wasip2` definitions are removed. WASI Preview 2 requires a separate Component Model proposal.

J32 has two provider acceptance paths: the Go-compatible JavaScript shim and Emscripten. They share Go source semantics but not necessarily the same import names, glue, or artifact packaging. J64 initially uses Emscripten. W32 runs on a WASI runtime and may link C through wasi-libc. Provider and capability selection is part of the build and cache identity.

## Compatibility contract

- J32 and W32 implement the official Go WebAssembly sizes, alignment, standard build constraints, runtime behavior, and applicable standard-library surface. Binary compatibility with Go compiler object files is not required.
- The J32 Go provider reuses the selected GOROOT's `syscall/js` source and does not expose emval as its Go-facing API. Its initial host adapter may reuse Emscripten glue, filesystem services, and the WebAssembly libffi backend; this is source/API compatibility, not stock `wasm_exec.js` binary compatibility.
- J32 and J64 expose `syscall/js` through their selected JavaScript provider. C code remains available through the explicit C ABI; arbitrary Emscripten-dependent libraries require the Emscripten provider.
- W32 uses the Go WASI Preview 1 host contract and can use wasi-libc without becoming a separate C profile.
- Host adapters own imports, startup, filesystems, callbacks, timers, process exit, and artifact sidecars. Runtime scheduling, GC, panic/defer/recover, reflection semantics, and caller metadata remain provider-independent where possible.

## Reflection and foreign calls

`reflect.Value.Call`, `CallSlice`, methods, and `reflect.MakeFunc` are required capabilities, not profile definitions. J32/GoJS, J32/Emscripten, and J64/Emscripten use the WebAssembly libffi backend from [#2549](https://github.com/xgo-dev/llgo/pull/2549), which provides generic dynamic calls without generating a bridge for every function signature. W32/WASI has no JavaScript table adapter, so it uses compact compiler-generated typed bridges deduplicated by lowered signature and emitted only when whole-program reachability finds a dynamic reflection call. The selected backend must preserve GC roots, suspension, panic/recover, closures, aggregate ABI lowering, and deterministic errors. Typed-bridge size and compile-time cost must remain confined to WASI and are measured in acceptance.

LLGo's Core Wasm C ABI is unrelated to the WIT Canonical ABI. Future WASI Preview 2 support will add generated WIT lift/lower adapters outside the ordinary Go and C calling conventions; it is not part of W1-W3.

## Acceptance

A feature is implemented only when CI executes it on every applicable path. The minimum hosted matrix contains four paths: J32/GoJS, J32/Emscripten, J64/Emscripten Memory64, and W32/WASI. Tests use real Node, browser, and Wasmtime execution where applicable and cover `llgo build/run/test`, artifacts, host callbacks and exit, C boundaries, reflection, GC and suspension, `test/**`, `test/std`, and every applicable GOROOT case. The complete GOROOT corpus runs on the canonical Go-compatible J32/GoJS path, while all four paths run GOROOT sentinels and the complete applicable repository package suite; this avoids multiplying more than two thousand compiler conformance cases by provider paths whose differences are covered by target integration tests. Only reviewed `xfail` and `notapplicable` classifications may be excluded. Each W-stage PR carries focused executable CI; compile-only coverage does not count. Final acceptance also checks compiler coverage, `cprintf`/`println`/`fmtprintf` and reflection size, runtime benchmarks, native and embedded regressions, and removal of diagnostic or superseded changes.

## Remaining PR plan: W1-W3

### W1: profiles and ABI foundation

Replace the five-profile split with J32/J64/W32 while preserving stable target identifiers and the `wasm`/`wasip1` compatibility aliases, remove unsupported target definitions, implement the official Go 64-bit word model over Memory32, define checked Go/C/host boundary conversion, and include profile/provider identity in compilation and caches. Add ABI, target, cache, C-boundary, and basic execution CI for all four acceptance paths.

### W2: host, reflection, and standard-library completeness

Complete the GoJS, Emscripten, and WASI providers; consolidate output/FS behavior with [#2539](https://github.com/xgo-dev/llgo/pull/2539); preserve synchronous nested callbacks, external events, memory growth, and exit status; select the measured reflection backend; and close applicable standard-library gaps. Run Node, browser, Wasmtime, reflection, GC/suspension, and focused `test/std` CI in this PR.

### W3: full compatibility acceptance and consolidation

Run and classify the full applicable `test/**` suite on all four paths, the complete applicable GOROOT corpus on J32/GoJS, and GOROOT sentinels on every path; retire unnecessary skips, finish issues exposed by those tests, audit the accumulated diff, split out independently useful fixes, and enforce coverage, size, performance, native, and embedded gates. W3 completes the currently supported WebAssembly scope.

## Deferred work

W64, WASI Preview 2 and WIT components, threads and Atomics, WasmGC, Exception Handling, JSPI and Stack Switching, multiple workers, and parallel goroutines are later capability work. Unsupported environments continue to use the single-worker scheduler and LLGo linear-memory GC.

---

# LLGo WebAssembly 提案

状态：草案。跟踪 issue：[xgo-dev/llgo#2152](https://github.com/xgo-dev/llgo/issues/2152)。

R1 至 R3（包括 R2.1）已经合并，作为本提案已交付的基础继续保留。W1-W3 在此基础上完成当前支持的 WebAssembly profile，再推进高级引擎特性。

## 已交付里程碑：R1-R3（已合并）

| 阶段 | 已交付范围 | 已合并 PR |
| --- | --- | --- |
| R1：单 worker runtime | Emscripten Fiber 与 WASI Asyncify 调度；channel/select/sync 阻塞、timer 与 Sleep、host 唤醒，以及代表性 `llgo test` 执行。 | [#2472](https://github.com/xgo-dev/llgo/pull/2472) |
| R2：线性内存 GC | 非移动回收、编译器及挂起 goroutine 的根、协作 safepoint，以及 panic/recover 和 Asyncify replay 过程中的根链恢复。 | [#2487](https://github.com/xgo-dev/llgo/pull/2487) |
| R2.1：对象生命周期 | Finalizer、cleanup callback、弱引用失效、回调顺序与取消，以及实际执行的生命周期回归测试。 | [#2492](https://github.com/xgo-dev/llgo/pull/2492) |
| R3：工具链流程 | 编译后不执行的测试产物与 named-target 路由；明确 Asyncify/优化流水线的处理归属；实际执行调度、GC、生命周期、callback 和测试命令集成检查。 | [#2511](https://github.com/xgo-dev/llgo/pull/2511), [#2488](https://github.com/xgo-dev/llgo/pull/2488) |

这些已合并阶段构成 runtime/工具链基础。新的 Go 数据模型、独立 host provider、标准库完整性，以及完整兼容验收和体积/性能验收，仍由下述 W1-W3 完成。

## Profile 模型

Hosted profile 是 Memory ABI 与 Host ABI 的组合。源码兼容、C 互操作、runtime provider 和可选引擎特性都是 profile 的能力，不再拆成额外 profile。

| Profile | Memory ABI | Host ABI | W1-W3 范围 |
| --- | --- | --- | --- |
| J32 | Memory32 | JavaScript | 必须完成 |
| J64 | Memory64 | JavaScript | 必须完成 |
| W32 | Memory32 | WASI Preview 1 | 必须完成 |
| W64 | Memory64 | WASI | 延期 |

Memory32 使用 Core Wasm `i32` 地址和 ILP32 C ABI；Memory64 使用 `i64` 地址和 LP64 C ABI。Go 语言模型不是另一个 profile：`int`、`uint`、`uintptr` 和指针存储遵循官方 Go WebAssembly 的 64 位 word 模型，合法 Memory32 地址的高位为零。C 类型通过 `github.com/goplus/lib/c` 表达，Go/C 或 host 边界执行必要的带检查宽度转换；Go 与 C 的 `int` 不需要同宽。

保留的公共入口如下：

| 入口 | Profile | 初始 provider |
| --- | --- | --- |
| `GOOS=js GOARCH=wasm` | J32 | Go 兼容 JavaScript host shim |
| `llgo build -target emscripten` | J32 | Emscripten |
| `llgo build -target emscripten-memory64` | J64 | Emscripten Memory64 |
| `GOOS=wasip1 GOARCH=wasm` 或 `llgo build -target wasi` | W32 | WASI Preview 1，可选 wasi-libc |

`-target wasm` 保留为 `emscripten` 的 alias，`-target wasip1` 保留为 `wasi` 的 alias。现有 target JSON 名称以及 `llgo.wasm.emscripten` 等 build tag 保持稳定；J32/J64/W32 是 profile 元数据，不替换这些标识。删除当前不支持的 `-target wasm-unknown` 和 `-target wasip2` 定义。WASI Preview 2 需要单独的 Component Model 提案。

J32 有两条 provider 验收路径：Go 兼容 JavaScript shim 和 Emscripten。二者共享 Go 源码语义，但 import 名称、glue 和产物打包不要求相同。J64 初期使用 Emscripten；W32 运行在 WASI runtime 上，并可通过 wasi-libc 链接 C。Provider 与 capability 选择必须进入构建和缓存标识。

## 兼容约定

- J32 和 W32 实现官方 Go WebAssembly 的尺寸、对齐、标准 build constraints、runtime 行为和适用标准库，不要求与 Go 编译器 object 文件二进制兼容。
- J32 Go provider 复用所选 GOROOT 的 `syscall/js` 源码，并且不把 emval 暴露为 Go 侧 API。初始 host adapter 可以复用 Emscripten glue、文件系统服务和 WebAssembly libffi 后端；这里保证的是源码/API 兼容，而不是 stock `wasm_exec.js` 二进制兼容。
- J32 和 J64 通过所选 JavaScript provider 提供 `syscall/js`。C 代码通过显式 C ABI 使用；依赖 Emscripten runtime 的任意 C 库仍要求 Emscripten provider。
- W32 使用 Go WASI Preview 1 host contract，并可使用 wasi-libc，不再因此拆出单独 C profile。
- Host adapter 负责 imports、启动、文件系统、回调、定时器、进程退出和产物 sidecar。调度、GC、panic/defer/recover、反射语义和 caller metadata 在可行范围内保持 provider 无关。

## 反射与外部调用

`reflect.Value.Call`、`CallSlice`、方法和 `reflect.MakeFunc` 是必须能力，不是 profile 定义。J32/GoJS、J32/Emscripten 和 J64/Emscripten 使用 [#2549](https://github.com/xgo-dev/llgo/pull/2549) 的 WebAssembly libffi 后端，以通用动态调用避免为每个函数签名生成 bridge。W32/WASI 没有 JavaScript table adapter，因此使用按 lowering 后签名去重的 compact typed bridge，并且只在 whole-program 可达性分析发现动态反射调用时生成。最终后端必须正确处理 GC root、挂起、panic/recover、闭包、聚合 ABI lowering 和确定性错误；typed bridge 的体积与编译时间开销必须严格限制在 WASI，并在验收中测量。

LLGo Core Wasm C ABI 与 WIT Canonical ABI 无关。未来 WASI Preview 2 将在普通 Go/C 调用约定之外生成 WIT lift/lower adapter，不属于 W1-W3。

## 验收

功能只有在 CI 对所有适用路径实际执行后才算完成。最小 hosted 矩阵包含四条路径：J32/GoJS、J32/Emscripten、J64/Emscripten Memory64、W32/WASI。测试按适用范围在真实 Node、浏览器和 Wasmtime 中运行，覆盖 `llgo build/run/test`、产物、host 回调与退出、C 边界、反射、GC 与挂起、`test/**`、`test/std` 以及全部适用 GOROOT case。完整 GOROOT corpus 在规范性的 Go 兼容 J32/GoJS 路径运行，四条路径都运行 GOROOT sentinel 和完整的适用仓库 package suite；这样无需把两千多个编译器一致性 case 机械乘以 host provider，而 provider 差异由 target 集成测试覆盖。只允许排除经过审查的 `xfail` 和 `notapplicable`。每个 W 阶段 PR 自带聚焦的可执行 CI，compile-only 不算覆盖。最终验收还检查编译器覆盖率、`cprintf`/`println`/`fmtprintf` 与反射体积、runtime benchmark、native/embedded 回归，以及诊断和被取代变更的清理。

## 后续 PR 规划：W1-W3

### W1：Profile 与 ABI 基础

把五 profile 模型收敛为 J32/J64/W32，同时保留稳定 target 标识及 `wasm`/`wasip1` 兼容 alias，删除不支持的 target 定义，实现 Memory32 上的官方 Go 64 位 word 模型，定义带检查的 Go/C/host 边界转换，并把 profile/provider 纳入编译与缓存标识。本 PR 为四条验收路径加入 ABI、target、cache、C 边界与基础执行 CI。

### W2：Host、反射与标准库完整性

完成 GoJS、Emscripten 和 WASI provider；结合 [#2539](https://github.com/xgo-dev/llgo/pull/2539) 统一输出与 FS；保证同步嵌套回调、外部事件、内存增长和退出状态；根据测量选择反射后端；补齐适用标准库缺口。本 PR 运行 Node、浏览器、Wasmtime、反射、GC/挂起和聚焦的 `test/std` CI。

### W3：完整兼容验收与收敛

在四条路径上运行并分类全部适用 `test/**`，在 J32/GoJS 上运行完整适用 GOROOT corpus，并在每条路径运行 GOROOT sentinel；清理不必要的 skip，修复测试暴露的问题，审计累计 diff，把可独立复用的修复拆出，并执行覆盖率、体积、性能、native 和 embedded gate。W3 完成当前支持的 WebAssembly 范围。

## 延期范围

W64、WASI Preview 2/WIT component、threads/Atomics、WasmGC、Exception Handling、JSPI/Stack Switching、多 worker 和并行 goroutine 都是后续 capability 工作。不支持这些能力的环境继续使用单 worker 调度和 LLGo 线性内存 GC。

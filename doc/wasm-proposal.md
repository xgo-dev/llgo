# LLGo WebAssembly Proposal

Status: active. Profile and source-compatibility work is tracked in [xgo-dev/llgo#2152](https://github.com/xgo-dev/llgo/issues/2152); the supported execution and toolchain decisions are tracked in [#2632](https://github.com/xgo-dev/llgo/issues/2632).

The supported build uses the LLGo-patched Binaryen release. Browser execution keeps Emscripten Fiber/Asyncify and advances to a bounded Web Worker pool. W32 advances to WASI threads on WAMR; single-thread WASI is retired only after threaded GC passes. These execution milestones proceed alongside the profile/ABI, host-provider, and compatibility-acceptance work.

R1 through R3, including R2.1, are merged and remain the delivered foundation of this proposal. The remaining implementation work builds on that foundation to complete the supported WebAssembly profiles before advanced engine features are added.

## Design overview

- Go source layer: all three profiles use the official Go WebAssembly 64-bit word model. Go `int`, `uint`, `uintptr`, and pointer storage are 64-bit; valid Memory32 addresses use only the low 32 bits. Most pure Go standard-library code can therefore be shared, and implementations should prefer the same selected GOROOT sources.

- Memory ABI:

  - J32 (wasm32 with JavaScript) and W32 (wasm32 with WASI Preview 1): Core Wasm Memory32, `i32` addresses, and the ILP32 C ABI.
  - J64 (wasm64 with JavaScript): Memory64, `i64` addresses, and the LP64 C ABI, extending the same Go data model to a wider address space.
  - Differences belong in address lowering, Go/C boundaries, libffi, memory accesses, and host glue; ordinary Go source does not require separate forks for these differences.
  - C types use `github.com/goplus/lib/c`; Go and C `int` need not have the same width. The compiler and boundary adapters perform the required checked width conversions.

- Host ABI:

  - JavaScript: `syscall/js`, event-loop integration, browser/Node services, and JavaScript callbacks.
  - WASI: imports for filesystem access, clocks, randomness, arguments, exit, and other WASI services.
  - Standard-library host files selected by `js && wasm` and `wasip1 && wasm` therefore remain different.

- Provider:

  - J32-GoJS (wasm32 with the Go-compatible JavaScript provider) and J32-Emscripten (wasm32 with the Emscripten JavaScript provider) share Go API semantics, while import names, JavaScript glue, filesystem implementations, and artifact packaging may differ.
  - Both are providers of the same J32 (wasm32 with JavaScript) profile and share its Go data model; provider selection does not define another Go ABI.

```text
Shared Go sources and Go data model
│
├── Memory32 ── ILP32 C boundary
│      ├── JavaScript / GoJS provider
│      ├── JavaScript / Emscripten provider
│      └── WASI Preview 1 provider
│
└── Memory64 ── LP64 C boundary
       └── JavaScript / Emscripten provider
```

Assembly/runtime boundary: reuse applicable host-independent Go wasm assembly through Plan 9 translation to LLVM IR and target lowering. Code tied to the gc compiler's calling convention, stack layout, GC, scheduler, or runtime entry points requires LLGo adaptation. Source/API compatibility does not require identical generated function signatures or binary compatibility with gc compiler objects or artifacts for other targets.

## Delivered milestones: R1-R3 (merged)

| Milestone | Delivered scope | Merged PRs |
| --- | --- | --- |
| R1: single-worker runtime | Emscripten Fiber and WASI Asyncify scheduling; blocking channels/select/sync, timers and Sleep, host wakeups, and representative public `llgo test` execution. | [#2472](https://github.com/xgo-dev/llgo/pull/2472) |
| R2: linear-memory GC | Non-moving collection, compiler and suspended-goroutine roots, cooperative safepoints, and root-chain recovery across panic/recover and Asyncify replay. | [#2487](https://github.com/xgo-dev/llgo/pull/2487) |
| R2.1: object lifecycles | Finalizers, cleanup callbacks, weak-reference expiration, callback ordering and cancellation, and executable lifecycle regressions. | [#2492](https://github.com/xgo-dev/llgo/pull/2492) |
| R3: toolchain workflows | Compile-only test artifacts and named-target routing; explicit ownership of the Asyncify/optimization pipeline; executable scheduler, GC, lifecycle, callback, and test-command integration checks. | [#2511](https://github.com/xgo-dev/llgo/pull/2511), [#2488](https://github.com/xgo-dev/llgo/pull/2488) |

These merged milestones establish the runtime/toolchain baseline. The revised Go data model, independent host providers, standard-library completeness, and full compatibility and size/performance acceptance remain the implementation work defined below.

## Profile model

A hosted profile is the product of a memory ABI and a host ABI. Source compatibility, C interoperability, runtime providers, and optional engine features are capabilities of that profile, not additional profiles.

| Profile | Memory ABI | Host ABI | Current scope |
| --- | --- | --- | --- |
| J32 | Memory32 | JavaScript | required |
| J64 | Memory64 | JavaScript | required |
| W32 | Memory32 | WASI Preview 1 | required |
| W64 | Memory64 | WASI | deferred |

The supported public entries are:

| Entry | Profile | Initial provider |
| --- | --- | --- |
| `GOOS=js GOARCH=wasm` | J32 | Go-compatible JavaScript host shim |
| `llgo build -target emscripten` | J32 | Emscripten |
| `llgo build -target emscripten-memory64` | J64 | Emscripten Memory64 |
| `GOOS=wasip1 GOARCH=wasm` or `llgo build -target wasi` | W32 | WASI Preview 1, optionally with wasi-libc |

`-target wasm` remains an alias of `emscripten`, and `-target wasip1` remains an alias of `wasi`. Existing target JSON names and build tags such as `llgo.wasm.emscripten` remain stable; J32/J64/W32 are profile metadata, not replacements for those identifiers. The unsupported `-target wasm-unknown` and `-target wasip2` definitions are removed. WASI Preview 2 requires a separate Component Model proposal.

These four entries are acceptance paths for three profiles. Memory ABI, host/provider, and capability selection are part of the build and cache identity.

## Compatibility contract

- All supported profiles, including J64 (wasm64 with JavaScript), follow the shared Go sizes, alignment, and source-reuse rules above, standard build constraints, and the applicable Go runtime behavior and standard-library surface for their selected host.
- The J32 Go provider reuses the selected GOROOT's `syscall/js` source and does not expose emval as its Go-facing API. Its initial host adapter may reuse Emscripten glue, filesystem services, and the WebAssembly libffi backend; this is source/API compatibility, not stock `wasm_exec.js` binary compatibility.
- J32 and J64 expose `syscall/js` through their selected JavaScript provider. C code remains available through the explicit C ABI; arbitrary Emscripten-dependent libraries require the Emscripten provider.
- W32 uses the Go WASI Preview 1 host contract and can use wasi-libc without becoming a separate C profile.
- Host adapters own imports, startup, filesystems, callbacks, timers, process exit, and artifact sidecars. Runtime scheduling, GC, panic/defer/recover, reflection semantics, and caller metadata remain provider-independent where possible.

The opt-in bounded Emscripten worker mode keeps each goroutine on one physical worker. Emscripten `syscall/js` handles are JavaScript-realm-local, so a goroutine that uses them keeps ordinary descendants on that worker. `github.com/xgo-dev/llgo/runtime/wasmworkers.GoIndependent` starts work in the pool only when its closure carries no JS value or thread-local C state. Passing a `js.Value` between unrelated workers is not yet supported; resolving that boundary is required before treating multi-worker mode as the default Go-compatible JavaScript provider.

## Reflection and foreign calls

`reflect.Value.Call`, `CallSlice`, methods, and `reflect.MakeFunc` are required capabilities, not profile definitions. J32/GoJS, J32/Emscripten, and J64/Emscripten use the WebAssembly libffi backend from [#2549](https://github.com/xgo-dev/llgo/pull/2549), which provides generic dynamic calls without generating a bridge for every function signature. W32/WASI has no JavaScript table adapter, so it uses compact compiler-generated typed bridges deduplicated by lowered signature and emitted only when whole-program reachability finds a dynamic reflection call. The selected backend must preserve GC roots, suspension, panic/recover, closures, aggregate ABI lowering, and deterministic errors. Typed-bridge size and compile-time cost must remain confined to WASI and are measured in acceptance.

LLGo's Core Wasm C ABI is unrelated to the WIT Canonical ABI. Future WASI Preview 2 support will add generated WIT lift/lower adapters outside the ordinary Go and C calling conventions; it is not part of the current scope.

## Acceptance

A feature is implemented only when CI executes it on every applicable path. The minimum hosted matrix contains four paths: J32/GoJS, J32/Emscripten, J64/Emscripten Memory64, and W32/WASI. Tests use real Node, browser, and Wasmtime execution where applicable and cover `llgo build/run/test`, artifacts, host callbacks and exit, C boundaries, reflection, GC and suspension, `test/**`, `test/std`, and every applicable GOROOT case. The complete GOROOT corpus runs on the canonical Go-compatible J32/GoJS path, while all four paths run GOROOT sentinels and the complete applicable repository package suite; this avoids multiplying more than two thousand compiler conformance cases by provider paths whose differences are covered by target integration tests. Only reviewed `xfail` and `notapplicable` classifications may be excluded. Each implementation PR carries focused executable CI; compile-only coverage does not count. Final acceptance also checks compiler coverage, `cprintf`/`println`/`fmtprintf` and reflection size, runtime benchmarks, native and embedded regressions, and removal of diagnostic or superseded changes.

## Remaining implementation work

### Profiles and ABI foundation

Replace the five-profile split with J32/J64/W32 while preserving stable target identifiers and the `wasm`/`wasip1` compatibility aliases, remove unsupported target definitions, implement the official Go 64-bit word model over Memory32, define checked Go/C/host boundary conversion, and include profile/provider identity in compilation and caches. Add ABI, target, cache, C-boundary, and basic execution CI for all four acceptance paths.

### Host providers, reflection, and standard-library completeness

Complete the GoJS, Emscripten, and WASI providers; consolidate output/FS behavior with [#2539](https://github.com/xgo-dev/llgo/pull/2539); preserve synchronous nested callbacks, external events, memory growth, and exit status; select the measured reflection backend; and close applicable standard-library gaps. Run Node, browser, Wasmtime, reflection, GC/suspension, and focused `test/std` CI for these paths.

### Full compatibility acceptance and consolidation

Run and classify the full applicable `test/**` suite on all four paths, the complete applicable GOROOT corpus on J32/GoJS, and GOROOT sentinels on every path; retire unnecessary skips, finish issues exposed by those tests, audit the accumulated diff, split out independently useful fixes, and enforce coverage, size, performance, native, and embedded gates. This completes the currently supported WebAssembly scope.

## Deferred work

W64, WASI Preview 2 and WIT components, a WasmGC heap, and JSPI/stackless execution remain later research. EH encoding is selected through the comparison in [#2632](https://github.com/xgo-dev/llgo/issues/2632); existing validated EH behavior remains supported during that work. Browser multi-worker execution and W32 WASI threads are required milestones of #2632, with the single-worker browser mode retained.

---

# LLGo WebAssembly 提案

状态：推进中。Profile 与源码兼容性由 [xgo-dev/llgo#2152](https://github.com/xgo-dev/llgo/issues/2152) 跟踪；正式执行架构与工具链决定由 [#2632](https://github.com/xgo-dev/llgo/issues/2632) 跟踪。

正式构建使用 LLGo 打补丁的 Binaryen 版本。浏览器保留 Emscripten Fiber/Asyncify，并推进到有上限的 Web Worker 池。W32 以 WAMR 上的 WASI threads 为目标；只有在线程 GC 验收通过后才移除单线程 WASI。这些运行时工作与 profile/ABI、host provider 和兼容性验收工作并行推进。

R1 至 R3（包括 R2.1）已经合并，作为本提案已交付的基础继续保留。后续实现工作在此基础上完成当前支持的 WebAssembly profile，再推进高级引擎特性。

## 设计总览

- Go 源码层：三个 profile 都采用官方 Go WebAssembly 的 64 位 word model；Go `int`、`uint`、`uintptr` 和指针存储均为 64 位，合法 Memory32 地址只使用低 32 位。因此绝大多数纯 Go 标准库代码可以共用，实现应优先复用所选 GOROOT 的同一套源码。

- Memory ABI：

  - J32（wasm32 + JavaScript）和 W32（wasm32 + WASI Preview 1）：Core Wasm Memory32，使用 `i32` 地址，C ABI 为 ILP32。
  - J64（wasm64 + JavaScript）：Memory64，使用 `i64` 地址，C ABI 为 LP64，将同一 Go 数据模型扩展到更大的地址空间。
  - 差异集中在地址 lowering、Go/C 边界、libffi、内存访问和 host glue，无须为这些差异分叉普通 Go 源码。
  - C 类型使用 `github.com/goplus/lib/c`；Go 与 C 的 `int` 不必同宽。编译器和边界适配器执行必要的带检查宽度转换。

- Host ABI：

  - JavaScript：`syscall/js`、事件循环接入、浏览器/Node 服务、JS 回调。
  - WASI：文件系统、时钟、随机数、参数、退出等 WASI imports。
  - 因此，由 `js && wasm` 与 `wasip1 && wasm` 选择的标准库 host 文件仍然不同。

- Provider：

  - J32-GoJS（wasm32 + Go 兼容 JavaScript provider）与 J32-Emscripten（wasm32 + Emscripten JavaScript provider）的 Go API 语义相同，但 import 名称、JS glue、FS 实现和产物包装可以不同。
  - 它们属于同一 J32（wasm32 + JavaScript）profile 的两个 provider，共享 Go 数据模型；provider 选择不定义另一套 Go ABI。

```text
共同的 Go 源码与 Go 数据模型
│
├── Memory32 ── ILP32 C boundary
│      ├── JavaScript / GoJS provider
│      ├── JavaScript / Emscripten provider
│      └── WASI Preview 1 provider
│
└── Memory64 ── LP64 C boundary
       └── JavaScript / Emscripten provider
```

汇编/runtime 边界：适用且与 host 无关的 Go wasm 汇编通过 Plan 9 翻译为 LLVM IR，再按目标 lowering 以实现复用。依赖 gc 编译器调用约定、栈布局、GC、调度器或 runtime 入口的代码需要 LLGo 适配。源码/API 兼容不要求生成的函数签名相同，也不要求与 gc 编译器 object 文件或其他目标产物二进制兼容。

## 已交付里程碑：R1-R3（已合并）

| 阶段 | 已交付范围 | 已合并 PR |
| --- | --- | --- |
| R1：单 worker runtime | Emscripten Fiber 与 WASI Asyncify 调度；channel/select/sync 阻塞、timer 与 Sleep、host 唤醒，以及代表性 `llgo test` 执行。 | [#2472](https://github.com/xgo-dev/llgo/pull/2472) |
| R2：线性内存 GC | 非移动回收、编译器及挂起 goroutine 的根、协作 safepoint，以及 panic/recover 和 Asyncify replay 过程中的根链恢复。 | [#2487](https://github.com/xgo-dev/llgo/pull/2487) |
| R2.1：对象生命周期 | Finalizer、cleanup callback、弱引用失效、回调顺序与取消，以及实际执行的生命周期回归测试。 | [#2492](https://github.com/xgo-dev/llgo/pull/2492) |
| R3：工具链流程 | 编译后不执行的测试产物与 named-target 路由；明确 Asyncify/优化流水线的处理归属；实际执行调度、GC、生命周期、callback 和测试命令集成检查。 | [#2511](https://github.com/xgo-dev/llgo/pull/2511), [#2488](https://github.com/xgo-dev/llgo/pull/2488) |

这些已合并阶段构成 runtime/工具链基础。新的 Go 数据模型、独立 host provider、标准库完整性，以及完整兼容验收和体积/性能验收，仍由下述实现工作完成。

## Profile 模型

Hosted profile 是 Memory ABI 与 Host ABI 的组合。源码兼容、C 互操作、runtime provider 和可选引擎特性都是 profile 的能力，不再拆成额外 profile。

| Profile | Memory ABI | Host ABI | 当前范围 |
| --- | --- | --- | --- |
| J32 | Memory32 | JavaScript | 必须完成 |
| J64 | Memory64 | JavaScript | 必须完成 |
| W32 | Memory32 | WASI Preview 1 | 必须完成 |
| W64 | Memory64 | WASI | 延期 |

保留的公共入口如下：

| 入口 | Profile | 初始 provider |
| --- | --- | --- |
| `GOOS=js GOARCH=wasm` | J32 | Go 兼容 JavaScript host shim |
| `llgo build -target emscripten` | J32 | Emscripten |
| `llgo build -target emscripten-memory64` | J64 | Emscripten Memory64 |
| `GOOS=wasip1 GOARCH=wasm` 或 `llgo build -target wasi` | W32 | WASI Preview 1，可选 wasi-libc |

`-target wasm` 保留为 `emscripten` 的 alias，`-target wasip1` 保留为 `wasi` 的 alias。现有 target JSON 名称以及 `llgo.wasm.emscripten` 等 build tag 保持稳定；J32/J64/W32 是 profile 元数据，不替换这些标识。删除当前不支持的 `-target wasm-unknown` 和 `-target wasip2` 定义。WASI Preview 2 需要单独的 Component Model 提案。

这四个入口是三个 profile 的验收路径。Memory ABI、host/provider 和 capability 选择须进入构建和缓存标识。

## 兼容约定

- 包括 J64（wasm64 + JavaScript）在内的所有受支持 profile，都遵循上述共同 Go 尺寸、对齐及源码复用规则、标准 build constraints，以及所选 host 下适用的 Go runtime 行为和标准库。
- J32 Go provider 复用所选 GOROOT 的 `syscall/js` 源码，并且不把 emval 暴露为 Go 侧 API。初始 host adapter 可以复用 Emscripten glue、文件系统服务和 WebAssembly libffi 后端；这里保证的是源码/API 兼容，而不是 stock `wasm_exec.js` 二进制兼容。
- J32 和 J64 通过所选 JavaScript provider 提供 `syscall/js`。C 代码通过显式 C ABI 使用；依赖 Emscripten runtime 的任意 C 库仍要求 Emscripten provider。
- W32 使用 Go WASI Preview 1 host contract，并可使用 wasi-libc，不再因此拆出单独 C profile。
- Host adapter 负责 imports、启动、文件系统、回调、定时器、进程退出和产物 sidecar。调度、GC、panic/defer/recover、反射语义和 caller metadata 在可行范围内保持 provider 无关。

当前有界 Emscripten Worker 模式需显式启用，每个 goroutine 固定在一个物理 Worker。Emscripten 的 `syscall/js` 句柄属于创建它的 JavaScript realm，因此使用这些句柄的 goroutine 会让普通子 goroutine 留在同一 Worker。只有闭包不携带 JS 值或 C 线程局部状态时，才可用 `github.com/xgo-dev/llgo/runtime/wasmworkers.GoIndependent` 将独立任务分配到池中。尚不支持在无亲缘关系的 Worker 间传递 `js.Value`；在把多 Worker 模式作为默认的 Go 兼容 JavaScript provider 前，必须解决这一边界。

## 反射与外部调用

`reflect.Value.Call`、`CallSlice`、方法和 `reflect.MakeFunc` 是必须能力，不是 profile 定义。J32/GoJS、J32/Emscripten 和 J64/Emscripten 使用 [#2549](https://github.com/xgo-dev/llgo/pull/2549) 的 WebAssembly libffi 后端，以通用动态调用避免为每个函数签名生成 bridge。W32/WASI 没有 JavaScript table adapter，因此使用按 lowering 后签名去重的 compact typed bridge，并且只在 whole-program 可达性分析发现动态反射调用时生成。最终后端必须正确处理 GC root、挂起、panic/recover、闭包、聚合 ABI lowering 和确定性错误；typed bridge 的体积与编译时间开销必须严格限制在 WASI，并在验收中测量。

LLGo Core Wasm C ABI 与 WIT Canonical ABI 无关。未来 WASI Preview 2 将在普通 Go/C 调用约定之外生成 WIT lift/lower adapter，不属于当前范围。

## 验收

功能只有在 CI 对所有适用路径实际执行后才算完成。最小 hosted 矩阵包含四条路径：J32/GoJS、J32/Emscripten、J64/Emscripten Memory64、W32/WASI。测试按适用范围在真实 Node、浏览器和 Wasmtime 中运行，覆盖 `llgo build/run/test`、产物、host 回调与退出、C 边界、反射、GC 与挂起、`test/**`、`test/std` 以及全部适用 GOROOT case。完整 GOROOT corpus 在规范性的 Go 兼容 J32/GoJS 路径运行，四条路径都运行 GOROOT sentinel 和完整的适用仓库 package suite；这样无需把两千多个编译器一致性 case 机械乘以 host provider，而 provider 差异由 target 集成测试覆盖。只允许排除经过审查的 `xfail` 和 `notapplicable`。每个实现 PR 自带聚焦的可执行 CI，compile-only 不算覆盖。最终验收还检查编译器覆盖率、`cprintf`/`println`/`fmtprintf` 与反射体积、runtime benchmark、native/embedded 回归，以及诊断和被取代变更的清理。

## 后续实现工作

### Profile 与 ABI 基础

把五 profile 模型收敛为 J32/J64/W32，同时保留稳定 target 标识及 `wasm`/`wasip1` 兼容 alias，删除不支持的 target 定义，实现 Memory32 上的官方 Go 64 位 word 模型，定义带检查的 Go/C/host 边界转换，并把 profile/provider 纳入编译与缓存标识。为四条验收路径加入 ABI、target、cache、C 边界与基础执行 CI。

### Host provider、反射与标准库完整性

完成 GoJS、Emscripten 和 WASI provider；结合 [#2539](https://github.com/xgo-dev/llgo/pull/2539) 统一输出与 FS；保证同步嵌套回调、外部事件、内存增长和退出状态；根据测量选择反射后端；补齐适用标准库缺口。为这些路径运行 Node、浏览器、Wasmtime、反射、GC/挂起和聚焦的 `test/std` CI。

### 完整兼容验收与收敛

在四条路径上运行并分类全部适用 `test/**`，在 J32/GoJS 上运行完整适用 GOROOT corpus，并在每条路径运行 GOROOT sentinel；清理不必要的 skip，修复测试暴露的问题，审计累计 diff，把可独立复用的修复拆出，并执行覆盖率、体积、性能、native 和 embedded gate。此项完成当前支持的 WebAssembly 范围。

## 延期范围

W64、WASI Preview 2/WIT component、WasmGC 堆以及 JSPI/stackless 执行仍属于后续研究。EH 编码按照 [#2632](https://github.com/xgo-dev/llgo/issues/2632) 的对比结果确定；期间保留已验证的 EH 行为。浏览器多 worker 和 W32 WASI threads 是 #2632 的必需里程碑，浏览器仍保留单 worker 模式。

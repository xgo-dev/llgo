# Multi-Module Layout, Middle-End SSAOpt & Incremental Cache Blueprint (Proposal #3215)

This document records the concrete architecture and delivery plan for separating
LLGo's pure-Go library surface—organized 1:1 after `go/*` and
`golang.org/x/tools/go/*`—from the LLVM/CGo compiler CLI, layering a pure-Go
compiler middle-end (`cmd/compiler/ssaopt`) over `llgo/ssa`, and pairing it with
a compact binary package-summary cache and a target-accurate stdlib `go/types`
patch.

## Target Module Graph (`golang.org/x/tools`-Aligned)

```text
github.com/xgo-dev/llgo              (root go.mod — strict Go baseline: go 1.20, zero external deps, zero replace)
├── directive/                       Pure-Go //llgo:*, //go:*, //export, //llgointernal:* parsing & semantics
├── target/                          Pure-Go target triple, arch, and ABI/layout spec model
├── types/                           1:1 go/types + first-class LLGo directives, CallConv, layout, GC, ABI, SIMD [API-guarded]
│   └── typeutil/                    1:1 golang.org/x/tools/go/types/typeutil                                    [API-guarded]
├── ssa/                             1:1 golang.org/x/tools/go/ssa + source-level facts (CallKind, Intrinsic, RecvNilCheck) [API-guarded]
│   └── ssautil/                     1:1 golang.org/x/tools/go/ssa/ssautil
├── callgraph/                       1:1 golang.org/x/tools/go/callgraph                                         [API-guarded]
│   └── rta/                         1:1 golang.org/x/tools/go/callgraph/rta                                     [API-guarded]
├── exportdata/                      1:1 golang.org/x/tools/go/gcexportdata (format v4 with LLGo metadata)       [API-guarded]
└── runtime/go.mod                   (separate module, go 1.20)

github.com/xgo-dev/llgo/cmd          (cmd/go.mod — latest Go toolchain, go 1.27.0)
├── llgo/                            CLI entrypoint (`llgo`, package main)
├── xtool/                           LLVM/C toolchain bindings (package main tools)
├── chore/                           Repository maintenance utilities (package main tools)
├── internal/                        `llgo` CLI subcommands & flag handling (aligned with Go's cmd/go/internal/*)
└── compiler/                        Compiler implementation packages
    ├── build/                       Compilation driver, cache orchestration, linking & packaging
    ├── frontcache/                  Frontend `.llgotypes` & private compiler summary/body cache
    ├── pkgmeta/                     Compact binary per-package summaries (`.llgoex`: escape, inline, call edges)
    ├── goescape/                    Pure-Go escape analysis (decoupled from LLVM)
    ├── ssaopt/                      Pure-Go, zero-LLVM Go-semantic middle-end (`FunctionPlan`, passes & plan verifier)
    ├── cl/                          Go SSA → LLVM IR frontend translator (relocated from `cmd/cl`)
    ├── llvmgen/                     LLVM IR code generator, ABI/GC/DWARF/SIMD lowering
    └── ...                          (packages, typepatch, plan9asm, cabi, crosscompile, env, locality, dcepass, ...)

go.work                              Coordinates `.`, `./cmd`, `./runtime`, `./test`, `./dev`, `./benchmark`
```

### Package Taxonomy Mapping (`go/*` & `golang.org/x/tools/go/*` ↔ `llgo/*`)

| Upstream counterpart | `github.com/xgo-dev/llgo` package | Role |
|---|---|---|
| `go/types` | `llgo/types` | 1:1 non-deprecated `go/types` parity + first-class LLGo metadata (`Package`/`Func`/`Var` directives, `CallConv`, `Locality`, `WasmImport`/`WasmExport`, `NoInline`/`NoInterface`/`NoEscape`/`UintptrEscapes`/`NoSplit`, `HasClosureEnv`, `Skip*`) & canonical `Context` layout/ABI/SIMD (`types/api_guard_test.go`) |
| `x/tools/go/types/typeutil` | `llgo/types/typeutil` | 1:1 `Hasher`, `Map`, `MethodSetCache`, `Callee`, `StaticCallee`, `Dependencies`, `IntuitiveMethodSet` |
| `x/tools/go/ssa` + `ssautil` | `llgo/ssa`, `llgo/ssa/ssautil` | 1:1 `Program`, `Package`, `Function`, `BasicBlock`, `Value`, `Instruction`, all 37 instructions + source-level LLGo facts (`CallKind`, `Intrinsic`, `RecvNilCheck`); backend codegen policies (`NoScan`, `StaticBytes`) belong in `cmd/compiler/ssaopt` (`cmd/compiler/cl/xtools_parity_guard_test.go`) |
| `x/tools/go/callgraph` + `callgraph/rta` | `llgo/callgraph`, `llgo/callgraph/rta` | 1:1 `Graph`/`Node`/`Edge`/helpers and Rapid Type Analysis (`rta.Analyze`/`Result`) |
| `x/tools/go/gcexportdata` | `llgo/exportdata` | 1:1 `Read`, `Write`, `ReadBundle`, `WriteBundle`, `NewImporter` (`types.ImporterFrom`); format v4 round-trips all LLGo `Package`/`Func`/`Method`/`Var` metadata |
| *(LLGo-specific)* | `llgo/directive`, `llgo/target` | Directive parsing & target/ABI spec model |

## Single Canonical Type Context & Direct Lowering Contract (M0)

Downstream compiler stages consume `llgo/types` and `llgo/ssa` directly without
recomputing target sizes or cloning/patching semantic type graphs:

1. **Single shared `*llgo/types.Context` (`Sizes` source of truth)**:
   `llvmgen.NewProgram(target)` builds one target-configured `*types.Context`
   (`prog.TypeContext()`), which is shared across `packages.Deduper`,
   `types.Config.Context`/`Sizes`, `cmd/compiler/goescape`, `cmd/compiler/ssaopt`,
   `cmd/compiler/cl`, `cmd/compiler/llvmgen` (including DWARF `di.go`), and
   `cmd/compiler/llvmgen/abi`. Ad-hoc `StdSizes` / `effectiveTypeSizes` overrides
   are eliminated.
2. **Zero semantic type cloning (`Expr.Type.RawType()` is always clean `llgo/types.Type`)**:
   `cmd/compiler/llvmgen` no longer clones the `types.Type` graph (`gocvt`) or
   synthesizes fake `struct { $f; $data }` types for Go closures. Go
   `*types.Signature` (`CallConv == types.Go`) lowers directly to `vkClosure`
   (`{ ptr, ptr }` or named LLVM struct for named function types), while native
   `//llgo:cdecl`/`//llgo:stdcall` signatures lower to `vkFuncPtr` (`ptr`).
   Runtime `*abi.Type` descriptors remain 100% binary compatible with
   `runtime/abi` and `reflect` (`_llgo_closure$...`, `TFlagClosure`,
   `Kind=Struct`, `$f`/`$data` fields).
3. **Root-module guardrail — Source facts vs. backend decisions**:
   - **Keep source semantics and analysis view stable in root `llgo/types` & `llgo/ssa`**;
     code generation and optimization policies belong to the compiler (`cmd/compiler/ssaopt`).
   - `RecvNilCheck` (implicit pointer dereference on receiver) and type pointer
     presence (`types.Context.Layout` / `HasPointers`) are source/type facts.
   - Allocation scan policy (`AllocAtomicU` / `NoScan`), stack vs. heap vs.
     interface-box placement, large static byte array folding (`StaticBytes` /
     init statement elision), and check elimination are compiler decisions owned
     by `cmd/compiler/ssaopt`.

## Compiler Middle-End Layering (`llgo/ssa` + `cmd/compiler/ssaopt`)

Rather than duplicating LLVM's generic machine/scalar optimization pipeline or
forking the public `llgo/ssa` API away from `x/tools/go/ssa`, LLGo keeps
`llgo/ssa` as the stable source-level SSA and introduces a **pure-Go, zero-LLVM**
compiler middle-end at `cmd/compiler/ssaopt` (mirroring the upper Go-semantic
half of Go's `cmd/compile/internal/ir` + generic SSA operations):

1. **Go-semantic operation granularity preserved before LLVM IR**:

   | Operation | Information retained before LLVM lowering |
   |---|---|
   | **Interface conversion** | Distinguish concrete boxing (`MakeInterface`), interface-to-interface (`ChangeInterface`), and type assertions (`TypeAssert`); preserve dynamic type and nil-interface vs. typed-nil distinction; boxing placement decided by escape + `FunctionPlan`. |
   | **Nil check** | Explicitly distinguish pointer deref, implicit receiver deref (`RecvNilCheck`), and nil interface; preserve defined panic evaluation order even at `-O0`; elimination requires a dominating proof. |
   | **Memory allocation** | Retain Go type, zero-init, and GC scan requirements (via `types.Context`); stack vs. heap vs. `NoScan` (`AllocAtomicU`) vs. box elimination are explicit plan decisions. |
   | **Interface call (`Invoke`)** | Retain interface receiver, method identity, and Go signature so devirtualization runs before indirect function pointer lowering while preserving any required nil check. |
   | **Indexing** | Distinguish array, slice, string, and map; evaluate bounds checks on the original index width before element address computation (e.g., `uint64` indices on Wasm32 checked prior to pointer-width truncation). |

2. **`FunctionPlan` first, optional private MIR only when justified**:
   - **V1 (`M2`)** produces a read-only `ssaopt.FunctionPlan` per `*ssa.Function`,
     consuming pure-Go layout facts (`*types.Context`) and escape analysis
     (`goescape`, decoupled from LLVM backend programs) and emitting explicit
     decisions (nil-check requirements/elisions, allocation & box placement,
     `NoScan`, static byte array init folding, bounds check elisions, devirt
     targets) consumed by `cmd/compiler/cl`.
   - **Verification**: CFG/Phi invariants continue to be verified by
     `ssa.VerifyFunction`; `ssaopt.VerifyPlan` adds plan-specific invariants
     (valid instruction anchors, every eliminated check has a dominating proof,
     single-stage ownership of checks, and `-O0` semantic preservation).
   - **When to escalate to a private `cmd/compiler/mir`**: Only if a concrete
     pass (such as GC safepoints / write barriers requiring explicit memory-token
     effects) cannot be cleanly expressed and verified with `ssa + FunctionPlan`,
     or profiling demonstrates that pointer-graph construction/decoding is a
     primary bottleneck. If introduced later, it lowers from `llgo/ssa` without
     rewriting the AST frontend.

### Module Invariants

1. **Root `go.mod` (`github.com/xgo-dev/llgo`)**:
   - Declares `go 1.20` (matching `runtime/go.mod`) for maximum downstream
     library compatibility; any Go 1.22+/1.23+ stdlib hooks (such as
     `go/types.Alias`) are isolated behind `//go:build` files.
   - Contains **zero** `github.com/xgo-dev/llvm`, **zero** `golang.org/x/tools`,
     **zero** CGo dependencies, and **zero** `replace` directives.
   - Verified in CI by:
     ```bash
     GOWORK=off CGO_ENABLED=0 go test ./...
     ! GOWORK=off go list -f '{{join .Deps "\n"}}' ./... | grep -qE 'llvm|x/tools'
     ```
2. **Compiler `cmd/go.mod` (`github.com/xgo-dev/llgo/cmd`)**:
   - Declares the latest Go toolchain version matching `.go-version`.
   - Holds `github.com/xgo-dev/llvm` and `golang.org/x/tools` (`packages` /
     `astutil` only; zero `x/tools/go/ssa` in the compiler), and requires
     `github.com/xgo-dev/llgo` and `github.com/xgo-dev/llgo/runtime`.
   - `cmd/compiler/ssaopt` and `cmd/compiler/goescape` maintain **zero**
     `github.com/xgo-dev/llvm` dependency so middle-end analyses and passes are
     fast, pure-Go unit-testable, and reusable across backends.
3. **Workspace (`go.work`)**:
   - Lists `use ( . ./cmd ./runtime ./test ./dev ./benchmark )`.

## Patched Standard Library `go/types` Contract

When LLGo compiles the Go standard library `go/types` package itself (via
`runtime/_patch/go/types/`, alongside existing `runtime/_patch/go/build` and
`runtime/_patch/go/importer`), `go/types` is patched so that `SizesFor` (for
`"llgo"` / `runtime.Compiler`) and default `Sizes` compute target-accurate
`Sizeof`, `Alignof`, and `Offsetsof` matching LLGo's runtime layout rules:

- 2-word Go function/closure values (`{codePtr, envPtr}` = `2 * wordSize`),
- `structs.HostLayout` field alignment,
- target word/alignment rules (64-bit, 32-bit, Wasm32/64 wide Go storage), and
- Go's trailing zero-sized struct field padding.

This patch strictly models standard Go types and does **not** attempt to expose
LLGo-only extension attributes (`//llgo:cdecl`, `//llgo:stdcall`, `NativePtr`,
`SIMDVec`, etc.). Full LLGo physical/ABI extensions live in
`github.com/xgo-dev/llgo/types`, whose `Context.GoSizes()` serves as the pure-Go
reference implementation.

## Incremental Cache Contract (M1)

Decouple **Dev Incremental Mode** (separate per-package compilation) from
**Release Whole-Program Mode** (`-Os`/whole-program escape/RTA/`-deadcodedrop`/`-lto=full`)
using two cleanly separated cache layers alongside each cached package `.a`:

1. **Public type export layer (`llgo/exportdata` v4 `.llgotypes`)**:
   - Persists binary `llgo/types.Package` scopes including all LLGo directives
     and metadata (explicitly covering `pkg.Module == nil` stdlib and
     runtime-patched effective scopes) so unchanged dependencies skip Go source
     parsing, `types.Checker`, and `llgo/ssa` construction in production builds.
2. **Private, independently versionable compiler sidecars (`cmd/compiler/{frontcache,pkgmeta,build}`)**:
   - **`LLGODEP5` depfiles**: pure-Go `os.Stat` validation of transitive C header
     dependencies without spawning `clang++ -E`.
   - **`.linksnap`**: binary `packageLinkSnapshot` + PCLN metadata so cache hits
     never allocate an LLVM context or invoke `cl.NewPackageEx`.
   - **`.llgoex` (`pkgmeta.PackageMeta`) & on-demand bodies**: compact binary
     escape/inline/call summaries plus on-demand decoding of generic templates
     and inline-candidate bodies only when required by a dirty importer.

## Staged Delivery & Acceptance Gates (M0 → M3)

| Milestone | Scope | Independent Acceptance Gate |
|---|---|---|
| **M0 (#3220 + #3254)** | Pure-Go Go 1.20 core, native `llgo/ssa` cutover, multi-module split, LLGo extensions internalized (`directive`/`types`/`exportdata` v4/`ssa`), single canonical `prog.TypeContext()`, `gocvt` removal | Root Go 1.20/1.22/1.27 matrix + zero `llvm`/`x/tools` deps; full `cmd/compiler/llvmgen`, `cmd/cl`, `cmd/compiler/build`, and ABI/closure/CGo/Wasm tests green |
| **`cl` Relocation** | Mechanical move `cmd/cl` → `cmd/compiler/cl` (updating imports, test generators, fixture paths, CI, coverage) | Zero semantic diff; `cmd/compiler/cl`, `cmd/chore/*`, and CI workflows green |
| **M1 (Incremental Cache Hit Skip)** | Wire `.llgotypes` (`exportdata` v4) + private summary/on-demand body cache into production `packages`/`build` loading path | Warm 1-line edit verifies both wall-clock reduction **and** actual count of unchanged dependency packages skipping parse, type-check, and SSA build |
| **M2 (`cmd/compiler/ssaopt` v1)** | Decouple `PrepareEscapeAnalysis` from LLVM; add pure-Go `ssaopt` (`FunctionPlan` + `VerifyPlan`); pilot receiver nil-check single-owner pipeline (`RecvNilCheck` → `FunctionPlan` → `cl`); migrate `Alloc.NoScan` & `StaticBytes` codegen policies from root `ssa` into `ssaopt` | Zero `llvm` imports in `ssaopt`/`goescape`; `ssa.VerifyFunction` + `ssaopt.VerifyPlan` green; `-O0` semantics & panic evaluation order verified by execution tests |
| **M3 (Go-Semantic Passes)** | Expand `ssaopt` `FunctionPlan` across interface boxing placement, slice/array/string bounds check elimination (preserving wide-index Wasm32 safety), and interface devirtualization (preserving nil checks) | Golden IR + execution regressions green; evaluate private MIR only if concrete memory-effect passes or profiling justify it |

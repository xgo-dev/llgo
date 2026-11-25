# Multi-Module Layout & Incremental Cache Cutover Blueprint (Proposal #3215)

This document records the concrete cutover plan for separating LLGo's pure-Go
library surface—organized after `golang.org/x/tools/go/*`—from the LLVM/CGo
compiler CLI, pairing it with a compact binary package-summary cache and a
target-accurate stdlib `go/types` patch, so third-party Go consumers can import
LLGo packages without LLVM/CGo/`replace`/bleeding-edge Go while iterative builds
skip whole-program frontend and LLVM re-materialization on cache hits.

## Target Module Graph (`golang.org/x/tools`-Aligned)

```text
github.com/xgo-dev/llgo              (root go.mod — strict Go baseline: go 1.20, zero external deps, zero replace)
├── directive/                       Pure-Go //llgo:* and //go:* directive parsing & semantics
├── target/                          Pure-Go target triple, arch, and config model
├── types/                           1:1 go/types + LLGo semantic/physical layout, GC, ABI, SIMD  [API-guarded]
│   └── typeutil/                    1:1 golang.org/x/tools/go/types/typeutil                     [API-guarded]
├── ssa/                             1:1 golang.org/x/tools/go/ssa core types, methods & 37 instrs [API-guarded]
├── callgraph/                       1:1 golang.org/x/tools/go/callgraph                          [API-guarded]
│   └── rta/                         1:1 golang.org/x/tools/go/callgraph/rta                      [API-guarded]
├── exportdata/                      1:1 golang.org/x/tools/go/gcexportdata (Read/Write/Bundle/Importer) [API-guarded]
└── runtime/go.mod                   (separate module, go 1.20)

github.com/xgo-dev/llgo/cmd          (cmd/go.mod — latest Go toolchain, go 1.27.0)
├── llgo/                            CLI entrypoint (`llgo`)
├── internal/                        CLI subcommands & flag handling
├── cl/                              Go frontend adapter (go/types + x/tools/go/ssa → llgo/ssa & LLVM)
├── compiler/                        Compiler internals (build, llvmgen, pkgmeta, cabi, crosscompile, env, packages, ...)
├── xtool/                           LLVM/C toolchain bindings
└── chore/                           Repository maintenance utilities

go.work                              Coordinates `.`, `./cmd`, `./runtime`, `./test`, `./dev`, `./benchmark`
```

### Package Taxonomy Mapping (`go/*` & `golang.org/x/tools/go/*` ↔ `llgo/*`)

| Upstream counterpart | `github.com/xgo-dev/llgo` package | Role |
|---|---|---|
| `go/types` | `llgo/types` | 1:1 non-deprecated `go/types` parity + LLGo ABI/layout/CallConv/SIMD extensions (`types/api_guard_test.go`) |
| `x/tools/go/types/typeutil` | `llgo/types/typeutil` | 1:1 `Hasher`, `Map`, `MethodSetCache`, `Callee`, `StaticCallee`, `Dependencies`, `IntuitiveMethodSet` |
| `x/tools/go/ssa` | `llgo/ssa` | 1:1 `Program`, `Package`, `Function`, `BasicBlock`, `Value`, `Instruction`, `CallInstruction`, `CallCommon`, `Member`, `Node`, and all 37 instructions |
| `x/tools/go/callgraph` + `callgraph/rta` | `llgo/callgraph` + `llgo/callgraph/rta` | 1:1 `Graph`/`Node`/`Edge`/helpers and Rapid Type Analysis (`rta.Analyze`/`Result`) |
| `x/tools/go/gcexportdata` | `llgo/exportdata` | 1:1 `Read`, `Write`, `ReadBundle`, `WriteBundle`, `NewImporter` (`types.ImporterFrom`) |
| *(LLGo-specific)* | `llgo/directive`, `llgo/target` | Directive parsing & target/ABI spec model |

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
     ! GOWORK=off go list -f '{{join .Deps "\n"}}' ./directive ./target ./types ./ssa | grep -qE 'llvm|x/tools'
     ```
     *(After PR 6 moves compiler packages into `cmd/` and adds `exportdata/` + `analysis/`,
     the same check covers `./...` across the entire root module.)*
2. **Compiler `cmd/go.mod` (`github.com/xgo-dev/llgo/cmd`)**:
   - Declares the latest Go toolchain version matching `.go-version` so
     `golang.org/x/tools/go/ssa` parses the latest Go language syntax.
   - Holds `github.com/xgo-dev/llvm`, `golang.org/x/tools` (and any temporary
     `replace` directive while awaiting upstream `x/tools` fixes), and requires
     `github.com/xgo-dev/llgo` and `github.com/xgo-dev/llgo/runtime`.
3. **Workspace (`go.work`)**:
   - Lists `use ( . ./cmd ./runtime )` so contributors run tests seamlessly
     across all three modules without manual `replace` edits.

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
`SIMDVec`, etc.), since `go/types` has no fields for those extensions. Full
LLGo physical/ABI extensions live in `github.com/xgo-dev/llgo/types`, whose
`Context.GoSizes()` serves as the pure-Go reference implementation for standard
Go layout rules (and supplies directive-aware layout to the compiler frontend's
`go/types.Config.Sizes`).

## Incremental Cache Sidecar Contract (PR 5–6 Prerequisite)

Before the physical module cutover, PRs 5–6 of #3215 decouple **Dev
Incremental Mode** (separate per-package compilation) from **Release
Whole-Program Mode** (`-Os`/whole-program escape/RTA/`-deadcodedrop`/`-lto=full`)
by persisting compact **binary** sidecars (via `exportdata`) alongside each
cached package `.a`:

1. **Depfile fingerprint sidecar (`LLGoFiles`)**: persist `-MMD -MF` header
   dependencies on first C-wrapper preprocess so warm builds validate headers
   via pure-Go `os.Stat` instead of serially invoking `clang++ -E`.
2. **Link & PCLN snapshot sidecar**: persist `packageLinkSnapshot`
   (`needAbiInit`, `methodByIndex`, `methodByName`, `abiSymbols`, `abiTypes`,
   `hasLocalExports`, `cExports`) and binary PCLN records so cache hits never
   invoke `cl.NewPackageEx` or allocate an LLVM context.
3. **Binary frontend & `PackageSummary` sidecar (`llgo/exportdata`)**: persist
   binary `types.Package` export data (explicitly covering `pkg.Module == nil`
   stdlib and runtime-patched effective scopes) plus binary `PackageSummary`
   (escape, inline, and call-graph summaries consumed by `llgo/analysis/...`)
   so cache hits skip source parse, `go/types`, `go/ssa`, and whole-program
   escape analysis.

## Atomic Cutover Checklist (PR 7)

Perform the physical module split in a single dedicated PR once PRs 1–6 of
#3215 have merged to `main`, avoiding wide cross-PR merge conflicts:

1. **Relocate LLVM/`x/tools`-dependent compiler packages into `cmd/`**:
   - Move `cl/`, `internal/` (including `internal/llvmgen/`), `xtool/`, and
     `chore/` into `cmd/internal/...` (or a sibling `codegen/llvm` module if an
     embeddable LLVM backend API is desired).
   - Root pure-Go packages (`directive/`, `target/`, `types/`, `ssa/`,
     `exportdata/`, `analysis/`) remain in `github.com/xgo-dev/llgo` without
     import-path changes.
   - Update internal import paths accordingly.
2. **Create `cmd/go.mod` and `go.work`**:
   - Move compiler dependencies and `replace` directives from root `go.mod` into
     `cmd/go.mod`.
   - Lower root `go.mod` to `go 1.20` and prune root `go.sum` with
     `GOWORK=off go mod tidy`.
3. **Update release, install, and CI touchpoints**:
   - `.goreleaser.yaml`: set `dir: cmd` and `main: ./llgo` (updating `-X`
     ldflags paths to match relocated `env` and `xtool/env/llvm` package paths).
   - `install.sh` / `install.ps1`: update source-install commands to build from a
     source checkout under `./cmd` (e.g., `cd cmd && go build ./llgo` or
     `go install ./llgo` inside the checkout) while `cmd/go.mod` carries any
     `replace` directive for `golang.org/x/tools`, since Go rejects versioned
     `go install ...@<version>` on modules with `replace` directives. Enable
     versioned `go install github.com/xgo-dev/llgo/cmd/llgo@<version>` only
     after all `replace` directives are removed from `cmd/go.mod`.
   - `.github/workflows/*.yml`: run `GOWORK=off CGO_ENABLED=0 go test ./...` at
     the repo root to guard pure-Go Go 1.20 library hygiene, and run
     compiler/LLVM test matrices with `go.work` enabled (or
     `cd cmd && go test ./...`).
   - Update `.github/scripts/test_pixi_env.py` (introduced in PR #3218) to
     validate `dev/pixi.toml` against `cmd/go.mod` and `.go-version`.
4. **Verify end-to-end**:
   - `GOWORK=off CGO_ENABLED=0 go test ./...` (root module at Go 1.20)
   - `cd cmd && go build ./llgo && go test ./...`
   - Full CI matrix across Linux, macOS, and Windows (MSVC/MinGW).

# Multi-Module Layout & Incremental Cache Cutover Blueprint (Proposal #3215)

This document records the concrete cutover plan for separating LLGo's pure-Go
library surface from the LLVM/CGo compiler CLI—and pairing it with a binary
package-summary cache—so third-party Go consumers can import LLGo packages
without LLVM/CGo/`replace`/bleeding-edge Go while iterative builds skip
whole-program frontend and LLVM re-materialization on cache hits.

## Target Module Graph

```text
github.com/xgo-dev/llgo              (root go.mod — lower Go baseline, e.g. go 1.24)
├── directive/                       Pure-Go //llgo:* and //go:* directive parsing & semantics
├── target/                          Pure-Go target triple, arch, and config model
├── types/                           Pure-Go LLGo type model (Closure, NativePtr, HostLayout, SIMD)
├── ir/                              Pure-Go mid-level SSA IR, verifier, and package summary codec
└── runtime/go.mod                   (existing separate module, unchanged)

github.com/xgo-dev/llgo/cmd          (cmd/go.mod — latest Go toolchain, e.g. go 1.27+)
├── llgo/                            CLI entrypoint (`llgo`)
├── llgen/                           IR generation tool
├── internal/                        CLI subcommands and compiler-internal packages
└── compiler/ (or cmd/internal/...)  x/tools + LLVM lowering (`cl`, `ssa`, `internal/build`, ...)

go.work                              Coordinates `.`, `./cmd`, and `./runtime` in local dev & CI
```

### Module Invariants

1. **Root `go.mod` (`github.com/xgo-dev/llgo`)**:
   - Declares a conservative Go version (e.g., `go 1.24`) suitable for broad
     library adoption.
   - Contains **zero** `github.com/xgo-dev/llvm`, **zero** `golang.org/x/tools`,
     **zero** CGo dependencies, and **zero** `replace` directives.
   - Verified in CI by:
     ```bash
     GOWORK=off CGO_ENABLED=0 go test ./...
     ! GOWORK=off go list -f '{{join .Deps "\n"}}' ./directive ./target ./types ./ir | grep -qE 'llvm|x/tools'
     ```
2. **Compiler `cmd/go.mod` (`github.com/xgo-dev/llgo/cmd`)**:
   - Declares the latest Go toolchain version matching `.go-version` so
     `golang.org/x/tools/go/ssa` parses the latest Go language syntax.
   - Holds `github.com/xgo-dev/llvm`, `golang.org/x/tools` (and any temporary
     `replace` directive while awaiting upstream `x/tools` fixes), and requires
     `github.com/xgo-dev/llgo` and `github.com/xgo-dev/llgo/runtime`.
3. **Workspace (`go.work`)**:
   - Lists `use ( . ./cmd ./runtime )` so contributors run tests seamlessly
     across all three modules without manual `replace` edits.

## Incremental Cache Sidecar Contract (PR 5–6 Prerequisite)

Before the physical module cutover, Wave 2 of #3215 decouples **Dev
Incremental Mode** (separate per-package compilation) from **Release
Whole-Program Mode** (`-Os`/whole-program escape/RTA/`-deadcodedrop`/`-lto=full`)
by persisting compact **binary** sidecars alongside each cached package `.a`:

1. **Depfile fingerprint sidecar (`LLGoFiles`)**: persist `-MMD -MF` header
   dependencies on first C-wrapper preprocess so warm builds validate headers
   via pure-Go `os.Stat` instead of serially invoking `clang++ -E`.
2. **Link & PCLN snapshot sidecar**: persist `packageLinkSnapshot`
   (`needAbiInit`, `methodByIndex`, `methodByName`, `abiSymbols`, `abiTypes`,
   `hasLocalExports`, `cExports`) and binary PCLN records so cache hits never
   invoke `cl.NewPackageEx` or allocate an LLVM context.
3. **Binary frontend & `ir.PackageSummary` sidecar**: persist binary
   `types.Package` export data (explicitly covering `pkg.Module == nil` stdlib
   and runtime-patched effective scopes) plus binary `ir.PackageSummary`
   (escape, inline, and call-graph summaries) so cache hits skip source parse,
   `go/types`, `go/ssa`, and whole-program escape analysis.

## Atomic Cutover Checklist (PR 7)

Perform the physical package relocation in a single dedicated PR once the
prerequisite PRs of #3215 (Wave 1 PRs 1–4 and Wave 2 incremental/IR PRs 5–6)
have merged to `main`, avoiding wide cross-PR merge conflicts:

1. **Relocate LLVM/`x/tools`-dependent compiler packages** (`cl/`, `ssa/`,
   `internal/`, `xtool/`, `chore/`) into the compiler module (`cmd/` or a
   sibling `compiler/` module if an embeddable compiler API is desired alongside
   `cmd/`). Update internal import paths accordingly.
2. **Create `cmd/go.mod` and `go.work`**:
   - Move compiler dependencies and `replace` directives from root `go.mod` into
     `cmd/go.mod`.
   - Lower root `go.mod` `go` directive and prune root `go.sum` with
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
     the repo root to guard pure-Go library hygiene, and run compiler/LLVM test
     matrices with `go.work` enabled (or `cd cmd && go test ./...`).
   - Update `.github/scripts/test_pixi_env.py` (introduced in PR #3218) to
     validate `dev/pixi.toml` against `cmd/go.mod` and `.go-version`.
4. **Verify end-to-end**:
   - `GOWORK=off CGO_ENABLED=0 go test ./directive/... ./target/... ./types/... ./ir/...`
   - `cd cmd && go build ./llgo && go test ./...`
   - Full CI matrix across Linux, macOS, and Windows (MSVC/MinGW).

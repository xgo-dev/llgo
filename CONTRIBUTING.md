# Contributing to LLGo

LLGo is an LLVM-based Go compiler with C, Python, JavaScript, WebAssembly, and embedded integrations. This guide covers the repository-specific workflow for code contributors. See the [README](README.md) for installation and usage.

## Contribution workflow

- Fork the repository, create a focused branch, and open a pull request against `xgo-dev/llgo:main`.
- Keep changes scoped and avoid rewriting unrelated files. Diagnose baseline failures instead of hiding them with skips, exclusions, or weakened checks.
- Describe the behavior change, compatibility implications, tests run, platforms exercised, and any validation gaps in the pull request.
- Use an issue to discuss substantial proposals or behavior changes before investing in a large implementation.

## Project structure

- `cmd/llgo` - Main compiler command
- `cl/` - Go package to LLVM IR compilation
- `ssa/` - LLVM IR generation with Go SSA semantics
- `internal/build/` - Build orchestration
- `runtime/` - LLGo runtime library
- `chore/` - Development tools (litgen, llpyg, ssadump, etc.)
- `_demo/` - C/C++, Python, and other integration examples
- `_cmptest/` - Go/LLGo output comparison tests

## Development environment

For detailed dependency requirements and installation instructions, see the [Dependencies](README.md#dependencies) and [How to install](README.md#how-to-install) sections in the README.

CI uses LLVM 19 and pinned Go patch releases; check [`.github/workflows/llgo.yml`](.github/workflows/llgo.yml) and [`.github/workflows/goroot.yml`](.github/workflows/goroot.yml) for exact versions. Native development supports macOS and Linux; use WSL2 or Linux containers on Windows.

## Testing and validation

Behavior changes require focused regression tests; documentation-only and mechanical changes do not need artificial tests. Start with the affected package, then broaden validation:

```bash
go test ./path/to/package
go test ./...
```

The nested `runtime` Go module is not covered by root-level `go test`, `go build`, or `go vet`; run the corresponding command there when it is affected, for example `(cd runtime && go test ./...)`.

Install the [documented dependencies](README.md#dependencies), including development libraries for Python and other integrations. If one is unavailable, report the exact omitted tests and reason; omission is not a pass.

Prefer the development wrapper for LLGo execution tests; it builds the current checkout and selects its runtime tree:

```bash
./dev/llgo.sh test ./path/to/package
```

After focused tests pass, `./dev/local_ci.sh` runs the main local checks when dependencies are available. See [`dev/README.md`](dev/README.md) for details.

### Coverage

- The Codecov patch check must pass; new deterministic logic and error paths should normally be covered.
- From the module containing the target package, check focused coverage with `go test -coverprofile=coverage.out ./path/to/package` and `go tool cover -func=coverage.out`.
- Linux and macOS coverage is combined; validate host-specific changes on the matching host when possible.
- [`.github/codecov.yml`](.github/codecov.yml) lists paths excluded from coverage. Add an exclusion only for generated, tooling, fixture, or otherwise non-meaningful code; never exclude production logic merely to make a PR pass, and explain every ignore change in the PR.

### Update IR test expectations

When `ssa/` or `cl/` changes generated IR, refresh only the affected expectations and review every generated diff:

```bash
go run ./chore/litgen path/to/LITTEST/in.go
```

Do not regenerate unrelated output. Supported scopes and the marker format are documented in [`dev/README.md`](dev/README.md#6-refresh-ir-checks).

### Compatibility and target validation

- Go compatibility covers source and observable behavior, not gc's internal ABI. Run standard-library tests with both `go test ./test/std/...` and `./dev/llgo.sh test ./test/std/...`.
- Run official Go cases with `bash ./dev/test_goroot.sh -- -directive-mode ci`; see [`test/goroot/README.md`](test/goroot/README.md) for filtering, multiple toolchains, full coverage, and sharding.
- Run native tests on the matching host. Use `dev/docker.sh` for Linux amd64/arm64 validation, `dev/test_wasm.sh` for Wasm, and `dev/test_embed.sh` for embedded build plus emulator smoke.
- Cross-compilation is not execution validation. Do not weaken failures to make a change pass, and state any target that could not be run.
- Changes to runtime ABI, archive/link metadata, target selection, or generated IR need focused multi-target tests. Use `// LITTEST` checks where IR shape matters and describe compatibility implications in the pull request.

The host matrix, CI coverage, dependencies, and target-specific follow-up commands are in [`dev/README.md`](dev/README.md#platform-and-target-validation).

### Performance, size, and validation record

- For compiler, runtime, linker, ABI, or hot-path changes, run focused benchmarks and inspect the paired Linux/macOS results. Repeat material differences because small changes may be runner noise. See [`benchmark/baseline/README.md`](benchmark/baseline/README.md).
- For changes that may affect binary layout or size, use `llgo build -size` as described in [`doc/size-report.md`](doc/size-report.md).
- In the pull request, record commands and targets, distinguish execution from build-only checks, and identify gaps. Required Linux/macOS checks must pass; a `continue-on-error` lane is not authoritative.

## Code quality

### Format code

```bash
gofmt -w path/to/changed.go
```

Format every changed Go file before committing, but do not rewrite unrelated files.

For changed shell scripts, run `bash -n path/to/changed.sh` and `shellcheck path/to/changed.sh` when ShellCheck is available.

### Run static analysis

Run `go vet ./path/to/package` for affected packages. Repository-wide vet currently reports lock-copy diagnostics in `ssa/type_cvt.go` and possible `unsafe.Pointer` misuse in `cl/builtin_test.go`; do not claim a clean run, suppress new diagnostics, or silently expand this baseline.

## Common development tasks

Use `./dev/llgo.sh version` to build the current checkout with the development configuration and check the resulting command. Installation and tool-building commands are maintained in the [README](README.md#how-to-install).

## Debugging

### Disable garbage collection

The `nogc` build tag is a targeted diagnostic mode that changes runtime semantics; it does not replace validation with the default GC configuration:

```bash
./dev/llgo.sh run -tags nogc .
```

See [Garbage Collection](README.md#garbage-collection-gc) and [`doc/defer-tls-gc.md`](doc/defer-tls-gc.md) for the supported modes and runtime design.

### `LLGO_ROOT`

Do not set `LLGO_ROOT` unconditionally. Development wrappers derive it for the current checkout, and an installed `llgo` does not necessarily require it. Set it explicitly only to select a non-standard source/runtime tree.

## Important notes

Examples live under `_demo/`, whose underscore keeps ordinary `go` package discovery from including them. C and C++ integration uses LLGo directives and target ABIs, including `go:linkname` where appropriate; follow [`doc/How-to-support-a-C&C++-Library.md`](doc/How-to-support-a-C&C++-Library.md) instead of assuming every binding uses the same mechanism.

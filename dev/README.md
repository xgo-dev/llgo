# Dev tooling

This directory contains reproducible host environments and scripts for running
LLGo locally or inside reusable Linux dev containers.

## Host development environment

Choose [mise](https://mise.jdx.dev/installing-mise.html) 2026.10.7 or newer, or
[Pixi](https://pixi.prefix.dev/latest/installation/) 0.81.0 or newer. Both
provide Go and install LLVM/Clang/LLD 22, pkg-config, and native libraries
from conda-forge. Their configuration and lockfiles live in this directory.
Use Go directly to run or build LLGo in either environment.

From the repository root, enter the mise environment:

```sh
mise -C dev trust
mise -C dev install --locked
mise -C dev en
```

Or enter the Pixi environment:

```sh
cd dev
pixi shell --locked
```

Both examples start the shell inside `dev/`. Return to the repository root:

```sh
cd ..
go run ./cmd/llgo version
go build ./cmd/llgo
```

The build creates `llgo` (`llgo.exe` on Windows). The environment stays loaded
when you change directories; type `exit` to leave the shell.

Both environments are checked in CI on these platforms:

| Host | Architectures |
| --- | --- |
| Linux | x86-64, ARM64 |
| macOS | Intel, ARM64 |
| Windows | x86-64, native PowerShell/CMD |

macOS requires the Xcode Command Line Tools (`xcode-select --install`). Windows
requires the Visual Studio C++ Build Tools with the Windows SDK and the
x86-64 C++ toolchain. Python, LLDB, cJSON, and cross-compilers are optional and
are not included in either environment.

Run the shared checks from the repository root in either development shell:

```sh
bash dev/check_devenv.sh
```

On Windows PowerShell, run `.\dev\check_devenv.ps1`. These scripts build LLGo,
run a compiled Go program, and check GC, libffi, OpenSSL, SQLite, libuv, and zlib.

For a single command without entering a shell, use these commands from the
repository root. mise runs inside `dev/`; Pixi uses the current directory:

```sh
mise -C dev exec -- go run ../cmd/llgo version
pixi run --manifest-path dev/pixi.toml --locked go run ./cmd/llgo version
```

Both configurations enable the LLVM Go bindings' `byollvm` tag and obtain
headers and linker flags from `llvm-config`. Include `byollvm` if you supply
`-tags` explicitly. The shared Linux wrappers in `conda-tools/` activate the
compiler's prefix and sysroot; `devenv_windows_ldflags.ps1` supplies Windows
linker flags while preserving Visual Studio's SDK/CRT library discovery.

mise installs packages in its user data directory; keep `MISE_DATA_DIR` local
to each host if you override it. Pixi stores its environment in `dev/.pixi/`,
which is ignored by Git. Do not share that directory between different hosts.

To update mise dependencies, edit `dev/mise.toml` (or `.go-version` for Go):

```sh
mise -C dev lock --platform linux-x64,linux-arm64,macos-x64,macos-arm64,windows-x64
mise -C dev install --locked
```

For Pixi, edit `dev/pixi.toml` and update its lockfile:

```sh
pixi update --manifest-path dev/pixi.toml
pixi install --manifest-path dev/pixi.toml --locked
```

Keep Pixi's Go major/minor version aligned with `.go-version`. Patch versions
may differ between providers. Open a new shell, run the
shared checks, and commit each changed configuration with its lockfile.

## Containers and scripts

## Prerequisites

- Docker installed and running
- Docker Compose v2 (`docker compose`, not `docker-compose`)

## 1) Start a Linux container, then run `dev/llgo.sh` / `dev/llgo_wasm.sh`

Start an interactive shell (pick one):

```bash
./dev/docker.sh amd64
./dev/docker.sh arm64
./dev/docker.sh i386
```

Notes:
- `amd64` uses the `pydeps` image target (includes extra Python demo deps like `numpy`/`torch`).
- `arm64` and `i386` use the smaller `base` target (no extra Python ML deps).

Inside the container, run tests/builds using the repo scripts:

```bash
./dev/llgo.sh test ./...
./dev/llgo.sh test ./test

# WASI/WASM (wasip1/wasm)
./dev/llgo_wasm.sh build ./...
```

Notes:
- `dev/docker.sh` starts in the same repo subdirectory you launched it from.
- `dev/llgo.sh` and `dev/llgo_wasm.sh` must be run from within `LLGO_ROOT` (the repo) and will error otherwise.

## 2) Start a Linux container, run one command, then exit

```bash
./dev/docker.sh amd64 bash -lc './dev/llgo.sh test ./test'
```

## 3) Run on the host (no container)

From anywhere inside the repo:

```bash
./dev/llgo.sh test ./test
./dev/llgo_wasm.sh build ./...
```

## 4) Run versioned tests

Run one exact Go toolchain against its default representative/full package set:

```bash
./dev/test_go_version.sh 1.20
./dev/test_go_version.sh 1.24 ./test/std/bytes
```

Run the complete Go 1.20 through Go 1.27 integration matrix:

```bash
./dev/test_go_versions.sh
```

This integration command is intentionally sequential and may take tens of
minutes locally. CI uses the same driver to consolidate the focused Go 1.20
through Go 1.26 compatibility sets in one job; Go 1.27 runs the complete test
tree in the primary platform matrix.

The corresponding wasm runtime commands are:

```bash
./dev/test_wasm_runtime_go_version.sh 1.24
./dev/test_wasm_runtime_go_versions.sh
```

The native runtime module has matching single-version and integration entries:

```bash
./dev/test_runtime_go_version.sh 1.20
./dev/test_runtime_go_versions.sh
```

Any standalone task can be run under an exact target toolchain with:

```bash
./dev/with_go_version.sh 1.20 ./dev/test_helloworld.sh 1.20
```

All scripts select exact toolchains, set `GOTOOLCHAIN=local` while testing, and
build llgo and repository tools with the version pinned in `.go-version`.

## 5) Run local CI (covers most checks)

```bash
./dev/local_ci.sh
```

This script creates a temporary workspace, runs formatting/build/tests, runs the
complete versioned llgo test integration script, and then runs demo checks.
You can control demo parallelism via `LLGO_DEMO_JOBS` (defaults to up to 4 jobs).

## 6) `dev/docker.sh` (composition-friendly)

`dev/docker.sh` is a thin wrapper around `docker compose`:

```bash
./dev/docker.sh <arch> [command...]
```

- `<arch>` must be `amd64`, `arm64`, or `i386`.
- If `[command...]` is omitted, it starts an interactive `bash`.
- If `[command...]` is provided, it runs that command and exits.
- You must run it from within the repo (within `LLGO_ROOT`), and it will start in the matching repo subdirectory inside the container.

## 7) Refresh test goldens

LLGo has separate refresh flows for runtime data and LLVM IR checks:

- `gentests` for runtime-output and package-metadata golden files
- `litgen` for the curated set of source-embedded, autogenerated `// LITTEST`
  FileCheck snapshots

### `gentests`

Run:

```bash
go run ./chore/gentests
```

Behavior:

- Refreshes `expect.txt` for the built-in runtime test suites using the existing execution flow.
- Refreshes `meta-expect.txt` for package-metadata tests.
- Preserves the runtime-output skip convention where `expect.txt` containing only `;` means "do not refresh".

LLVM IR is not part of the `gentests` workflow; it is checked from the marked
Go source with FileCheck.

### `litgen`

Autogenerated checks stay in the Go source file. A test opts in with an
autogenerated note immediately after `// LITTEST`; the note also records the
arguments required to reproduce the checks:

```go
// LITTEST
// NOTE: Assertions have been autogenerated by chore/litgen UTC_ARGS: --function=run --check-globals=smart
```

Create or replace an autogenerated region explicitly:

```bash
go run ./chore/litgen --function=run --check-globals=smart path/to/in.go
```

Refresh existing autogenerated tests recursively, using the arguments recorded
in each source file:

```bash
go run ./chore/litgen -u cl
```

Check that committed autogenerated checks are current without modifying files:

```bash
go run ./chore/litgen -u --check cl
```

Behavior:

- Accepts one or more paths.
- If the path is a `.go` file, it refreshes only that file. The file must start with `// LITTEST`.
- If the path is a directory, it walks that directory recursively and processes marked source files in a stable order.
- `-u`/`--update-only` updates only tests that already carry the autogenerated note. Handwritten checks are never silently replaced.
- `--check` reports stale autogenerated checks and does not write files.
- `--function` is repeatable and selects functions by regular expression. Use
  `--all-functions` only when the entire generated module is genuinely the test
  contract.
- `--check-globals=none|smart|all` controls global checks. `smart` keeps globals
  referenced by selected functions.
- Generated checks abstract numeric LLVM SSA values, block suffixes, numeric
  globals, and generated cgo symbol hashes to reduce irrelevant churn.
- Does not update runtime-output or package-metadata goldens.

Use `litgen` only when a test intentionally checks a broad IR shape. Prefer
short, handwritten FileCheck assertions for a single lowering or ABI property.
Runtime output does not replace focused IR checks; it only removes the need for
a second full-output snapshot.

Use 100 FileCheck directive lines as an audit threshold rather than a size cap.
For a larger handwritten fixture, every group should correspond to an explicit
semantic contract. Convert long contiguous IR-shape checks to an opted-in
`litgen` snapshot instead.

### Marker convention

Source-embedded IR checks are enabled by putting this marker on the first line of the source file:

```go
// LITTEST
```

The generated directives are consumed by the existing `littest`/FileCheck path in the compiler tests.

The plain marker retains the existing IR stage for the fixture's effective
target. To check that stage for several platforms independently of the host,
list a cross-compilation matrix explicitly:

```go
// LITTEST darwin/arm64 linux/amd64
```

The harness generates IR once for every listed GOOS/GOARCH pair and enables
`CHECK` together with the corresponding architecture and specific prefixes,
such as `CHECK,ARM64,DARWIN-ARM64` or `CHECK,AMD64,LINUX-AMD64`. Keep portable
assertions under `CHECK`, assertions shared by an architecture under `ARM64`,
`AMD64`, and similar prefixes, and only OS-specific differences under the exact
target prefix. Do not hide known platform differences in regular-expression
alternatives.

The fixture's current effective target is always checked as well. Targets on
the marker add cross-compilation coverage rather than replacing the platform
running the test, and a listed target equal to the current target is
deduplicated. An unlisted CI platform therefore still exercises all portable
`CHECK` assertions; add its explicit prefix when it has a distinct IR contract.

A test can instead check the module after target ABI lowering and before LLVM
optimization:

```go
// LITTEST: POST-ABI
```

Without an autogenerated note, post-ABI checks remain handwritten and
`litgen -u` leaves them untouched.

A post-ABI fixture can request several GOOS/GOARCH configurations on the same
marker. The harness generates IR once per listed target while the target
prefixes keep the differing assertions in one file. Runtime output, when
present, is still executed once with the fixture's normal run configuration.

```go
// LITTEST: POST-ABI linux/amd64 linux/arm64
```

To opt a post-ABI target matrix into automatic maintenance, add the same
autogenerated note used by an ordinary fixture. Automatic post-ABI
generation requires at least one explicit GOOS/GOARCH target on the marker so
the result is reproducible. For either matrix stage, `litgen` generates each
requested target, emits identical directives as shared `CHECK` lines, and keeps
same-architecture directives under the GOARCH prefix. Only remaining
differences use exact target prefixes.

```go
// LITTEST: POST-ABI linux/amd64 linux/arm64
// NOTE: Assertions have been autogenerated by chore/litgen UTC_ARGS: --function=f32ToI32 --check-globals=none
```

Cross-target IR generation does not require running target binaries. A fixture
that imports `C`, directly or through a dependency, is the exception: cgo also
needs a target C compiler, headers, and sysroot. Keep such fixtures on plain
`// LITTEST` unless the test environment supplies that complete cross-cgo
toolchain; their explicit target-prefixed assertions are then exercised by the
matching platform CI.

Example:

- [cl/_testlibc/setjmp/in.go](../cl/_testlibc/setjmp/in.go) demonstrates a handwritten default-stage cross-target matrix.
- [cl/_testgo/postabi/in.go](../cl/_testgo/postabi/in.go) demonstrates a handwritten post-ABI check with shared assertions and target-specific prefixes.
- [cl/_testdata/floatint/in.go](../cl/_testdata/floatint/in.go) demonstrates an automatically maintained post-ABI target matrix.

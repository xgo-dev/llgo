# Contributing to LLGo

Fork the repository, create a focused branch, and open a pull request against
`xgo-dev/llgo:main`. Discuss substantial language, runtime or compatibility
changes in an issue first. Keep unrelated edits and generated output out of the PR.

This guide covers building and changing LLGo. To install a release and write
programs with it, start with the [README](README.md#how-to-install).

## Get the source

```sh
git clone https://github.com/xgo-dev/llgo.git
cd llgo
```

Configure your fork as the push remote before publishing a branch.
Choose one of the environments below. Run subsequent commands from the
repository root unless a command explicitly changes directory. Use the versions
pinned in `.go-version`, the environment lockfiles, and the
[dependency setup action](.github/actions/setup-deps/action.yml).

## Development environments

Choose [mise](https://mise.jdx.dev/installing-mise.html) 2026.10.7 or newer, or
[Pixi](https://pixi.prefix.dev/latest/installation/) 0.81.0 or newer. Both
provide Go and install LLVM/Clang/LLD 22, pkg-config, and native libraries
from conda-forge. Their configuration and lockfiles live in `dev/`.
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

Both environments are checked on these platforms. Daily checks run in the
Development environments workflow in [llgo-compat/ci](https://github.com/llgo-compat/ci).

| Host | Architectures | CI cadence |
| --- | --- | --- |
| Linux | x86-64 | Daily |
| Linux | ARM64 | Daily |
| macOS | Intel, ARM64 | Daily |
| Windows | x86-64, native PowerShell/CMD | Daily |

macOS requires the Xcode Command Line Tools (`xcode-select --install`). Windows
requires the Visual Studio C++ Build Tools with the Windows SDK and the
x86-64 C++ toolchain. Python, LLDB, cJSON, and cross-compilers are optional and
are not included in either environment.

Run the shared checks from the repository root in either development shell:

```sh
bash dev/check_devenv.sh
```

To run the checks locally on Windows PowerShell, use `.\dev\check_devenv.ps1`.
These scripts build LLGo,
run a compiled Go program, and check GC, libffi, OpenSSL, SQLite, libuv, and zlib.

For a single command without entering a shell, use these commands from the
repository root. mise runs inside `dev/`; Pixi uses the current directory:

```sh
mise -C dev exec -- go run ../cmd/llgo version
pixi run --manifest-path dev/pixi.toml --locked go run ./cmd/llgo version
```

Both configurations enable the LLVM Go bindings' `byollvm` tag and obtain
headers and linker flags from `llvm-config`. Include `byollvm` if you supply
`-tags` explicitly. The shared Linux wrappers in `dev/conda-tools/` activate the
compiler's prefix and sysroot; the shared flag helpers quote LLVM paths and
preserve Visual Studio's SDK/CRT library discovery on Windows.
The Unix pkg-config wrapper handles Conda prefixes containing spaces.
Windows uses the compatible `pkgconf` implementation for paths containing spaces.

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

## Linux containers

Requirements:

- Docker installed and running
- Docker Compose v2 (`docker compose`, not `docker-compose`)

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

To run a command and exit:

```bash
./dev/docker.sh amd64 bash -lc './dev/llgo.sh test ./test'
```

## Manual development setup

Use these steps if you manage the host toolchain yourself. Install Go from
`.go-version` first. The platform snippets below install dependencies and build
the current checkout; run them from the repository root.

### Build dependencies

- [Go 1.27](https://go.dev) for building LLGo; CI validates user packages separately with Go 1.20 through Go 1.27
- [LLVM 22](https://llvm.org)
- [Clang 22](https://clang.llvm.org)
- [LLD 22](https://lld.llvm.org)
- [LLDB](https://lldb.llvm.org) (LLVM 22 packages on Linux/Windows; Xcode's Apple LLDB on macOS)
- [pkg-config 0.29+](https://gitlab.freedesktop.org/pkg-config/pkg-config)
- [bdwgc/libgc 8.0+](https://www.hboehm.info/gc/)
- [libffi](https://sourceware.org/libffi/)
- [OpenSSL 3.0+](https://www.openssl.org/)
- [zlib 1.2+](https://github.com/madler/zlib)
- [Python 3.12+](https://www.python.org) (optional, for [github.com/goplus/lib/py](https://pkg.go.dev/github.com/goplus/lib/py))

SQLite, cJSON, and libuv are also used by integration tests and demos. Install
optional Python packages and target toolchains only for the suites you run.
LLDB is needed for debugger checks.

### macOS

<!-- embedme doc/_readme/scripts/install_macos.sh#L2-L1000 -->

```sh
brew update
brew install llvm@22 lld@22 bdw-gc openssl cjson libffi libuv pkg-config
brew install python@3.12 # optional
brew link --force --overwrite llvm@22 lld@22 libffi
./install.sh
```

Native macOS builds use LLVM/LLD 22 and require a macOS SDK older than 27.
LLGo keeps the active SDK when it meets that requirement; otherwise it selects
the newest installed older SDK from Xcode or Command Line Tools. SDK 27 and
newer are excluded. Set `SDKROOT` to pin an installed older SDK; an unsupported
or missing explicit SDK is an error. If no older SDK is installed, install one
before building. The selected SDK is used for both compilation and linking.

Homebrew's versioned LLVM 22 formula does not ship LLDB and there is no `lldb@22` formula. LLGo checks common Homebrew and system locations, then `lldb` on `PATH`; use `LLGO_LLDB` or `llgo lldb -lldb` to select one explicitly.

### Debian/Ubuntu

Install the build dependencies from the repository root:

<!-- embedme doc/_readme/scripts/install_ubuntu.sh#L2-L1000 -->

```sh
set -eu
llvm_key="$(mktemp)"
trap 'rm -f "$llvm_key"' EXIT
wget -4 --retry-on-host-error --tries=3 --timeout=15 --waitretry=2 -O "$llvm_key" https://apt.llvm.org/llvm-snapshot.gpg.key
sudo install -m 0644 "$llvm_key" /usr/share/keyrings/llvm-snapshot.asc
echo "deb [signed-by=/usr/share/keyrings/llvm-snapshot.asc] https://apt.llvm.org/$(lsb_release -cs)/ llvm-toolchain-$(lsb_release -cs)-22 main" | sudo tee /etc/apt/sources.list.d/llvm.list
sudo apt-get update
sudo apt-get install -y llvm-22-dev clang-22 libclang-22-dev lld-22 lldb-22 libunwind-22-dev libc++-22-dev pkg-config libgc-dev libssl-dev zlib1g-dev libffi-dev libcjson-dev libsqlite3-dev libuv1-dev
sudo apt-get install -y python3-dev # optional
./install.sh

```

### Alpine Linux

```sh
apk add go llvm22-dev clang22-dev lld22 lldb pkgconf gc-dev libunwind-dev openssl-dev zlib-dev libffi-dev cjson-dev sqlite-dev libuv-dev
apk add python3-dev # optional
apk add g++ # build only
export LLVM_CONFIG=/usr/lib/llvm22/bin/llvm-config
export CGO_CPPFLAGS="$($LLVM_CONFIG --cppflags)"
export CGO_CXXFLAGS=-std=c++17
export CGO_LDFLAGS="$($LLVM_CONFIG --ldflags) $($LLVM_CONFIG --libs all)"
./install.sh
```

### Fedora Linux

For source builds, Fedora 44 and 45 ship LLVM 22 as the default LLVM stack.
Fedora 43 still ships LLVM 21 and is not a supported default-toolchain
environment for this LLGo release.

```sh
sudo dnf install -y llvm-devel clang-devel lld lldb libcxx-devel llvm-libunwind-devel \
  pkgconf-pkg-config gc-devel openssl-devel libffi-devel libuv-devel \
  cjson-devel sqlite-devel zlib-ng-compat-devel
llvm-config --version # must report 22.x
./install.sh
```

### Windows

Use [WINDOWS.md](WINDOWS.md) for the native SDK/CRT and library setup. MSVC
requires Visual Studio C++ Build Tools and the matching Windows SDK; MinGW
requires the matching native LLVM toolchain and libraries.

The recommended GNU-hosted setup is an MSYS2 `CLANG64` shell. Install the LLVM 22 stack and LLGo's native dependencies, then provide the versioned pkg-config metadata used by the Go/C++ bindings:

```sh
pacman -S --needed \
  mingw-w64-clang-x86_64-{clang,llvm,llvm-tools,lld,lldb,compiler-rt,libc++,libunwind} \
  mingw-w64-clang-x86_64-{gc,libffi,libuv,openssl,cjson,sqlite3,zlib,pkgconf}
test "$(llvm-config --version | cut -d. -f1)" = 22
pc_dir="$MINGW_PREFIX/lib/pkgconfig"
mkdir -p "$pc_dir"
printf '%s\n' \
  'Name: LLVM 22' \
  'Description: LLVM 22 host compiler and linker flags' \
  "Version: $(llvm-config --version)" \
  "Cflags: $(llvm-config --cflags)" \
  "Libs: $(llvm-config --ldflags --libs all --system-libs)" \
  > "$pc_dir/llvm-22.pc"
```

The native MSVC CI profile uses LLVM's official 22.1.8 development archive for headers, libraries, Clang, LLD, and all code-generation backends. That archive does not contain LLDB, so the profile also extracts LLDB from the matching official `win64` or `woa64` installer. LLVM 22 has no official Win32 installer; the Windows 386 lane uses the LLVM 22.1.8-based llvm-mingw 20260616 builtins and qualifies the official x64 LLDB under WoW64. The complete pinned setup, checksums, and generated `llvm-22.pc` are in `.github/actions/setup-deps/action.yml`.


### Install a source checkout

From the configured checkout:

<!-- embedme doc/_readme/scripts/install_llgo.sh#L4-L4 -->

```sh
./install.sh
```

`install.sh` builds the local checkout when run from the repository. To avoid
installing over a release while iterating, use `go run ./cmd/llgo`,
`go build ./cmd/llgo`, or the `dev/llgo.sh` wrapper instead.

## Work with the current checkout

Set `LLGO_ROOT` when invoking a custom-built compiler so it uses this checkout's
runtime and targets:

```sh
export LLGO_ROOT="$PWD"
go run ./cmd/llgo version
go build -o llgo ./cmd/llgo
./llgo run ./_demo/c/hello
```

On PowerShell, use `$env:LLGO_ROOT = (Get-Location).Path` and `./llgo.exe`.
The mise/Pixi environments and `dev/llgo.sh` set the checkout root for you.
The Bash wrapper builds the current checkout before invoking it:

```sh
./dev/llgo.sh version
```

For script invocation contracts, versioned test matrices,
and golden-file maintenance, see [dev/README.md](dev/README.md).

## Choose tests for the change

Add a regression that fails before a bug fix, or exercises the new behavior.
Start with the affected package and test, then run the relevant integration
checks. Documentation-only changes do not need artificial code tests.

Compiler compatibility is checked against applicable upstream `GOROOT/test`
cases using the pinned Go toolchain. User packages are also checked with exact
Go 1.20 through Go 1.27 toolchains. Applicable differences belong in
[`xfail.yaml`](test/goroot/xfail.yaml); mechanisms outside LLGo's compatibility
goals belong in [`notapplicable.yaml`](test/goroot/notapplicable.yaml).

| Area | Validation entry points |
| --- | --- |
| Compiler and LLVM lowering | Go tests in `cl/`, `ssa/` and `internal/build/`; focused IR assertions where needed |
| Language and runtime behavior | LLGo execution tests in `test/go/` and `test/std/`; Go/LLGo comparisons in `_cmptest/` |
| Official Go cases and source versions | [GOROOT runner](test/goroot/README.md) and [versioned tests](dev/README.md#versioned-tests) |
| WASI threads, GC and exceptions | [WASI qualification](dev/wasm-wasi-validation.md), using the pinned [Wasmer installer](dev/install_wasmer.sh) |
| Browser and Wasm standard library | [Browser checks](dev/test_wasm_browser.sh), [stdlib audit](dev/wasmstdlib/README.md) and [debugger qualification](dev/browser-debugging.md#qualification) |
| Embedded targets and debuggers | [Target commands](doc/Embedded_Cmd.md), emulator fixtures in `_demo/embed/`, and [debugger tests](doc/debugging.md) |

For example, from the repository root in Bash:

```sh
export LLGO_ROOT="$PWD"
go test ./cl
./dev/llgo.sh test ./test/std/runtime
```

`runtime/` is a separate Go module: root-level `go test ./...` does not include
its host-side tests. Run affected host-testable packages with an explicit module
directory, such as `(cd runtime && go test ./internal/traceback)`, as well as the
relevant LLGo execution tests.

For broader checks, see [local CI](dev/README.md#local-ci) and the
[workflow definitions](.github/workflows). Install the dependencies required by
the selected suites, including optional Python/demo libraries where applicable.
Cross-compilation, emulator execution and physical-device validation are
different results; record which one was performed.

## Maintain focused IR checks

Protect the compiler property under test with small handwritten `// LITTEST`
assertions. Declare target differences with explicit target matrices and
FileCheck prefixes. Runtime-output tests and IR checks protect different contracts.

Use `chore/litgen` for explicitly opted-in autogenerated snapshots. Review
generated changes and verify only the relevant scope; do not regenerate
unrelated checks. `chore/gentests` still maintains runtime-output and package
metadata goldens. See [test goldens](dev/README.md#refresh-test-goldens) for
the commands and conventions.

## Validate and describe the PR

- Format Go changes and run static analysis for the affected packages. Review
  formatter output rather than committing unrelated changes. Diagnose existing
  failures separately from regressions introduced by the PR.
- Cover new deterministic logic and error paths. Check focused coverage with
  `go test -coverprofile=coverage.out ./path/to/package` and
  `go tool cover -func=coverage.out`. Follow the [coverage guide](doc/test-coverage.md)
  and [Codecov configuration](.github/codecov.yml); do not exclude production
  code or weaken assertions to pass a check.
- For performance or layout changes, use the [benchmark workflow](benchmark/baseline/README.md)
  and [size reports](doc/size-report.md). Compare matching toolchains, targets
  and options, and distinguish final files from memory reservations or debug assets.
- For documentation edits, run the [documentation checks](#documentation-checks).
- Give the PR a concise title and explain the changed behavior, why it is
  needed, compatibility implications, and the commands/platforms tested. State
  skipped or unavailable validation explicitly. Address review comments and
  CI failures before calling the contribution ready.

## Documentation checks

Keep installation and usage in `README.md`, and contributor setup and
validation in this guide. Detailed script contracts and fixture conventions
live in `dev/README.md` and the relevant test README.

Run these checks from the repository root:

```sh
npx --yes embedme --verify README.md
npx --yes embedme --verify CONTRIBUTING.md
lychee --include-fragments README.md CONTRIBUTING.md dev/README.md
```

The [documentation workflow](.github/workflows/doc-link-checker.yml) runs these
checks. Embedded install examples come from `doc/_readme/scripts/`, which the
[installation workflow](.github/workflows/doc.yml) also executes. Change the
source script before updating an embedded example. `embedme` interprets a
single-token `//` comment on the first line of a code block as a filename;
start examples containing compiler directives with a real declaration, import,
or explanatory comment instead.

## Repository tools

* [gentests](chore/gentests): It refreshes runtime-output and package-metadata golden data under `cl/_test*`. LLVM IR checks live in Go sources as `// LITTEST` FileCheck directives.
* [litgen](chore/litgen): It maintains explicitly opted-in, source-embedded FileCheck snapshots. It supports function/global selection, update-only operation, stale-check verification, and stable LLVM value abstractions. Small handwritten checks remain manual.

For local workflows and test-golden refresh commands, see [dev/README.md](dev/README.md#refresh-test-goldens).

Build the tools from the configured repository root:

```sh
go install -v ./cmd/...
go install -v ./chore/...
```

## Key modules

The compiler pipeline is `internal/build` → `cl` → `ssa` → LLVM.

| Location | Responsibility |
| --- | --- |
| `cmd/llgo/` | Compiler command and Go-compatible command interface |
| `internal/build/` | Package loading, toolchain selection, compilation and linking |
| `cl/` | Go package compilation and lowering to LLVM IR |
| `ssa/` | LLVM IR generation through Go SSA semantics and interfaces |
| `runtime/` | Runtime library, maintained as a separate Go module |
| `chore/` | Repository maintenance tools |
| `_demo/`, `_cmptest/` | User examples and Go/LLGo comparison programs; the leading `_` excludes them from standard Go package discovery |

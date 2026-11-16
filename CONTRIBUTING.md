# Contributing to LLGo

Fork the repository and open a focused pull request against `xgo-dev/llgo:main`.
Discuss substantial language, runtime, or compatibility changes in an issue first.
For release installation, see the [README](README.md#how-to-install).

## Get the source

```sh
git clone https://github.com/xgo-dev/llgo.git
cd llgo
```

Run commands from the repository root unless shown otherwise. Use the versions
pinned in `.go-version`, the lockfiles, and the
[dependency setup action](.github/actions/setup-deps/action.yml).

## Development environments

Choose [mise](https://mise.jdx.dev/installing-mise.html) 2026.10.7 or newer, or
[Pixi](https://pixi.prefix.dev/latest/installation/) 0.81.0 or newer. Both
provide Go and install LLVM/Clang/LLD 22, pkg-config, and native libraries
from conda-forge. Configuration and lockfiles live in `dev/`.

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

The build creates `llgo` (`llgo.exe` on Windows). Type `exit` to leave the shell.

Daily checks run in [llgo-compat/ci](https://github.com/llgo-compat/ci) on Linux
amd64/arm64, macOS Intel/ARM64, and Windows amd64.

macOS requires the Xcode Command Line Tools (`xcode-select --install`). Windows
requires the Visual Studio C++ Build Tools with the Windows SDK and the
x86-64 C++ toolchain. Python, LLDB, cJSON, and cross-compilers are optional and
are not included in either environment.

Run the shared checks from the repository root in either development shell:

```sh
bash dev/check_devenv.sh
```

On PowerShell, use `.\dev\check_devenv.ps1`. The checks build LLGo, run a
compiled program, and exercise the native libraries.

For a single command without entering a shell, use these commands from the
repository root. mise runs inside `dev/`; Pixi uses the current directory:

```sh
mise -C dev exec -- go run ../cmd/llgo version
pixi run --manifest-path dev/pixi.toml --locked go run ./cmd/llgo version
```

Both environments set `GOFLAGS=-tags=byollvm`; include `byollvm` when overriding
`-tags`. Keep each host's mise/Pixi environment local to that host.

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

Keep Pixi's Go major/minor version aligned with `.go-version`. Run the shared
checks in a new shell and commit configuration changes with their lockfiles.

## Linux containers

Install Docker with Compose v2, then start a shell (pick one):

```bash
./dev/docker.sh amd64
./dev/docker.sh arm64
./dev/docker.sh i386
```

The amd64 image includes Python demo dependencies; arm64 and i386 use the
smaller base image.

Inside the container, run tests/builds using the repo scripts:

```bash
./dev/llgo.sh test ./...
./dev/llgo.sh test ./test

# WASI/WASM (wasip1/wasm)
./dev/llgo_wasm.sh build ./...
```

To run a command and exit:

```bash
./dev/docker.sh amd64 bash -lc './dev/llgo.sh test ./test'
```

## Manual development setup

Install the Go version in `.go-version`, then follow your platform's setup
below to install dependencies and build the checkout.

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

Tests and demos also use SQLite, cJSON, and libuv. Debugger checks need LLDB;
Python and target toolchains depend on the selected suites.

### macOS

<!-- embedme doc/_readme/scripts/install_macos.sh#L2-L1000 -->

```sh
brew update
brew install llvm@22 lld@22 bdw-gc openssl cjson libffi libuv pkg-config
brew install python@3.12 # optional
brew link --force --overwrite llvm@22 lld@22 libffi
./install.sh
```

Use a macOS SDK older than 27; set `SDKROOT` to select one explicitly.
Use Xcode's LLDB or select another with `LLGO_LLDB`.

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

Fedora 44 and 45 provide LLVM 22.

```sh
sudo dnf install -y llvm-devel clang-devel lld lldb libcxx-devel llvm-libunwind-devel \
  pkgconf-pkg-config gc-devel openssl-devel libffi-devel libuv-devel \
  cjson-devel sqlite-devel zlib-ng-compat-devel
llvm-config --version # must report 22.x
./install.sh
```

### Windows

Follow [WINDOWS.md](WINDOWS.md) for SDK/CRT and native libraries.
For an MSYS2 `CLANG64` source build, install LLVM 22 and its pkg-config metadata:

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

MSVC source builds need the full LLVM 22 development archive. The
[dependency setup action](.github/actions/setup-deps/action.yml) pins downloads
and configures LLDB and `llvm-22.pc`.

### Install a source checkout

From the configured checkout:

<!-- embedme doc/_readme/scripts/install_llgo.sh#L4-L4 -->

```sh
./install.sh
```

For local iteration, build and run the checkout as shown below.

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
The mise/Pixi environments set the checkout root. The Bash wrapper sets it
and builds the compiler before running:

```sh
./dev/llgo.sh version
```

See [dev/README.md](dev/README.md) for script options and test workflows.

## Choose tests for the change

Add a regression for changed behavior. Start with the affected test, then run
the relevant integration checks.

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
[workflows](.github/workflows). Record whether you built, ran in an emulator,
or tested on a device.

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

- Format Go changes and run static analysis and tests for the affected packages.
- Check [coverage](doc/test-coverage.md), [benchmarks](benchmark/baseline/README.md),
  or [binary sizes](doc/size-report.md) when relevant.
- Run the [documentation checks](#documentation-checks) for documentation edits.
- Describe the behavior change and list validation commands, platforms, results,
  and any checks you could not run.

## Documentation checks

```sh
npx --yes embedme --verify README.md
npx --yes embedme --verify CONTRIBUTING.md
lychee --include-fragments README.md CONTRIBUTING.md dev/README.md
```

The [documentation workflow](.github/workflows/doc-link-checker.yml) runs these
checks. Edit embedded examples in `doc/_readme/scripts/` first; the
[installation workflow](.github/workflows/doc.yml) also executes those scripts.

## Repository tools

- [gentests](chore/gentests): refresh runtime-output and package-metadata goldens.
- [litgen](chore/litgen): maintain opted-in FileCheck snapshots in Go sources.

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

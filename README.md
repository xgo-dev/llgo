LLGo - A Go compiler based on LLVM
=====

[![Build Status](https://github.com/xgo-dev/llgo/actions/workflows/go.yml/badge.svg)](https://github.com/xgo-dev/llgo/actions/workflows/go.yml)
[![GitHub release](https://img.shields.io/github/v/tag/xgo-dev/llgo.svg?label=release)](https://github.com/xgo-dev/llgo/releases)
[![Coverage Status](https://codecov.io/gh/xgo-dev/llgo/branch/main/graph/badge.svg)](https://codecov.io/gh/xgo-dev/llgo)
[![Benchmark](https://img.shields.io/badge/benchmark-LLGo_vs_Go-yellowgreen.svg)](https://xgo-dev.github.io/benchmarks/)
[![GoDoc](https://pkg.go.dev/badge/github.com/xgo-dev/llgo.svg)](https://pkg.go.dev/github.com/xgo-dev/llgo)
[![XGo](https://img.shields.io/badge/project-XGo-blue.svg)](https://github.com/goplus/xgo)

LLGo is a Go compiler based on LLVM in order to better integrate Go with the C ecosystem, including Python and JavaScript. It's a subproject of [the XGo project](https://github.com/goplus/xgo).

LLGo aims to expand the boundaries of Go/XGo, providing limitless possibilities such as:

* Game development
* AI and data science
* WebAssembly
* Embedded development
* ...

How can these be achieved?

```
LLGo := Go * C ecosystem
```

LLGo is compatible with the C ecosystem through the C **Application Binary Interface (ABI)**, while LLGo is compatible with Go at the **source-code level**. The C ecosystem includes languages that expose C-compatible interfaces (e.g. C/C++, Python, JavaScript, Objective-C, and Swift).


## Go support

LLGo is compatible with Go 1.20+ source code and supports the complete Go 1.27 language syntax, as well as `cgo`.

Compiler compatibility is checked against applicable upstream [`GOROOT/test`](test/goroot/README.md) cases using the pinned Go 1.27 toolchain. User projects and packages under `test/` are additionally tested with exact Go 1.20 through Go 1.27 toolchains. Remaining applicable differences are recorded in [`xfail.yaml`](test/goroot/xfail.yaml); gc-specific mechanisms outside LLGo's compatibility goals are documented in [`notapplicable.yaml`](test/goroot/notapplicable.yaml).

### Runtime

LLGo uses a different runtime from the standard Go toolchain. Native goroutines map 1:1 to OS threads with fixed native stacks, so direct C calls require no Go-to-C stack or scheduler transition, avoiding the cgo overhead that makes frequent C calls costly in standard Go.

The default garbage collector is conservative [BDWGC](https://www.hboehm.info/gc/) (also known as libgc). Bare-metal embedded targets instead use a TinyGo-derived conservative mark-and-sweep collector.

Garbage collection can be disabled with the `nogc` build tag. For example:

```sh
llgo run -tags nogc .
```

### Standard libraries

LLGo fully supports the Go standard library on supported native platforms. CI requires compatibility coverage for every public package and exported symbol in the primary Go toolchain, and runs focused [`test/std`](test/std/README.md) compatibility sets with each older supported toolchain.

Other targets may not provide every OS service or implementation-specific runtime behavior.

| Target | Current coverage |
| --- | --- |
| Native | Linux amd64/arm64, macOS amd64/arm64, and Windows amd64/arm64 (MSVC and MinGW) release builds; primary CI on Linux amd64, macOS arm64, and Windows toolchain profiles |
| WebAssembly | `js/wasm` and `wasip1/wasm` builds; WASI and Emscripten CI coverage |
| Embedded | [`-target`](doc/Embedded_Cmd.md) configurations for supported boards and MCUs, with selected QEMU/emulator smoke tests |

Named WebAssembly targets select an ecosystem C ABI independently of the Go
source tags: `-target emscripten` (and the legacy `-target wasm` alias) emits an
ES module plus its sibling wasm32 module, `-target emscripten-memory64` emits the
same pair with an LP64 C data model, and `-target wasi`/`-target wasip1` emits a
WASI Preview 1 module. Raw `GOOS=js/wasip1 GOARCH=wasm` remains a separate
compatibility path reserved for convergence with the standard Go platform ABI.


## C/C++ support

LLGo lets you import and call C/C++ libraries directly, without wrappers or cgo overhead.

### Interop mechanism

LLGo uses `go:linkname` to bind a Go declaration directly to a C ABI symbol:

<!-- embedme doc/_readme/llgo_call_c/call_c.go#L3-L6 -->

```go
import _ "unsafe" // for go:linkname

//go:linkname Sqrt C.sqrt
func Sqrt(x float64) float64
```

You can use this directly in your own code:

<!-- embedme doc/_readme/llgo_call_c/call_c.go -->

```go
package main

import _ "unsafe" // for go:linkname

//go:linkname Sqrt C.sqrt
func Sqrt(x float64) float64

func main() {
	println("sqrt(2) =", Sqrt(2))
}
```

Or organize such bindings into a package, as [c/math](https://github.com/goplus/lib/tree/main/c/math/math.go) does:

<!-- embedme doc/_readme/llgo_call_cmath/call_cmath.go -->

```go
package main

import "github.com/goplus/lib/c/math"

func main() {
	println("sqrt(2) =", math.Sqrt(2))
}
```

Because calls into C compile to native calls against the C ABI, there is no Go-to-C stack or scheduler transition, so frequent C calls stay cheap.

On Windows, bind APIs declared with `WINAPI` or `__stdcall` through the
`stdcall.` namespace. The convention is distinct on 386; Windows amd64 and
arm64 use their unified native C ABI. An explicitly decorated 386 name such as
`_MessageBoxW@16` is also accepted and is normalized to `MessageBoxW` on
64-bit targets.

```go
//go:linkname MessageBoxW stdcall.MessageBoxW
func MessageBoxW(hwnd uintptr, text, caption *uint16, flags uint32) int32

//llgo:type stdcall
type Callback func(context uintptr) uintptr
```

`stdcall.` declarations and `//llgo:type stdcall` apply only to non-variadic
function types. A native callback is one function pointer, so a Go callback
must be a direct function reference; pass state through an explicit context
pointer rather than a capturing closure.

### C/C++ standard libraries

LLGo provides Go bindings for the C/C++ standard library:

| Package | Description |
| --- | --- |
| [c](https://pkg.go.dev/github.com/goplus/lib/c) | C standard library core |
| [c/syscall](https://pkg.go.dev/github.com/goplus/lib/c/syscall) | System calls |
| [c/sys](https://pkg.go.dev/github.com/goplus/lib/c/sys) | System headers |
| [c/os](https://pkg.go.dev/github.com/goplus/lib/c/os) | OS interfaces |
| [c/math](https://pkg.go.dev/github.com/goplus/lib/c/math) | Math functions |
| [c/math/cmplx](https://pkg.go.dev/github.com/goplus/lib/c/math/cmplx) | Complex math |
| [c/math/rand](https://pkg.go.dev/github.com/goplus/lib/c/math/rand) | Random number generation |
| [c/pthread](https://pkg.go.dev/github.com/goplus/lib/c/pthread) | POSIX threads |
| [c/pthread/sync](https://pkg.go.dev/github.com/goplus/lib/c/pthread/sync) | Thread synchronization |
| [c/sync/atomic](https://pkg.go.dev/github.com/goplus/lib/c/sync/atomic) | Atomic operations |
| [c/time](https://pkg.go.dev/github.com/goplus/lib/c/time) | Time functions |
| [c/net](https://pkg.go.dev/github.com/goplus/lib/c/net) | Networking |
| [cpp/std](https://pkg.go.dev/github.com/goplus/lib/cpp/std) | C++ standard library core |

Here is a simple example calling the C `printf` function:

<!-- embedme doc/_readme/llgo_simple/simple.go -->

```go
package main

import "github.com/goplus/lib/c"

func main() {
	c.Printf(c.Str("Hello world\n"))
}
```

`c.Str` is not a runtime conversion from a Go string to a C string — it is a built-in instruction that `llgo` recognizes and compiles directly into a C string constant.

Additional demos are available in the `_demo` directory (prefixed with `_` so the `go` command skips them):

* [hello](_demo/c/hello/main.go): call C `printf` and `fprintf` with Go and C strings
* [qsort](_demo/c/qsort/qsort.go): call a C function that takes a callback (e.g. `qsort`)

To run a demo (see [How to install](#how-to-install) if `llgo` isn't installed yet):

```sh
cd <demo-directory>  # e.g. cd _demo/c/hello
llgo run .
```

### Other frequently used libraries

Beyond the standard library, LLGo can import libraries from across the C/C++ ecosystem. Bindings are currently maintained by hand; automating this process, as is already done for Python library imports, is planned for the future.

Available bindings include:

* [c/bdwgc](https://pkg.go.dev/github.com/goplus/lib/c/bdwgc)
* [c/cjson](https://pkg.go.dev/github.com/goplus/lib/c/cjson)
* [c/clang](https://pkg.go.dev/github.com/goplus/lib/c/clang)
* [c/ffi](https://pkg.go.dev/github.com/goplus/lib/c/ffi)
* [c/libuv](https://pkg.go.dev/github.com/goplus/lib/c/libuv)
* [c/llama2](https://pkg.go.dev/github.com/goplus/lib/c/llama2)
* [c/lua](https://pkg.go.dev/github.com/goplus/lib/c/lua)
* [c/neco](https://pkg.go.dev/github.com/goplus/lib/c/neco)
* [c/openssl](https://pkg.go.dev/github.com/goplus/lib/c/openssl)
* [c/raylib](https://pkg.go.dev/github.com/goplus/lib/c/raylib)
* [c/sqlite](https://pkg.go.dev/github.com/goplus/lib/c/sqlite)
* [c/zlib](https://pkg.go.dev/github.com/goplus/lib/c/zlib)
* [cpp/inih](https://pkg.go.dev/github.com/goplus/lib/cpp/inih)
* [cpp/llvm](https://pkg.go.dev/github.com/goplus/lib/cpp/llvm)

Examples built on these bindings:

* [llama2-c](_demo/c/llama2-c): inference Llama 2 (the first LLGo AI example)
* [mkjson](https://github.com/goplus/lib/tree/main/c/cjson/_demo/mkjson/mkjson.go): create a JSON object and print it
* [sqlitedemo](https://github.com/goplus/lib/tree/main/c/sqlite/_demo/sqlitedemo/demo.go): a basic SQLite demo
* [tetris](https://github.com/goplus/lib/tree/main/c/raylib/_demo/tetris/tetris.go): a Tetris game based on raylib


## Python support

You can import a Python library in LLGo!

You can import Python libraries into `llgo` through `llpyg` (see [Development tools](#development-tools)). Available bindings include:

* [py](https://pkg.go.dev/github.com/goplus/lib/py) (abi)
* [py/std](https://pkg.go.dev/github.com/goplus/lib/py/std) (builtins)
* [py/sys](https://pkg.go.dev/github.com/goplus/lib/py/sys)
* [py/os](https://pkg.go.dev/github.com/goplus/lib/py/os)
* [py/math](https://pkg.go.dev/github.com/goplus/lib/py/math)
* [py/json](https://pkg.go.dev/github.com/goplus/lib/py/json)
* [py/inspect](https://pkg.go.dev/github.com/goplus/lib/py/inspect)
* [py/statistics](https://pkg.go.dev/github.com/goplus/lib/py/statistics)
* [py/numpy](https://pkg.go.dev/github.com/goplus/lib/py/numpy)
* [py/pandas](https://pkg.go.dev/github.com/goplus/lib/py/pandas)
* [py/torch](https://pkg.go.dev/github.com/goplus/lib/py/torch)
* [py/matplotlib](https://pkg.go.dev/github.com/goplus/lib/py/matplotlib)

Third-party libraries such as pandas and PyTorch must be installed separately.

Here is an example:

<!-- embedme doc/_readme/llgo_call_py/call_py.go -->

```go
package main

import (
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
	"github.com/goplus/lib/py/std"
)

func main() {
	x := math.Sqrt(py.Float(2))       // x = sqrt(2)
	std.Print(py.Str("sqrt(2) ="), x) // print("sqrt(2) =", x)
}
```

It is equivalent to the following Python code:

<!-- embedme doc/_readme/llgo_call_py/call_math.py -->

```py
import math

x = math.sqrt(2)
print("sqrt =", x)
```

Here, We call `py.Float(2)` to create a Python number 2, and pass it to Python’s `math.sqrt` to get `x`. Then we call `std.Print` to print the result.

Let's look at a slightly more complex example. For example, we use `numpy` to calculate:

<!-- embedme doc/_readme/llgo_py_list/py_list.go -->

```go
package main

import (
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/numpy"
	"github.com/goplus/lib/py/std"
)

func main() {
	a := py.List(
		py.List(1.0, 2.0, 3.0),
		py.List(4.0, 5.0, 6.0),
		py.List(7.0, 8.0, 9.0),
	)
	b := py.List(
		py.List(9.0, 8.0, 7.0),
		py.List(6.0, 5.0, 4.0),
		py.List(3.0, 2.0, 1.0),
	)
	x := numpy.Add(a, b)
	std.Print(py.Str("a+b ="), x)
}
```

Here we define two 3x3 matrices a and b, add them to get x, and then print the result.

The `_demo/py/` directory contains some python related demos:

* [basic](_demo/py/basic/main.go): call Python math, statistics, variadic builtin, iterator, and print APIs
* [scientific](_demo/py/matrix/matrix.go): convert nested lists through NumPy and PyTorch

To run these demos (If you haven't installed `llgo` yet, please refer to [How to install](#how-to-install)):

```sh
cd <demo-directory>  # eg. cd _demo/py/basic
llgo run .
```

## Dependencies

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

## How to install

Follow these steps to install the `llgo` command, whose usage is similar to the `go` command:

### on macOS

<!-- embedme doc/_readme/scripts/install_macos.sh#L2-L1000 -->

```sh
brew update
brew install llvm@22 lld@22 bdw-gc openssl cjson libffi libuv pkg-config
brew install python@3.12 # optional
brew link --force --overwrite llvm@22 lld@22 libffi
# curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
./install.sh
```

Homebrew's versioned LLVM 22 formula does not ship LLDB and there is no
`lldb@22` formula. LLGo checks common Homebrew and system locations, then
`lldb` on `PATH`; use `LLGO_LLDB` or `llgo lldb -lldb` to select one explicitly.

### on Linux

#### Debian/Ubuntu

<!-- embedme doc/_readme/scripts/install_ubuntu.sh#L2-L1000 -->

```sh
echo "deb http://apt.llvm.org/$(lsb_release -cs)/ llvm-toolchain-$(lsb_release -cs)-22 main" | sudo tee /etc/apt/sources.list.d/llvm.list
wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y llvm-22-dev clang-22 libclang-22-dev lld-22 lldb-22 libunwind-22-dev libc++-22-dev pkg-config libgc-dev libssl-dev zlib1g-dev libffi-dev libcjson-dev libsqlite3-dev libuv1-dev
sudo apt-get install -y python3.12-dev # optional
#curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
./install.sh
```

#### Alpine Linux

```sh
apk add go llvm22-dev clang22-dev lld22 lldb pkgconf gc-dev libunwind-dev openssl-dev zlib-dev libffi-dev cjson-dev sqlite-dev libuv-dev
apk add python3-dev # optional
apk add g++ # build only
export LLVM_CONFIG=/usr/lib/llvm22/bin/llvm-config
export CGO_CPPFLAGS="$($LLVM_CONFIG --cppflags)"
export CGO_CXXFLAGS=-std=c++17
export CGO_LDFLAGS="$($LLVM_CONFIG --ldflags) $($LLVM_CONFIG --libs all)"
curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
```

#### Fedora Linux 44 or newer

Fedora 44 and 45 ship LLVM 22 as the default LLVM stack. Fedora 43 still
ships LLVM 21 and is not a supported default-toolchain environment for this
LLGo release.

```sh
sudo dnf install -y llvm-devel clang-devel lld lldb libcxx-devel llvm-libunwind-devel \
  pkgconf-pkg-config gc-devel openssl-devel libffi-devel libuv-devel \
  cjson-devel sqlite-devel zlib-ng-compat-devel
llvm-config --version # must report 22.x
```

docker alpine 386 llgo environment
```
export GCC_ROOT_DIR=$(gcc -print-search-dirs | grep 'install:' | awk -F': ' '{print $2}')
export LDFLAGS="-L$GCC_ROOT_DIR -B$GCC_ROOT_DIR -Wl,-dynamic-linker,/lib/ld-musl-i386.so.1"
llgo run .
```

### on Windows

The release workflow builds four integrated Windows archives:
`llgo<VERSION>.windows-{amd64,arm64}-{msvc,mingw}.tar.gz`. Check the
[release assets](https://github.com/xgo-dev/llgo/releases) for availability in
each version. Add the extracted `bin` directory to `PATH`; the archives keep
the same `runtime`, `targets`, and `crosscompile/clang` layout as Unix releases.
The MSVC compiler links LLVM statically; MinGW archives include the native
LLVM and C++ DLL dependencies beside `llgo.exe`.

Use the archive matching your native architecture and toolchain profile. Native
programs still need the corresponding SDK/CRT, Clang, and dependencies described
below: Visual Studio's C++ developer environment for MSVC, or MSYS2 `CLANG64`
(`amd64`) / `CLANGARM64` (`arm64`) for MinGW. The bundled ESP Clang remains the
upstream x64 Windows payload, including in ARM64 archives, and runs through
Windows' x64 emulation there; `llgo.exe` itself is native ARM64 in those archives.

The recommended GNU-hosted setup is an MSYS2 `CLANG64` shell. Install the LLVM
22 stack and LLGo's native dependencies, then provide the versioned pkg-config
metadata used by the Go/C++ bindings:

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

The native MSVC CI profile uses LLVM's official 22.1.8 development archive for
headers, libraries, Clang, LLD, and all code-generation backends. That archive
does not contain LLDB, so the profile also extracts LLDB from the matching
official `win64` or `woa64` installer. LLVM 22 has no official Win32 installer;
the Windows 386 lane uses the LLVM 22.1.8-based llvm-mingw 20260616 builtins and
qualifies the official x64 LLDB under WoW64. The complete pinned setup,
checksums, and generated `llvm-22.pc` are in
`.github/actions/setup-deps/action.yml`.

### Install from source

<!-- embedme doc/_readme/scripts/install_llgo.sh#L2-L1000 -->

```sh
git clone https://github.com/xgo-dev/llgo.git
cd llgo
./install.sh
```

## Development tools

* [pydump](_xtool/pydump): It is the first production program compiled with `llgo` rather than `go`. It outputs symbol information (functions, variables, and constants) from a Python library in JSON format, preparing for the generation of corresponding packages in `llgo`.
* [pysigfetch](https://github.com/goplus/hdq/tree/main/chore/pysigfetch): It generates symbol information by extracting information from Python's documentation site. This tool is not part of the `llgo` project, but we depend on it.
* [llpyg](chore/llpyg): It is used to automatically convert Python libraries into Go packages that `llgo` can import. It depends on `pydump` and `pysigfetch` to accomplish the task.
* [llgen](chore/llgen): It is used to compile Go packages into LLVM IR files (*.ll).
* [gentests](chore/gentests): It refreshes runtime-output and package-metadata golden data under `cl/_test*`. LLVM IR checks live in Go sources as `// LITTEST` FileCheck directives.
* [litgen](chore/litgen): It maintains explicitly opted-in, source-embedded FileCheck snapshots. It supports function/global selection, update-only operation, stale-check verification, and stable LLVM value abstractions. Small handwritten checks remain manual.
* [ssadump](chore/ssadump): It is a Go SSA builder and interpreter.

For local workflows and test-golden refresh commands, see [dev/README.md](dev/README.md#6-refresh-test-goldens).

How do I generate these tools?

<!-- embedme doc/_readme/scripts/install_full.sh#L2-L1000 -->

```sh
git clone https://github.com/xgo-dev/llgo.git
cd llgo
go install -v ./cmd/...
go install -v ./chore/...  # compile all tools except pydump
export LLGO_ROOT=$PWD
cd _xtool
llgo install ./...   # compile pydump
go install github.com/goplus/hdq/chore/pysigfetch@v0.8.1  # compile pysigfetch
```

## Key modules

Below are the key modules for understanding the implementation principles of `llgo`:

* [ssa](https://pkg.go.dev/github.com/xgo-dev/llgo/ssa): It generates LLVM IR files (LLVM SSA) using the semantics and interfaces of Go SSA. Although `LLVM SSA` and `Go SSA` are both IR languages, they work at completely different levels. `LLVM SSA` is closer to machine code and abstracts over different instruction sets, while `Go SSA` is closer to a high-level language. We can think of it as the instruction set of the `Go computer`. `llgo/ssa` is not limited to the `llgo` compiler. If we view it as providing the high-level expressive power of `LLVM`, it is very useful. Its advanced SSA form lets clients use LLVM without operating directly on machine-code semantics.
* [cl](https://pkg.go.dev/github.com/xgo-dev/llgo/cl): It is the core of the llgo compiler. It converts a Go package into LLVM IR files. It depends on `llgo/ssa`.
* [internal/build](https://pkg.go.dev/github.com/xgo-dev/llgo/internal/build): It strings together the entire compilation process of `llgo`. It depends on `llgo/ssa` and `llgo/cl`.

LLGo - A Go compiler based on LLVM
=====

[![Build Status](https://github.com/xgo-dev/llgo/actions/workflows/go.yml/badge.svg)](https://github.com/xgo-dev/llgo/actions/workflows/go.yml)
[![GitHub release](https://img.shields.io/github/v/tag/xgo-dev/llgo.svg?label=release)](https://github.com/xgo-dev/llgo/releases)
[![Coverage Status](https://codecov.io/gh/xgo-dev/llgo/branch/main/graph/badge.svg)](https://codecov.io/gh/xgo-dev/llgo)
[![LLGo bindings](https://img.shields.io/badge/llarhub-LLGo_Bindings-brightgreen.svg)](https://github.com/xgo-dev/llgo/blob/main/doc/LLGo-Bindings.md)
[![Benchmark](https://img.shields.io/badge/benchmark-LLGo_vs_Go-yellowgreen.svg)](https://llgo-test.github.io/benchmarks/)
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

See [compatibility coverage](CONTRIBUTING.md#choose-tests-for-the-change) for
validation details and recorded differences from the standard Go compiler.

### Runtime

LLGo uses a different runtime from the standard Go toolchain. Native goroutines map 1:1 to OS threads with fixed native stacks, so direct C calls require no Go-to-C stack or scheduler transition, avoiding the cgo overhead that makes frequent C calls costly in standard Go.

The default garbage collector is conservative [BDWGC](https://www.hboehm.info/gc/) (also known as libgc). Bare-metal embedded targets instead use a TinyGo-derived conservative mark-and-sweep collector.

Garbage collection can be disabled with the `nogc` build tag. For example:

```sh
llgo run -tags nogc .
```

### Standard libraries

LLGo fully supports the Go standard library on supported native platforms.
See [standard-library compatibility](test/std/README.md) for coverage details.

Other targets may not provide every OS service or implementation-specific runtime behavior.

| Target | Current coverage |
| --- | --- |
| Native | Linux amd64/arm64, macOS amd64/arm64, and Windows 386/amd64/arm64 (MSVC and MinGW) release builds |
| WebAssembly | `js/wasm` and `wasip1/wasm` builds; WASI and Emscripten profiles |
| Embedded | [`-target`](doc/Embedded_Cmd.md) configurations for supported boards and MCUs, with QEMU/emulator support for selected targets |

The supported WebAssembly profiles combine a memory ABI with a host ABI.
`-target emscripten` emits a Memory32 ES module and its sibling wasm module,
`-target emscripten-memory64` emits the corresponding Memory64 pair, and
`-target wasi` emits a Memory32 WASI Preview 1 module. Raw
`GOOS=js GOARCH=wasm` and `GOOS=wasip1 GOARCH=wasm` select the Go-compatible
JavaScript and WASI host contracts. See the [WebAssembly proposal](doc/wasm-proposal.md)
for the compatibility contract and acceptance plan.

WebAssembly goroutine stacks have fixed sizes. For deeply recursive workloads,
set `-goroutine-stack-size=2MB` when building (the existing `-pthread-stack-size`
flag remains an alias). This sets both the goroutine stack and its Asyncify
buffer; it does not change the main goroutine's stack. JavaScript-hosted builds
check stack bounds and abort on overflow instead of corrupting linear memory.
These checks remain enabled in optimized builds: checking each stack-pointer
update adds code and execution overhead, but a return-time cookie check alone
cannot prevent a goroutine from overwriting adjacent memory.


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

Or organize such bindings into a package, as [c/math](https://github.com/goplus/lib/tree/main/c/math) does:

<!-- embedme doc/_readme/llgo_call_cmath/call_cmath.go -->

```go
package main

import "github.com/goplus/lib/c/math"

func main() {
	println("sqrt(2) =", math.Sqrt(2))
}
```

Because calls into C compile to native calls against the C ABI, there is no Go-to-C stack or scheduler transition, so frequent C calls stay cheap.

#### Promoted C variadic methods

Forwarding an incoming C ellipsis through a promoted-method wrapper, including
an interface call, is currently supported only on hosted native targets without
an explicit `-target` profile. The underlying C method must have a pointer
receiver. The wrapper and callee must have matching fixed LLVM prototypes and
calling conventions, with no result conversion. Fixed parameters must be
scalars, with zero or one scalar result: booleans, integers, floating-point
values, and pointers qualify.

Value receivers, aggregate fixed parameters or results, multiple results,
Wasm, and explicit `-target` profiles (including custom native profiles) remain
outside this support. Selecting an unsupported forwarding wrapper through an
interface conversion, method expression, or bound method produces a compiler
error naming the method, target, and reason forwarding is unsupported. Call the C-linked method
or function directly with explicit C-compatible arguments when forwarding is
unavailable. Unused synthetic wrappers do not prevent those direct calls.

#### Windows API

On Windows, bind APIs declared with `WINAPI` or `__stdcall` through the `stdcall.` namespace. The convention is distinct on 386; Windows amd64 and arm64 use their unified native C ABI. An explicitly decorated 386 name such as `_MessageBoxW@16` is also accepted and is normalized to `MessageBoxW` on 64-bit targets.

```go
//go:linkname MessageBoxW stdcall.MessageBoxW
func MessageBoxW(hwnd uintptr, text, caption *uint16, flags uint32) int32

//llgo:stdcall
type Callback func(context uintptr) uintptr
```

`stdcall.` declarations and `//llgo:stdcall` apply only to non-variadic function types. A native callback is one function pointer, so a Go callback must be a direct function reference; pass state through an explicit context pointer rather than a capturing closure.

`//llgo:cdecl` and `//llgo:stdcall` function values can be passed directly to `syscall.NewCallback`, `syscall.NewCallbackCDecl`, and `runtime.SetFinalizer`. On Windows, the type's convention governs calls into the function; independently, `NewCallback` creates a stdcall entry and `NewCallbackCDecl` creates a cdecl entry.

#### Calling-convention directives

A function type that models a C or Windows callback carries a calling-convention directive so the generated function pointer matches the native ABI:

* `//llgo:cdecl` selects the C calling convention.
* `//llgo:stdcall` selects the Windows stdcall convention (distinct only on 386; Windows amd64 and arm64 use their unified native C ABI).

```go
import (
	"unsafe"

	"github.com/goplus/lib/c"
)

//llgo:cdecl
type Handler func(data unsafe.Pointer) c.Int

//llgo:stdcall
type WinProc func(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr
```

The older `//llgo:type C` and `//llgo:type stdcall` spellings remain supported for backward compatibility but are **deprecated**: the compiler accepts them and prints a warning naming the replacement (`//llgo:type C` → `//llgo:cdecl`, `//llgo:type stdcall` → `//llgo:stdcall`). Prefer the new directives in new code. A type may carry only one calling-convention directive; combining them (for example `//llgo:cdecl` together with `//llgo:stdcall` or `//llgo:type stdcall`) is reported as an error.

#### Host-layout structs

Use a leading blank `structs.HostLayout` field for a struct exchanged with native
code. LLGo recognizes the marker from type information, including aliases,
defined marker types, and generic instantiations.

```go
import "structs"

//llgo:cdecl
type Callback func(int32) int32

type Record struct {
    _        structs.HostLayout
    Callback Callback
    Data     *byte
}
```

The marker selects native field layout, not a calling convention. A plain Go
`func` field still carries its closure environment; use an explicitly annotated
function type for a native callback. Nested struct fields keep their own layout,
and containing structs do not inherit the marker. Ordinary Go fields retain
their storage representation, including when their address is passed to another
function. On profiled wasm32 targets, Go pointers and `uintptr` still occupy
eight bytes; explicitly annotated callback fields also retain their existing
Go storage width. HostLayout does not change callback reflection or conversion
semantics. Host records requiring four-byte address slots must use explicit
fixed-width storage and convert at the interface boundary. A nonempty marked
struct ending in a zero-sized field retains Go's protective trailing padding.

The older struct-level `//llgo:cdecl` and `//llgo:type C` declarations remain
accepted for compatibility. A struct that still carries a legacy type directive
keeps its legacy source-layout behavior, including when a HostLayout marker is
added directly or through a field. Remove the struct directive when adopting
HostLayout. When migrating, give any native callback fields
explicit callback types rather than relying on the outer struct's directive.
If Go and native pointer widths differ, use target-specific address/size slots
and boundary accessors, retaining Go references for any addresses the host saves.
The marker is a real zero-sized field: reflection field indices and positional
struct literals must be adjusted when adding it.
The runtime uses an internal compatibility marker on Go 1.20–1.22 and an alias
to the standard `structs.HostLayout` on Go 1.23 and later.

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
* [c/cjson](https://pkg.go.dev/github.com/llarhub/cjson)
* [c/clang](https://pkg.go.dev/github.com/goplus/lib/c/clang)
* [c/ffi](https://pkg.go.dev/github.com/goplus/lib/c/ffi)
* [c/libuv](https://pkg.go.dev/github.com/goplus/lib/c/libuv)
* [c/llama2](https://pkg.go.dev/github.com/goplus/lib/c/llama2)
* [c/lua](https://pkg.go.dev/github.com/goplus/lib/c/lua)
* [c/neco](https://pkg.go.dev/github.com/goplus/lib/c/neco)
* [c/openssl](https://pkg.go.dev/github.com/goplus/lib/c/openssl)
* [c/raylib](https://pkg.go.dev/github.com/llarhub/raylib)
* [c/sqlite](https://pkg.go.dev/github.com/llarhub/sqlite)
* [c/zlib](https://pkg.go.dev/github.com/goplus/lib/c/zlib)
* [cpp/inih](https://pkg.go.dev/github.com/goplus/lib/cpp/inih)
* [cpp/llvm](https://pkg.go.dev/github.com/goplus/lib/cpp/llvm)

Examples built on these bindings:

* [llama2-c](_demo/c/llama2-c): inference Llama 2 (the first LLGo AI example)
* [mkjson](https://github.com/llarhub/cjson/tree/main/demo/mkjson): create a JSON object and print it
* [sqlitedemo](https://github.com/llarhub/sqlite/tree/main/demo/sqlitedemo): a basic SQLite demo
* [tetris](https://github.com/llarhub/raylib/tree/main/demo/tetris): a Tetris game based on raylib


## Python support

You can import a Python library in LLGo!

You can import Python libraries into `llgo` through [llpyg](https://github.com/xgo-dev/llpyg). Available bindings include:

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

## How to install

### Quick install

**macOS and Linux:**

<!-- embedme doc/_readme/scripts/quick_install.sh#L3-L1000 -->

```sh
curl -fsSL https://raw.githubusercontent.com/xgo-dev/llgo/main/install.sh | bash
```

**Windows (PowerShell):**

<!-- embedme doc/_readme/scripts/quick_install.ps1#L2-L2 -->

```powershell
irm https://raw.githubusercontent.com/xgo-dev/llgo/main/install.ps1 | iex
```

Open a new terminal after installation, then verify:

<!-- embedme doc/_readme/scripts/verify_install.sh#L3-L1000 -->

```sh
llgo version
```

### Manual installation

#### Homebrew (macOS)

<!-- embedme doc/_readme/scripts/install_brew.sh#L3-L1000 -->

```sh
brew install llgo
```

#### WinGet (Windows)

Until `XGo.LLGo` is published, use the legacy package below. See [WinGet migration](WINDOWS.md#migrating-an-older-winget-installation) when the unified package becomes available.

<!-- embedme doc/_readme/scripts/install_winget.ps1#L3-L3 -->

```powershell
winget install XGo.LLGo.MSVC
```

#### APT / YUM (Linux)

For Debian/Ubuntu without LLVM 23 packages, first enable the matching [apt.llvm.org repository](https://apt.llvm.org/). RPM installation requires Fedora 44 or newer. These repository entries use HTTPS and disable package signature checks (`trusted=yes` / `gpgcheck=0`).

Debian/Ubuntu:

<!-- embedme doc/_readme/scripts/install_apt.sh#L3-L1000 -->

```sh
echo 'deb [trusted=yes] https://pkgs.xgo.dev/apt/ /' | sudo tee /etc/apt/sources.list.d/goplus.list
sudo apt-get update
sudo apt-get install -y llgo
```

Fedora/RPM:

<!-- embedme doc/_readme/scripts/install_yum.sh#L3-L1000 -->

```sh
sudo tee /etc/yum.repos.d/goplus.repo <<'EOF'
[goplus]
name=XGo
baseurl=https://pkgs.xgo.dev/yum/
enabled=1
gpgcheck=0
EOF
sudo yum install -y llgo
```

For source builds and Pixi, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Go environment

Use `llgo env GOOS GOARCH` to query Go environment values and `llgo env LLGO_ROOT LLGO_RUNTIME_DIR LLGO_LLVM_VERSION` to inspect the LLGo installation and selected LLVM toolchain. With no variable names it prints both sets. `-json` provides machine-readable output, and `-target name` adds the resolved `LLGO_TARGET_*` fields for an embedded target without downloading tools or building libraries. For `LLGO_*` fields, `-changed` includes only values explicitly set in the process environment and omits derived values; Go fields retain the underlying `go env -changed` behavior. Paths for optional packaged tools are empty when the payload is not installed.

Go-only queries are delegated to the underlying Go toolchain and respect its environment and configuration. The `-w` and `-u` flags update the same GOENV configuration as `go env`; LLGo fields are read-only.

A real Go executable must remain in an absolute PATH directory. If `go` is a symlink or hard link to `llgo`, the environment command skips that link and searches the remaining PATH entries instead of invoking itself recursively.

## Module management

Use `llgo mod init`, `llgo mod tidy`, `llgo mod vendor`, and other `go mod` subcommands to manage modules with the underlying Go toolchain. Arguments, working directory, environment, standard streams, and exit status are preserved. As with `llgo env`, a real Go executable must remain in an absolute PATH directory; links from `go` back to LLGo are skipped to prevent recursion.

Module maintenance does not inject LLGo-specific build tags. Go's `tidy` and `vendor` already consider all build tags except `ignore`, including dependencies imported only by files with `//go:build llgo`.

## Package and project commands

`llgo list` provides a Go-compatible package query backed by the real Go toolchain while adding LLGo's source-selection build tags to package queries. Module queries keep caller-supplied and explicit target tags but do not add LLGo's defaults. It supports the list queries used by `golang.org/x/tools/go/packages`, including `list -f '{{context.GOARCH}} {{context.Compiler}}' -- unsafe`; the reported `gc` context identifies Go frontend and type-size compatibility, not LLGo's LLVM backend. Use `-target name` to apply a resolved LLGo target's GOOS, GOARCH, and build tags without installing its toolchain. When `go/packages` requests `-export=true`, paths in the `Export` field refer to Go frontend type data in Go's build cache, not LLGo package archives.

`llgo get`, `llgo fix`, `llgo generate`, and `llgo vet` use the Go toolchain with LLGo's build tags and accept `-target` to select the target GOOS, GOARCH, and tags. `llgo fmt` and `llgo work` forward arguments unchanged; `go fmt` already includes files excluded by build constraints.

`build`, `clean`, `install`, `run`, `test`, `version`, and `tool compile` remain LLGo-specific. The Go `bug`, `doc`, and `telemetry` commands are not exposed: the first and last operate Go services, while `go doc` cannot select LLGo-only sources because it has no build-tag option.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development environments, source
builds, repository structure, testing, and the pull-request workflow.

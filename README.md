llgo - A Go compiler based on LLVM
=====

[![Build Status](https://github.com/xgo-dev/llgo/actions/workflows/go.yml/badge.svg)](https://github.com/xgo-dev/llgo/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/xgo-dev/llgo)](https://goreportcard.com/report/github.com/xgo-dev/llgo)
[![GitHub release](https://img.shields.io/github/v/tag/xgo-dev/llgo.svg?label=release)](https://github.com/xgo-dev/llgo/releases)
[![Coverage Status](https://codecov.io/gh/xgo-dev/llgo/branch/main/graph/badge.svg)](https://codecov.io/gh/xgo-dev/llgo)
[![GoDoc](https://pkg.go.dev/badge/github.com/xgo-dev/llgo.svg)](https://pkg.go.dev/github.com/xgo-dev/llgo)
[![Language](https://img.shields.io/badge/language-XGo-blue.svg)](https://github.com/goplus/xgo)

LLGo is a Go compiler based on LLVM in order to better integrate Go with the C ecosystem including Python and JavaScript. It's a subproject of [the XGo project](https://github.com/goplus/gop).

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

LLGo is compatible with C ecosystem through the language's **Application Binary Interface (ABI)**, while LLGo is compatible with Go through its **syntax (source code)**. C ecosystem includes all languages that are ABI compatible with C (eg. C/C++, Python, JavaScript, Objective-C, Swift, etc).


## C/C++ standard libary support

You can import a C/C++ standard library in LLGo!

* [c](https://pkg.go.dev/github.com/goplus/lib/c)
* [c/syscall](https://pkg.go.dev/github.com/goplus/lib/c/syscall)
* [c/sys](https://pkg.go.dev/github.com/goplus/lib/c/sys)
* [c/os](https://pkg.go.dev/github.com/goplus/lib/c/os)
* [c/math](https://pkg.go.dev/github.com/goplus/lib/c/math)
* [c/math/cmplx](https://pkg.go.dev/github.com/goplus/lib/c/math/cmplx)
* [c/math/rand](https://pkg.go.dev/github.com/goplus/lib/c/math/rand)
* [c/pthread](https://pkg.go.dev/github.com/goplus/lib/c/pthread)
* [c/pthread/sync](https://pkg.go.dev/github.com/goplus/lib/c/pthread/sync)
* [c/sync/atomic](https://pkg.go.dev/github.com/goplus/lib/c/sync/atomic)
* [c/time](https://pkg.go.dev/github.com/goplus/lib/c/time)
* [c/net](https://pkg.go.dev/github.com/goplus/lib/c/net)
* [cpp/std](https://pkg.go.dev/github.com/goplus/lib/cpp/std)

Here is a simple example:

<!-- embedme doc/_readme/llgo_simple/simple.go -->

```go
package main

import "github.com/goplus/lib/c"

func main() {
	c.Printf(c.Str("Hello world\n"))
}
```

This is a simple example of calling the C `printf` function to print `Hello world`. Here, `c.Str` is not a function for converting a Go string to a C string, but a built-in instruction supported by `llgo` for generating a C string constant.

The `_demo` directory contains some C standard libary related demos (it start with `_` to prevent the `go` command from compiling it):

* [hello](_demo/c/hello/hello.go): call C `printf` to print `Hello world`
* [concat](_demo/c/concat/concat.go): call C `fprintf` with `stderr`
* [qsort](_demo/c/qsort/qsort.go): call C function with a callback (eg. `qsort`)

To run these demos (If you haven't installed `llgo` yet, please refer to [How to install](#how-to-install)):

```sh
cd <demo-directory>  # eg. cd _demo/c/hello
llgo run .
```


## How to support C/C++ and Python

LLGo use `go:linkname` to link an extern symbol througth its ABI:

<!-- embedme doc/_readme/llgo_call_c/call_c.go#L3-L6 -->

```go
import _ "unsafe" // for go:linkname

//go:linkname Sqrt C.sqrt
func Sqrt(x float64) float64
```

You can directly integrate it into [your own code](_demo/c/linkname/linkname.go):

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

Or put it into a package (see [c/math](https://github.com/goplus/lib/tree/main/c/math/math.go)):

<!-- embedme doc/_readme/llgo_call_cmath/call_cmath.go -->

```go
package main

import "github.com/goplus/lib/c/math"

func main() {
	println("sqrt(2) =", math.Sqrt(2))
}
```


## Python support

You can import a Python library in LLGo!

And you can import any Python library into `llgo` through a program called `llpyg` (see [Development tools](#development-tools)). The following libraries have been included in `llgo`:

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

Note: For third-party libraries (such as pandas and pytorch), you still need to install the library files.

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

* [callpy](_demo/py/callpy/callpy.go): call Python standard library function `math.sqrt`
* [pi](_demo/py/pi/pi.go): print python constants `math.pi`
* [statistics](_demo/py/statistics/statistics.go): define a python list and call `statistics.mean` to get the mean
* [matrix](_demo/py/matrix/matrix.go): a basic `numpy` demo

To run these demos (If you haven't installed `llgo` yet, please refer to [How to install](#how-to-install)):

```sh
cd <demo-directory>  # eg. cd _demo/py/callpy
llgo run .
```


## Other frequently used libraries

LLGo can easily import any libraries from the C ecosystem. Currently, this import process is still manual, but in the future, it will be automated similar to Python library imports.

The currently supported libraries include:

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

Here are some examples related to them:

* [llama2-c](_demo/c/llama2-c): inference Llama 2 (It's the first llgo AI example)
* [mkjson](https://github.com/goplus/lib/tree/main/c/cjson/_demo/mkjson/mkjson.go): create a json object and print it
* [sqlitedemo](https://github.com/goplus/lib/tree/main/c/sqlite/_demo/sqlitedemo/demo.go): a basic sqlite demo
* [tetris](https://github.com/goplus/lib/tree/main/c/raylib/_demo/tetris/tetris.go): a tetris game based on raylib


## Go syntax support

All Go syntax (including `cgo`) is already supported. Here are some examples:

* [concat](_demo/c/concat/concat.go): define a variadic function
* [genints](_demo/c/genints/genints.go): various forms of closure usage (including C function, recv.method and anonymous function)
* [errors](_cmptest/errors/errors.go): demo to implement error interface
* [defer](_cmptest/defer/defer.go): defer demo
* [goroutine](_demo/go/goroutine/goroutine.go): goroutine demo


### Defer

LLGo now supports `defer` within loops, matching Go's semantics of executing defers in LIFO order for every iteration. The usual caveat from Go still applies: be mindful of loop-heavy defer usage because it allocates per iteration.


### Garbage Collection (GC)

By default, LLGo implements `gc` based on [bdwgc](https://www.hboehm.info/gc/) (also known as [libgc](https://www.hboehm.info/gc/)).

However, you can disable gc by specifying the `nogc` tag. For example:

```sh
llgo run -tags nogc .
```


## Go packages support

Here are the Go packages that can be imported correctly:

* [unsafe](https://pkg.go.dev/unsafe)
* [unicode](https://pkg.go.dev/unicode)
* [unicode/utf8](https://pkg.go.dev/unicode/utf8)
* [unicode/utf16](https://pkg.go.dev/unicode/utf16)
* [math](https://pkg.go.dev/math)
* [math/big](https://pkg.go.dev/math/big) (partially)
* [math/bits](https://pkg.go.dev/math/bits)
* [math/cmplx](https://pkg.go.dev/math/cmplx)
* [math/rand](https://pkg.go.dev/math/rand)
* [net/url](https://pkg.go.dev/net/url)
* [errors](https://pkg.go.dev/errors)
* [context](https://pkg.go.dev/context)
* [io](https://pkg.go.dev/io)
* [io/fs](https://pkg.go.dev/io/fs)
* [io/ioutil](https://pkg.go.dev/io/ioutil)
* [log](https://pkg.go.dev/log)
* [flag](https://pkg.go.dev/flag)
* [sort](https://pkg.go.dev/sort)
* [bytes](https://pkg.go.dev/bytes)
* [bufio](https://pkg.go.dev/bufio)
* [strings](https://pkg.go.dev/strings)
* [strconv](https://pkg.go.dev/strconv)
* [path](https://pkg.go.dev/path)
* [path/filepath](https://pkg.go.dev/path/filepath)
* [sync/atomic](https://pkg.go.dev/sync/atomic)
* [sync](https://pkg.go.dev/sync) (partially)
* [syscall](https://pkg.go.dev/syscall) (partially)
* [runtime](https://pkg.go.dev/runtime) (partially)
* [os](https://pkg.go.dev/os) (partially)
* [os/exec](https://pkg.go.dev/os/exec) (partially)
* [fmt](https://pkg.go.dev/fmt) (partially)
* [reflect](https://pkg.go.dev/reflect) (partially)
* [time](https://pkg.go.dev/time) (partially)
* [encoding](https://pkg.go.dev/encoding)
* [encoding/binary](https://pkg.go.dev/encoding/binary)
* [encoding/hex](https://pkg.go.dev/encoding/hex)
* [encoding/base32](https://pkg.go.dev/encoding/base32)
* [encoding/base64](https://pkg.go.dev/encoding/base64)
* [encoding/csv](https://pkg.go.dev/encoding/csv)
* [net/textproto](https://pkg.go.dev/net/textproto)
* [hash](https://pkg.go.dev/hash)
* [hash/adler32](https://pkg.go.dev/hash/adler32)
* [hash/crc32](https://pkg.go.dev/hash/crc32) (partially)
* [hash/crc64](https://pkg.go.dev/hash/crc64)
* [hash/maphash](https://pkg.go.dev/hash/maphash) (partially)
* [crypto](https://pkg.go.dev/crypto)
* [crypto/md5](https://pkg.go.dev/crypto/md5)
* [crypto/sha1](https://pkg.go.dev/crypto/sha1)
* [crypto/sha256](https://pkg.go.dev/crypto/sha256)
* [crypto/sha512](https://pkg.go.dev/crypto/sha512) (partially)
* [crypto/hmac](https://pkg.go.dev/crypto/hmac) (partially)
* [crypto/rand](https://pkg.go.dev/crypto/rand) (partially)
* [crypto/subtle](https://pkg.go.dev/crypto/subtle) (partially)
* [regexp](https://pkg.go.dev/regexp)
* [regexp/syntax](https://pkg.go.dev/regexp/syntax)
* [go/token](https://pkg.go.dev/go/token)
* [go/scanner](https://pkg.go.dev/go/scanner)
* [go/parser](https://pkg.go.dev/go/parser)


## Dependencies

- [Go 1.21+](https://go.dev)
- [LLVM 18](https://llvm.org)
- [Clang 18](https://clang.llvm.org)
- [LLD 18](https://lld.llvm.org)
- [pkg-config 0.29+](https://www.freedesktop.org/wiki/Software/pkg-config/)
- [bdwgc/libgc 8.0+](https://www.hboehm.info/gc/)
- [OpenSSL 3.0+](https://www.openssl.org/)
- [zlib 1.2+](https://www.zlib.net)
- [Python 3.12+](https://www.python.org) (optional, for [github.com/goplus/lib/py](https://pkg.go.dev/github.com/goplus/lib/py))

## How to install

Follow these steps to generate the `llgo` command (its usage is the same as the `go` command):

### on macOS

<!-- embedme doc/_readme/scripts/install_macos.sh#L2-L1000 -->

```sh
brew update
brew install llvm@19 lld@19 bdw-gc openssl cjson libffi libuv pkg-config
brew install python@3.12 # optional
brew link --overwrite llvm@19 lld@19 libffi
# curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
./install.sh
```

### on Linux

#### Debian/Ubuntu

<!-- embedme doc/_readme/scripts/install_ubuntu.sh#L2-L1000 -->

```sh
echo "deb http://apt.llvm.org/$(lsb_release -cs)/ llvm-toolchain-$(lsb_release -cs)-19 main" | sudo tee /etc/apt/sources.list.d/llvm.list
wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y llvm-19-dev clang-19 libclang-19-dev lld-19 libunwind-19-dev libc++-19-dev pkg-config libgc-dev libssl-dev zlib1g-dev libcjson-dev libsqlite3-dev libuv1-dev
sudo apt-get install -y python3.12-dev # optional
#curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
./install.sh
```

#### Alpine Linux

```sh
apk add go llvm19-dev clang19-dev lld19 pkgconf gc-dev libunwind-dev openssl-dev zlib-dev
apk add python3-dev # optional
apk add g++ # build only
export LLVM_CONFIG=/usr/lib/llvm19/bin/llvm-config
export CGO_CPPFLAGS="$($LLVM_CONFIG --cppflags)"
export CGO_CXXFLAGS=-std=c++17
export CGO_LDFLAGS="$($LLVM_CONFIG --ldflags) $($LLVM_CONFIG --libs all)"
curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
```

docker alpine 386 llgo environment
```
export GCC_ROOT_DIR=$(gcc -print-search-dirs | grep 'install:' | awk -F': ' '{print $2}')
export LDFLAGS="-L$GCC_ROOT_DIR -B$GCC_ROOT_DIR -Wl,-dynamic-linker,/lib/ld-musl-i386.so.1"
llgo run .
```

### on Windows

TODO

### Install from source

<!-- embedme doc/_readme/scripts/install_llgo.sh#L2-L1000 -->

```sh
git clone https://github.com/xgo-dev/llgo.git
cd llgo
./install.sh
```

## Development tools

* [pydump](_xtool/pydump): It's the first program compiled by `llgo` (NOT `go`) in a production environment. It outputs symbol information (functions, variables, and constants) from a Python library in JSON format, preparing for the generation of corresponding packages in `llgo`.
* [pysigfetch](https://github.com/goplus/hdq/tree/main/chore/pysigfetch): It generates symbol information by extracting information from Python's documentation site. This tool is not part of the `llgo` project, but we depend on it.
* [llpyg](chore/llpyg): It is used to automatically convert Python libraries into Go packages that `llgo` can import. It depends on `pydump` and `pysigfetch` to accomplish the task.
* [llgen](chore/llgen): It is used to compile Go packages into LLVM IR files (*.ll).
* [gentests](chore/gentests): It refreshes the built-in golden test data under `cl/_test*`, including `out.ll` and `expect.txt`. Directories that use source-embedded `// LITTEST` checks are skipped for `out.ll` regeneration.
* [litgen](chore/litgen): It generates and refreshes source-embedded `// LITTEST` FileCheck directives from the current LLVM IR for marked Go source files.
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

* [ssa](https://pkg.go.dev/github.com/xgo-dev/llgo/ssa): It generates LLVM IR files (LLVM SSA) using the semantics (interfaces) of Go SSA. Although `LLVM SSA` and `Go SSA` are both IR languages, they work at completely different levels. `LLVM SSA` is closer to machine code, which abstracts different instruction sets. While `Go SSA` is closer to a high-level language. We can think of it as the instruction set of the `Go computer`. `llgo/ssa` is not just limited to the `llgo` compiler. If we view it as the high-level expressive power of `LLVM`, you'll find it very useful. Prior to `llgo/ssa`, you had to operate `LLVM` using machine code semantics. But now, with the advanced SSA form (in the semantics of Go SSA), you can conveniently utilize `LLVM`.
* [cl](https://pkg.go.dev/github.com/xgo-dev/llgo/cl): It is the core of the llgo compiler. It converts a Go package into LLVM IR files. It depends on `llgo/ssa`.
* [internal/build](https://pkg.go.dev/github.com/xgo-dev/llgo/internal/build): It strings together the entire compilation process of `llgo`. It depends on `llgo/ssa` and `llgo/cl`.

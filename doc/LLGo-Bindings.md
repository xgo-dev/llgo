# LLGo Bindings

LLGo is a Go compiler based on LLVM that integrates Go with the C ecosystem. Besides importing ordinary Go packages, LLGo can import C/C++ libraries and Python libraries through *bindings*: Go packages that map directly onto the C ABI. You use a binding the same way you use any other Go package: `go get` it (or let `go.mod` resolve it), `import` it, and build with `llgo`.

This page lists the bindings that are currently available.

- [C/C++ Bindings](#cc-bindings)
- [Python Bindings](#python-bindings)

---

## C/C++ Bindings

All C/C++ bindings live in the [LLAR Hub](https://github.com/llarhub) organization ("C/C++ Library Hub for XGo/LLGo"), one repository per library. Each repository is its own Go module, so the import path is simply `github.com/llarhub/<name>`:

```
go get github.com/llarhub/cjson
```

Before building, install the underlying C/C++ library (and `pkg-config`) on your machine; each binding's README explains the exact packages for macOS, Linux and Windows, and which upstream library version the binding was generated from. Most bindings are generated with [llcppg](https://github.com/xgo-dev/llcppg), the LLGo binding generator for C/C++ libraries.

| Name | Import Path | Description | Release | GoDoc |
| --- | --- | --- | --- | --- |
| [bdw-gc](https://github.com/llarhub/bdw-gc) | `github.com/llarhub/bdw-gc` | Boehm-Demers-Weiser garbage collector: allocate GC-managed memory and control collection (finalizers, heap tuning, leak detection). | [![release](https://img.shields.io/github/v/tag/llarhub/bdw-gc.svg?label=release)](https://github.com/llarhub/bdw-gc/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/bdw-gc.svg)](https://pkg.go.dev/github.com/llarhub/bdw-gc) |
| [cjson](https://github.com/llarhub/cjson) | `github.com/llarhub/cjson` | cJSON: parse, build and serialize JSON documents, including the JSON Pointer / Patch / Merge Patch helpers from `cJSON_Utils`. | [![release](https://img.shields.io/github/v/tag/llarhub/cjson.svg?label=release)](https://github.com/llarhub/cjson/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/cjson.svg)](https://pkg.go.dev/github.com/llarhub/cjson) |
| [clang-c](https://github.com/llarhub/clang-c) | `github.com/llarhub/clang-c` | libclang C API: parse, analyze and traverse C/C++ ASTs. | [![release](https://img.shields.io/github/v/tag/llarhub/clang-c.svg?label=release)](https://github.com/llarhub/clang-c/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/clang-c.svg)](https://pkg.go.dev/github.com/llarhub/clang-c) |
| [libuv](https://github.com/llarhub/libuv) | `github.com/llarhub/libuv` | libuv: cross-platform asynchronous I/O (event loops, TCP/UDP sockets, file-system operations, and more). | [![release](https://img.shields.io/github/v/tag/llarhub/libuv.svg?label=release)](https://github.com/llarhub/libuv/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/libuv.svg)](https://pkg.go.dev/github.com/llarhub/libuv) |
| [libxml2](https://github.com/llarhub/libxml2) | `github.com/llarhub/libxml2` | libxml2: parse, query (XPath), validate and serialize XML and HTML documents. | [![release](https://img.shields.io/github/v/tag/llarhub/libxml2.svg?label=release)](https://github.com/llarhub/libxml2/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/libxml2.svg)](https://pkg.go.dev/github.com/llarhub/libxml2) |
| [libxslt](https://github.com/llarhub/libxslt) | `github.com/llarhub/libxslt` | libxslt (with libexslt): load XSLT stylesheets and transform XML documents; built on the libxml2 binding. | [![release](https://img.shields.io/github/v/tag/llarhub/libxslt.svg?label=release)](https://github.com/llarhub/libxslt/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/libxslt.svg)](https://pkg.go.dev/github.com/llarhub/libxslt) |
| [llvm-c](https://github.com/llarhub/llvm-c) | `github.com/llarhub/llvm-c` | LLVM's official C API: a stable, ABI-safe interface to the IR builder, modules and JIT. | [![release](https://img.shields.io/github/v/tag/llarhub/llvm-c.svg?label=release)](https://github.com/llarhub/llvm-c/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/llvm-c.svg)](https://pkg.go.dev/github.com/llarhub/llvm-c) |
| [lua](https://github.com/llarhub/lua) | `github.com/llarhub/lua` | Lua C API: embed the Lua interpreter, run scripts, and exchange values and functions with Lua. | [![release](https://img.shields.io/github/v/tag/llarhub/lua.svg?label=release)](https://github.com/llarhub/lua/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/lua.svg)](https://pkg.go.dev/github.com/llarhub/lua) |
| [openssl](https://github.com/llarhub/openssl) | `github.com/llarhub/openssl` | OpenSSL: TLS/SSL, hashing, symmetric and public-key cryptography, signatures, X.509 certificates. | [![release](https://img.shields.io/github/v/tag/llarhub/openssl.svg?label=release)](https://github.com/llarhub/openssl/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/openssl.svg)](https://pkg.go.dev/github.com/llarhub/openssl) |
| [py](https://github.com/llarhub/py) | `github.com/llarhub/py` | Python/C API: embed a Python interpreter, execute Python code, and exchange objects between host and script. (For importing Python libraries as Go packages, see [Python Bindings](#python-bindings).) | [![release](https://img.shields.io/github/v/tag/llarhub/py.svg?label=release)](https://github.com/llarhub/py/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/py.svg)](https://pkg.go.dev/github.com/llarhub/py) |
| [raylib](https://github.com/llarhub/raylib) | `github.com/llarhub/raylib` | raylib: 2D/3D games and graphical applications (windowing, drawing, textures, models, audio, input). | [![release](https://img.shields.io/github/v/tag/llarhub/raylib.svg?label=release)](https://github.com/llarhub/raylib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/raylib.svg)](https://pkg.go.dev/github.com/llarhub/raylib) |
| [sqlite](https://github.com/llarhub/sqlite) | `github.com/llarhub/sqlite` | SQLite3 C API: embedded SQL databases with prepared statements, transactions and custom functions. | [![release](https://img.shields.io/github/v/tag/llarhub/sqlite.svg?label=release)](https://github.com/llarhub/sqlite/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/sqlite.svg)](https://pkg.go.dev/github.com/llarhub/sqlite) |
| [zlib](https://github.com/llarhub/zlib) | `github.com/llarhub/zlib` | zlib: compress and decompress data in DEFLATE, zlib and gzip formats. | [![release](https://img.shields.io/github/v/tag/llarhub/zlib.svg?label=release)](https://github.com/llarhub/zlib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/zlib.svg)](https://pkg.go.dev/github.com/llarhub/zlib) |
| [libcxx](https://github.com/llarhub/libcxx) | `github.com/llarhub/libcxx` | libc++ (LLVM's C++ standard library): link against and interoperate with libc++ containers, strings and runtime types. | [![release](https://img.shields.io/github/v/tag/llarhub/libcxx.svg?label=release)](https://github.com/llarhub/libcxx/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/libcxx.svg)](https://pkg.go.dev/github.com/llarhub/libcxx) |
| [llvm](https://github.com/llarhub/llvm) | `github.com/llarhub/llvm` | LLVM C++ API: direct access to LLVM's IR builder, module and pass infrastructure. | [![release](https://img.shields.io/github/v/tag/llarhub/llvm.svg?label=release)](https://github.com/llarhub/llvm/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/llvm.svg)](https://pkg.go.dev/github.com/llarhub/llvm) |
| [v8](https://github.com/llarhub/v8) | `github.com/llarhub/v8` | V8 JavaScript engine: embed, compile and execute JavaScript through V8's C++ API. | [![release](https://img.shields.io/github/v/tag/llarhub/v8.svg?label=release)](https://github.com/llarhub/v8/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/llarhub/v8.svg)](https://pkg.go.dev/github.com/llarhub/v8) |

Notes:

- A release badge reads "no tags" until the repository publishes its first tag; the bindings are still usable from the `main` branch in the meantime.
- The C and C++ **standard libraries** are covered separately by [`github.com/goplus/lib`](https://github.com/goplus/lib) (`c`, `c/math`, `c/pthread`, `c/time`, `c/net`, `cpp/std`, and so on).

---

## Python Bindings

> **Note:** Python library support in LLGo is still at an early stage and should be considered a technical preview.

Python bindings are generated by `llpyg` (see the [LLGo homepage](https://github.com/xgo-dev/llgo#python-support)) and are shipped together in the [`github.com/goplus/lib`](https://github.com/goplus/lib) module under the `py/` directory. A binding package wraps a Python module as ordinary Go functions that operate on Python objects, so you import it and call it directly:

```
import (
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
	"github.com/goplus/lib/py/std"
)
```

You need Python 3.12 or newer installed. The standard-library bindings work out of the box; third-party libraries such as numpy, pandas, torch and matplotlib must be installed separately (for example with `pip`) before you build.

| Name | Import Path | Description | Release | GoDoc |
| --- | --- | --- | --- | --- |
| [py](https://github.com/goplus/lib/tree/main/py) | `github.com/goplus/lib/py` | Core Python ABI layer: object construction (`py.Float`, `py.Str`, `py.List`, ...) shared by every other Python binding. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py.svg)](https://pkg.go.dev/github.com/goplus/lib/py) |
| [py/std](https://github.com/goplus/lib/tree/main/py/std) | `github.com/goplus/lib/py/std` | Python builtins (`print`, `len`, `range`, and friends). | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/std.svg)](https://pkg.go.dev/github.com/goplus/lib/py/std) |
| [py/sys](https://github.com/goplus/lib/tree/main/py/sys) | `github.com/goplus/lib/py/sys` | Python `sys` module: interpreter and runtime information. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/sys.svg)](https://pkg.go.dev/github.com/goplus/lib/py/sys) |
| [py/os](https://github.com/goplus/lib/tree/main/py/os) | `github.com/goplus/lib/py/os` | Python `os` module: operating-system interfaces. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/os.svg)](https://pkg.go.dev/github.com/goplus/lib/py/os) |
| [py/math](https://github.com/goplus/lib/tree/main/py/math) | `github.com/goplus/lib/py/math` | Python `math` module: mathematical functions. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/math.svg)](https://pkg.go.dev/github.com/goplus/lib/py/math) |
| [py/json](https://github.com/goplus/lib/tree/main/py/json) | `github.com/goplus/lib/py/json` | Python `json` module: encode and decode JSON. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/json.svg)](https://pkg.go.dev/github.com/goplus/lib/py/json) |
| [py/inspect](https://github.com/goplus/lib/tree/main/py/inspect) | `github.com/goplus/lib/py/inspect` | Python `inspect` module: introspect live objects. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/inspect.svg)](https://pkg.go.dev/github.com/goplus/lib/py/inspect) |
| [py/statistics](https://github.com/goplus/lib/tree/main/py/statistics) | `github.com/goplus/lib/py/statistics` | Python `statistics` module: mean, median, variance and other descriptive statistics. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/statistics.svg)](https://pkg.go.dev/github.com/goplus/lib/py/statistics) |
| [py/numpy](https://github.com/goplus/lib/tree/main/py/numpy) | `github.com/goplus/lib/py/numpy` | [NumPy](https://numpy.org): n-dimensional arrays and numerical computing. Requires `pip install numpy`. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/numpy.svg)](https://pkg.go.dev/github.com/goplus/lib/py/numpy) |
| [py/pandas](https://github.com/goplus/lib/tree/main/py/pandas) | `github.com/goplus/lib/py/pandas` | [pandas](https://pandas.pydata.org): data frames and data analysis. Requires `pip install pandas`. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/pandas.svg)](https://pkg.go.dev/github.com/goplus/lib/py/pandas) |
| [py/torch](https://github.com/goplus/lib/tree/main/py/torch) | `github.com/goplus/lib/py/torch` | [PyTorch](https://pytorch.org): tensors and deep learning. Requires `pip install torch`. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/torch.svg)](https://pkg.go.dev/github.com/goplus/lib/py/torch) |
| [py/matplotlib](https://github.com/goplus/lib/tree/main/py/matplotlib) | `github.com/goplus/lib/py/matplotlib` | [Matplotlib](https://matplotlib.org): plotting and visualization. Requires `pip install matplotlib`. | [![release](https://img.shields.io/github/v/tag/goplus/lib.svg?label=release)](https://github.com/goplus/lib/releases) | [![GoDoc](https://pkg.go.dev/badge/github.com/goplus/lib/py/matplotlib.svg)](https://pkg.go.dev/github.com/goplus/lib/py/matplotlib) |

Notes:

- Need a Python library that is not listed? Generate your own binding with `llpyg`, which converts a Python library into a Go package that LLGo can import. It relies on `pydump` and `pysigfetch`; see the [Development tools](https://github.com/xgo-dev/llgo#development-tools) section of the LLGo README.

---

## References

- LLGo: <https://github.com/xgo-dev/llgo>
- LLAR Hub (C/C++ bindings): <https://github.com/llarhub>
- goplus/lib (standard-library and Python bindings): <https://github.com/goplus/lib>
- llcppg (C/C++ binding generator): <https://github.com/xgo-dev/llcppg>

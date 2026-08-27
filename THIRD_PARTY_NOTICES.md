# Third-party notices

LLGo is licensed under Apache License 2.0. The following components retain
their own licenses. The referenced license files are included in source
checkouts, release archives, and packaged distributions. The nested `runtime`
module carries its own copy of LLGo's Apache-2.0 license in `runtime/LICENSE`.

## TinyGo

LLGo contains source derived from the [TinyGo project](https://github.com/tinygo-org/tinygo):

- the conservative bare-metal collector in `runtime/internal/runtime/tinygogc/gc_tinygo.go`;
- firmware image support in `internal/firmware/esp.go`, `nrfutil.go`, `objcopy.go`, and `uf2.go`;
- the marked flashing-support portions of `internal/flash/flash.go`;
- serial monitoring and panic-location support in `internal/monitor/monitor.go`;
- the deprecated archive builder in `chore/_deprecated/ar/ar.go`;
- target configurations and support files under `targets`.

Reference snapshots contemporaneous with the initial LLGo imports are:

- [runtime GC](https://github.com/tinygo-org/tinygo/tree/79ab77facd8b4d7ea39257f85d37f094f52770d2/src/runtime);
- [firmware builders](https://github.com/tinygo-org/tinygo/tree/3869f76887feef6c444308e7e1531b7cac1bbd10/builder);
- [monitor implementation](https://github.com/tinygo-org/tinygo/blob/020664591ab3a995d6d0aab5097c6fab838a925c/monitor.go);
- [initial target configuration import](https://github.com/tinygo-org/tinygo/tree/8c5886060f022a36768b5c29327759846021a868/targets).

The TinyGo-derived portions remain subject to the BSD 3-Clause License in
[`LICENSES/TinyGo-BSD-3-Clause.txt`](LICENSES/TinyGo-BSD-3-Clause.txt).
Independently written LLGo code remains subject to the repository's Apache
License 2.0.

## The Go project

LLGo distributes Go-derived source under `runtime`, `targets/wasm_exec.js`,
`chore/ssadump`, and selected compatibility tests.
The `llgo` executable also contains the Go standard library and packages from
`golang.org/x/mod`, `golang.org/x/sync`, `golang.org/x/sys`, and
`golang.org/x/tools`. The authoritative versions are recorded in `go.mod` and
the executable's Go build information.

These portions are licensed under the Go project's BSD 3-Clause License in
[`LICENSES/Go-BSD-3-Clause.txt`](LICENSES/Go-BSD-3-Clause.txt). The nested
runtime module also carries a copy at
[`runtime/LICENSES/Go-BSD-3-Clause.txt`](runtime/LICENSES/Go-BSD-3-Clause.txt).

## Source incorporated into LLGo

| Component | Distributed in | License |
| --- | --- | --- |
| `github.com/sigurn/crc16` and LLGo adaptations | `internal/crc16`, compiled CLI | [MIT](LICENSES/CRC16-MIT.txt) |
| `github.com/marcinbor85/gohex` and LLGo adaptations | `internal/gohex`, compiled CLI | [MIT](LICENSES/GoHex-MIT.txt) |
| `github.com/blakesmith/ar` and LLGo adaptations | `xtool/ar`, source and development tools | [MIT](LICENSES/BlakeSmith-AR-MIT.txt) |
| Hardware-vendor target support | marked files under `targets/device` and `targets/rp2040-boot-stage2.S` | [vendor notices](LICENSES/Target-Device-Notices.txt), including the [Nordic BSD license](LICENSES/Nordic-BSD-3-Clause.txt) |

The original file-level notices remain in the target support sources. Firmware
distributors must reproduce the applicable notices when those sources are
included in a firmware image.

## Go modules compiled into the llgo executable

This list is limited to modules in the shipped Darwin and Linux executable;
development- and test-only modules are not included.

| Module | License |
| --- | --- |
| `github.com/creack/goselect` | [MIT](LICENSES/Goselect-MIT.txt) |
| `github.com/goplus/cobra` | [Apache-2.0](LICENSES/Cobra-Apache-2.0.txt); compiled `pflag` package: [BSD-3-Clause](LICENSES/Cobra-pflag-BSD-3-Clause.txt) |
| `github.com/mattn/go-tty` | [MIT](LICENSES/GoTTY-MIT.txt) |
| `github.com/qiniu/x` | [Apache-2.0](LICENSES/Qiniu-X-Apache-2.0.txt) |
| `github.com/xgo-dev/llvm` | [Apache-2.0 WITH LLVM-exception](LICENSES/XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt) |
| `github.com/xgo-dev/plan9asm` | [Apache-2.0](LICENSES/XGo-Plan9Asm-Apache-2.0.txt) |
| `go.bug.st/serial` | [BSD-3-Clause](LICENSES/Go-Serial-BSD-3-Clause.txt) |
| `go.yaml.in/yaml/v3` | [MIT and Apache-2.0](LICENSES/Go-YAML-LICENSE.txt), [NOTICE](LICENSES/Go-YAML-NOTICE.txt) |
| `golang.org/x/mod`, `x/sync`, `x/sys`, and `x/tools` | [BSD-3-Clause](LICENSES/Go-BSD-3-Clause.txt) |

## LLVM/Clang

LLGo can download an Espressif-maintained ESP LLVM/Clang toolchain matching
the LLVM major linked into LLGo:

- LLVM 19: [`19.1.2_20250905-3`](https://github.com/goplus/espressif-llvm-project-prebuilt/releases/tag/19.1.2_20250905-3)
- LLVM 21: [`21.1.3_20260816`](https://github.com/goplus/espressif-llvm-project-prebuilt/releases/tag/21.1.3_20260816)

Current LLGo release archives use the LLVM 19 payload and include it under
`crosscompile/clang`, because the shipped `llgo` executable dynamically links
its LLVM library.

LLVM, Clang, LLD, libc++, libc++abi, libunwind, compiler-rt, and other
LLVM-project components are licensed under Apache License 2.0 with LLVM
Exceptions. The complete license is reproduced in
[`LICENSES/XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt`](LICENSES/XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt).
Archive extraction preserves upstream license files.

## Components downloaded for cross-compilation

LLGo downloads the following components directly from their upstream release
locations when a selected target needs them. They are stored in the user's LLGo
cache; they are not vendored in this source tree. Their license files remain in
the extracted source or SDK directory.

The Go packages under `internal/crosscompile/compile/libc` and
`internal/crosscompile/compile/rtlib` are LLGo build manifests, not copies of
the C library sources. The manifest code itself is compiled into `llgo` and is
covered by LLGo's Apache-2.0 license. At cross-compilation time, those manifests
select files from the downloaded picolibc, ESP newlib, or compiler-rt tree and
build static archives in that same cached tree. The archives are then linked
into the target program, so they are not part of the `llgo` executable or LLGo
release archive, but their code can be incorporated into the resulting
firmware.

| Component | Download source | License location |
| --- | --- | --- |
| WASI SDK 25 | [`WebAssembly/wasi-sdk`](https://github.com/WebAssembly/wasi-sdk/releases/tag/wasi-sdk-25) | upstream `LICENSE` and license files in the SDK |
| picolibc/newlib sources | [`goplus/picolibc`](https://github.com/goplus/picolibc) | upstream `COPYING.picolibc` and `COPYING.NEWLIB` |
| ESP newlib sources | [`goplus/newlib`](https://github.com/goplus/newlib/tree/esp-4.3.0_20250211-patch7) | upstream `COPYING.NEWLIB` and applicable file notices |
| compiler-rt sources | [`goplus/compiler-rt` LLVM 19](https://github.com/goplus/compiler-rt/tree/xtensa_release_19.1.2) or [LLVM 21](https://github.com/goplus/compiler-rt/tree/xtensa_release_21.1.3_20260408), selected with the linked LLVM major | upstream `LICENSE.TXT` (Apache-2.0 WITH LLVM-exception) |

Firmware or other binaries built from downloaded C library sources may carry
their own redistribution requirements. Distributors of those outputs should
retain the applicable license and copyright notices from the cached source;
in particular, picolibc and ESP newlib use per-file notices collected by their
upstream `COPYING` files.

## External tools and system libraries

BDWGC, OpenSSL, libffi, cJSON, SQLite, zlib, Python, Emscripten, QEMU,
OpenOCD, flashing utilities, and platform SDK/system libraries are installed or
provided separately. LLGo may link to or invoke them, but does not copy them
into its source tree or release archives, except for LLVM-project components
explicitly described above. Their upstream licenses therefore apply to the
separate installations and to any redistributed output that incorporates them;
they are not relicensed by LLGo.

## Redistribution

Distributions containing the source, the `llgo` executable, the bundled
toolchain, or firmware linked with third-party runtime or target support must
retain or reproduce the applicable copyright notices, license conditions,
disclaimers, and NOTICE text described above.

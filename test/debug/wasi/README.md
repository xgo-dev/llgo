# WASI source debugger acceptance

Install LLVM 22, LLGo, Node.js 22+, native LLDB 18+ (or Apple LLDB), and build
the optional [`llgo-wasi-debug` host](host/README.md) with Rust 1.97+. The host
embeds Wasmtime 50.0.1 and the maintained `wasi-common`/Wiggle 36.0.20 LTS
preview1 implementation. It supplies real WASI threads without depending on
the removed upstream CLI threads option or an unmaintained engine.

```sh
LLGO=/path/to/llgo LLGO_WASI_HOST=/path/to/llgo-wasi-debug LLGO_LLDB=/path/to/lldb \
  python3 test/debug/wasi/runtest.py --report /tmp/wasi-debug-qualification.json
```

Acceptance first runs [`fdsharing`](fdsharing/main.go): a worker is created
before a file is opened, writes and closes the new descriptor, and the main
thread verifies the contents and the process-wide close. Both `/tmp` and
relative working-directory paths are checked. Separate runs verify that
`proc_exit` from the main thread or a worker preserves a nonzero process exit
status. A host that independently
recreates or snapshots the descriptor table per instance fails this check.

Both embedded and external DWARF sessions then exercise the shared
[`../wasm`](../wasm) fixture, without replacing imports, disabling EH, or
substituting a single-thread module. Acceptance checks:

- C++ source breakpoint, initialized local and source stepping; Go caller stacks.
- Nine recursive Go frames on two real parked worker threads, with distinct
  local arrays read through their own translated DWARF locations.
- Logical G IDs, parent/source stacks, thread exit and registry cleanup.
- Shared-memory growth and GC before inspection; Unicode strings, nested
  slices, imported/named interface payloads, typed nils, functions, closure
  captures, growing and indirect maps, closed/wrapped channels and zero-sized values.
- A second all-stop snapshot after resume, container mutations, channel
  draining, worker exit and another GC; inspection rejected after process exit.
- Deployment/debug sizes, source and artifact SHA-256 identities, dirty-source
  status, and exact tool versions in the JSON report.

The private debugger module must reproduce the deployed code/data exactly.
Package tests reject a changed sidecar even when it advertises the original
build ID. Every runtime-memory read verifies the process identity and stop ID.
Compiler/schema tests keep the opt-in artificial memory anchor aligned with
the canonical schema; ordinary Wasm builds and native builds do not emit it.
It reserves one native pointer slot per debugger frame. The anchor's standard
indirect DWARF expression avoids an engine-private VMContext offset dependency.
The browser and WASI frontend share the same JavaScript value/stack decoders.

The host's build excludes persistent compilation caching. With a private fresh
Wasmtime CLI cache, the first LLDB session can step successfully while a second
session traps on the previous breakpoint instruction. Wasmtime registers JIT
code before serializing a compilation-cache miss, so an attached debugger's
code patches can enter that cache. See
[Wasmtime #15290](https://github.com/bytecodealliance/wasmtime/issues/15290), also
reproduced with a six-line C program on Wasmtime 48.0.1.

Qualification on 2026-11-15 used Wasmtime 50.0.1, WASI 36.0.20 LTS, Rust 1.97.0,
Go 1.27.3, Clang/LLVM 22.1.8 and Node.js 26.8.1:

| Native host | LLDB | Embedded bytes | External bytes | Debug sidecar bytes |
| --- | --- | ---: | ---: | ---: |
| macOS arm64 | Apple 2103.0.34.103 / Swift 6.4 | 2,099,266 | 723,561 | 2,099,266 |

The shared-descriptor test and both complete source sessions passed. Sizes
are fixture/build-specific. The same acceptance is a `cross-compile` workflow
gate on native Linux amd64; its uploaded JSON records the exact checked commit
and tool identities. Earlier qualification with Wasmtime CLI 46.0.3 passed on
macOS arm64 and Linux amd64, but does not qualify this replacement host on Linux.
Cross-architecture Docker emulation cannot qualify a native debugger. Physical
probe qualification remains in [`../hardware`](../hardware/README.md).

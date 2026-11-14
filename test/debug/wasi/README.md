# WASI source debugger acceptance

Install LLVM 22, LLGo, Node.js 22+, native LLDB 18+ (or Apple LLDB), and
[Wasmtime 46.0.3](https://github.com/bytecodealliance/wasmtime/releases/tag/v46.0.3).
Wasmtime 47+ removed WASI threads. Set `LLGO_WASMTIME` to the supported executable;
other Wasmtime versions used by unrelated tests can remain on PATH.

```sh
LLGO=/path/to/llgo LLGO_WASMTIME=/path/to/wasmtime LLGO_LLDB=/path/to/lldb \
  python3 test/debug/wasi/runtest.py --report /tmp/wasi-debug-qualification.json
```

Both embedded and external DWARF sessions exercise the shared
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
- Deployment/debug sizes and exact tool versions in the JSON report.

The private debugger module must reproduce the deployed code/data exactly.
Package tests reject a changed sidecar even when it advertises the original
build ID. Every runtime-memory read verifies the process identity and stop ID.
Compiler/schema tests keep the opt-in artificial memory anchor aligned with
the canonical schema; ordinary Wasm builds and native builds do not emit it.
It reserves one native pointer slot per debugger frame. The anchor's standard
indirect DWARF expression avoids an engine-private VMContext offset dependency.
The browser and WASI frontend share the same JavaScript value/stack decoders.

Debug sessions always pass `-Ccache=n`. With a private fresh Wasmtime cache,
the first LLDB session can step successfully while a second session traps on
the previous breakpoint instruction. Wasmtime registers JIT code before
serializing a compilation-cache miss, so an attached debugger's code patches
can enter that cache. See [Wasmtime #15290](https://github.com/bytecodealliance/wasmtime/issues/15290),
also reproduced with a six-line C program on Wasmtime 48.0.1.
Engine-cache hits must not be used as a debugging gate.

Qualification on 2026-11-14: macOS arm64, Go 1.27.3, Clang/LLVM 22.1.8,
Apple LLDB 2103.0.34.103 / Swift 6.4, Wasmtime 46.0.3 (`c5569e6aa`),
Node.js 26.8.1. Both complete sessions passed. Embedded deployment: 2,099,289
bytes; external deployment: 723,561 bytes; full debug sidecar: 2,099,289 bytes.
Sizes are fixture/build-specific. Linux qualification is also an explicit
`cross-compile` workflow gate. Cross-architecture Docker emulation cannot
qualify a native debugger. Physical probe qualification remains in
[`../hardware`](../hardware/README.md).

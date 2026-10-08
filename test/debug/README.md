# Debugger acceptance tests

This directory contains integration suites that invoke real debugger tools.
They run explicitly in CI and require an installed LLGo compiler and the tools
listed by each suite.

For debugger usage and artifact options, see the
[debugging guide](../../doc/debugging.md). This directory documents acceptance
entry points and their tool requirements.

- [Runtime inspection](runtime/README.md) shares native Go and C fixtures between
  LLDB and GDB, covering variables, runtime values, goroutines, and backtraces.
- [Embedded transports](embedded/README.md) exercises Cortex-M QEMU sessions
  through GDB Remote and LLDB, including explicit image download into an empty
  simulator, then checks that DWARF leaves flash bytes unchanged.
- [Physical probes](hardware/README.md) reuses the embedded fixture on a real
  target. It requires explicit opt-in and is not part of normal CI.
- [Wasm logical stacks](wasm/README.md) exercises the installed Chrome extension
  with Memory32/64, parked goroutines, GC, and 1/2/4 workers.

Run the native LLDB suite from the repository root:

```sh
bash test/debug/runtime/runtest.sh -v
```

Run GDB's native values, registry and main-stack acceptance with Python-enabled
GDB 12+ (the Windows ARM64 jobs use GDB 18+):

```sh
LLGO_GDB_INTEGRATION=1 go test ./cmd/internal/gdb -run '^TestGDBIntegration$' -count=1 -v
```

Complete blocked-worker unwind is a separate strict acceptance test:

```sh
LLGO_GDB_INTEGRATION=1 go test ./cmd/internal/gdb -run '^TestGDBCompleteWorkerUnwind$' -count=1 -v
```

The latter remains a required gate on Linux and Windows amd64/386. Stock GDB
cannot currently complete that test on Intel macOS (dyld shared-library support)
or Windows ARM64 (PAC masks), although native values, the goroutine registry
and the main application stack are tested there. See the
[runtime coverage matrix](runtime/README.md#gdb-runtime-acceptance). These
limitations do not apply to the native LLDB suite.

Run the embedded suite with QEMU, target-aware GDB, LLDB and LLVM tools installed:

```sh
bash test/debug/embedded/runtest.sh
```

The Targets workflow runs GDB and LLDB separately on Linux x64, macOS arm64 and
Windows x64 hosts against the same Cortex-M3 target. It verifies that the physical-probe
entry point refuses to halt or load a device without its documented opt-in.
Passing QEMU acceptance does not imply a physical-probe test was performed.

The runtime and embedded fixtures retain their own `go.mod` files, including
the `lldbtest` module name used by runtime debugger type assertions. The nested
modules exclude these programs
from root-module `go test ./test/...` and `llgo test ./test/...` enumeration.
The LLGo workflow invokes the LLDB suite in its native platform jobs and the
GDB suite in its Linux and all six Windows ABI/architecture jobs. A separate
Intel macOS job exercises both debuggers; its GDB wrapper starts the executable
suspended with `posix_spawn`, then attaches natively using the ephemeral runner's
passwordless sudo permission to obtain task ports. This avoids GDB's Darwin
fork/exec startup hang while preserving the complete values/registry gate.
The Windows debugger is installed separately from the pinned compiler and
LLDB dependencies. Apple Silicon has no native GDB process target: use LLDB
there; GDB remote debugging remains available. Unit tests and package-specific `testdata` remain
beside their implementation.

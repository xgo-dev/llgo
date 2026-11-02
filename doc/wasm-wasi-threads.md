# W32 WASI threads

`llgo build/run/test -target wasi` (including the `wasip1` alias and raw
`GOOS=wasip1 GOARCH=wasm`) uses wasi-libc's pthread startup, shared linear
memory, and the threaded linear collector by default. WASI output no longer
runs Binaryen Asyncify. Native and embedded pthread backends are unchanged.

The LLGo Wasmer release `llgo-v7.5.0.1` is the default runner. Install it with
`bash dev/install_wasmer.sh`, and add the printed cache `bin` directory to `PATH`.
The installer verifies the release archive's SHA-256 digest. This fork carries
the exception/thread and compiled-module-cache fixes needed by LLGo. The public
run/test commands enable standard Wasm EH and SIMD and configure the stack and
directory mappings. Cold and warm module-cache runs preserve guest output and
exit status.

LLVM emits standard EH directly with `-wasm-use-legacy-eh=false`, including at
LTO link time. The module imports `wasi.thread-spawn` in addition to Preview 1
and shared `env.memory`. Explicit worker `Goexit` returns to wasi-libc's thread
entry through a thread-local setjmp boundary, after running Go defers and C
pthread cleanup handlers. Libc then runs TLS destructors, removes the thread
from its list, and reclaims detached stack/TLS storage at a subsequent thread
creation or exit. This follows the same cleanup path as ordinary return and
does not import WASIX `thread_exit`.

The exit adapter uses SDK 25's `__pthread_create` entry and cleanup-record ABI;
SDK upgrades must rerun its regression tests. It also wraps C-created pthreads
and preserves joinable return values. The initial Go thread uses the runtime's
existing parked-main Goexit path; arbitrary C `pthread_exit` on that initial
thread or from a TLS destructor is not supported.

On Unix, the runner grants the absolute package working directory and `/tmp`.
On Windows, those guest paths are `/work` and `/tmp`, mapped to the package
and host temporary directories. Guest flags follow a `--` separator. Only
`PWD`, `PATH` and the optional `LLGO_STRESS_PROFILE` are forwarded into the guest.
The GOROOT comparison runs both W32 artifacts with Wasmer. Only the separate
`dev/wasmstdlib` official-Go reference profile uses Wasmtime; that does not imply
support for executing LLGo's threaded W32 artifact there.

The installer supports the fork's macOS arm64, Linux amd64/aarch64/riscv64,
and Windows amd64 archives. The release has no prebuilt macOS Intel archive;
on that host, build the CLI from source and put it on `PATH`.

The former `LLGO_WASI_THREADS=1` opt-in is unnecessary and remains accepted.
Setting it to `0`/`false`/`off` now produces a migration error instead of silently
building a different runtime. The `llgo.wasi_threads` source tag remains part
of compilation/cache identity. `-tags nogc` is still available; the default
enables GC. Single-thread WASI context switching and its allocator wrappers
have been removed.

WASI retains one goroutine per host pthread. It does not use the browser's
bounded worker scheduler, and `LLGO_WASM_WORKERS` remains a browser-only option.
Host thread and shared-memory limits therefore bound the number of concurrent
WASI goroutines. A C call that cannot acknowledge a safepoint causes that
collection attempt to be skipped; the collector does not scan an actively
mutating foreign stack.

## Memory and process-stack budgets

A target under `targets/` can inherit `wasi` or `wasip1` and set byte budgets:

```json
{
  "inherits": ["wasip1"],
  "wasm-initial-memory": 16777216,
  "wasm-max-memory": 134217728,
  "wasm-stack-size": 1048576
}
```

Save this as `targets/wasi-small.json` under `LLGO_ROOT`, then build with
`llgo build -target=wasi-small`. The initial and maximum memory sizes must be
multiples of 64 KiB, at most 4 GiB. The process stack must be 16-byte aligned
and smaller than 4 GiB. Omitted or zero fields inherit the parent's setting;
otherwise the toolchain defaults apply. These fields are supported by W32 WASI
targets only. The linker rejects budgets too small for the actual static data
and process stack. The initial/maximum comparison is also left to the linker,
after explicit flags and response files can override inherited defaults.

The initial size covers the whole linear memory, including static data and the
process stack. Memory may grow up to the maximum; a small initial size does not
cap runtime allocation. The process stack is separate from goroutine/pthread
stacks (`-stacksize`) and the Wasmer host stack (`--stack-size`).

Explicit linker flags remain authoritative. An explicit `--initial-memory`,
`--initial-heap`, or user response file suppresses the target's initial-memory
default. Stack/maximum linker flags follow the existing driver argument order.
Without a configured initial size, LLGo retains the 54 MiB initial heap
budget in addition to static data and the process stack. The default process
stack is 10 MiB and maximum memory is 512 MiB, including on the raw
`GOOS=wasip1 GOARCH=wasm` path.

Run `python3 dev/test_wasm_wasi_threads.py` for pthread/GC, EH, filesystem,
run/test, selected standard-library, and GOROOT checks. Run
`python3 dev/test_wasm_debug_info.py --profile w32` to validate final Go/C++
DWARF and source-line mapping at O0/O2. `dev/test_wasm_target_profiles.sh`
covers named and raw profile builds. Full compatibility audit results are
reported separately and must not be inferred from these focused checks.

The pthread/GC groundwork was introduced in #2669 and #2695. The former WAMR
stability patches are superseded by this runner migration. Browser filesystem
integration is tracked in #2696.
WasmGC, W64, JSPI, and WASI Preview 2/components remain separate proposals.

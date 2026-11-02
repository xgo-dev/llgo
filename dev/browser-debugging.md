# Browser source debugging

Build and open an LLGo program with its actual Emscripten host:

```sh
llgo debug -target=emscripten -chrome /path/to/chrome-for-testing ./app
llgo debug -target=emscripten-memory64 -debug-artifact=external -o app.mjs ./app
```

The command enables DWARF and selects O0 unless an optimization level was
specified. The `wasm` target alias and raw `GOOS=js GOARCH=wasm` source mode are
also supported. Memory width, selected profile, and compiler worker settings
are preserved. An explicit output path keeps the artifacts after the session;
temporary output directories and the isolated Chrome profile are cleaned up
when Chrome exits.

Use Chrome for Testing or Chromium with support for loading an unpacked
extension through command-line flags. `-chrome` and `LLGO_CHROME` select the
executable. The extension, source files, generated JavaScript host, module and
optional DWARF sidecar are served from a local loopback session. DevTools opens
automatically; choose the original source in Sources, set a breakpoint, and run
the program from the session page.

`-source-map` accepts repeated `FROM=TO` mappings. The longest matching source
prefix wins:

```sh
llgo debug -target=emscripten \
  -source-map=/builder/project=/work/project \
  -source-map=/builder/project/vendor=/work/vendor ./app
```

`-browser-devtools=false` disables automatic DevTools opening. Additional
Chrome presentation arguments follow `--`: `--window-size=WIDTH,HEIGHT`,
`--window-position=X,Y`, `--start-maximized` or `--headless=new`. Profile,
extension and web security flags cannot be overridden. With DevTools enabled,
a missing extension handshake fails visibly before Wasm instantiation after
five seconds; open DevTools and reload to retry. Disabling browser DevTools
explicitly skips the extension handshake.

Source files are restricted to directories selected by the current build,
the module directory and explicit source-map destinations. A source path in
DWARF alone never authorizes reading an unrelated local file; symlinks cannot
escape these roots. Loopback CORS accepts only the extension identity generated
for this session. Native GDB Remote options `-remote` and `-server`
do not apply to browser sessions.

Worker coordination enables a random loopback Chrome debugging port in the
isolated profile. Chrome's CDP endpoint allows other local users/processes to
control that browser; use it on a machine whose local users you trust.
Disabling DevTools also disables this port and worker coordination.

Embedded and external DWARF are both accepted. For external DWARF, keep the
referenced `.debug.wasm` file within the module directory: its build ID must match the
module. The sidecar is debugger data; it does not replace the JavaScript host
or executable module. Missing or mismatched sidecars are reported before
starting the program.

`llgo debug` enables `llgo.wasm.debugger` for browser builds. Add `$goroutines`
to DevTools Watch to inspect live logical goroutines, including parked fibers:
IDs, parent IDs, state, last processor, and innermost-first function/file/line
stacks. For an artifact built separately, pass `-tags=llgo.wasm.debugger`.
The tag participates in the package cache key. Ordinary builds do not enable
the registry or extra caller-frame tracking.

`llgo debug` coordinates the session page and its descendant dedicated workers.
Pausing one target pauses the others; resuming or stepping one resumes the group, and
the next stop pauses the group again. New workers join the group and exited
workers leave it. Other tabs and extension workers are not controlled.
Workers created during a stop stay at Chrome's startup barrier until the
group resumes.

Shared-memory inspection waits for all targets to stop and rejects values if
the stop or worker membership changes during the read. Expanding an object from
an older stop reports that it must be inspected again. A disconnected control
transport disables coordinated inspection with an error. Registry/stack guards
remain an additional check.

Expand a waiting goroutine's frame in `$goroutines` to inspect its memory
locals. Debug builds save each frame's native base; the reader accepts a
`DW_OP_fbreg` location only when one scope and location cover the entire
function. Recursive frames therefore use their own addresses after GC and
fiber switches. Register values, changing locations, ambiguous names and
unwound panic snapshots remain unavailable. Ordinary builds retain no frame
bases. Existing artifacts without saved bases still show source stacks.

Selecting a logical frame does not switch the execution context. DevTools'
normal frame selector and local-variable pane refer to the paused Wasm frame.
Exact parked PCs, optimized register reconstruction and complete runtime-object
views remain a later phase.

Runtime views expand strings, slices, arrays, interface payloads, maps and
buffered channels, including nested values and zero-sized elements. Named
interface payloads use runtime import-path metadata to find their DWARF types;
unresolved types retain the summary without guessing a payload layout. Function
values show their Wasm table index, known function name and closure-context
address. Expand `captures` to inspect the compiler-described environment;
reference captures retain their pointer indirection and observe later writes.
This uses the final module's static function table and artificial `$env` DWARF
type, including nested closures and bound methods. Missing or ambiguous types
keep the raw code/data view. Dynamic table rebinding is outside this static
mapping contract. Inspection never calls the function. Container expansion is
bounded to 100 elements and map scans to 4096 buckets.

WASI source sessions remain unavailable for current W32 pthread artifacts.
Their shared `env.memory`, `wasi.thread-spawn` and `wasix_32v1.thread_exit` imports
require a compatible runtime debugger; the current Wasmtime backend cannot
provide that contract. `llgo debug -target=wasi` reports the limitation and
`llgo run -target=wasi` continues to use Wasmer. No thread imports are replaced
with synthetic stubs.

## Qualification

`dev/test_wasm_browser_debug.py` records tool versions; its verbose Go tests
record artifact build IDs and deployment/symbol sizes. Use the repository's pinned Binaryen from
`.github/actions/setup-binaryen`; a stock binary with the same major version
does not establish the patched DWARF contract.

The 2026-10-20 macOS arm64 run used Go 1.27.2, LLVM 22.1.8, Emscripten 6.0.8,
Binaryen `llgo-v132.3`, Node 26.8.1 and Chrome for Testing 152.0.7977.54.
All ten sessions passed: Memory32/64 with embedded/external DWARF, plus
1/2/4-worker goroutine fixtures for each memory width (four workers use
external DWARF). The worker sessions check source locals, parked stacks after
GC, goroutine exit, coordinated stepping, worker creation/termination and
startup barriers. Four-worker lifecycle/step acceptance also passed 20
consecutive runs. These results qualify browser coordination, not the
remaining optimized-variable or WASI debugger work.

The 2026-10-21 run on the same toolchain passed twelve sessions with saved
frame bases, adding two-worker `-Os` coverage for each memory width. The goroutine fixtures read distinct arrays from four/five
recursive waiting frames after forced memory growth and GC, then verify the
values again after resuming. A recovered goroutine parks in a later defer to
check that cleared panic snapshots cannot expose unwound frame addresses.
Unit tests reject register/changing/ambiguous
locations, panic snapshots and stale stops, and check recursive object release.

The 2026-10-22 run passed all twelve sessions with deep runtime values after
memory growth and GC: nested strings/slices, named interface payloads from main
and imported packages, typed nils, function/closure addresses, maps during
evacuation, indirect keys/values, wrapped closed channels and zero-sized
elements. A second stop verifies mutations and drained channels; object release
checks include every expanded child. The compiler now uses the runtime's
canonical package paths in DWARF. Native macOS LLDB O0/O2 panic, inline and
aggregate acceptance also passes with this change.

The 2026-10-27 macOS arm64 run used Go 1.27.3, LLVM 22.1.8, Emscripten
6.0.8, Binaryen `llgo-v132.3` and Chrome for Testing 152.0.7977.54. All twelve
sessions passed with typed captures, including nested closures, bound methods
and independent counter closures before and after mutation. Captures remain
readable after memory growth and GC, and releasing the function view releases
its expanded children. Native LLDB O0/O2 regression acceptance also passes.

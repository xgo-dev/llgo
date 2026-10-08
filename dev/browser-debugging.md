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

For multi-worker programs, pause the other threads in DevTools before reading
shared runtime state. The extension reports registry/stack changes observed
during inspection; these guards do not replace pausing all workers. The
frontend does not automatically coordinate worker pauses. Its logical stack
view does not expose parked-frame locals or switch the execution context.
Debugger frame selection and local variables still refer to the paused Wasm
execution frame. Optimized DWARF locations can be unavailable or optimized out;
complete runtime-object views remain a later phase.

WASI source sessions remain unavailable for current W32 pthread artifacts.
Their shared `env.memory`, `wasi.thread-spawn` and `wasix_32v1.thread_exit` imports
require a compatible runtime debugger; the current Wasmtime backend cannot
provide that contract. `llgo debug -target=wasi` reports the limitation and
`llgo run -target=wasi` continues to use Wasmer. No thread imports are replaced
with synthetic stubs.

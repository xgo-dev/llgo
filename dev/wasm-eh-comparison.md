# WebAssembly exception-handling comparison

Run `python3 dev/compare_wasm_eh.py` with Emscripten, Node, `wasm-tools`,
`llvm-dwarfdump`, and the selected Binaryen installation on `PATH`. Set
`EM_BINARYEN_ROOT` to select a complete Binaryen installation; optionally set
`LLGO` to include GoJS, Emscripten wasm32, and Memory64 panic/recover tests. Pass `--browser`
to execute all six C++ variants and, with `LLGO`, the Go O0/O2/Thin/Full LTO cases
and the Go/C++ boundary matrix in Chrome as well.
Individual tool paths can be set with `EMXX`, `NODE`, `WASM_TOOLS`,
`LLVM_DWARFDUMP`, and `WASMOPT`; `WASMOPT` takes precedence over
`EM_BINARYEN_ROOT`.
For reproducible DWARF verification, use Emscripten 6.0.8 with the
[`llgo-v132.3` Binaryen release](https://github.com/xgo-dev/binaryen/releases/tag/llgo-v132.3)
selected through `EM_BINARYEN_ROOT` (and `WASMOPT` if set). Upstream Binaryen
132 can report DWARF range errors unrelated to this EH comparison.
Emscripten 6.0.8-generated glue requires Node 24.15.0 or newer. The script
checks this before building: Node 22 cannot run the glue even with
`--experimental-wasm-exnref`, so `NODE` must point at a compatible binary.

## Browser runtime policy

GoJS, Emscripten wasm32, and Emscripten Memory64 use native Wasm SjLj for
`setjmp`/`longjmp`, matching WASI's choice of native exception handling.
Compilation and linking select `-sSUPPORT_LONGJMP=wasm` and
`-fwasm-exceptions`. The latter also keeps Embind and C++ wrappers on the
native ABI; a Go entry package may still reference C++ exception support
through its bindings. Explicit environment flags selecting JavaScript SjLj
or disabling longjmp are rejected because they conflict with Go panic/recover.

The matching-LLVM Memory64 IR compiler selects `-wasm-enable-sjlj` instead
of `-enable-emscripten-sjlj`. Browser builds retain Emscripten's legacy **Wasm**
EH encoding for Asyncify; this is distinct from the removed JavaScript SjLj
mechanism. W32 continues to emit the standardized Wasm EH encoding.

Go's defer/panic/recover semantics still use the runtime's SjLj abstraction.
This change does not replace that abstraction with direct Go-to-LLVM exception
lowering, nor does it change Fiber/Asyncify scheduling. Engines must support
native Wasm exceptions; a JavaScript fallback is no longer selected.

Emscripten warns about mixing Asyncify with `-fwasm-exceptions`. Binaryen has
partial support: suspension while a Wasm catch is active is unsupported. The
boundary acceptance matrix explicitly enables its existing `asyncify-asserts`
checks through `BINARYEN_EXTRA_PASSES` at O0/O2/Thin/Full LTO, independently of
Emscripten `ASSERTIONS=0`. Environment options and package link directives may
not request `asyncify-ignore-unwind-from-catch`.

Do not enable `asyncify-asserts` globally for LLGo applications: this option also
checks uninstrumented functions, including the reflection closure trampolines
that deliberately forward a suspension without replaying their temporary
argument buffers. The existing `TestReflectCallAndMethod` passes with the
default profile and traps with those extra assertions. The pinned Binaryen
does not expose a catch-only assertion switch. This PR therefore keeps the
default assertion policy, uses the extra checks only in the focused acceptance
matrix, and makes no new Binaryen or Emscripten patch. The generic warning remains.

The focused checks are runtime assertions, not a static suspension analysis or
complete support for suspending from arbitrary foreign exception handlers. An unsupported
path can still compile, and passing fixtures do not qualify arbitrary SDKs,
nested exception/control-flow combinations, or user-provided Asyncify exclusion
lists. Keep the warning and the explicit foreign-exception boundary below.

The Go regression verifies the absence
of JS `invoke_*` imports and the presence of native EH, then executes deferred
panic recovery and Goexit across suspension and GC at O0/O2 and O2 Thin/Full
LTO in Node and Chrome. LTO flags reach both compiler and linker drivers, and
SDK `emar` indexes bitcode so a newer SDK is not read by an older host LLVM.
The scheduler, timers, GC, lifecycle, callbacks, and multi-worker suites remain
required acceptance checks. C++ exceptions must still be caught inside a C++
wrapper and returned as a C ABI status; unwinding a foreign exception through
Go or a suspended goroutine is not supported by this change.

The Go/C++ matrix tests all three browser targets at O0/O2/Thin/Full LTO, with
general Emscripten assertions disabled. After a caught C++ exception returns,
Go sleeps, collects, panics, and sleeps/collects again in its recovering defer.
Separate direct and indirect callbacks deliberately sleep while the C++ catch
is still active; both must enter the callback, fail with an `unreachable` trap,
and never report a resumed callback. A timeout or an unrelated crash fails the
test. These negative callbacks do not throw and are declared `noexcept` on the
C++ side; they do not establish coverage of every C++ cleanup/unwind shape.

The catch-status fixture uses a `noinline` implementation behind a volatile
function pointer. LLVM's `noinline` attribute alone does not keep the boundary
through Binaryen's post-link inlining. Code after a source-level catch can also
be moved inside the Wasm catch by optimization, so merely moving a sleep after
the closing brace is insufficient.

Qualification also found that a late Go defer could allocate its jump buffer
below the fixed Wasm shadow-stack frame. A caught native C++ exception could
discard that allocation and later calls overwrite it, leaving a subsequent Go
panic unable to find its setjmp. Wasm jump buffers are now reserved in the
function entry block; this regression must pass with optimization and LTO.

## Encoding comparison and historical measurements

The script compiles the same C++ `throw`/`catch` and `setjmp`/`longjmp` fixture
with Emscripten's legacy EH mode and its direct standard `exnref` mode. It
also runs Binaryen's `--translate-to-exnref` pass on the legacy module. Each
variant is validated, checked for the expected EH instruction family and valid
DWARF structure, then executed with Node at `-O0` and `-O2`.
The translated variant reuses legacy Emscripten JS glue with only its companion
Wasm filename changed; it is not a full Emscripten exnref link.

Local result on 2026-09-27 (Emscripten 6.0.8-git,
[LLGo Binaryen `llgo-v132.3`](https://github.com/xgo-dev/binaryen/releases/tag/llgo-v132.3),
Node 26.8.1, wasm-tools 1.258.0, LLVM 22.1.8):

| Optimization | Legacy Wasm EH | Direct exnref | Binaryen translation |
| --- | ---: | ---: | ---: |
| `-O0` | 1,104,279 B | 1,107,030 B | 1,145,446 B |
| `-O2` | 238,326 B | 239,045 B | 238,312 B |

These historical columns compare candidate Wasm EH encodings. At the time,
the Go/C++ boundary used Emscripten JS EH, caught C++ inside the wrapper, and
returned a C ABI status to Go. The current native policy is described above.

All six variants passed validation, DWARF verification, and execution. The
separate Go panic/recover baseline passed. These measurements are a smoke
comparison, not a performance result. The `--browser` run passed
all six C++ variants, the Go panic/recover baseline, and both Go/C++ wrappers
in Chrome. These checks never let a foreign exception unwind through a Go
frame or a suspended goroutine.

The optional LLGo boundary fixture additionally passed at O0 and O2: Go
calls a C++ wrapper, the wrapper throws and catches internally, returns a C
ABI status, and Go carries that status into a panic and recovers it. The historical
wrapper used Emscripten's JS exception mode (`-sDEFAULT_TO_CXX` and
`-sDISABLE_EXCEPTION_CATCHING=0`) with `-fexceptions` on its C++ source.
Placing the native file under `_wrap/` is necessary here: if a `.cpp` file
also sits in the Go package directory, automatic C++ source collection and
`LLGoFiles` compile it twice, and the linker may select the object lacking
the requested EH flags. That historical wrapper result established a status-translation boundary
with the then-current Asyncify backend. By itself it did not establish that
a C++ exception could unwind through Go or qualify native EH for the whole
LLGo browser link.

## Supported EH boundary

C++ wrappers compile and link with `-fwasm-exceptions`, catch their own
exceptions, and return a C ABI status that Go may translate to a panic. The
isolated direct `exnref` and post-link translation measurements do not establish
whole-module browser support for those encodings. W32-WASI is qualified
separately with Wasmer 7.5.0 and LLVM's direct standard EH lowering, including
SjLj and LTO.

## Threaded WASI regression

`python3 dev/test_wasm_wasi_threads.py` tests W32 standard EH with the Wasmer
installed by `dev/install_wasmer.sh`. It repeats cross-function panic/recover,
C `setjmp`/`longjmp`, and deferred worker `Goexit` with GC on and off. Main/init
`Goexit` must execute the defer and report deadlock; an unrecovered worker
panic and a raw Wasm exception escaping `_start` must fail. A spawned-thread
probe combines SIMD calls, a v128 exception payload and atomic notification.

### Historical WAMR results (before the Wasmer migration)

WAMR 2.4.5 previously called `wasm_set_exception` while transferring a caught
exception to its Wasm caller. With threads enabled, that publishes a
cluster-wide termination signal before the caller can catch the exception.
Sibling threads can then exit early or leave a channel waiter hung. The former local
interpreter patch unwound directly to a Wasm caller and preserved the terminal
exception path when the exception escapes to the native invocation boundary.
The POSIX signal-handler backport from WAMR #5119 was applied separately.
Those build patches are no longer part of the default runner.

On macOS arm64, the same deferred-Goexit artifact passed 40/50 runs with stock
WAMR 2.4.5 (six hangs and four premature successful exits), 44/50 with the
signal-handler fix alone on WAMR main, and 100/100 with both fixes on 2.4.5.
The threaded acceptance suite and the expanded GC/nogc EH probes passed with
both fixes. These finite runs establish regression coverage, not a guarantee
that every WAMR threading issue is resolved.

The browser comparison was rerun on 2026-09-29 with the pinned LLGo Binaryen
`llgo-v132.3`: all six C++ encoding/optimization variants passed in Node and
Chrome, as did the Go baseline and the Go/C++ catch-status wrappers at O0/O2.
That historical run did not change the browser encoding. Whole-module exnref
across Go/Asyncify remains outside the supported contract.

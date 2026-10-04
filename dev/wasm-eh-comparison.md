# WebAssembly exception-handling comparison

Run `python3 dev/compare_wasm_eh.py` with Emscripten, Node, `wasm-tools`,
`llvm-dwarfdump`, and the selected Binaryen installation on `PATH`. Set
`EM_BINARYEN_ROOT` to select a complete Binaryen installation; optionally set
`LLGO` to include the existing Go panic/recover smoke test. Pass `--browser`
to execute all six C++ variants and, with `LLGO`, the Go baseline and both
Go/C++ wrappers in Chrome as well.
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

These columns compare candidate Wasm EH encodings, not LLGo's current browser
output. The current Go/C++ boundary uses Emscripten JS EH, catches C++ inside
the wrapper, and returns a C ABI status to Go.

All six variants passed validation, DWARF verification, and execution. The
separate Go panic/recover baseline passed. These measurements are a smoke
comparison, not a performance result. The `--browser` run passed
all six C++ variants, the Go panic/recover baseline, and both Go/C++ wrappers
in Chrome. These checks never let a foreign exception unwind through a Go
frame or a suspended goroutine.

The optional LLGo boundary fixture additionally passed at O0 and O2: Go
calls a C++ wrapper, the wrapper throws and catches internally, returns a C
ABI status, and Go carries that status into a panic and recovers it. The
wrapper uses Emscripten's JS exception mode (`-sDEFAULT_TO_CXX` and
`-sDISABLE_EXCEPTION_CATCHING=0`) with `-fexceptions` on its C++ source.
Placing the native file under `_wrap/` is necessary here: if a `.cpp` file
also sits in the Go package directory, automatic C++ source collection and
`LLGoFiles` compile it twice, and the linker may select the object lacking
the requested EH flags. This wrapper result establishes a safe status
translation boundary with the current Asyncify backend. It does not establish
that a C++ exception can unwind through Go or that direct Wasm EH can be
enabled for the whole LLGo browser link.

## Supported EH boundary

Keep the current validated Emscripten/LLGo EH encoding for browser builds.
C++ exceptions are caught inside a C++ wrapper; the wrapper returns a C ABI
status that Go may translate to a panic. Do not enable direct `exnref` or a
post-link translation for the whole LLGo module based on the isolated C++
comparison. Both remain compatible candidates for a later full-link test,
provided panic/recover, Asyncify suspension, Go/C/JS callbacks, final DWARF,
and the chosen runtime all pass together. W32-WASI is qualified separately:
it uses Wasmer 7.5.0 and LLVM's direct standard EH lowering, including SjLj
and LTO. This does not change the browser/Asyncify encoding.

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
Browser encoding remains unchanged. Whole-module exnref across Go/Asyncify remains outside
the supported contract.

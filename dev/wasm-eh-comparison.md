# WebAssembly exception-handling comparison

Run `python3 dev/compare_wasm_eh.py` with Emscripten, Node, `wasm-tools`,
`llvm-dwarfdump`, and the selected Binaryen installation on `PATH`. Set
`EM_BINARYEN_ROOT` to select a complete Binaryen installation; optionally set
`LLGO` to include the existing Go panic/recover smoke test. Pass `--browser`
to execute all six C++ variants and both Go/C++ wrappers in Chrome as well.
Individual tool paths can be set with `EMXX`, `NODE`, `WASM_TOOLS`,
`LLVM_DWARFDUMP`, and `WASMOPT`; `WASMOPT` takes precedence over
`EM_BINARYEN_ROOT`.

The script compiles the same C++ `throw`/`catch` and `setjmp`/`longjmp` fixture
with Emscripten's legacy EH mode and its direct standard `exnref` mode. It
also runs Binaryen's `--translate-to-exnref` pass on the legacy module. Each
variant is validated, checked for the expected EH instruction family and valid
DWARF structure, then executed with Node at `-O0` and `-O2`.

Local result on 2026-09-23 (Emscripten 6.0.8-git,
[LLGo Binaryen `llgo-v132.2`](https://github.com/xgo-dev/binaryen/releases/tag/llgo-v132.2),
Node 26.8.1, wasm-tools 1.258.0, LLVM 22.1.8):

| Optimization | Legacy EH | Direct exnref | Binaryen translation |
| --- | ---: | ---: | ---: |
| `-O0` | 1,104,113 B | 1,106,868 B | 1,145,311 B |
| `-O2` | 238,216 B | 238,931 B | 238,204 B |

All six variants passed validation, DWARF verification, and execution. The
separate Go panic/recover baseline passed. These measurements are a smoke
comparison, not a performance result. A second run with `--browser` passed
all six C++ variants and both Go/C++ wrappers in Chrome 153.0.8010.53.
Neither run permits a foreign exception to unwind through a Go frame or a
suspended goroutine.

The optional LLGo boundary fixture additionally passed at O0 and O2: Go
calls a C++ wrapper, the wrapper throws and catches internally, returns a C
ABI status, and Go translates that status into a panic and recovers it. The
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
and the chosen runtime all pass together. [WAMR's documented EH support](https://github.com/bytecodealliance/wasm-micro-runtime/blob/main/doc/build_wamr.md)
is currently limited to legacy EH in its classic interpreter, so the browser
comparison alone does not change the W32/WAMR execution contract.

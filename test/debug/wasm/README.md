# Wasm goroutine debugger acceptance

Run `python3 dev/test_wasm_browser_debug.py` from the repository root with
`LLGO` and `LLGO_BROWSER_CHROME` set. LLVM, Emscripten, Node 22+ and the pinned
LLGo Binaryen must be available.

The fixture keeps two goroutines parked in recursive Go calls, forces GC,
and stops at a C++ source breakpoint. The installed DevTools extension reads
their IDs, parent IDs, state, last processor and logical source stacks through
`$goroutines`. A second breakpoint checks that their records disappear after
exit and another GC. The probe also expands and releases the watch objects.

Acceptance covers Memory32/64 and 1/2/4 workers, with embedded and external
DWARF. The test attaches to every worker, installs the source breakpoint in
each Wasm instance and pauses all worker targets before inspecting shared
memory. This uses Chrome's debugger transport; it does not simulate runtime
state or replace thread imports.

Logical frames provide function/file/line, not parked-frame DWARF locals.
The frontend requires users to pause the other threads in DevTools before
inspection; it does not automatically halt unrelated workers. Registry and
frame sequence guards reject changes observed during a read, but do not
provide a globally simultaneous snapshot while workers are running.

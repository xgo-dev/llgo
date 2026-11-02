# Wasm goroutine debugger acceptance

Run `python3 dev/test_wasm_browser_debug.py` from the repository root with
`LLGO` and `LLGO_BROWSER_CHROME` set. LLVM, Emscripten, Node 22+ and the pinned
LLGo Binaryen must be available.

The fixture keeps two goroutines parked in recursive Go calls, forces GC,
and stops at a C++ source breakpoint. The installed DevTools extension reads
their IDs, parent IDs, state, last processor and logical source stacks through
`$goroutines`. A second hit of the same breakpoint checks that their records disappear after
exit and another GC. The probe also expands and releases the watch objects.

Acceptance covers Memory32/64 and 1/2/4 workers, with embedded and external
DWARF. The test attaches to every worker and installs the source breakpoint in
each Wasm instance. The product coordinator pauses the page and all workers;
the test does not issue those auxiliary pause/resume commands. It verifies
stable stop IDs, worker creation and termination while stopped, local values,
and execution through a step, a second hit of the source breakpoint and program completion.
New workers remain at Chrome's startup barrier during a stop; acceptance
checks that they cannot update shared memory until the group resumes.

Logical frames provide function/file/line. For waiting frames with saved base
addresses, the extension also reads locals whose DWARF locations stay in
linear memory across the executable scope. The fixture checks distinct arrays
in four/five recursive frames after memory growth and GC. Register-dependent
or changing locations and unwound panic snapshots remain unavailable.
The same stops inspect nested strings/slices, named interface payloads from
main and imported packages, typed nils, functions and closure addresses, a map
during bucket evacuation, indirect map keys/values, closed wrapped channels,
and zero-sized container elements. After resuming, a second stop verifies
mutations and channel draining; releasing the root also releases its children.
The extension checks the coordinated stop before and after inspection, and
rejects expanded objects from an older stop. Registry and frame sequence
guards remain an additional check. This uses Chrome's debugger transport and
the real runtime; it does not replace thread imports or qualify WASI guest
debugging.

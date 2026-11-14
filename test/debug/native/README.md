# Native source-debug acceptance

Run `bash test/debug/native/runtest.sh` with the current `llgo` in PATH.
The runner uses default native DWARF at O0 and O2, then the user-facing
`llgo debug` launcher at O2 with ThinLTO and Full LTO. It fails if any LLDB Python
assertion fails. `LLGO_LLDB` selects an installed LLDB.

The O0 cases require the shared runtime panic entry and the exact Go caller
line for explicit panic, integer division by zero, and invalid memory access.
The O2 case requires leaf and middle inline frames in source order, an actual
noinline caller, and step-over reaching the caller's following source line.
An O2 aggregate case also checks that addressable locals and parameters show
their current values after mutation through pointers.
Go/C/Go callbacks and host fault stacks are covered by the existing
`test/debug/runtime/mixed` suite.

The inline leaf calls a volatile C probe so whole-program optimization cannot
legitimately eliminate the instructions used for its source breakpoint.
Darwin LTO sessions require `dsymutil`; the launcher creates and removes a
private dSYM without changing the build output's debug packaging.

This dedicated panic/inline acceptance suite currently runs on Linux and
macOS CI. Windows is not yet covered by this suite: the runner and its
fault/inline assertions still need qualification with the Windows debugger
launch environment. Windows CI separately runs the runtime LLDB integration
suite and `TestDWARFPCLNLineSites`; these cover values/stacks and runtime line
lookup, respectively, but do not establish the panic/inline guarantees above.

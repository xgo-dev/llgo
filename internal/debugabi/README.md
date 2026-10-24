# LLGo debugger contract

`schema_v1.json` is shared by LLDB, GDB, and portable debugger frontends. Schema
version 1 describes the 16-byte record; runtime layout version 2 describes the
current runtime. Runtime layout changes require a new version even when the
record encoding stays unchanged.

Native debug builds retain `__llgo_debugger_abi_v1` and the original
`__llgo_debugger_marker_v1`. A marker by itself cannot identify current runtime
layouts. Adapters require a valid structured record with matching pointer size
and byte order. An unknown version, malformed record, or missing schema disables
runtime presentation while preserving ordinary debugger commands.

The record has no selectable C ABI mode. Bytes 12 and 15 are reserved and must
be zero. `NewRecord`, `MarshalBinary`, and `ParseRecord` implement the same
encoding that portable frontends can put in the `llgo.debugger` custom section.
Installing that section and external DWARF sidecars belongs to build artifact
processing, independently of native runtime inspection.

Native goroutine inspection reads the existing traceback registry through
`llgo_debugger_threads_v1`; it does not register another list of goroutines or
suspend target threads itself. Read the registry only while the debugger has
stopped all threads. Each record provides a Go ID, parent, state, and OS thread
ID. LLDB and GDB match that ID against their native thread lists for backtraces.
Goroutines not yet attached to a thread have no native backtrace. Targets without
this registry, including Wasm fibers and bare metal, report the capability as
unavailable rather than treating a goroutine as a host thread.

On a native Windows ARM64 host, the LLDB adapter supplies an unset code-address
mask from the operating system's user-address bounds before unwinding. This
allows LLDB to interpret authenticated return addresses in Windows library
frames. Explicit debugger masks, data addresses, remote targets, and crash dumps
are unchanged. Backtraces still depend on the debugger's native unwind support;
this does not add an ARM64 PE unwind-table decoder to LLDB.

Run `llgo lldb program` or `llgo gdb program` to load the installed adapters.
Both support `llgo status`, `llgo goroutines`, and `llgo goroutine ID bt`, plus
runtime-aware string, slice, interface, function, map, and channel values.

Interface summaries in the native LLDB/GDB adapters currently identify the
dynamic type and nil state; the raw data field remains available through normal
debugger field inspection. They do not decode the concrete payload. The schema's
`runtime_type.kind` (`Kind_`) and
`direct_interface_flag` (`KindDirectIface`, 32) describe the runtime contract
for portable frontends and future concrete-value inspection. They are not a
claim that native payload decoding is implemented. A decoder must check this
flag: direct interface data is the value itself, while indirect interface data
points to value storage. The interface's payload address alone is insufficient
to decide whether to dereference it. Debug-artifact processing transports this
schema and record; it does not provide a concrete-value decoder either.

Portable readers can use the optional `runtime_type.kind_types`, `uncommon_flag`,
`named_flag`, `uncommon_type` and `package_path` fields to resolve named interface
payloads by import path. Reflection's package-name spelling alone is ambiguous.
The uncommon record follows the kind-specific metadata structure; readers use
its DWARF size rather than fixed host offsets. `pointer_kind` and
`pointer_element` allow pointer payloads to resolve their element recursively.
Missing metadata leaves the concrete value unavailable. These fields describe
the existing layout; they do not change its version or native adapter behavior.

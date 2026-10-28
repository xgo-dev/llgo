# Explicit purego ABI adapter

This directory owns the Linux/amd64 native call backend for the **explicit**
purego v0.9.1 compatibility profile. It is not a translation rule for arbitrary
`syscall15X`, `runtime.cgocall`, or `crosscall2` symbols. An adapter must verify
the source module and replace the original backend before selecting these
assets. Nothing here automatically changes a package's calling convention.

`native/frame.h` specifies the packed frame: a C function address, six integer
argument registers, nine shared stack spill slots, eight vector registers,
and a reserved word. The C ABI entry returns RAX/RDX and XMM0/XMM1 in that
frame. The caller keeps all Go argument owners alive until the synchronous
call returns. It must not pass a Go function value or ABI0 entry as the C
function address.

The native bridge has its own frame and unwind information. It does not use
the Go-runtime trampoline's R12 scratch register or positive BP-relative
argument-block save slot. Both violate an ordinary C caller's contract. AL
contains a valid SysV upper bound for variadic vector arguments.

## Verification

Select LLVM 22 with `LLVM_CONFIG`, then run:

```sh
go test ./internal/puregobridge -count=1 -v
```

Every host checks the actual Linux/amd64 ELF object and unwind/stack metadata.
On Linux/amd64, the same command also executes a C oracle for strlen, all
15 integer slots, eight vector registers, mixed spilled arguments, both pairs
of result registers, float32, variadic floats, nested calls, stack alignment,
callee-saved registers, and caller-frame canaries. Fault-injected R12,
BP-offset, and AL regressions must fail controlled assertions. A missing or
wrong-version LLVM tool is an error.

These checks establish the C call transport only. They do **not** qualify Go
callbacks, foreign-thread GC ownership, the source adaptation, purego's full
API, or a Turso query. Those require separate LLGo compile/link/run tests.

## Staging and callbacks

`Adapt` accepts only the exact module/version, Linux/amd64, and cgo-enabled
profile. It checks hashes of the original backend, frame definition, call
marshaller, and dynamic-loader entrypoints before returning module-relative
replacements/removals. Apply those changes to a private stage, and select it
with a build-local modfile replacement. Never rewrite the shared module cache.
The staged Go file selects the native source through `LLGoFiles` and imports
the dedicated `runtime/compat/purego` callback package.

Callbacks use libffi's actual C signature and invoke the retained Go closure
through reflection. CDecl is omitted only after the adapter checks its type.
Unsupported signatures fail before executable allocation. A bounded table
retains at most 2000 successful callbacks for process lifetime, matching
purego's existing ownership contract. Failed closure binding frees executable
storage and does not publish roots or consume a table slot.

The C entry registers an adopted thread before reflection or allocation.
Nested calls reuse the G; newly adopted threads release their retained
collector/G registration through the existing TLS lifecycle. As in the
runtime's Windows callback entry, this boundary does not install a Go defer
that would need to unwind across libffi. Callbacks must handle their own
panics; this profile does not promise arbitrary panic propagation through C.

The separate LLGo fixture `runtime/_test/puregobridge` checks captured roots,
15 integer arguments, mixed floats/spills, narrow signed/unsigned returns,
pointer and bool values, two qsort closures, CDecl, void callbacks, concurrent
creation, foreign pthread GC with nested callbacks, and limit rejection. Run
it with LLGo on Linux/amd64; building its C helper alone is not an execution
pass. The real-library strlen and Turso probes remain separate acceptance
tests for the complete staged application.

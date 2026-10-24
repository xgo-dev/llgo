# GOROOT runtime reuse

LLGo selects source from the **same resolved GOROOT and Go version as the
application**. The build parser uses its existing source overlay to replace
declaration-only runtime files in memory. The selected bytes participate in
the ordinary package fingerprint; `//line` records retain their GOROOT source
locations. Neither source tree is modified.

## Current ownership

| Code | Owner | Boundary |
| --- | --- | --- |
| `countrunes`, `decoderune`, `encoderune`, UTF-8 constants | GOROOT `runtime/utf8.go` | Compiled into LLGo's low-level runtime; no Go scheduler, stack, GC or metadata dependencies |
| String iteration and string/rune conversions | LLGo low-level runtime | Existing compiler entry points; a small adapter handles Go 1.27's unsigned decoder offsets |
| `setCrashFD` (Go 1.23+) | GOROOT `runtime/runtime.go` | Only this declaration is selected, including its linkname |
| `crashFD`, `panicking`, fatal output and descriptor initialization | LLGo standard-runtime replacement | Atomic operations and output delivery remain LLGo-owned |
| Other standard-runtime replacement declarations | LLGo | Unchanged until individually reviewed |
| `newproc`, stack switching, TLS, GC metadata, write barriers | LLGo low-level runtime/backend | Full Go runtime and scheduler assembly remain gated |

`internal/build/runtime_source.go` is the checked selection list. UTF-8
imports, compiler/build directives, new declarations, and changed helper
signatures fail explicitly.
The selected crash-output function's signature is checked; new dependencies
must type-check against LLGo's remaining runtime. The normal type checker
rejects duplicate definitions. This is an initial migration, not a complete
closure-ABI audit for #2165.

The declarations in `runtime/internal/runtime/utf8.go` and
`runtime/internal/lib/runtime/crash_fd_go123.go` let `cmd/go` collect package
metadata before LLGo applies its source view. They do not contain alternate
algorithm implementations. The `utf8_index*.go` files retain the LLGo-owned
`decodeNonASCII` adapter; only their metadata-only `decoderune` declaration
is removed by the overlay. Adding another reused subsystem requires an
explicit selection, compatible boundary types, removal of its old bodies,
and runtime tests.

String slicing keeps separate runtime entries for signed indices that fit a
native `int` and for wider or unsigned indices. Go 1.27's reused decoder uses
an unsigned offset; it must not prevent LTO from specializing the common signed
path. The compiler checks the original index width before narrowing, and the
generic entry preserves wide values and signedness in bounds-panic messages.

## Ownership and upgrade audit

From an LLGo development environment, generate a source ownership report:

```sh
go run ./dev/runtimeaudit -goos=linux -goarch=amd64 -o=runtime-linux.json
```

`-dir` selects the application's module directory and therefore its resolved
Go toolchain. The report records that Go version, GOOS, GOARCH, build tags,
tool tags and `CGO_ENABLED`. It includes LLGo's default tags (`llgo`, `purego`,
`math_big_pure_go`) and the target's closure ABI tag. Source selection uses
the build's source overlays and alternative package declaration filtering.
Paths use `$GOROOT` and `$LLGO` prefixes so
reports can be compared across checkouts. The command fails on duplicate
declarations, conflicting Go implementations/linknames, or unresolved runtime
references; it writes the JSON report before failing for unresolved references.

The inventory covers declarations in both runtime layers, `runtime/abi`, the
merged `internal/abi` source view, public GOROOT runtime types/constants, and
runtime entry points supplied by other standard-library packages. Each record
includes its source owner, location, function signature, linkname, Go body,
file imports and function directives. Legacy coverage-main hooks appear as
compiler-generated entries with their instrumentation/version condition.
The requirement list includes selected
standard-library runtime/ABI selectors and linkname declarations, plus literal
compiler runtime lookups. Package-scoped linknames may appear in another file.

Compiler sources contain lowering for multiple targets. Literal lookups whose
declarations exist only outside the selected source view appear separately in
`other_target_compiler_uses`; type-dependent lookup expressions appear in
`dynamic_compiler_uses`. These lists need compiler/execution coverage. This is
a source inventory, not a call graph or proof of linker reachability: unused
standard-library declarations are checked too. Go declarations without bodies
record external boundaries. Assembly `TEXT` entries record source locations;
preprocessor conditions, generated offsets and ABI compatibility still require
the existing translator and execution tests. No assembly is enabled by the audit.

Hosted Wasm source selection needs the same profile tags as the build:

```sh
go run ./dev/runtimeaudit -goos=js -goarch=wasm \
  -tags=llgo.wasm.gc.linear,llgo.wasm.emscripten -o=runtime-js.json
go run ./dev/runtimeaudit -goos=wasip1 -goarch=wasm \
  -tags=llgo.wasm.gc.linear,llgo.wasi_threads,llgo.wasm.workers \
  -o=runtime-wasi.json
```

`TestRuntimeOwnership` checks Darwin arm64, Linux amd64, Windows
amd64/arm64/386 and these two Wasm selections against the active toolchain.
Set `LLGO_RUNTIME_AUDIT_DIR` to an existing directory when running that test
to retain its JSON reports. Negative fixtures cover new hooks, duplicate
owners, package-scoped aliases, variable alias storage and source/assembly
selection. Run this gate alongside the normal compile/execution matrix when
upgrading Go; a source report alone does not establish runtime compatibility.

The initial audit exposed hooks that were too narrowly selected:
`runtime.getAuxv` now exists on platforms without a process auxiliary vector,
where it returns an empty slice, and `runtime.nanotime` uses the platform clock
on Windows/Wasm as well as Unix. The `runtime.fastrand64` compatibility entry
now delegates to the platform random generator. Go 1.21's non-`purego`
`hash/maphash` and external packages use this entry; LLGo's default `purego`
selection does not require it. Direct linkname execution tests cover these hooks.

## Replacement-package assembly decision (#2128)

The current syscall implementation uses a source patch that suppresses GOROOT
assembly. It is no longer an alternate replacement package. The same selective
assembly question still applies.

| Candidate | Dependencies / assumptions | Decision |
| --- | --- | --- |
| Linux `syscall.rawSyscallNoError` | Kernel return registers; callers use the first result for process/user IDs and `umask` | Keep the existing libc bridge. This small helper does not remove libc from an LLGo executable. No claim of a general two-register raw-syscall ABI is made. |
| Linux amd64 `syscall.gettimeofday` | `runtime.vdsoGettimeofdaySym`, ELF symbol/version lookup and initialization before foreign C-library entry; raw syscall fallback | Keep libc's time implementation and errno adapter. libc already owns vDSO selection and kernel fallback; duplicating that loader for this single helper has no demonstrated benefit. |
| Linux `syscall.rawVforkSyscall` | Special stack/return-address treatment around process creation | Exclude; it must not be enabled as a side effect of selecting the same `.s` file. |
| Runtime syscall/clock assembly | Go `g`/TLS, scheduler/syscall state and platform-specific register conventions | Keep platform backends; not self-contained replacement candidates. |
| `mstart`, `mcall`, `systemstack`, `morestack` | Go stack growth, g0, register ABI, stack maps and preemption | Blocked by the scheduler/stack/GC compatibility gate. |

The inventory does not establish multiple independent helpers with a material
benefit from a new replacement-assembly selector. The proposal's necessity
gate therefore selects **keep bridges**. Ordinary-package Plan 9 translation
and its ABI checks remain available; this decision introduces no vDSO loader
or assembly allowlist.

`TestGettimeofdayAndRawFallback` covers Linux amd64 and arm64; the assembly
candidate above is specific to amd64. It compares the libc path with a direct kernel
time syscall and checks return/errno behavior, including a preceding failed
syscall. This is an explicit raw-syscall reference, not forced execution of
libc's internal fallback or proof that libc used vDSO. Native executable and
C-library tests exercise the same adapters without owning process startup.
Android, unsupported architectures, Wasm and embedded backends retain their
existing selection.

## Validation

- Source identity, source-change cache invalidation, missing/changed helpers
  and new-dependency rejection are tested beside the build implementation.
- Runtime tests cover every Unicode scalar value, invalid scalar values,
  all two-byte inputs, truncated/overlong UTF-8 and string/rune iteration.
- String-slice tests cover native, narrow, wide and unsigned indices, empty
  strings and exact bounds-panic messages; IR tests check 32/64-bit dispatch.
- The existing `runtime/debug` descriptor test exercises `SetCrashOutput`.
- Run the runtime/syscall tests in executable, `c-shared` and `c-archive`
  modes on the relevant native targets; run UTF-8 on the Wasm profiles too.

Remaining #2165 work includes qualifying dynamic compiler lookups and external
boundaries, further standard-runtime subsystems, and metadata, stack and
scheduler ABI compatibility. Hidden closure-environment passing is already
implemented by #2255; it does not establish full Go runtime ABI compatibility.
These source selections do not
establish compatibility with the full upstream runtime.

# PCLN metadata packaging

Status: initial implementation. This design is stacked on the typed Go linker
options introduced by #2113.

## Goals

LLGo supports three independent packaging policies for the runtime function
and line metadata used by `runtime.Caller`, `runtime.Callers`,
`runtime.FuncForPC`, and stack traces or pprof symbolization on the available
unwinding paths:

- `-pclntab=embedded` keeps the metadata in the executable. This is the
  default and preserves existing behavior.
- `-pclntab=external` writes `<executable>.pclntab`. The executable loads that
  optional file on the first safe symbolization request. A missing, stale, or
  corrupt file is a cached soft failure; the program continues with the
  available native fallback.
- `-pclntab=none` does not generate the metadata and does not link the external
  loader or its path/probe code.

`-pclntab=external` supports native executable builds without `-target` on
Darwin/Linux amd64/arm64, and hosted Wasm executables: WASI W32, Emscripten
J32/J64, and `GOOS=js GOARCH=wasm`. `ModeGen`, `c-archive`, `c-shared`, native
named targets, and other OS/architecture combinations reject it.

The policy is deliberately separate from Go's `-ldflags=-s` and `-w`:

- Native executable and archive builds preserve DWARF by default. As with
  cmd/link, Darwin `c-shared` defaults to omitting DWARF. `-w=false` explicitly
  preserves it and `-w` explicitly omits it.
- Wasm and named embedded targets retain their default omission policy.
  `-w=false` enables metadata when the selected target supports it.
- `-s` records native symbol-table omission intent and implies `-w` unless
  `-w` is explicitly set; the explicit value wins regardless of argument order.
- `-pclntab` controls LLGo runtime symbolization metadata independently.

`ModeGen` remains DWARF-free by default and emits it only for an explicit
`-w=false`. Fixed targets that always omit DWARF reject preservation; backends
without an omit capability reject omission. Archives do not have a final
linker step: `-w` controls their compiled members.

Use `-O0` for stable parameter and local-variable homes. Optimized builds run
LLVM and C ABI lowering while retaining source metadata; debugger variables
can still be optimized out. At O0, C ABI lowering preserves authoritative
aggregate homes so assignments and declarations agree. Optimized builds retain
parameter-home reuse even when DWARF is enabled.
The former `LLGO_DEBUG` and `LLGO_DEBUG_SYMBOLS` environment switches are not
part of this policy.

The build tests verify default DWARF. The LLDB suite verifies O0 parameter
updates and mixed Go/C callbacks; the native acceptance suite verifies runtime
panic source locations and O2 inline frames. Darwin
cold and warm cache builds both use persistent archive paths in their debug
maps, and runtime PC-line anchors remain present when DWARF is enabled.

Consequently all combinations are meaningful. For example,
`-ldflags=-s -pclntab=embedded` keeps Go-compatible runtime symbolization,
while `-ldflags=-s -pclntab=external` keeps it in an optional deployment
artifact.

## Package boundaries

Go-compatible build syntax is parsed once in `internal/goflags`, which is
shared by command entry points and the `flags.txt` golden-test driver.
LLGo-specific command flags such as `-pclntab` remain in
`cmd/internal/flags`. Both paths write only typed `internal/build.Config`
fields; the build package does not parse command strings.

The golden-test `flags.txt` format accepts native Go build flags. It accepts
`-flag=value` and `-flag value`, single or double dashes, blank lines, comments,
shell-style quoting, and multiple ordinary flags on one line. An unquoted
argument-list form such as `-ldflags=-s -w=false` consumes the remainder of its
line; quote that value to put another build flag on the same line. The debug
fixture uses `-ldflags=-w=false`, so ModeGen debug metadata follows the same
linker semantics as the CLI.

This is syntax compatibility, not complete linker-backend support. Among
`-ldflags` options, this change translates only `-s` and `-w` into typed LLGo
semantics; all other linker tokens remain in the normalized raw flag list and
do not yet affect LLGo's native link. Only unpatterned and `all=` `-ldflags`
values are currently interpreted; other package patterns are rejected. The
existing `-gcflags` frontend subset (`-lang`, `-N`, and `-l`) is separate.

IR golden tests remain non-debug unless a fixture explicitly selects
`-w=false`, so their `CHECK-NEXT` fixtures continue to describe stable LLVM IR
and do not depend on incidental debug metadata. Native integration and LLDB
suites exercise both default and explicit-DWARF executable paths.

`internal/build` validates the selected target, enables compiler metadata,
coordinates link finalization, and owns output paths. `internal/pclnpost`
contains host-side ELF/Mach-O inspection and rewriting; `internal/wasmdebug`
owns the Wasm identity-slot rewrite. `internal/pclnmap` owns the versioned
sidecar format. Build tags select the native loader/decoder in
`runtime/internal/lib/runtime/pclntab_external.go` or the Wasm implementation
in `runtime/internal/pcln`. Both runtime decoders keep their format constants
and validation rules synchronized with the host encoder.

This separation keeps both binary-format policy and command syntax out of the
compiler pipeline.

## Link and strip order

Externalization needs final linked PCs and the native symbol table to remove
LTO inline copies. It therefore cannot be implemented only by suppressing IR
generation. The required final-artifact order is:

1. Generate funcinfo and line records, but keep the external payload out of
   the synthetic main module.
2. Link the final executable with temporary link-site sections and a fixed
   identity slot.
3. Analyze the unmodified executable, join final PCs to the encoded records,
   and stage the sidecar.
4. Write the executable identity and clear link-only site data.
5. Apply any future native symbol-table removal requested by `-s`.
6. Perform final platform signing after all executable mutations.
7. Atomically publish the sidecar, then report the executable and sidecar as
   separate outputs.

#2113 currently implements typed `-s`/`-w` intent and DWARF omission; native
symbol-table removal is a later phase. When that phase is added, it must use
the ordering above rather than adding an independent post-link rewrite before
PCLN analysis. The current external detach step signs a mutated Mach-O itself;
that signing responsibility must move to the end of the unified pipeline when
a later native-strip mutation is added.

`-pclntab=none` takes the generation-time path: compiler metadata and link
sites are not produced. `-pclntab=embedded` keeps the existing post-link
prebuilt-table rewrite.

Darwin amd64/arm64 LTO executables with embedded pclntab and DWARF collect entry
addresses from the final native symbol table, joined to the funcinfo catalog
by symbol ID. A data-only carrier in the entry module reserves space for the
prebuilt table; the metadata relink grows it when the linked text range needs
more buckets. This preserves function bodies and dead-code elimination without
entry assembly or function-address roots. `LLGO_PCLNPOST=0` retains the legacy
dynamic-symbol fallback and its optimized source-location limitations.
PC-line anchors retain precise runtime source locations and carry no debug location.
Linux and Windows retain entry sites because they cannot reconstruct every
function address through a dynamic symbol lookup. External mode retains both
categories to construct the sidecar.

## Sidecar identity and addressing

The header records the format and runtime ABI versions, target, pointer size, section descriptors, payload checksum, and pairing identity. The executable has the same identity in an initialized 32-byte slot.

- Native targets use the SHA-256 of the linked executable before detachment. The code addresses are relative to the lowest file-backed load segment and relocate with the image base from `dladdr`.
- Hosted Wasm hashes the header and non-custom sections with the identity placeholder in place. Stripping custom debug and name sections preserves the pairing.

## Loading and failure semantics

The default lazy policy starts loading on the first safe public symbolization query. Hosted Wasm also supports the startup policies below. Fatal paths consult the loaded state and use fallback diagnostics while metadata is unavailable.

One caller reads and validates the sidecar. Lazy queries may wait on safe paths. With the explicit Wasm policies, queries use whatever is already available. Both success and failure are cached for the process lifetime.

The native implementation uses one bounded read, capped at 512 MiB, instead
of `mmap`. It opens nonblocking and accepts only regular files, so an optional
FIFO cannot hang the first query. Read-only mapping remains a possible
follow-up optimization.

The format is intentionally private to LLGo and versioned. Its unit is a
physical function: compiler-generated wrappers and adapters use ordinary
function records, while calling conventions and closure environment transport
remain outside PCLN.

## Hosted Wasm

```sh
llgo build -target=wasi -Oz -ldflags=-s -pclntab=external -o app.wasm ./app
llgo build -target=emscripten -Oz -ldflags=-s -pclntab=external -o app.mjs ./app
llgo build -target=emscripten-memory64 -Oz -ldflags=-s -pclntab=external -o app.mjs ./app
```

Each command writes `app.wasm.pclntab`, including when the requested output
is JavaScript glue. This is runtime metadata, independent of external DWARF.
The default remains `embedded`. A successful embedded/none rebuild removes
the previous module's sidecar.

Wasm has function-table indexes rather than native return addresses. The
small entry-site array remains linked; source records, strings and location
IDs are stored in the sidecar. In ordinary external builds, shadow-stack
instrumentation records numeric location IDs instead of rooting function and
file strings in every function. Symbolization resolves these IDs only after
the map is ready. Builds with `llgo.wasm.debugger` retain their existing
shadow-frame strings for the interactive debugger.

The linker reserves an initialized 32-byte identity slot. Finalization prunes
the metadata, computes the pairing token, and patches only that slot. Closed
programs that use logical locations keep the function records those locations describe, including inlined frames. When the runtime uses physical function-table indexes, the map also retains entries based on linked names. Code offsets, section sizes and DWARF
remain intact. The v4 sidecar records the physical Wasm pointer width (4 or 8
bytes); this does not change Go's `uintptr` contract. It contains no native
entry/PC relocation records. Final stripping and DWARF packaging remain
independent and happen after pairing.

Wasm executables with `-pclntab=external` can select a loading policy with
`-pclntab-load`:

- `lazy` is the default. The first safe symbolization query starts the load.
- `required` validates the sidecar before application package initialization
  and stops with an error when the map is unavailable.
- `background` starts a loader goroutine at entry. Until it publishes the
  map, symbol queries return the available fallback without waiting.
- `none` keeps the sidecar for offline use and performs no runtime reads.

An unresolved traceback includes logical location IDs. To resolve it with a
matching module and sidecar, run `llgo tool symbolize app.wasm < panic.log`.

The host resolves the sidecar as follows:

- WASI reads the sibling of `argv[0]`. For a relative executable path, an
  absolute `PWD` supplies its directory. The host must preopen that directory;
  an inaccessible sidecar is an ordinary soft failure. Hosts may instead set
  `LLGO_PCLNTAB` to an absolute guest path. The default `llgo run`/`test`
  Wasmer adapter preopens the build artifact directory and supplies this path,
  including for temporary outputs and Windows host paths.
- Emscripten resolves the actual module URL and appends `.pclntab` to its
  pathname, preserving any query. Browsers fetch over HTTP(S); Node reads a
  `file:` URL. The HTTP request and body read have a 5-second timeout.
- On JS hosts, the provider starts an asynchronous read. Startup policies wait through the core scheduler on the worker that started the request. Lazy loads yield through the standard-library scheduler timers. Both paths leave Asyncify waits on the system stack.

Wasm reads at most 64 MiB and accepts only regular filesystem files. Format,
ABI, target, pairing identity, checksum, section bounds and all referenced
record/string IDs are validated before table pointers are published. The
successful C buffer stays resident without a second full-size Go copy;
rejected buffers are freed. Both success and failure are cached for the
process lifetime. Under the lazy and background policies, a missing, truncated, corrupt, stale,
oversized or timed-out sidecar leaves available PCs and fallback diagnostics. Installing a sidecar after the first failed query requires a
restart. Fatal traceback paths never start or wait for a load; they use ready
metadata or immediately fall back. Resolved synthetic frames cache their
function/file strings; immutable location IDs keep their PCs stable. A frame
captured before loading can resolve later when the map becomes ready.

`dev/test_wasm_pclntab.py --browser` exercises real WASI/Node hosts and HTTP
browser loading, including lazy loading, terminal caching, synthetic-frame string/PC reuse, concurrent callers
with explicit GC, fatal tracebacks without loading, malformed records, missing
files, FIFO rejection and HTTP timeout. It runs in the existing Wasm runtime CI job.

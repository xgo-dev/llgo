# Final WebAssembly size optimization

Use LLGo's cross-platform size levels and Go-compatible linker stripping:

```sh
llgo build -Oz -ldflags=-s -o app ./app
llgo build -target=wasi -Oz -ldflags=-s -o app.wasm ./app
llgo build -target=emscripten -Oz -ldflags=-s -o app.mjs ./app
```

Explicit `-Os` and `-Oz` select the corresponding LLVM size optimization on
all targets. Explicit `-Oz` also runs a final Binaryen `-Oz` pass for hosted
W32, J32 and J64 executables after required linker/Asyncify transformations. The pass
uses an explicit feature allowlist; only J64 enables memory64, and WASI retains its multimemory capability. Inferred target
optimization defaults and `-Os` retain their existing pipeline. Generation, libraries
and firmware conversions continue to use their normal compiler optimization.

Closed WASI executables built with explicit `-Oz` also use Go reachability
pruning with LTO off or with `-lto=full`. ThinLTO, external LTO passes,
opaque native inputs and unknown host exports keep conservative linking.
Coverage, logical debugger frames and frontend-cache builds also keep the
conservative path. Output-only linker diagnostics such as
`LDFLAGS=-Wl,--print-map` use the same reachability analysis. With `-x` or
`-compiler-verbose`, look for `enable closed-program WASI size pruning`
or a `retain WASI code and metadata` diagnostic. The shared check reports
opaque inputs with `retain global initializers: native inputs in …`. Builds
outside the eligible configuration skip the release pruning check.
The compiler can specialize default formatting for concrete values, including
direct value methods it can prove cannot panic. It can also reuse Go's format
parser with a scalar-only dispatcher when all arguments have known builtin
types. A `retain generic fmt` diagnostic identifies open link inputs,
foreign entry points, unknown native input, or a reachable general formatter.

Development LLGo builds also specialize typed value formatting on closed
Emscripten J32/J64, macOS arm64, and supported Linux programs with
`-Oz -lto=full -deadcodedrop -globaldce=false`. Native links can prune type
metadata using the same reachability proof as WASI, with assembly references
and native entry points included in the analysis. Native archives can guard
static map construction in standard-library leaf packages. The final link
can omit a map when its initializer accepts only static data and the program
has no reachable use of the map. This configuration can replace
`encoding/binary.Encode`, `Decode`, and `Size` with typed codecs for fixed-width
layouts and standard byte orders. Mixed or dynamic uses retain the standard
codec. With `-x`, selected codecs report
`llgo: specialize fixed-size binary encoding`.

Emscripten Full LTO size builds repeat Binaryen size passes while the
deployment module shrinks. Embedded and external DWARF profiles use a single
pass.
With DCE, the final link can call a proven allocation-free and synchronous
executable directly. The direct entry skips Go runtime startup, its scheduler
and Asyncify. The executable package also omits its caller frames and polls.
When the link needs no Go runtime objects, Emscripten provides the allocator
used by C and the host API. Small console writes use stack storage. A final
link without the collector can still use these runtime helpers when all their
live references resolve. Dependency archives keep their normal code.
For other closed single-worker size builds, Full LTO can prove that the
scheduler is unused even when the program needs Go runtime initialization.
The final link first tries direct execution with unresolved scheduler markers.
If the linker still needs a marker or another Asyncify dependency, LLGo links
with the normal scheduler using the same archives.
The JavaScript SjLj profile preserves call boundaries in frames that use
`setjmp` to limit duplicate unwind checks after LLVM inlining.

The linear Wasm collector attaches finalizer and cleanup hooks on first
registration. The single-worker timer scheduler likewise attaches its hooks
when a real timer is first started or reset. Full LTO can discard each
implementation when its registration is unreachable.

Named Emscripten size executables with DCE and Full LTO also try Closure's
SIMPLE optimization at final link. SIMPLE preserves property names used by
host code through `Module.FS` and other runtime exports. Emscripten's default
optimizer handles builds where Closure is unavailable or fails. Builds with
explicit linker or Closure settings follow the selected driver configuration.
The Wasm manifest records a successful automatic invocation as
`js_minifier: "closure-simple"`.

Closed Emscripten size builds can omit the setup of unreferenced Go standard
files. The final link selects the setup to omit from the cached `os` archive.
Code that uses or reassigns a standard stream keeps its initialization.
Single-worker Full LTO size builds with supported Go sources read host
filesystem properties during initialization and create Go JavaScript handles
on first use.

Closed WASI and Emscripten size links can also omit unused standard-library
slice literals, literal errors and interface values made from constants.
Slice tables remain initialized when reachable code can access their backing
data through another variable or symbol. Literal errors use the same checks
on initializer references after verifying the selected `errors.New`
implementation. The final link chooses the initializers to keep from the
shared package archives.

Wasm value-method wrappers precompute nil receiver panic messages when the
compiler knows both names. Caller instrumentation reconstructs strings from
the original compiler literals using their known data pointers and lengths.

With DCE and Full LTO, size builds can defer the private reflection lookup
in `errors` to the concrete-target check in `errors.As`, which caches the
descriptor on first use. Programs that only construct or compare errors can
release that reflection code at link time.

In `-Oz -deadcodedrop -lto=full` builds, the compiler can lower direct
`encoding/json.Marshal` calls to typed encoders. Closed shapes can use
integers, booleans, strings, pointers, arrays, slices, structs and string-keyed
maps. Custom marshalers, dynamic inputs and other reachable codec operations
retain the standard encoder. The verbose diagnostic is
`specialize statically typed JSON encoding`.
When the entry has no other JSON uses and the JSON initializers only build
private state, the compiler omits those calls from the executable's startup.
Explicit blank imports and other JSON consumers retain initialization. The
standard-library archives are shared across both cases. The same build profile
can lower supported `encoding/json.Unmarshal` calls to typed decoders on closed
WASI, Emscripten, macOS arm64, Linux amd64/arm64 and bare-metal builds. This path
uses the v2-backed legacy API, enabled by default in Go 1.27 or with
`GOEXPERIMENT=jsonv2` in Go 1.26. Unknown native references keep codec startup.
Linux amd64 size builds use Go's software span scanner for the standard
runtime package. LLGo's collector is selected independently.
Allocation profile counters use zero-initialized storage, and their size
classes are computed from each bucket's index.

Ordinary Wasm builds reuse already-required backing-allocation roots for
pointer and slice views, preserving cold-path publication points. Loads, phi
nodes and allocating conversions keep independent roots. Frame bookkeeping
stays inline to avoid extra calls in ordinary builds. Go reachability pruning
can introduce owner roots and outline frame bookkeeping at `-Oz`.

Wasm type descriptors use fixed-size equality/hash helpers for regular-memory
values and direct hashers for scalar keys. Padded structs, blank fields and
interfaces keep their general algorithms. Private map buckets retain allocator
layout without source-type reflection metadata. Exact pointer finalizers use
the linear-GC dispatcher directly; dynamic types, foreign callbacks and debugger
frames retain the general path. These local lowerings let ordinary linker DCE
discard unnecessary generic algorithms and descriptors.

`-ldflags=-s` strips optional Wasm names and DWARF, as it strips native symbols
and debug information. `-ldflags="-s -w=0"` strips names while retaining DWARF
and source maps. `-w` alone controls DWARF and does not request name stripping.
The built-in metadata stripper requires no Binaryen when size optimization is
disabled. Code/data section bytes, runtime PCLN, target features, build identity,
external DWARF references and unknown custom sections are retained.

For explicit Wasm metadata policy, `-wasm-strip=debug` removes DWARF, name and
source-map metadata; `-wasm-strip=all` additionally removes producer metadata.
The default is `none` unless `-s` requests stripping. Explicit
`-wasm-strip=none` conflicts with `-s`. PCLN policy remains independently
controlled by `-pclntab`.

To defer runtime symbol strings and line tables as well, use
`-pclntab=external`. Hosted W32/J32/J64 and `GOOS=js GOARCH=wasm` write an
optional `app.wasm.pclntab` beside the module. Ordinary shadow frames then
record location IDs instead of embedding source strings. The first safe
symbol query loads and validates the map; missing or failed loads are cached
and execution continues with fallback diagnostics. Browser fetches time out
after 5 seconds. Fatal tracebacks never initiate loading. Programs that never
query symbols need no sidecar request. See
[PCLN packaging](design/pclntab-packaging.md#hosted-wasm) for deployment and
failure semantics.

Fatal tracebacks without loaded metadata include logical location IDs. Resolve
them with the original Wasm module and its adjacent `.pclntab` file.

```sh
llgo tool symbolize app.wasm < panic.log
```

For Emscripten, size mode sets `EXPORT_ALL=0`. Explicit native/public exports
and the existing runtime methods (`cwrap`, `allocateUTF8`, `stringToUTF8`,
`UTF8ToString`, `FS`, `setValue`, `getValue`, `ENV`) remain available; incidental
JS implementation symbols are no longer published on `Module`. `EXPORT_ALL`
controls publication, not reachability; the final pass does not remove these
explicit runtime APIs.

## Toolchain and debugging

Use LLGo Binaryen `llgo-v132.3`, selected through `WASMOPT` or
`EM_BINARYEN_ROOT`. Browser builds must also set `EM_BINARYEN_ROOT` so
Emscripten uses the patched tools during linking, not only the final pass.
CI pins the release archive checksum. Size optimization
checks the Binaryen 132 version string and records the executable SHA-256;
the version string alone cannot distinguish upstream Binaryen from LLGo's
patched build.

Debug packaging happens after optimization:

```sh
llgo build -target=emscripten -Oz \
  -debug-artifact=external -wasm-strip=debug -o app.mjs ./app
```

Embedded DWARF is incompatible with explicit stripping. External DWARF keeps a
matching `app.debug.wasm`; stripping applies only to the deployed module.
When retaining DWARF, the final pass skips Binaryen 132's `precompute`,
`precompute-propagate` and `local-cse` passes, which lose surviving Go call-site
line mappings. Other `-Oz` passes retain Binaryen's DWARF compatibility checks.
Emscripten source maps requested through `LDFLAGS=-gsource-map` are regenerated
for the optimized code, retaining their URL and source locations after publication.
Debug stripping removes the map file and URL together.
CI checks Go/C++ source locations and executes the final modules for W32/J32/J64.

## Artifact manifest

An enabled final pipeline writes `app.wasm-manifest.json` beside the output.
It records compiler/source-Go/LLVM/link-driver identity, target profile and
pointer width, PCLN and final optimization modes, Binaryen identity/arguments
(empty for built-in stripping), effective linker driver arguments (including
environment flags and generated defaults), declared and allowed Wasm features, and each artifact's byte count,
gzip-9 byte count and SHA-256. Its build ID covers sorted artifact names and
hashes. Tool names and absolute flag paths are normalized to omit local machine
directories; the manifest is a build record, not a portable linker command.
Debugger build IDs remain paired with the external DWARF container.

`payload` includes Wasm, JS/HTML loaders, worker assets and runtime symbols;
`debug` counts external debugger containers and source maps separately. Gzip totals sum files
compressed independently. Totals exclude the manifest itself; LLGo's artifact
report includes its actual size. Include it when measuring a deployment that
ships the manifest. External debug containers contain the module's standard
sections as well as DWARF, so their size is not just the DWARF payload.

The complete output set is staged before publication. Ordinary publication
failures restore the previous files; an unsuccessful restore retains its backup
and reports the recovery path. The manifest is published last. This is not an
atomic multi-file deployment: serve a versioned directory or verify artifact
hashes before switching readers. Successful builds remove obsolete debug/glue
siblings, and a successful rebuild without final optimization or stripping removes
the old manifest.

## Measurement

Darwin arm64, Go 1.27.2, LLVM 22.1.8, Emscripten 6.0.8 and LLGo Binaryen
`llgo-v132.3`; fixtures `benchmark/binary_size/{cprintf,println,fmtprintf}`.
The baseline uses LLGo's default optimization and `-debug-artifact=none`;
the final column additionally selects explicit `-Oz` and metadata stripping.
The named targets already default to LLVM `Oz`. Neither column uses
development DCE or LTO. Baseline: `057851875`. The measurements do not include the separate initializer/metadata pruning in PR #2908. All 18 final
products executed successfully in Wasmer or the matching Node runner.

Sizes below are **complete artifact bytes / independently compressed Brotli-11
bytes**, including generated JS, `wasm_fs.js` and the final column's manifest.
There are no deferred runtime assets in these cases. Small browser programs
can have a slightly larger compressed total because of the manifest.

| Profile | Example | Baseline | Final | Raw reduction |
| --- | --- | ---: | ---: | ---: |
| W32 | cprintf | 160,243 / 44,666 | 118,610 / 40,517 | 25.98% |
| W32 | println | 159,933 / 44,678 | 118,358 / 40,387 | 26.00% |
| W32 | fmtprintf | 2,371,607 / 514,578 | 1,937,708 / 495,349 | 18.30% |
| J32 | cprintf | 252,135 / 70,298 | 248,991 / 70,371 | 1.25% |
| J32 | println | 251,679 / 70,215 | 248,508 / 70,129 | 1.26% |
| J32 | fmtprintf | 3,139,290 / 703,680 | 3,072,575 / 700,958 | 2.13% |
| J64 | cprintf | 242,604 / 66,670 | 239,736 / 66,807 | 1.18% |
| J64 | println | 242,194 / 66,485 | 239,296 / 66,574 | 1.20% |
| J64 | fmtprintf | 2,908,050 / 642,098 | 2,848,135 / 640,863 | 2.06% |

## Optional runtime symbols

Measured against main `0539ea6e2` (including #3108), with the same `-Oz
-debug-artifact=none -ldflags=-s` settings on both sides. Only
`-pclntab=embedded` versus `external` differs. Darwin arm64, Go 1.27.3,
LLVM 22.1.8, Emscripten 6.0.8 and LLGo Binaryen `llgo-v132.3`;
fixtures `benchmark/binary_size/{cprintf,println,fmtprintf}`. Every before/after
product executed in Wasmer or the matching Node runner. No DWARF is present.

These are **uncompressed Wasm module bytes**, with the optional map reported
separately. JS glue and manifest bytes are excluded from this table. The
sidecar is deferred, not compressed: shipping/loading it increases total bytes,
and first symbolization uses additional resident memory. Small programs can
grow because the identity/loader support exceeds their removed metadata.

| Profile | Example | Embedded module | External module | Change | External PCLN file |
| --- | --- | ---: | ---: | ---: | ---: |
| W32 | cprintf | 12,745 | 12,829 | +0.66% | 256 |
| W32 | println | 14,490 | 15,348 | +5.92% | 256 |
| W32 | fmtprintf | 456,380 | 382,223 | -16.25% | 385,880 |
| J32 | cprintf | 151,656 | 154,393 | +1.80% | 84,616 |
| J32 | println | 150,855 | 153,584 | +1.81% | 84,616 |
| J32 | fmtprintf | 1,111,100 | 729,940 | -34.30% | 397,400 |
| J64 | cprintf | 138,876 | 141,371 | +1.80% | 84,616 |
| J64 | println | 138,077 | 140,580 | +1.81% | 84,616 |
| J64 | fmtprintf | 1,046,497 | 665,101 | -36.45% | 397,320 |

For `fmtprintf`, the data section drops from 159,501 to 83,599 bytes (W32),
474,571 to 90,613 (J32), and 474,801 to 90,892 (J64). This reduction comes
from runtime metadata and shadow-frame source strings in ordinary data
sections, rather than removing DWARF. The code section changes slightly to
support lazy loading.

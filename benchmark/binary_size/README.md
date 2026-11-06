# Binary size workloads

These fixed programs keep LLGo binary-size measurements comparable across
linker and PCLN packaging changes. The comprehensive linker/PCLN matrix is
maintained by `llgo-test/benchmarks`; `benchmark/baseline` reuses the three
smallest workloads for the per-commit lightweight size and timing gate.

- `cprintf`: the smallest C-library hello world, using only `lib/c.Printf`.
- `println`: a hello world using the built-in `println`.
- `fmtprintf`: a hello world using `fmt.Printf`.
- `texttemplate`: a heavier standard-library program using `text/template`.
- `nilfault`: an import-free abnormal path that dereferences a nil pointer.
- `reflectmetadata`: enumerates method names/signatures without observing
  `reflect.Method.Func`; its unused method references the JSON encoder.
- `reflectcall`: calls that JSON method through `reflect.Method.Func`, checking
  that method pruning preserves observable function values.
- `reflectcallbacks`: models Gorm's loop over nine hook names and calls the
  two hooks implemented by a model. An unrelated JSON method exposes the cost
  of treating the loop variable as an arbitrary method name.

For the native macOS Deadcode Drop/ThinLTO experiment, build a development
compiler and a plugin matching the selected linker, then run:

```sh
go build -tags dev -o /tmp/llgo-size ./cmd/llgo
python3 dev/measure_deadcode_thinlto.py \
  --compiler /tmp/llgo-size --plugin /path/to/LLGOLTOPlugin.dylib \
  --output /tmp/llgo-size-results --repeat 4
```

Use `--linker /path/to/ld64.lld` to select an explicit linker for every row.
The probe checks program output, records section sizes and symbols, and
normalizes all artifacts with hidden exports and post-PCLN local-symbol
stripping/signing. Builds use `-a -Os -debug-artifact=none`, retain embedded
runtime PCLN, and rotate the order of Drop, Thin+Drop, Full+DCE and Full+DCE+P.
Timing covers forced package rebuilds and linking, excluding the common final
strip/sign step; OS and tool caches may be warm. Run without competing builds
for timing comparisons. It is a size/build-time probe, not a runtime benchmark.

To include the complete Gorm schema test executable, prepare a separate Go
module with `gorm.io/gorm@v1.31.2` and its test dependencies, then run:

```sh
python3 dev/measure_deadcode_thinlto.py \
  --compiler /tmp/llgo-size --plugin /path/to/LLGOLTOPlugin.dylib \
  --linker /path/to/ld64.lld --gorm-module /path/to/gorm-probe-module \
  --cases reflectcallbacks gormschema --modes drop thin fullp \
  --output /tmp/llgo-gorm-size-results --repeat 3
```

The `gormschema` case uses `llgo test -c` and runs the complete schema suite for
every artifact. The module version is recorded with the result. See
`dev/deadcode_reflect_results.json` for this experiment; the earlier four-case
measurements remain in `dev/deadcode_thinlto_results.json`.

Thin+Drop associates unresolved `reflect.Value.MethodByName` calls with stable
source identities. After package optimization, a bounded LLVM analysis recovers
finite name sets from string constants, selects/phis, and readonly array/slice
loads. Each site's result refines only that source demand before Drop decides
method slots and regenerates ThinLTO summaries. It does not require a merged
FullLTO module or a linker plugin. Cached builds read the same optimized bitcode
from archives. Unknown calls, missing calls, writable or escaped storage, and
incomplete clone results retain generic reflection. Package-private source
ownership, linkname/assembly references and opaque native inputs are checked
before reading a global initializer as immutable. This refinement currently
shares the native macOS development-mode, single-root, no-DWARF restriction of
the Thin archive rewrite.

The first four programs must exit successfully with their expected output.
`nilfault` must exit unsuccessfully with a nil-pointer diagnostic in every
mode; for `external`, check it once with the sidecar present and once after the
sidecar has been removed. The runtime integration suite separately verifies
that a hardware-fault signal path never starts sidecar I/O.

Size reports for `-pclntab=external` must record the executable and its
`.pclntab` sidecar separately, with their sum shown explicitly. Run every
linker/PCLN variant at the same output path: DWARF records paths, so changing
the output path can otherwise make byte counts look different even when two
flag combinations have equivalent code-generation semantics.

External PCLN is built from the final linked entry and PC-line sites. Dead
function and PC-line records are removed together with unused strings, and
function indexes and hash tables are rebuilt before writing the sidecar. To
check this optimization, compare normal and Full-LTO `cprintf` and `fmtprintf`
builds with `-pclntab=external -ldflags=-w`, recording the sidecar's function,
PC-line and site counts as well as its bytes. The executable's text bytes must
remain unchanged.

Embedded native executables use the same final-site liveness and ownership
checks. When records can be removed, the build regenerates only the synthetic
entry object's metadata and relinks, then verifies the live set against the
original catalog. This reclaims the actual bytes without changing runtime
table formats or adding a runtime decoder. It adds one link when pruning is
needed. Builds without entry sites, library modes and named targets retain
their existing policy; the Go global DCE dev gate is unchanged.

This initial matrix applies to native Darwin/Linux executable builds;
generation, archive/shared, and fixed-target policies are outside this
benchmark.

Native Linux DWARF is zlib-compressed by default. Include an explicit
`-ldflags=-compressdwarf=false` control when comparing debug-information
volume, and distinguish executable size from compressed DWARF section size.
Darwin DWARF measurements must include a separately generated dSYM when
reporting complete debug artifacts. Size-optimized main executables restrict
automatic dynamic exports independently of DWARF; this preserves debugger
metadata and runtime PCLN. The default optimization level, LTO selection, and
development-only Go global-DCE policy are unchanged.

The comparison matrix is the Cartesian product of
`embedded|external|none` and these linker groups: default, `-w`, `-s`, and
`-s -w=false`. Native Linux/Darwin `-s` removes ordinary symbols after PCLN
finalization. Check that `-s` is smaller than `-w`, and that `-s -w=false`
preserves usable DWARF while reducing executable symbol overhead. ELF retains
dynamic symbols and debug sections when requested; Mach-O retains required
dynamic-binding entries and generates a matching `.dSYM` before stripping.
Record the complete dSYM bundle separately and include it in debug totals.

For WASI, compare each workload with the same compiler/toolchain and final
Binaryen `wasm-opt -Oz` pass. Record both the linked module and the final file,
then run the final file under the supported Wasmer runner. Keep three separate
columns: default, dev `-deadcodedrop`, and dev
`-deadcodedrop -lto=full -globaldce=false`. Full LTO otherwise selects the
existing LLVM GoGlobalDCE path, so those results are a different configuration.
Do not add `-s` to this comparison.

Optimized Wasm executables discard unused scalar global stores and safe blank
map initialization, preserving operand effects and potential panics. Globals
containing GC pointers remain roots even without explicit reads. Native inputs
with unknown references disable this analysis. Original dev DCE also clears
method metadata for types outside its interface/reflection domain; observable
method sets and structural metadata remain intact. WASI executables without
address-based `FuncForPC` entries prune embedded PCLN against linked function
names only when the name map covers every imported and defined function.
Sparse maps retain all records. Emscripten, library outputs and entry-bearing PCLN retain their current
packaging policy. `dev/test_wasm_wasi_threads.py` and
`dev/test_wasm_wasi_lto.py` check initialization, map/reflect behavior and source
locations, including an unrecovered panic, across the default and LTO paths.

## Startup and literal formatting

Optimized WASI executables can omit Go runtime startup and instrumentation when
all reachable initializers and functions are proven to need no Go allocation,
panic handling, callbacks or scheduler. The proof accepts restricted integer
and boolean operations, string values, print/println and constant C strings
passed to a no-argument C printf. Unknown
calls, recursion, external inputs/exports, owner-local globals, linker overrides,
coverage hooks, debugger frames and non-WASI targets retain the full runtime.
Package cache keys include this decision; development-only DCE flags retain
their existing policy.

Constant fmt.Printf/Sprintf calls with no formatting arguments can use a literal
path. Fprintf retains its original caller
identity for user Writers. Only escaped percent signs are decoded. Other formats
use the general formatter; Writer.Write calls, counts, errors and current Stdout
are preserved. Neither optimization applies at -O0 or with debugger frames.

For size comparisons, report linked bytes and final optimized bytes separately,
with the same Binaryen version and flags for each compiler. Include JavaScript,
manifest and symbol sidecars in deployment totals where present; a minimal WASI
result does not establish parity for fmt, browser or embedded workloads.

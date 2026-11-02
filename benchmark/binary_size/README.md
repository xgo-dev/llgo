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

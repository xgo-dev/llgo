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
- `reflectmetadata`: observes a method's name and signature without its Func.
- `reflectcall`: invokes the reflected method, so its body must survive.
- `reflectcallbacks`: looks up a fixed list of Gorm-style callback names and
  invokes two hooks; the unused `UnrelatedJSON` method pulls in encoding/json.

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

## IR-refined Drop with strong overrides

The native macOS development experiment refines unresolved MethodByName sites
from optimized package IR before running Drop. It uses the existing strong
type overrides and native object archives; it does not rewrite package method
tables or enable LTO. Only packages with unresolved reflection sites save an
analysis-only `.reflect.bc` cache sidecar. The snapshot is captured before
native codegen, is never linked, and contains no program-specific pruning.
Missing required snapshots cause cache misses. Global visibility is checked
again for every final program.

The regression suite checks positive name recovery, mutable/escaping names,
independent unknown lookups, cross-package mutation, and a cached dependency
containing the actual reflection loop. Both symbols and runtime output are
checked. The cache fixture asserts an actual cache hit on its second build:

```sh
go test -tags dev ./cl -run '^TestBuildAndCheckSymbolsFromTestdropIRReflection$' -count=1
go test -tags dev ./internal/build -run '^TestReflectIR' -count=1
```

For a controlled comparison, build ordinary-main, IR-refined-strong, and
IR-refined-Thin compilers against the same upstream revision, then run:

```sh
python3 dev/measure_deadcode_strong.py \
  --baseline /path/to/llgo-main \
  --strong /path/to/llgo-strong \
  --thin /path/to/llgo-thin \
  --gorm-module /path/to/prepared-gorm-module \
  --linker /path/to/ld64.lld \
  --output /tmp/llgo-strong-comparison --repeat 3
```

The prepared module must pin Gorm (the recorded experiment uses v1.31.2).
All variants use the same runtime/source root, output path, `-a -Os -p=10`,
embedded PCLN, omitted DWARF and final strip/sign steps. Run the matrix without
other compiler/test jobs. Downloads are prepared before timing. Every Gorm
binary runs the full schema test suite. The runner saves per-run commands,
compiler hashes, section sizes, function symbols and execution logs.

Compare method-slot decisions separately from final machine-code entries.
Equal LiveSlots do not imply equal function counts: ThinLTO also imports,
inlines, specializes, and removes code. The runner counts unique addresses in
`__TEXT,__text` and checks them against Mach-O `LC_FUNCTION_STARTS`; a missing
symbol alone is not proof of source-function DCE.

To compare Drop's actual decisions, use diagnostic compiler overlays without
changing either checkout:

```sh
python3 dev/audit_deadcode_live_slots.py \
  --strong-root /path/to/strong-checkout \
  --thin-root /path/to/thin-checkout \
  --gorm-module /path/to/prepared-gorm-module \
  --linker /path/to/ld64.lld \
  --output /tmp/llgo-live-slots
```

This exports the full type-to-live-slot maps and compares their contents. The
instrumented compilers are not used for timing or final-size measurements.

### Recorded macOS comparison, 2026-11-07

All three compilers use upstream `0c9212756`; the Thin control reapplies the two
commits from PR #3008 to that base. These are three-round medians from a serial
matrix, not the older PR's measurements. Raw sample metrics, compiler hashes,
analysis-map hashes and reproduction details are in
[`dev/deadcode_strong_results.json`](../../dev/deadcode_strong_results.json).

| Case | Mode | Bytes | Function entries | Wall seconds |
|---|---|---:|---:|---:|
| reflectcallbacks | main Drop | 1,687,600 | 3,099 | 7.16 |
| reflectcallbacks | IR-refined Drop + strong overrides | 1,027,616 | 1,330 | 7.11 |
| reflectcallbacks | IR-refined Drop + ThinLTO | 862,624 | 814 | 6.62 |
| gormschema | main Drop | 4,585,440 | 7,647 | 11.06 |
| gormschema | IR-refined Drop + strong overrides | 3,512,784 | 4,651 | 10.83 |
| gormschema | IR-refined Drop + ThinLTO | 3,314,240 | 3,632 | 15.11 |

The full LiveSlots maps match between strong overrides and ThinLTO: 79 types /
112 slots for reflectcallbacks, and 285 types / 649 slots for Gorm. Both remove
UnrelatedJSON. Final function counts differ even with identical method-slot
decisions. On Gorm, strong overrides are 23.39% smaller than ordinary main Drop,
5.99% larger than ThinLTO, and use 28.30% less wall time than ThinLTO.

All 18 matrix binaries execute successfully. All nine Gorm binaries pass the
62 top-level schema tests. Positive and conservative reflection regressions,
including an asserted dependency cache hit, also pass.

This remains an experiment: a separate workqueue v0.36.2 check builds in both
modes, but the strong-override branch reproduces `unreachable method called`
while main Drop passes. This is an explicit Draft PR blocker; it is not hidden
by the successful Gorm measurements and has not been fixed in this change.

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

## IR-refined Drop with selectable Strong and ThinLTO backends

One development compiler now provides both refined Drop paths:

| Options | Backend |
|---|---|
| `-deadcodedrop` | Strong entry overrides with native package archives |
| `-deadcodedrop -lto=thin -debug-artifact=none` | Entry overrides plus private bitcode slot rewriting and ThinLTO |

Both modes use optimized package IR snapshots captured before code generation,
the same finite method-name resolver, and the same `AnalyzeProgram` / `LiveSlots`
contract. Only packages with unresolved reflection sites save an analysis-only
`.reflect.bc` cache sidecar. Snapshots are never linked and contain no
program-specific pruning. Missing required snapshots cause cache misses;
whole-program visibility is checked again for each final program. Published
cache sidecars persist, while build-owned temporary snapshots are cleaned.

The current experiment requires a development compiler, native macOS executable
build/test, one initial package, host OS/architecture, no named target and no
`-genll`. Thin additionally requires `-debug-artifact=none`; Strong's refinement
does not depend on that debug setting. Embedded runtime PCLN is retained in the
size configuration. FullLTO plus GoGlobalDCE and the plugin remains a separate
comparison. See [Drop analysis and backend selection](../../dev/deadcode-drop-backends.md)
for the pipeline, conservative fallbacks and current review blocker.

The `m.Func` use proof is removed: reading only a method's name/signature no
longer licenses removal of its callable body. The retained precision improvement
recovers finite names at dynamic `reflect.Value.MethodByName` sites. A reachable
indexed lookup or unresolved generic reflection demand remains conservative.

The executable regression suite runs the same six fixtures in both backend
modes, checking positive name recovery, mutable/escaping names, independent
unknown lookups, cross-package mutation, and a cached dependency containing the
actual reflection loop. Both symbols and runtime output are checked, and the
cache fixture asserts a dependency cache hit on its second build:

```sh
go test -tags dev ./cl -run '^TestBuildAndCheckSymbolsFromTestdropIRReflection$' -count=1
go test -tags dev ./internal/build -run '^(TestReflectIR|TestDeadcodeThin|TestThinLTONamesLateEntryGlobals)' -count=1
```

To compare complete method-slot decisions with one compiler, use a diagnostic
Go overlay without changing the checkout:

```sh
python3 dev/audit_deadcode_live_slots.py \
  --root "$PWD" \
  --gorm-module /path/to/prepared-gorm-module \
  --linker /path/to/ld64.lld \
  --output /tmp/llgo-unified-live-slots --jobs 3
```

Prepare the module with Gorm v1.31.2 and its schema test dependencies. The helper
builds one diagnostic compiler, uses the same source/runtime root for both
selections, saves full type-to-slot maps, and fails on unequal contents. It also
records compiler hashes and per-backend map counts/hashes. For historical
comparisons, replace `--root` with both `--strong-root` and `--thin-root`; those
options cannot be mixed with `--root`.

For a size/build-time comparison against original Drop, build a baseline compiler
at the intended common upstream revision and pass the unified compiler for both
new modes:

```sh
python3 dev/measure_deadcode_strong.py \
  --baseline /path/to/llgo-main \
  --strong /path/to/llgo-unified \
  --thin /path/to/llgo-unified \
  --gorm-module /path/to/prepared-gorm-module \
  --linker /path/to/ld64.lld \
  --output /tmp/llgo-unified-comparison --repeat 3
```

Use uninstrumented compilers for these measurements. All variants use the same
runtime/source root, output path, `-a -Os -p=10`, embedded PCLN, omitted DWARF and
final strip/sign steps. Run without other compiler/test jobs and prepare module
downloads before timing. The runner records build commands, compiler hashes,
section sizes, function symbols and execution logs. Audit those logs for fatal
runtime diagnostics even if exit status is zero: the historical runner's
exit-code checks alone are insufficient for the known humanize failure.

Compare method-slot decisions separately from final machine-code entries.
Equal `LiveSlots` do not imply equal function counts: ThinLTO can still import,
inline, specialize, merge and remove code. The runner counts unique addresses
in `__TEXT,__text` and checks them against Mach-O `LC_FUNCTION_STARTS`; a missing
symbol alone is not proof of source-function DCE. Shared snapshot handling does
not guarantee every native/Thin optimizer input is identical, so compare the
actual maps for the workload being evaluated.

### Historical macOS comparison, 2026-11-07

The following table predates both the Func-use removal and the unified compiler.
All three controls use upstream `0c9212756`; the Thin control reapplies the two
commits from the earlier PR #3008 to that base. These are three-round medians
from a serial matrix using separate compiler controls, not new measurements of
the consolidated PR #3011. Raw sample metrics, compiler identities, analysis-map
hashes and reproduction details are in
[`dev/deadcode_strong_results.json`](../../dev/deadcode_strong_results.json).

| Case | Mode | Bytes | Function entries | Wall seconds |
|---|---|---:|---:|---:|
| reflectcallbacks | main Drop | 1,687,600 | 3,099 | 7.16 |
| reflectcallbacks | IR-refined Drop + strong overrides | 1,027,616 | 1,330 | 7.11 |
| reflectcallbacks | IR-refined Drop + ThinLTO | 862,624 | 814 | 6.62 |
| gormschema | main Drop | 4,585,440 | 7,647 | 11.06 |
| gormschema | IR-refined Drop + strong overrides | 3,512,784 | 4,651 | 10.83 |
| gormschema | IR-refined Drop + ThinLTO | 3,314,240 | 3,632 | 15.11 |

In that experiment, the complete `LiveSlots` maps match: 79 types / 112 slots
for reflectcallbacks, and 285 types / 649 slots for Gorm. Both remove
`UnrelatedJSON`. On Gorm, Strong is 23.39% smaller than original Drop, 5.99%
larger than ThinLTO, and has 28.30% lower median wall time in these samples.
These are scoped historical observations, not current-head speed guarantees.
All 18 artifacts pass their execution checks; all nine Gorm artifacts pass the
62 top-level schema tests. No database/CRUD workload was exercised.

The later frozen ablation and post-removal records are
[`dev/deadcode_mfunc_ablation_results.json`](../../dev/deadcode_mfunc_ablation_results.json)
and [`dev/deadcode_mfunc_removal_validation.json`](../../dev/deadcode_mfunc_removal_validation.json).
GORM and the six name-precision cases retain their size benefit without the
Func-use proof. Workqueue passes after removal in both earlier implementations,
superseding its historical failure as the current blocker. Humanize still
reports a fatal runtime diagnostic in those post-removal checks; it remains an
explicit Draft review blocker. Keep new unified-compiler results separate from
these historical records and never count failing artifacts as size wins.

The unified compiler's focused correctness, size and complete-slot comparisons
are recorded separately in `dev/deadcode_unified_validation.json`. Both modes
pass GORM's 62 schema tests, workqueue and all six forced/cached reflection
fixtures; humanize remains a known failure. These are consolidation checks,
not a new 22-case timing matrix.

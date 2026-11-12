# Drop analysis and backend selection

This implementation consolidates the earlier Strong (#3011) and ThinLTO (#3008)
experiments into [PR #3011](https://github.com/xgo-dev/llgo/pull/3011). The separate
[PR #3008](https://github.com/xgo-dev/llgo/pull/3008) is superseded. One development
compiler now provides both backends through the existing Drop and LTO options.

The SSA `reflect.Method.Func` use proof remains removed. `Type.Method` and
`Type.MethodByName` retain callable demands; constant names still use the
existing named demand. The retired metadata-only wire kind remains readable in
compatible version-3 experimental caches and is treated as conservative
callable reflection. This does not add support for older metadata file versions.
New compiler output does not emit that kind.

## Selecting a backend

| Command options | Analysis | Application |
|---|---|---|
| `-deadcodedrop` | Optimized-IR finite-name refinement and Drop | Strong entry overrides and native linking |
| `-deadcodedrop -lto=thin -debug-artifact=none` | Same refinement and Drop implementation | Entry overrides, private bitcode slot rewriting, then ThinLTO |

The refined paths currently require development-mode Drop, native macOS host
OS/architecture, an executable build or test, one initial package, no named
target and no `-genll`. Strong uses LTO off. Thin additionally requires
`-debug-artifact=none`, because temporary archives cannot provide persistent
DWARF debug-map paths. This flag omits DWARF; embedded runtime PCLN remains.
Outside those gates the new refinement/rewrite is not enabled. FullLTO plus the
existing plugin remains a separate comparison, not a third Drop backend.

For example, with the repository's Go/LLVM development prerequisites configured:

```sh
go build -tags dev -o /tmp/llgo-drop ./cmd/llgo
export LLGO_ROOT="$PWD"
export LDFLAGS="-Wl,-no_exported_symbols --ld-path=/opt/homebrew/opt/lld@23/bin/ld64.lld"
/tmp/llgo-drop build -a -Os -deadcodedrop -debug-artifact=none \
  -o /tmp/drop-strong ./benchmark/binary_size/reflectcallbacks
/tmp/llgo-drop build -a -Os -deadcodedrop -lto=thin -debug-artifact=none \
  -o /tmp/drop-thin ./benchmark/binary_size/reflectcallbacks
/tmp/drop-strong
/tmp/drop-thin
```

Adjust the linker path to the local installation. Both programs should print
`before-create` followed by `after-find`. Repeat without `-a` to exercise cache
reuse. Symbol assertions in the executable regression suite provide stronger
checks than this smoke example alone.

## Shared analysis input and contract

Both backends capture optimized package IR at the same pre-codegen stage in an
analysis-only `.reflect.bc` snapshot. Only packages with relevant reflection
sites need snapshots. A fresh build owns a temporary snapshot; cache publication
copies it to the package's fingerprinted sidecar. A cache hit missing a required
sidecar is rejected and rebuilt. Temporary snapshots are cleaned after use;
published cache sidecars persist. Neither kind of snapshot is a linker input,
and neither contains program-specific pruning.

Both modes read these snapshots through the same name-resolution path. The
snapshot is cached IR, not a cached whole-program proof: source ownership,
visibility, writes/escapes and opaque-input conditions are checked again for the
final program. The common resolver recovers complete finite name sets for
supported dynamic `reflect.Value.MethodByName` sites. Unknown or incomplete
proofs remain generic reflection; an unresolved observed clone invalidates its
site's finite result, and an entirely missing site has no proof.

Package metadata and the `reflectNames` map feed the common `GlobalSummary`,
which translates site demands to existing named or generic demands when queried.
The common `deadcode.AnalyzeProgram` then yields one `deadcode.Analysis`,
including `LiveSlots` and existing type/interface metadata decisions. Roots and
the fixed-point reachability rules are shared. A reachable generic demand, such
as `Method(0)`, still retains exported methods across candidate types and can
negate a finite-name refinement elsewhere.

```text
optimized package IR -> analysis-only snapshot -> finite-name proof
package metadata + reflectNames -> GlobalSummary -> AnalyzeProgram
                                      one Analysis / LiveSlots
                                  /                     \
                         Strong application            Thin application
```

The capture stage, snapshot protocol and analyzer are common; mode-specific LLVM
optimization can still produce different IR. Actual complete `LiveSlots` maps
must therefore be compared, rather than assuming equality for every program
from shared source code alone.

## Backend application

Strong keeps native package archives intact and emits program-specific strong
type-descriptor overrides into the entry module. The native linker discards
unreferenced bodies according to its normal rules.

Thin emits the same kind of entry overrides and additionally rewrites dead IFn
and TFn slots to `runtime.unreachableMethod` in link-private package bitcode
copies. The archive rewrite preserves native members, member order and weak
descriptor linkage. It verifies rewritten IR, names anonymous globals and
regenerates summaries with the ThinLTO bitcode writer. It does not rerun the
whole package optimization pipeline; normal ThinLTO optimization follows.

Shared package caches never receive program-specific slot pruning. Temporary
archives survive an embedded-PCLN relink and are cleaned afterward. Anonymous
globals introduced by late entry generation are named for both initial Thin
emission and PCLN relinking.

Identical `LiveSlots` means identical requested method-slot reachability, not
identical final machine-code function counts. Inlining, specialization, merging,
internalization and subsequent DCE can differ between the backends.

## Comparing the two backends in one compiler

The diagnostic helper builds one overlay-instrumented compiler and invokes it
with both backend selections against the same source/runtime root:

```sh
python3 dev/audit_deadcode_live_slots.py \
  --root "$PWD" \
  --gorm-module /path/to/prepared-gorm-module \
  --linker /path/to/ld64.lld \
  --output /tmp/llgo-unified-live-slots \
  --jobs 3
```

Prepare the module with `gorm.io/gorm@v1.31.2` and its schema test dependencies.
The output directory must be new or empty. `audit.json` records compiler paths
and hashes; `comparison.json` records full-map equality and per-backend type,
slot and hash metrics. The complete maps are saved separately. Any mismatch
fails the command. These diagnostic compilers are not timing/size controls.

For historical comparisons, the helper still accepts the paired
`--strong-root` and `--thin-root` options instead of `--root`. The Strong root
supplies the common program/runtime sources in that mode. Mixing the two forms,
providing only one historical root or using a nonpositive job count is rejected.

## Unified validation

`deadcode_unified_validation.json` records the one-compiler consolidation checks
on November 12. Both backends pass all six finite-name fixtures in forced and
cached builds, including an explicit cache-hit assertion. The snapshot/cache,
Thin archive, anonymous-global, metadata, resolver and graph tests also pass.

Complete one-compiler maps match for `reflectcallbacks` (79 types / 112 slots)
and `gormschema` (285 types / 649 slots), and match the canonical maps already
stored in `deadcode_mfunc_live_slots.json`. GORM and workqueue execute
successfully in both modes; each GORM artifact passes 62 schema tests. The
known humanize fatal remains in both modes. These checks do not establish a
new timing comparison or full database correctness.

## Historical evidence and remaining issue

The earlier measurement JSON files listed below predate this consolidation. They retain
original compiler identities and must not be presented as fresh measurements of
the unified compiler:

- `deadcode_strong_results.json` records the historical three-round comparison
  against original Drop, using separate compilers at a common upstream base.
- `deadcode_mfunc_ablation_results.json` contains the frozen 22-case, five-config
  macOS matrix, exact sizes, machine-code entries, wall/CPU times and corrected
  runtime status. GORM and all six name-precision cases retain the same sizes
  and entry counts with the Func-use proof disabled. The targeted
  `reflectmetadata` fixture loses its independent size benefit.
- `deadcode_mfunc_live_slots.json` records equal complete maps for the two
  post-removal implementations: `reflectcallbacks` has 79 types / 112 slots,
  and `gormschema` has 285 types / 649 slots. These are historical two-compiler
  checks, not universal equivalence or new unified-compiler results.
- `deadcode_mfunc_removal_validation.json` records post-removal local tests and
  smoke checks. GORM and workqueue pass in both implementations; each GORM
  artifact passes 62 schema tests, without a real database workload.

Workqueue's recovery after disabling the proof does not establish its earlier
failure's root cause. The known `dustin_humanize` fatal diagnostic remains a
correctness blocker; an exit code of zero or a PASS line does not override it.
PR #3011 remains Draft while that issue is unresolved. The unified-compiler
checks above are recorded separately from these frozen controls.

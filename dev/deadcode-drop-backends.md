# Drop analysis and backend selection

The SSA reflect.Method.Func-use proof is removed from this implementation.
Type.Method and Type.MethodByName retain callable demands; constant names still
retain only the existing named demand. The retired metadata wire kind remains
readable for old experimental caches and is treated as conservative callable
reflection. The optimized-IR finite method-name analysis remains enabled.

## Proposed combined interface

Keep `-deadcodedrop` as the feature switch. Use the existing LTO selection rather
than adding an independent option that can contradict `-lto`:

- `llgo build -deadcodedrop ...`: native Strong override backend.
- `llgo build -deadcodedrop -lto=thin ...`: Drop plus ThinLTO archive rewriting.
- Without `-deadcodedrop`, the existing non-Drop modes remain unchanged.

This is the proposed organization when combining the two experimental branches.
The current PRs remain separate implementations; these examples do not claim
that either branch already contains both optimized analysis-input paths. Current
native-macOS/development/single-root/debug-artifact restrictions still apply.
FullLTO plus the existing plugin remains a separate comparison, not a new Drop
backend. Unsupported combinations should be diagnosed, not silently change the
requested optimization mode.

## Shared analysis contract

Package metadata + optimized package IR + final-program visibility and roots
produce complete finite method-name sets. The common GlobalSummary feeds
`deadcode.AnalyzeProgram`, yielding one `deadcode.Analysis`, including `LiveSlots`
and the existing type/interface metadata decisions. Backend code must consume
that result rather than independently infer method liveness.

The Strong path currently captures an optimized `.reflect.bc` sidecar before
native code generation. The Thin path reads optimized archive bitcode. Sharing
an analyzer alone does not prove that those inputs are identical: a combined
implementation should share the optimization/name-resolution stage before
backend-specific lowering, and compare canonical complete LiveSlots maps in
regressions. Unknown or incomplete names must remain conservative in both paths.

## Backend application

Strong keeps native package archives intact and emits program-specific strong
type-descriptor overrides into the entry module. The native linker discards
unreferenced bodies according to its normal rules.

Thin still emits the entry overrides, and additionally rewrites dead IFn/TFn
slots in link-private package bitcode copies. It names anonymous globals and
regenerates ThinLTO summaries from the rewritten IR, then runs ThinLTO. Shared
package caches must never receive program-specific pruning. Temporary copies
must survive an embedded-PCLN relink and be cleaned up afterward.

Identical LiveSlots means identical requested method-slot reachability, not
identical final machine-code function counts. Inlining, specialization, merging,
internalization and subsequent DCE can differ between the backends.

## Measured ablation and remaining issue

`deadcode_mfunc_ablation_results.json` contains the frozen 22-case, five-config
macOS matrix, compiler identities, exact sizes, function entries, wall/CPU times
and corrected runtime status. GORM and all six name-precision cases retain the
same sizes and function counts with the Func-use proof disabled. The targeted
reflectmetadata fixture loses its independent size benefit. workqueue passes
with the proof disabled in both backends, but this does not establish the root
cause of its previous failure. dustin_humanize still reports a fatal diagnostic
with either setting; a zero exit code or a PASS line does not override it.
The PRs remain Draft while that correctness issue is unresolved.

After removing the proof, full-map diagnostics still match for reflectcallbacks
(79 types / 112 live slots) and gormschema (285 types / 649 live slots). Both
complete maps and comparison hashes are in `deadcode_mfunc_live_slots.json`.
This checks these fixtures; it is not a claim that all possible native/Thin IR
inputs necessarily produce identical name-resolution results.

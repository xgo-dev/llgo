# Link-phase ftab/findfunctab generation

Status: design + staged plan. Depends on #2012 (runtime funcinfo find index)
and benefits from #2015 (nanosecond monotonic clock, for honest benchmarks).

## Problem

#2012 builds the sorted function-entry table and the Go-style findfunctab at
**first use in the running process**, because LLVM IR generation does not know
final linked text order. This leaves four measured gaps against Go 1.26:

1. `cold.FirstFuncForPC`: 36µs on macOS / 12µs on Linux vs Go's 2.4µs / 375ns.
   The cold fast path (bounded linear scan of raw entry sections, then dladdr)
   is a transitional mechanism; Go needs none of it because the linker ships a
   sorted table.
2. LTO inlining duplicates the body-embedded entry-site inline asm into every
   inline site: `llgo_funcinfo_entry` grew ~4x on the multipkg benchmark and
   host-function PCs get registered under the inlinee's symbol ID. IR-level
   fixes were tried and ruled out (see Facts below); dedup must happen after
   final code generation.
3. The runtime keeps ~300 lines of transitional complexity: cold lookup
   budget, section scans, first-use sort, entry-PC slack matching.
4. pcvalue-style instruction-level line tables (the next alignment step with
   Go) need a per-function table keyed by final text order.

## Approach: post-link table generation

Insert a post-link step into `internal/build` after the final clang/lld link:

```
link -> post-link tool: parse binary -> sort/dedup -> build buckets -> write back
```

A separate linker plugin was considered and rejected: llgo drives stock
clang/lld and a plugin would need to be maintained per linker flavor
(ld64.lld, ld.lld) and per LTO mode. Editing the linked artifact is
linker-agnostic.

### Data flow

1. **Parse** the linked binary's metadata sections (`debug/elf`,
   `debug/macho` from the Go stdlib — the tool runs on the host):
   - `llgo_funcinfo_entry` / `__DATA,__llgo_fie`: `{pc, symbolID}` records.
   - `llgo_funcinfo_stubsite` / `__DATA,__llgo_stub`: same layout.
   - Zero records are skipped, as in the runtime today.
2. **Dedup by symbolID**: LTO inline copies register the same symbolID at
   several PCs. The true entry is the record whose PC lies inside the text
   range of the symbol that owns the symbolID; resolve via the binary's
   symbol table (`.symtab` / `nlist`). Records that fall inside a different
   function's range are inline copies — drop them. This is the fix for gap 2
   that IR-level metadata could not express.
3. **Sort** by PC; append a sentinel entry (end of text) so the runtime can
   use Go's forward-scan lookup shape (`internal/pclntab.LookupFuncIndex`).
4. **Build buckets** with `internal/pclntab.BuildFindFuncBuckets` — the
   faithful port of `cmd/link`'s algorithm that has been sitting unwired
   since #2012. Delta overflow is a hard error here, mirroring Go's linker;
   if it ever fires, fall back to leaving the prebuilt table absent.
5. **Write back** into a reserved section:
   - The main module already emits `__llgo_funcinfo_*` globals; add a
     `__llgo_pclntab_prebuilt` global sized from the collected package data
     (entry-record count is known at main-module emission time; LTO can only
     shrink it after dedup) plus a header {magic, version, count, anchorOff}.
   - The tool rewrites the section contents in place (same size or smaller;
     unused tail is zeroed) and flips the header magic to "valid".

### ASLR

Stored PCs must survive load-time slide. Store **offsets relative to an
anchor symbol** (`__llgo_pclntab_anchor`, placed in the same section). At
startup the runtime computes `slide = &anchor_runtime - anchorOff_stored`
and adds it during lookup (one add on the hot path, same as Go's
`datap.text` bias). Note the entry-site records themselves are already
rebased by the loader (they hold absolute pointers with relocations); the
prebuilt table deliberately holds offsets so the tool does not need to
emit relocations.

### Runtime integration

`initRuntimeFuncPCFramesOnce` gains a fast path: if the prebuilt header is
valid, adopt the table directly (no section scan, no sort, no bucket build)
— `FirstFuncForPC` becomes bucket-lookup cost, matching Go's shape. The
existing first-use construction remains as the fallback whenever the header
is invalid (older compilers, exotic formats, overflow bail-out), so the
change is strictly additive and safe to land incrementally.

## Staging

- **P1** `chore/pclnpost`: standalone tool, parse + dedup + sort + bucket
  build + stats printing; golden tests against binaries produced by the
  existing test programs. No behavior change.
- **P2** Reserve the section in `internal/build`, run the tool as a post-link
  step, wire the runtime fast path. Benchmarks: cold.FirstFuncForPC on both
  platforms; assert `llgo funcinfo: ... entries= prebuilt` via
  LLGO_FUNCINFO_DEBUG.
- **P3** Remove transitional runtime code (cold budget/scan, first-use sort
  path stays as fallback but slack matching can go once anchors are exact
  entries from the symbol table).
- **P4** pcvalue-style line tables keyed by the prebuilt function order
  (replaces the call-site pcline records; gives instruction-level FileLine).

## Established facts (verified in #2012 work; do not re-derive)

- Mach-O metadata sections need `live_support` + one lowercase-`l`
  linker-private symbol per record; ld64/lld `-dead_strip` then drops records
  exactly with their function. Verified with lld 19.1.7, including LTO.
- Boundary symbols: ELF `__start_/__stop_`; Mach-O `section$start$SEG$SECT`
  referenced from IR needs the `\x01` verbatim-name prefix or LLVM prepends
  an underscore and the linker stops recognizing it.
- Visible (non-`L`) labels inside Mach-O function bodies split the function
  into atoms that the linker may reorder — assembler-local labels only.
- `!associated` affects only linker GC; IR-level GlobalDCE deletes such
  globals regardless, and `llvm.compiler.used` pins dead functions through
  the records' initializers. This is why records stay body-embedded inline
  asm and dedup happens post-link.
- `internal/pclntab` is a faithful port of Go 1.26's findfunctab generation
  and lookup (uint8 deltas, overflow error, forward scan, sentinel); the
  runtime's in-process variant deliberately uses uint16 deltas because LLGo
  lacks Go's MINFUNC guarantee. The post-link table can use the faithful
  uint8 layout since dedup restores the one-record-per-function invariant.

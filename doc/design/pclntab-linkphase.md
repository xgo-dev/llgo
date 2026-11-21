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

### Function identity contract

PCLN is indexed by physical text functions, following Go's linker model.
Source functions and compiler-generated wrappers or adapters all contribute
ordinary function records and entry sites. The symbol names the physical
function; its display name may describe the corresponding Go operation.
Calling conventions, closure environments, and their transport mechanism are
not PCLN properties and must not introduce function-class-specific sections.

### Data flow

1. **Parse** the linked binary's metadata sections (`debug/elf`,
   `debug/macho` from the Go stdlib — the tool runs on the host):
   - `llgo_funcinfo_entry` / `__DATA,__llgo_fie` (Mach-O without physical
     compaction) / `__LLGO,__llgo_fie` (embedded executable LTO Mach-O):
     `{pc, symbolID}` records.
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
5. **Write back and compact** the entry carrier:
   - The compact table replaces the prefix of `llgo_funcinfo_entry`; if it
     does not fit, the binary is left unchanged and the runtime uses its
     first-use construction fallback.
   - Embedded executable LTO Mach-O normally puts the entry carrier in a
     dedicated `__LLGO` segment. Full LTO size builds collect entry addresses
     from the linked symbol catalog and size the reservation from live
     entries during the metadata relink. Their carrier shares `__DATA` to
     use space in the existing segment. Non-LTO, external-PCLN, c-shared, and
     c-archive Mach-O also use `__DATA`. ELF links the carrier immediately
     before `.bss`, at the file-backed tail of the final writable `PT_LOAD`.
     PC-line sites remain outside the disposable range.
   - ELF compaction rejects an image with a program segment after the carrier;
     only non-loaded sections and the section-header table may be shifted.
   - After fixing Mach-O chained pointers, section sizes, load commands,
     segment offsets, ELF program/section headers, and link-edit offsets, the
     tool removes the unused carrier suffix from the physical file. Virtual
     addresses of program text/data do not move; the omitted tail becomes
     zero-fill memory.
   - Rewriting is transactional: construct and reopen the complete staged
     image, re-sign an originally signed Mach-O, then atomically rename it.
     An unfamiliar segment shape, overlapping relocation/fixup range, signing
     failure, or verification failure leaves the original executable intact.

### Parallel executable publication on Linux

The test DAG waits for linking and postprocessing before executing each image,
but other roots may start subprocesses while that image is being staged.
`O_CLOEXEC` does not prevent a forked child from temporarily inheriting its
writable descriptor. Closing the parent's descriptor and atomically renaming
the file do not release the child's reference, so execution can fail with
`ETXTBSY` (see [Go issue 22315](https://go.dev/issue/22315)).

`pclnpost.stageBinary` holds `syscall.ForkLock` for reading from before opening
the staged image through closing its writable descriptor on Linux. Go takes
the write side when forking, so no child can inherit that descriptor. Multiple
image writers can still overlap, and already-running subprocesses are unaffected.
After closing the writer, Linux releases the guard and uses a read-only handle
for `fsync`, so slow storage cannot hold up process creation during the flush.
That handle is opened before applying the output mode, which may be execute-only.
The file is still synced before publication; directory `fsync` alone does
not ensure file-data durability. The write itself must stay guarded while its
writable descriptor exists. Other hosts retain sync-before-close behavior.
The guard also ends before verification or signing, which may start subprocesses
themselves. Parsing, table generation, linking, and test execution remain outside
the guard; no execution retry is needed for this publication race.

### ASLR

Stored table entries are offsets from the first function PC. The header keeps
that base as a runtime address: Mach-O rewrites its slot into the dyld chained
fixup chain, while supported non-PIE ELF outputs already use their runtime
address. The lookup hot path therefore only adds the stored entry offset.

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
- **P3** (done) Mach-O bind-record resolution: pointer slots naming exported
  Go functions are chained-fixup BIND nodes, not rebases; without decoding
  them through the imports table, exported records miss the prebuilt ftab and
  `FuncForPC` silently pays a dladdr per fresh pc (~6µs). Also: the prebuilt
  header's base slot is spliced back into the fixup chain as a live rebase
  node, so the runtime reads a dyld-slid runtime PC directly (no slide
  arithmetic). Transitional cold budget/scan stays as the fallback for
  non-rewritten binaries.
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
- Mach-O chained fixups encode anchors to exported symbols as BIND nodes
  (import ordinal + addend), even when the target is defined in the same
  image; only local-symbol anchors are rebases. Decode both.
- Adding fixup nodes does not pre-touch pages at load on modern macOS:
  dyld uses page-in linking (the kernel applies fixups lazily at first
  touch), so "sacrificial fixups to warm the table's pages" is not a
  viable optimization — measured no effect on first-lookup latency.
- `internal/pclntab` is a faithful port of Go 1.26's findfunctab generation
  and lookup (uint8 deltas, overflow error, forward scan, sentinel); the
  runtime's in-process variant deliberately uses uint16 deltas because LLGo
  lacks Go's MINFUNC guarantee. The post-link table can use the faithful
  uint8 layout since dedup restores the one-record-per-function invariant.

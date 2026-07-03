# Stage 5 notes: the FP unwinder's non-obvious invariants

Durable conclusions from bringing up the frame-pointer unwinder (PR #2019).
Read together with #2019's PR body and `pclntab-linkphase.md`. The full
session-by-session investigation log lives in the PR discussion.

## Invariants that must not silently degrade

- **pc-1 attribution and the walk bound must survive without the prebuilt
  table.** When a layout overflows the entry section, the link-phase rewrite
  backs off and `runtimePrebuiltFtab` stays empty. If `prebuiltTextContains`
  then answers false, two things turn off silently: the pc-1 return-address
  convention (frames get attributed to the *next* statement — a return
  address equals the following pcline anchor exactly) and the FP walk's
  text bound (libc tail frames decode as wild pcs). The fix is the
  first-use frame-table fallback bounds in `prebuiltTextContains`, and
  `fpCallers` building that table up front. Any future change to the table
  adoption path must keep this fallback alive.

- **Mid-function aligned pcs need pcline merging on amd64.** arm64
  instructions are 4-byte aligned, so a `ret-1` query is always unaligned
  and can't collide with a function entry or an aligned-branch lookup.
  amd64 entries and return addresses are byte-dense: `ret-1` can be
  4-aligned and can even equal another symbol's entry. Every path that
  resolves a function record must therefore also consult same-function
  pcline statement records — this is centralized in `refinePCSymbolLine`
  (symtab.go); do not add a new resolution path that bypasses it.

- **The two FP walkers must stay in sync.** `fpCallers`
  (runtime/internal/lib/runtime/unwind_llgo.go) and `llgo_stacktrace`
  (runtime/internal/clite/debug/_wrap/debug.c) implement the same chain
  discipline ([fp]/[fp+wordsize], strictly increasing, bounded stride,
  word alignment). The C walker serves pre-table paths (unrecovered-panic
  dump, last-resort fallback).

- **The FP attribute is target-gated.** `ssa.Program.NeedsFramePointer()`
  says where the chain is emitted (linux/darwin, non-embedded, non-wasm);
  the compiler records the decision in the per-binary `__llgo_fp_chain`
  byte, and the runtime's `fpUnwindAvailable` trusts that flag plus table
  presence. On ESP32-C3 the attribute interacted badly with the
  conservative GC ("clearing nested struct did not free all objects"), so
  keep embedded targets off unless that is understood.

## Diagnostic traps

- `nm` on `libexport.a` is useless: the archive nests `pkg-*.a` members.
  Inspect the linked dylib/executable instead.
- Statement labels can land exactly on return addresses; a "wrong line"
  report one statement late is the signature of a raw-pc (not pc-1) lookup.
- An amd64-only symbolization bug with green arm64 almost always traces to
  one of: byte-dense entries (collisions), 4-aligned ret-1 (aligned-branch
  path), or the empty-prebuilt fallback above.

## Local repro environments

- colima-qemu / container `llgo-amd64`: amd64 toolchain, stage5 clone at
  `/root/s5` (rebuild: fetch + reset + `go build -o /usr/local/bin/llgo
  ./cmd/llgo`).
- colima-llgo-perf / container `llgo-linux-final`: linux-arm64 bench matrix
  (mounts /work-s5, /work-2016, /work-2012).

## Merge queue

1. Merge order #2012 -> #2016 -> #2019, rebasing between each.
2. Semantics PRs 1918/1882/1892/1906 rebase after the merges (1918/1882
   overlap needs an ordering decision), then reimplement 1925/1903/
   1924-residual/1905, then P4 (zero-copy names, prebuilt pcline,
   !pcsections, section shrink) per #2004.

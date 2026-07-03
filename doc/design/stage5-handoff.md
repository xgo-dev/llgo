# Stage 5 session handoff (2026-07-03)

Working state for PR #2019 (`codex/stage5-unwinder`, head 7d41580bb at time of
writing). Read together with #2019's PR body and `pclntab-linkphase.md`.

## PR states

- #2012 `codex/runtime-pclntab-llvm` @83edf8823 — ALL GREEN (48 checks,
  codecov 91.33%). Merge-ready.
- #2016 `codex/pclntab-linkphase-p1` @fc3a21b05 — ALL GREEN incl. codecov
  88.86% (target 88.68%). Merge after #2012, then rebase #2019.
- #2019 — arm64 fully green locally (cl/test-go/internal-build/ssa/LLDB, and
  linux-arm64 test/go). TWO amd64-only failures remain, plus codecov pending
  a real run (its diff is mostly runtime-lib files codecov does not count;
  countable cl/ssa changes are covered).

## RESOLVED: the amd64 bug (5b96e84cc)

Final root cause: those layouts overflow the entry section, the link-phase
rewrite backs off, and with runtimePrebuiltFtab empty prebuiltTextContains()
answered false — silently disabling the pc-1 return-address convention and
the FP-walk text bound (a return address equals the next statement's anchor
exactly). Fixed by falling back to first-use frame-table bounds; fpCallers
ensures the table is built. amd64 test/go: ok (1201s). The earlier rounds'
fixes (aligned-branch pcline merge, walk bound, rounding removal) all
remain load-bearing. History below kept for archaeology.

## The historical amd64 investigation (resolved)

`TestRuntimeLineInfoAndStack` ("bad main frame: main.go:16") and
`TestRuntimeStatementLineInfo` ("bad Func.FileLine(pc-1): main.go:37") fail
on linux/amd64 only. Repro container: colima-qemu / container `llgo-amd64`
(has clang-19+lld+go+libunwind-19-dev; stage5 clone at /root/s5; rebuild:
`cd /root/s5 && git fetch origin codex/stage5-unwinder && git reset --hard
origin/codex/stage5-unwinder && go build -o /usr/local/bin/llgo ./cmd/llgo`).

Minimal probe /tmp/stline (source: scratchpad stline/main.go — check() does
runtime.Callers, walks CallersFrames to find its own frame, then
FuncForPC(frame.PC-1).FileLine(frame.PC-1)). Latest amd64 result:

    frameLine=11 (correct)  fileLinePCm1=14 (wrong, line of a later stmt)
    frame.PC=0x68c74b (odd return address — legal on amd64)

Paradox to resolve: Frames.Next and the probe both should funnel through
frameSymbol(0x68c74a) and its result cache, yet disagree. Prime suspects,
in order:
1. funcForPCSlow's hoisted prebuilt exact-entry lookup (pprof_runtime_stub
   _llgo.go): for pc-1 it can EXACT-MATCH a *different row's entry* on amd64
   (entries are byte-dense there; ret-1 can equal a stub/function entry).
   If prebuiltFrameIndexForEntry(pc-1) hits, FuncForPC returns that row's
   record line — plausibly line 14's record. arm64 can't hit this (4-byte
   alignment). Fix sketch: exact-entry fast path should not fire for pcs
   that are `ret-1` style queries… simplest correct rule: it's fine for the
   FUNC but FileLine must still use pcline at the query pc; or verify by
   printing which path returned in the container.
2. frameSymbolResultCache slot sharing pc vs pc-1 (exact pc compare should
   prevent it; verify store/load ordering anyway).

UPDATE (round 5, ff572a15b): walk now bounded to program text (kills the
wild libc-tail frames the diagnostic exposed) — still "bad main frame:
main.go:16" (expects FRAMES_MAIN_LINE=13, the checkFrames() call in
main; 16 = three statements later). A simple equivalent probe
(/tmp/mainframe on the container) reports main's frame line CORRECTLY, so
the failure needs the real probe's shape. Parked diagnostic: container
llgo-amd64, /tmp/probe_keep.log + /root/probe_dir — the actual generated
probe rebuilt with anchors-in-main dump (ANCHOR lines) and main's call
instructions. Compare: if labels for statements 14-16 precede the
checkFrames call's return address, amd64's scheduler is sinking label asm
relative to calls, and the durable fix is compiler-side anchoring (tie the
label to the call, e.g. emit label+call as one asm bundle or switch the
convention to label-at-ret with raw-ret lookups everywhere).

UPDATE (round 4, b6bc4ed27): ROOT CAUSE of the FileLine class found and
fixed — amd64 ret-1 can be 4-aligned (ret=0x293731 → query 0x293730), taking
FuncForPC's aligned funcPCFrameForPC branch which returned declaration lines
and never consulted pcline; now merges same-function statement records.
TestRuntimeStatementLineInfo passes on amd64. ONE check remains red:
TestRuntimeLineInfoAndStack "bad main frame: main.go:16" — checkFrames walks
CallersFrames, finds the main.main frame, expects FRAMES_MAIN_LINE (the
checkFrames() call, probe line ~13); gets 16 (≈3 statements later,
deterministic). Probe source = runtimeLineInfoProbe in
test/go/runtime_lineinfo_stack_test.go. Suspects: (a) which frame the walk
labels "main.main" (wrapper vs real main), (b) missing labels for
statements 14-15 in main making nearest-below skip forward — dump main's
pcline anchors vs its call sites in the container (same d.log recipe),
(c) frameSymbolResultCache interplay. Everything else on amd64 is green.

UPDATE (round 3): the exact-entry pcline refinement (15c1f3b77) did NOT fix
amd64 — identical failures at main.go:16 / main.go:37, deterministic across
builds. Hypothesis 1 (entry collision) is therefore wrong or incomplete.
A diagnostic run is parked in container llgo-amd64 at /tmp/stline/d.log
(probe output + `call runtime.Callers` disasm context + llgo_pcline hexdump
+ symbol addresses). Next session: read d.log, compute which anchor
nearest-below(ret-1) actually selects, and compare with the arm64 layout;
also check whether the probe's expected lines (16/37) correspond to the
STATEMENT AFTER the mark, i.e. whether amd64 pcline labels land after the
call while arm64's land before (LLVM x86 scheduler difference). If so, the
fix is to anchor labels to the call instruction explicitly (e.g. emit the
label asm and the call in one bundle, or record label-at-ret and make both
lookup paths use raw ret on amd64).

`bad main frame :16` in the full probe is likely the same mechanism through
checkFrames (expected FRAMES_MAIN_LINE, got function-decl-adjacent line).

Fixed already on this path (do not regress): fixed-width `&^3` rounding
removed (7d41580bb) — arm64 green after; text-range-only discrimination for
synthetic pcs.

## OPEN: llgo-workflow demo failure (c-shared exports, CI-only so far)

The "llgo (…)" CI jobs fail on #2019 (both ubuntu and macos-arm64; 2016's
same jobs are green). Decoded from the macos job log: in
_demo/go/export/test.sh, the c-SHARED step builds libexport.dylib
("Build succeeded", header matches), but linking use/main.c with
`-lexport` then fails with Undefined _ProcessXType/_CreateComplexData/…
— the dylib is missing the //export symbols under stage5. Local repro was
inconclusive (both 2016 and s5 show one tolerated Undefined block locally;
local env lacks the CI setup). NOTE: nm on libexport.a is useless — the
archive nests pkg-*.a members.

Next session: build c-shared with both compilers in a CI-like env and diff
`nm -gU libexport.dylib`; suspects: export-symbol retention vs -dead_strip
interacting with a stage5 emission change (framePointerAttr added in ssa
NewFuncEx InGo branch; collector trackability for methods/anons; clite
debug rewrite). Bisect the 11 stage5 commits with a dylib-export assertion
if the diff confirms. Also: use/Makefile hardcodes -lunwind (works while
system libunwind exists; unrelated to the failure but worth cleaning).

## Benchmarks in flight

Final #2019 matrices launched on BOTH platforms (mac: scratchpad/
bench-mac-final + bench-mac-finalscale; linux container llgo-linux-final:
/tmp/bench-lin-final + /tmp/bench-lin-finalscale). Variants go/2016/s5
±LTO; core = hot,deep,multipkg,cold,stdlib,plain 24x24 runs=5; scale =
deep 128/512 + bigfunc 32x200/16x2000 runs=3. When done, refresh #2019 PR
body tables (current tables are from the pre-fix head; deltas expected to
be small). Previous-head reference numbers live in the PR body.

## Container/bench inventory

- colima-llgo-perf: llgo-linux-final (mounts /work-s5=stage5,
  /work-2016=pclnpost, /work-2012=pclntab-llvm; compilers at /tmp/llgo-*;
  go126 wrapper at /usr/local/bin/go126). llgo-linux-matrix (older, 2012/
  2016/main mounts) still exists.
- colima-qemu: llgo-amd64 (amd64 toolchain, stage5 clone /root/s5,
  2016 mount /work-2016).
- Mac compilers: /tmp/llgo-2012, /tmp/llgo-2016, /tmp/llgo-s5, /tmp/llgo-main;
  PATH shims in scratchpad/bin (llgo->llgo-2016) and bin-s5 (llgo->llgo-s5).

## After amd64 green

1. Push, wait #2019 CI; codecov may need the same treatment as 2016 if any
   countable file is short (runtime-lib files are not counted).
2. Refresh #2019 body tables from bench-*-final outputs.
3. Merge order #2012 -> #2016 -> #2019 with a rebase between each.
4. Queue behind: semantics PRs 1918/1882/1892/1906 (already rebased onto
   old 2016 — need one more rebase after merges; 1918/1882 overlap needs an
   ordering decision), then reimplement 1925/1903/1924-residual/1905 on the
   new base, then P4 (zero-copy names, prebuilt pcline, !pcsections,
   section shrink) per #2004.

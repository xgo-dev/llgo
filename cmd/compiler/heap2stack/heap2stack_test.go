package heap2stack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNilModule(t *testing.T) {
	if err := Run(llvm.Module{}, llvm.TargetMachine{}); err == nil || !strings.Contains(err.Error(), "requires a module") {
		t.Fatalf("nil module must return a module error, got %v", err)
	}
}

func TestLoopAllocationObservablePointers(t *testing.T) {
	mod := parseModule(t, `
declare nonnull ptr @alloc(i64) allocsize(0)

; Publishing a pointer can keep it alive after the allocation block finishes.
define void @published(i1 %again, ptr %out) {
entry:
  br label %body
body:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store ptr %p, ptr %out
  br i1 %again, label %body, label %exit
exit:
  ret void
}

; An observed address is outside the proof for reusing a single entry slot.
define i1 @compared(i1 %again, ptr %other) {
entry:
  br label %body
body:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  %equal = icmp eq ptr %p, %other
  br i1 %again, label %body, label %exit
exit:
  ret i1 %equal
}
`)
	tm := tailTarget(t, mod)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	if err := Run(mod, tm); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"published", "compared"} {
		body := mod.NamedFunction(name).String()
		if !strings.Contains(body, "call ptr @alloc(") || strings.Contains(body, "alloca ") {
			t.Fatalf("%s must keep observable storage on the heap:\n%s", name, body)
		}
	}
}

func TestAllocationPlacement(t *testing.T) {
	llvm.InitializeNativeTarget()
	target, err := llvm.GetTargetFromTriple(llvm.DefaultTargetTriple())
	if err != nil {
		t.Fatal(err)
	}
	tm := target.CreateTargetMachine(llvm.DefaultTargetTriple(), "", "", llvm.CodeGenLevelDefault, llvm.RelocDefault, llvm.CodeModelDefault)
	defer tm.Dispose()
	mod := parseModule(t, `
declare nonnull ptr @alloc(i64) allocsize(0)
declare nonnull ptr @zero(i64) allocsize(0)
declare void @capture(ptr)

define i64 @local(i64 %x) {
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store i64 %x, ptr %p, align 8
  %v = load i64, ptr %p, align 8
  ret i64 %v
}
define i64 @zeroed() {
  %p = call ptr @zero(i64 16) allockind("alloc,zeroed")
  %v = load i64, ptr %p, align 8
  ret i64 %v
}
define ptr @escape() {
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  ret ptr %p
}
define void @captured() {
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  call void @capture(ptr %p)
  ret void
}
define ptr @dynamic(i64 %size) {
  %p = call ptr @alloc(i64 %size)
  ret ptr %p
}
define i64 @unoptimized(i64 %x) noinline optnone {
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store i64 %x, ptr %p, align 8
  %v = load i64, ptr %p, align 8
  ret i64 %v
}
; Atomic accesses keep a loop-local alloca alive through O2, consuming more
; stack on every iteration. Entry slots can instead be scalarized.
declare ptr @malloc(i64) allocsize(0) allockind("alloc,uninitialized") "alloc-family"="malloc"
define i64 @loop(i64 %n) {
entry:
  br label %body
body:
  %i = phi i64 [0, %entry], [%next, %body]
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store atomic i64 %i, ptr %p seq_cst, align 8
  %v = load atomic i64, ptr %p seq_cst, align 8
  %next = add i64 %i, 1
  %more = icmp ult i64 %next, %n
  br i1 %more, label %body, label %exit
exit:
  ret i64 %v
}
; Multiple entries make this cycle irreducible, so LoopInfo alone is
; insufficient. Exercise allocation attributes on the declaration as well.
define i64 @irreducible(i1 %first, i1 %again) {
entry:
  br i1 %first, label %left, label %right
left:
  %p = call ptr @malloc(i64 16)
  store atomic i64 42, ptr %p seq_cst, align 8
  %v = load atomic i64, ptr %p seq_cst, align 8
  br i1 %again, label %right, label %exit
right:
  br label %left
exit:
  ret i64 %v
}
; The old object and the new object coexist in the same iteration. Only
; the independent q allocation may reuse an entry slot.
define i64 @mixed_carried(i64 %n) {
entry:
  %initial = call ptr @zero(i64 16) allockind("alloc,zeroed")
  br label %body
body:
  %previous = phi ptr [%initial, %entry], [%p, %body]
  %i = phi i64 [0, %entry], [%next, %body]
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store i64 %i, ptr %p, align 8
  %old = load i64, ptr %previous, align 8
  %q = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store atomic i64 %old, ptr %q seq_cst, align 8
  %v = load atomic i64, ptr %q seq_cst, align 8
  %next = add i64 %i, 1
  %more = icmp ult i64 %next, %n
  br i1 %more, label %body, label %exit
exit:
  ret i64 %v
}
define i64 @zero_loop(i64 %n) {
entry:
  br label %body
body:
  %i = phi i64 [0, %entry], [%next, %body]
  %p = call ptr @zero(i64 16) allockind("alloc,zeroed")
  %v = load atomic i64, ptr %p seq_cst, align 8
  store atomic i64 99, ptr %p seq_cst, align 8
  %next = add i64 %i, 1
  %more = icmp ult i64 %next, %n
  br i1 %more, label %body, label %exit
exit:
  ret i64 %v
}
; HeapToStack still decides whether calls capture the pointer. The placement
; check must neither reject all calls nor treat passing an argument as safe
; without HeapToStack's own checks.
declare void @consume(ptr captures(none)) nofree
define void @call_loop(i1 %again) {
entry:
  br label %body
body:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  call void @consume(ptr %p)
  br i1 %again, label %body, label %exit
exit:
  ret void
}
define void @captured_loop(i1 %again) {
entry:
  br label %body
body:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  call void @capture(ptr %p)
  br i1 %again, label %body, label %exit
exit:
  ret void
}
; Reject the allocation before its uses are visited, including free calls.
; Otherwise the must-free fallback could bypass the placement restriction.
declare void @free(ptr allocptr) allockind("free") "alloc-family"="malloc"
define void @cross_block_free(i1 %again) {
entry:
  br label %body
body:
  %p = call ptr @malloc(i64 16)
  br label %use
use:
  call void @consume(ptr %p)
  call void @free(ptr %p)
  br i1 %again, label %body, label %exit
exit:
  ret void
}
; This helper must not move pre-existing allocas owned by other transforms.
define i64 @existing_alloca(i1 %again) {
entry:
  br label %body
body:
  %p = alloca i64
  store atomic i64 42, ptr %p seq_cst, align 8
  %v = load atomic i64, ptr %p seq_cst, align 8
  br i1 %again, label %body, label %exit
exit:
  ret i64 %v
}
; Derived pointers in the same block can reuse the allocation too.
define i64 @gep_loop(i1 %again) {
entry:
  br label %body
body:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  %q = getelementptr i8, ptr %p, i64 8
  store atomic i64 42, ptr %q seq_cst, align 8
  %v = load atomic i64, ptr %q seq_cst, align 8
  br i1 %again, label %body, label %exit
exit:
  ret i64 %v
}
; The same pointer can appear in both an argument and an operand bundle.
; Checking values instead of operand uses would misclassify the latter.
define void @bundle_loop(i1 %again) {
entry:
  br label %body
body:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  call void @consume(ptr %p) ["test"(ptr %p)]
  br i1 %again, label %body, label %exit
exit:
  ret void
}
define i64 @loop_caller(i64 %n) {
  %v = call i64 @loop(i64 %n)
  ret i64 %v
}
; Full Attributor in LLVM 22 crashes in invariant-load inference on this
; indirect-call result. Heap-to-stack does not require that analysis.
define i64 @indirect(ptr %fn) {
  %p = call ptr %fn()
  %v = load i64, ptr %p, align 8
  ret i64 %v
}
`)
	mod.SetTarget(llvm.DefaultTargetTriple())
	td := tm.CreateTargetData()
	defer td.Dispose()
	mod.SetDataLayout(td.String())
	if err := Run(mod, tm); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"local", "zeroed"} {
		body := mod.NamedFunction(name).String()
		if !strings.Contains(body, "alloca ") || strings.Contains(body, "call ptr @") {
			t.Fatalf("%s was not moved to stack:\n%s", name, body)
		}
	}
	if body := mod.NamedFunction("zeroed").String(); !strings.Contains(body, "@llvm.memset") {
		t.Fatalf("zero initialization lost:\n%s", body)
	}
	for _, name := range []string{"escape", "captured", "dynamic", "unoptimized"} {
		if body := mod.NamedFunction(name).String(); !strings.Contains(body, "@alloc(") {
			t.Fatalf("%s lost a required allocation:\n%s", name, body)
		}
	}
	blockText := func(bb llvm.BasicBlock) string {
		var text strings.Builder
		for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			text.WriteString(i.String())
		}
		return text.String()
	}
	checkCycles := func(expectSlots bool) {
		for _, name := range []string{"loop", "irreducible", "call_loop", "gep_loop"} {
			fn := mod.NamedFunction(name)
			if (expectSlots && !strings.Contains(blockText(fn.EntryBasicBlock()), "alloca ")) || strings.Contains(fn.String(), "@alloc(") || strings.Contains(fn.String(), "@malloc(") {
				t.Fatalf("%s must reuse entry storage:\n%s", name, fn.String())
			}
		}
		fn := mod.NamedFunction("mixed_carried")
		if (expectSlots && !strings.Contains(blockText(fn.EntryBasicBlock()), "alloca ")) || strings.Count(fn.String(), "@alloc(") != 1 {
			t.Fatalf("mixed loop must promote local storage and preserve carried storage:\n%s", fn.String())
		}
		for _, name := range []string{"loop", "irreducible", "mixed_carried", "zero_loop", "call_loop", "captured_loop", "cross_block_free", "gep_loop", "bundle_loop"} {
			fn := mod.NamedFunction(name)
			for bb := llvm.NextBasicBlock(fn.EntryBasicBlock()); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				if strings.Contains(blockText(bb), "alloca ") {
					t.Fatalf("%s retains non-entry stack allocation:\n%s", name, fn.String())
				}
			}
		}
		for _, name := range []string{"captured_loop", "cross_block_free", "bundle_loop"} {
			body := mod.NamedFunction(name).String()
			if !strings.Contains(body, "@alloc(") && !strings.Contains(body, "@malloc(") {
				t.Fatalf("%s lost its required heap allocation:\n%s", name, body)
			}
		}
	}
	checkCycles(true)
	zeroLoop := mod.NamedFunction("zero_loop")
	if strings.Contains(blockText(zeroLoop.EntryBasicBlock()), "@llvm.memset") || !strings.Contains(blockText(llvm.NextBasicBlock(zeroLoop.EntryBasicBlock())), "@llvm.memset") {
		t.Fatalf("zero initialization must stay inside the loop:\n%s", zeroLoop.String())
	}
	existing := mod.NamedFunction("existing_alloca")
	if strings.Contains(blockText(existing.EntryBasicBlock()), "alloca ") || !strings.Contains(blockText(llvm.NextBasicBlock(existing.EntryBasicBlock())), "alloca ") {
		t.Fatalf("moved an unrelated alloca:\n%s", existing.String())
	}
	opts := llvm.NewPassBuilderOptions()
	defer opts.Dispose()
	if err := mod.RunPasses("default<O2>", tm, opts); err != nil {
		t.Fatal(err)
	}
	checkCycles(false)
	for _, name := range []string{"local", "zeroed"} {
		body := mod.NamedFunction(name).String()
		if strings.Contains(body, "alloca ") || strings.Contains(body, "call ") {
			t.Fatalf("O2 retained local storage:\n%s", body)
		}
	}
	if body := mod.NamedFunction("zeroed").String(); !strings.Contains(body, "ret i64 0") {
		t.Fatalf("O2 changed the zero value:\n%s", body)
	}
}

func TestCrossBlockLoopReuse(t *testing.T) {
	mod := parseModule(t, `
declare nonnull ptr @alloc(i64) allocsize(0)
declare void @llvm.memset.p0.i64(ptr nocapture writeonly, i8, i64, i1 immarg)
declare void @observe(ptr)

; A loop-local allocation merged through a cross-block PHI whose loaded
; element (not the buffer address itself) flows to a tail call can reuse one
; entry stack slot. The non-tail memset must also lose its "tail" marker.
define ptr @cross_block_phi_reuse(i1 %opt, ptr %fallback, ptr %elem, i1 %again) {
entry:
  br label %loop
loop:
  br i1 %opt, label %allocblk, label %altblk
allocblk:
  %p = tail call ptr @alloc(i64 48) allockind("alloc,zeroed")
  tail call void @llvm.memset.p0.i64(ptr align 8 %p, i8 0, i64 24, i1 false)
  %slot = getelementptr inbounds i8, ptr %p, i64 40
  store ptr %elem, ptr %slot, align 8
  br label %merge
altblk:
  br label %merge
merge:
  %cur = phi ptr [ %p, %allocblk ], [ %fallback, %altblk ]
  %p2 = getelementptr inbounds i8, ptr %cur, i64 40
  %loaded = load ptr, ptr %p2, align 8
  tail call void @observe(ptr %loaded)
  br i1 %again, label %loop, label %exit
exit:
  ret ptr %loaded
}

; When a derived PHI from an earlier iteration remains live across the next
; allocation in a separate block, storage must stay on the heap.
define i64 @cross_block_carried(i1 %again, ptr %init) {
entry:
  br label %header
header:
  %prev = phi ptr [ %init, %entry ], [ %p, %latch ]
  br label %allocblk
allocblk:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store i64 7, ptr %p, align 8
  %old = load i64, ptr %prev, align 8
  br label %latch
latch:
  br i1 %again, label %header, label %exit
exit:
  ret i64 %old
}
`)
	tm := tailTarget(t, mod)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	if err := Run(mod, tm); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	reuse := mod.NamedFunction("cross_block_phi_reuse").String()
	if strings.Contains(reuse, "@alloc(") || !strings.Contains(reuse, "alloca ") {
		t.Fatalf("cross_block_phi_reuse was not promoted to stack:\n%s", reuse)
	}
	if strings.Contains(reuse, "tail call void @llvm.memset") {
		t.Fatalf("cross_block_phi_reuse retained invalid tail marker on promoted alloca memset:\n%s", reuse)
	}
	carried := mod.NamedFunction("cross_block_carried").String()
	if !strings.Contains(carried, "@alloc(") || strings.Contains(carried, "alloca ") {
		t.Fatalf("cross_block_carried lost required heap lifetime:\n%s", carried)
	}
}

func TestRunPostInlineAndCleanupPromoted(t *testing.T) {
	noAllocMod := parseModule(t, `
declare ptr @other()

define ptr @just_call() {
entry:
  %p = call ptr @other()
  ret ptr %p
}
`)
	noAllocTM := tailTarget(t, noAllocMod)
	if promoted, err := RunPostInline(noAllocMod, noAllocTM); err != nil || len(promoted) != 0 {
		t.Fatalf("RunPostInline(noAllocMod) = (%v, %v), want (nil, nil)", promoted, err)
	}

	mod := parseModule(t, `
declare nonnull ptr @alloc(i64) allocsize(0)
declare ptr @other()
declare void @use(ptr nocapture)

define ptr @no_alloc(i64 %x) {
entry:
  %p = call ptr @other()
  ret ptr %p
}

define i64 @already_optnone(i64 %x) optnone noinline {
entry:
  ret i64 %x
}

define void @dynamic_alloc(i64 %n) {
entry:
  %p = call ptr @alloc(i64 %n) allockind("alloc,uninitialized")
  call void @use(ptr %p)
  ret void
}

define i64 @promotable_post_inline() {
entry:
  %p = call ptr @alloc(i64 8) allockind("alloc,uninitialized")
  store i64 42, ptr %p, align 8
  %v = load i64, ptr %p, align 8
  ret i64 %v
}

define i64 @derived_phi_live(i1 %again, ptr %init) {
entry:
  br label %header
header:
  %prev = phi ptr [ %init, %entry ], [ %gep, %latch ]
  br label %allocblk
allocblk:
  %old = load i64, ptr %prev, align 8
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store i64 %old, ptr %p, align 8
  br label %derive
derive:
  %gep = getelementptr inbounds i8, ptr %p, i64 8
  br label %latch
latch:
  br i1 %again, label %header, label %exit
exit:
  %res = load i64, ptr %gep, align 8
  ret i64 %res
}
`)
	tm := tailTarget(t, mod)
	if promoted, err := RunPostInline(llvm.Module{}, tm); err == nil || promoted != nil {
		t.Fatalf("RunPostInline(nil) = (%v, %v), want (nil, error)", promoted, err)
	}
	if err := CleanupPromoted(llvm.Module{}, tm, nil); err != nil {
		t.Fatalf("CleanupPromoted(nil) = %v, want nil", err)
	}
	if err := CleanupPromoted(mod, tm, nil); err != nil {
		t.Fatalf("CleanupPromoted(empty) = %v, want nil", err)
	}

	promoted, err := RunPostInline(mod, tm)
	if err != nil {
		t.Fatal(err)
	}
	if len(promoted) != 1 || promoted[0].Name() != "promotable_post_inline" {
		t.Fatalf("RunPostInline promoted = %v, want [promotable_post_inline]", promoted)
	}
	optnone := llvm.AttributeKindID("optnone")
	if !mod.NamedFunction("no_alloc").GetEnumFunctionAttribute(optnone).IsNil() {
		t.Fatal("RunPostInline did not restore optnone on skipped no_alloc")
	}
	if mod.NamedFunction("already_optnone").GetEnumFunctionAttribute(optnone).IsNil() {
		t.Fatal("RunPostInline stripped existing optnone from already_optnone")
	}
	if !strings.Contains(mod.NamedFunction("dynamic_alloc").String(), "@alloc(") {
		t.Fatal("RunPostInline unexpectedly modified dynamic_alloc")
	}

	if err := CleanupPromoted(mod, tm, promoted); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	cleaned := mod.NamedFunction("promotable_post_inline").String()
	if strings.Contains(cleaned, "@alloc(") || strings.Contains(cleaned, "alloca ") || !strings.Contains(cleaned, "ret i64 42") {
		t.Fatalf("CleanupPromoted did not simplify promoted storage:\n%s", cleaned)
	}
}

func TestIsDerivedLiveAtAllocAndClearTail(t *testing.T) {
	mod := parseModule(t, `
declare void @llvm.memset.p0.i64(ptr nocapture writeonly, i8, i64, i1 immarg)

define i64 @phi_and_block_cases(i1 %cond, ptr %init) {
entry:
  br label %allocblk
allocblk:
  %early = load i64, ptr %init, align 8
  %slot = alloca [16 x i8], align 8
  %g1 = getelementptr inbounds i8, ptr %slot, i64 0
  %dup = select i1 %cond, ptr %g1, ptr %g1
  tail call void @llvm.memset.p0.i64(ptr align 8 %dup, i8 0, i64 16, i1 false)
  br i1 %cond, label %defblk, label %afterblk
defblk:
  %derived = getelementptr inbounds i8, ptr %slot, i64 8
  br label %afterblk
afterblk:
  %merged = phi ptr [ %derived, %defblk ], [ %init, %allocblk ]
  %late = load i64, ptr %merged, align 8
  %sum = add i64 %early, %late
  ret i64 %sum
}

define i64 @loop_live_use(i1 %cond, ptr %init) {
entry:
  br label %header
header:
  %carried_other = phi ptr [ %init, %entry ], [ %next, %latch ]
  %carried_after = phi ptr [ %init, %entry ], [ %next, %latch ]
  br label %allocblk
allocblk:
  %early = load i64, ptr %carried_other, align 8
  %slot = alloca [16 x i8], align 8
  %after = load i64, ptr %carried_after, align 8
  br i1 %cond, label %useblk, label %latch
useblk:
  %late = load i64, ptr %carried_other, align 8
  br label %latch
latch:
  %use_val = phi i64 [ %late, %useblk ], [ %early, %allocblk ]
  %next = getelementptr inbounds i8, ptr %slot, i64 8
  %sum = add i64 %use_val, %after
  br i1 %cond, label %header, label %exit
exit:
  ret i64 %sum
}
`)
	fn := mod.NamedFunction("phi_and_block_cases")
	var allocBB, defBB llvm.BasicBlock
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		switch bb.AsValue().Name() {
		case "allocblk":
			allocBB = bb
		case "defblk":
			defBB = bb
		}
	}
	var slot, derived llvm.Value
	order := make(map[llvm.Value]int)
	pos := 0
	for i := allocBB.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
		order[i] = pos
		pos++
		if i.Name() == "slot" {
			slot = i
		}
	}
	for i := defBB.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
		if i.Name() == "derived" {
			derived = i
		}
	}
	// Duplicate worklist path in clearTailOnMemIntrinsics (select references %g1 twice).
	clearTailOnMemIntrinsics(slot)
	if strings.Contains(fn.String(), "tail call void @llvm.memset") {
		t.Fatalf("clearTailOnMemIntrinsics kept tail on memset:\n%s", fn.String())
	}
	// If reachable includes defBB (the incoming block of %merged PHI), PHI check returns true.
	if !isDerivedLiveAtAlloc(derived, defBB, allocBB, order[slot], order, map[llvm.BasicBlock]bool{defBB: true}) {
		t.Fatal("expected derived value feeding PHI from reachable predecessor to be live")
	}
	// With empty reachable set, the PHI incoming block is unreachable -> not live.
	if isDerivedLiveAtAlloc(derived, defBB, allocBB, order[slot], order, map[llvm.BasicBlock]bool{}) {
		t.Fatal("unexpected liveness with empty reachable set")
	}

	// Verifier-valid loop fixture for non-PHI uses (pre-alloc, post-alloc, and cross-block).
	loopFn := mod.NamedFunction("loop_live_use")
	var headerBB, loopAllocBB llvm.BasicBlock
	for bb := loopFn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		switch bb.AsValue().Name() {
		case "header":
			headerBB = bb
		case "allocblk":
			loopAllocBB = bb
		}
	}
	var carriedOther, carriedAfter, loopSlot llvm.Value
	for i := headerBB.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
		switch i.Name() {
		case "carried_other":
			carriedOther = i
		case "carried_after":
			carriedAfter = i
		}
	}
	loopOrder := make(map[llvm.Value]int)
	pos = 0
	for i := loopAllocBB.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
		loopOrder[i] = pos
		pos++
		if i.Name() == "slot" {
			loopSlot = i
		}
	}
	// Loop-carried pointer used before %slot in allocblk and after %slot in reachable useblk.
	reachNoHeader := forwardReachableWithout(loopAllocBB, headerBB)
	if !isDerivedLiveAtAlloc(carriedOther, headerBB, loopAllocBB, loopOrder[loopSlot], loopOrder, reachNoHeader) {
		t.Fatal("expected loop-carried pointer used in reachable useblk to be live")
	}
	// When useblk is not reachable, only the pre-alloc load in allocblk remains -> not live.
	if isDerivedLiveAtAlloc(carriedOther, headerBB, loopAllocBB, loopOrder[loopSlot], loopOrder, map[llvm.BasicBlock]bool{}) {
		t.Fatal("unexpected liveness when only pre-alloc use is reachable")
	}
	// Loop-carried pointer used after %slot in the same allocblk -> live even with empty reachable set.
	if !isDerivedLiveAtAlloc(carriedAfter, headerBB, loopAllocBB, loopOrder[loopSlot], loopOrder, map[llvm.BasicBlock]bool{}) {
		t.Fatal("expected loop-carried pointer used after allocation in allocblk to be live")
	}
}

func parseModule(t *testing.T, source string) llvm.Module {
	t.Helper()
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	path := filepath.Join(t.TempDir(), "input.ll")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mod.Dispose)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid test IR module: %v", err)
	}
	return mod
}

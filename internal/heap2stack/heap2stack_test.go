package heap2stack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

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
	return mod
}

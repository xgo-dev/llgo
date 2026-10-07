package abi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestShouldLowerArrayCopy(t *testing.T) {
	for _, tc := range []struct {
		length, ptrSize int
		size            uint64
		want            bool
	}{
		{length: 1, size: 64, ptrSize: 8, want: false},
		{length: 2, size: 16, ptrSize: 8, want: false},
		{length: 4, size: 32, ptrSize: 8, want: false},
		{length: 5, size: 40, ptrSize: 8, want: true},
		{length: 256, size: 1024, ptrSize: 8, want: true},
		{length: 2, size: 16, ptrSize: 4, want: false},
		{length: 3, size: 24, ptrSize: 4, want: true},
	} {
		if got := ShouldLowerArrayCopy(tc.length, tc.size, tc.ptrSize); got != tc.want {
			t.Errorf("ShouldLowerArrayCopy(%d, %d, %d) = %v, want %v", tc.length, tc.size, tc.ptrSize, got, tc.want)
		}
		if got := ShouldSnapshotAggregateLoad(true, tc.length, false, tc.size, tc.ptrSize, false); got && tc.size < MinAggregateCopySize {
			t.Errorf("ShouldSnapshotAggregateLoad(array %d, %d) = true, want false below 4KiB (stack snapshot)", tc.length, tc.size)
		}
		if got := ShouldSnapshotAggregateLoad(true, tc.length, false, tc.size, tc.ptrSize, true); got && tc.size < MinAggregateCopySize {
			t.Errorf("wasm ShouldSnapshotAggregateLoad(array %d, %d) = true, want false below 4KiB", tc.length, tc.size)
		}
	}
	if !ShouldSnapshotAggregateLoad(false, 0, true, MinAggregateCopySize, 8, false) {
		t.Fatal("struct at 4KiB should snapshot")
	}
	if ShouldSnapshotAggregateLoad(false, 0, true, MinAggregateCopySize-1, 8, false) {
		t.Fatal("struct below 4KiB should not snapshot")
	}
}

func TestLargeAggregateThreshold(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()
	l := largeAggregateLowerer{td: td}

	if l.isLargeAggregate(ctx.Int64Type()) {
		t.Fatal("scalar type was classified as a large aggregate")
	}
	if l.isLargeAggregate(llvm.ArrayType(ctx.Int8Type(), int(MaxImplicitStackVarSize))) {
		t.Fatal("aggregate at the implicit stack limit was classified as large")
	}
	if !l.isLargeAggregate(llvm.ArrayType(ctx.Int8Type(), int(MaxImplicitStackVarSize+1))) {
		t.Fatal("aggregate above the implicit stack limit was not classified as large")
	}
	l.copyMinSize = MinAggregateCopySize
	if l.isLargeCopy(ctx.Int64Type()) {
		t.Fatal("scalar type was classified as an aggregate copy")
	}
}

func TestLowerAggregateCopies(t *testing.T) {
	const testIR = `
define void @copy(ptr %src, ptr %dst) {
entry:
  %value = load [4096 x i8], ptr %src
  store [4096 x i8] %value, ptr %dst
  ret void
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "wasm_copy.ll")
	if err := os.WriteFile(path, []byte(testIR), 0o644); err != nil {
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
	defer mod.Dispose()
	td := llvm.NewTargetData("e-p:32:32-i64:64-n32:64-S128")
	defer td.Dispose()
	config := AggregateLoweringConfig{GoWordSize: 8, GCRoots: true, Wasm: true}
	if got := LowerAggregateCopies(td, mod, config); got != 1 {
		t.Fatalf("lowered %d copies, want 1", got)
	}
	if got := LowerAggregateCopies(td, mod, config); got != 0 {
		t.Fatalf("second pass lowered %d copies, want 0", got)
	}
	if body := mod.NamedFunction("copy").String(); !strings.Contains(body, "@llvm.memmove") {
		t.Fatalf("copy was not lowered to memmove:\n%s", body)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
	}
}

func TestLowerAggregateCopiesAllocUDebugLoc(t *testing.T) {
	const testIR = `
define void @copy(ptr %src, ptr %dst, ptr %other) !dbg !3 {
entry:
  %v = load [4096 x i8], ptr %src, !dbg !4
  call void @mutate(ptr %src), !dbg !4
  store [4096 x i8] %v, ptr %dst, !dbg !4
  store [4096 x i8] %v, ptr %other, !dbg !4
  ret void, !dbg !4
}
declare void @mutate(ptr)
!llvm.dbg.cu = !{!0}
!llvm.module.flags = !{!2}
!0 = distinct !DICompileUnit(language: DW_LANG_C, file: !1, producer: "t", isOptimized: false, runtimeVersion: 0, emissionKind: FullDebug)
!1 = !DIFile(filename: "t.c", directory: "/")
!2 = !{i32 2, !"Debug Info Version", i32 3}
!3 = distinct !DISubprogram(name: "copy", scope: !1, file: !1, line: 1, type: !5, spFlags: DISPFlagDefinition, unit: !0)
!4 = !DILocation(line: 1, column: 1, scope: !3)
!5 = !DISubroutineType(types: !6)
!6 = !{null}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "dbg_copy.ll")
	if err := os.WriteFile(path, []byte(testIR), 0o644); err != nil {
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
	defer mod.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()
	if got := LowerAggregateCopies(td, mod, AggregateLoweringConfig{GoWordSize: 8}); got != 1 {
		t.Fatalf("lowered %d copies, want 1:\n%s", got, mod.String())
	}
	body := mod.NamedFunction("copy").String()
	if !strings.Contains(body, "AllocU") || !strings.Contains(body, "!dbg") {
		t.Fatalf("AllocU snapshot missing debug location:\n%s", body)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("debug function with AllocU snapshot failed verify: %v\n%s", err, mod.String())
	}
}

func TestLowerMultiElementArrayCopies(t *testing.T) {
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()
	config := AggregateLoweringConfig{GoWordSize: 8}

	t.Run("polynomial copy", func(t *testing.T) {
		mod := parseAggregateIR(t, `
define void @copy(ptr %src, ptr %dst) {
entry:
  %v = load [256 x i32], ptr %src
  store [256 x i32] %v, ptr %dst
  ret void
}
`)
		if got := LowerAggregateCopies(td, mod, config); got != 1 {
			t.Fatalf("lowered %d copies, want 1:\n%s", got, mod.String())
		}
		body := mod.NamedFunction("copy").String()
		if !strings.Contains(body, "@llvm.memmove") || strings.Contains(body, "load [256 x i32]") {
			t.Fatalf("multi-element array copy was not lowered to memmove:\n%s", body)
		}
	})

	t.Run("small two-element array", func(t *testing.T) {
		mod := parseAggregateIR(t, `
define void @copy(ptr %src, ptr %dst) {
entry:
  %v = load [2 x i64], ptr %src
  store [2 x i64] %v, ptr %dst
  ret void
}
`)
		before := mod.String()
		if got := LowerAggregateCopies(td, mod, config); got != 0 || mod.String() != before {
			t.Fatalf("register-sized array copy was rewritten:\n%s", mod.String())
		}
	})

	t.Run("above CanSSA size", func(t *testing.T) {
		mod := parseAggregateIR(t, `
define void @copy(ptr %src, ptr %dst) {
entry:
  %v = load [5 x i64], ptr %src
  store [5 x i64] %v, ptr %dst
  ret void
}
`)
		if got := LowerAggregateCopies(td, mod, config); got != 1 {
			t.Fatalf("lowered %d copies, want 1:\n%s", got, mod.String())
		}
		body := mod.NamedFunction("copy").String()
		if !strings.Contains(body, "@llvm.memmove") || strings.Contains(body, "load [5 x i64]") {
			t.Fatalf("40-byte array copy was not lowered to memmove:\n%s", body)
		}
	})

	t.Run("one-element array", func(t *testing.T) {
		mod := parseAggregateIR(t, `
define void @copy(ptr %src, ptr %dst) {
entry:
  %v = load [1 x i64], ptr %src
  store [1 x i64] %v, ptr %dst
  ret void
}
`)
		before := mod.String()
		if got := LowerAggregateCopies(td, mod, config); got != 0 || mod.String() != before {
			t.Fatalf("one-element array copy was rewritten:\n%s", mod.String())
		}
	})

	t.Run("call argument", func(t *testing.T) {
		mod := parseAggregateIR(t, `
declare void @take([2 x i64])
define void @pass(ptr %src) {
entry:
  %v = load [2 x i64], ptr %src
  call void @take([2 x i64] %v)
  ret void
}
`)
		before := mod.String()
		if got := LowerAggregateCopies(td, mod, config); got != 0 || mod.String() != before {
			t.Fatalf("array used as a call argument was rewritten:\n%s", mod.String())
		}
	})

	t.Run("snapshot roots", func(t *testing.T) {
		mod := parseAggregateIR(t, `
declare void @mutate(ptr)
define ptr @copy(ptr %src, ptr %dst, ptr %other) {
entry:
  %v = load [5 x i64], ptr %src
  call void @mutate(ptr %src)
  store [5 x i64] %v, ptr %dst
  store [5 x i64] %v, ptr %other
  ret ptr %src
}
`)
		cfg := AggregateLoweringConfig{GoWordSize: 8, GCRoots: true, Wasm: false}
		if got := LowerAggregateCopies(td, mod, cfg); got != 1 {
			t.Fatalf("lowered %d copies, want 1:\n%s", got, mod.String())
		}
		body := mod.NamedFunction("copy").String()
		if strings.Contains(body, "AllocU") {
			t.Fatalf("40-byte array snapshot used heap AllocU:\n%s", body)
		}
		if !strings.Contains(body, "alloca") {
			t.Fatalf("40-byte array snapshot missing stack alloca:\n%s", body)
		}
	})

	t.Run("pre-C-ABI large aggregate pass", func(t *testing.T) {
		mod := parseAggregateIR(t, `
define void @copy(ptr %src, ptr %dst) {
entry:
  %v = load [256 x i32], ptr %src
  store [256 x i32] %v, ptr %dst
  ret void
}
`)
		before := mod.String()
		LowerLargeAggregates(td, mod, config)
		if mod.String() != before {
			t.Fatalf("LowerLargeAggregates rewrote a 1KiB array copy before C ABI:\n%s", mod.String())
		}
	})
}

func TestLowerMultiElementArrayCopyStackReuse(t *testing.T) {
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()
	config := AggregateLoweringConfig{GoWordSize: 8}

	t.Run("disjoint switch cases share one slot", func(t *testing.T) {
		mod := parseAggregateIR(t, switchCopyIR(8))
		if got := LowerAggregateCopies(td, mod, config); got != 8 {
			t.Fatalf("lowered %d copies, want 8:\n%s", got, mod.String())
		}
		if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
		}
		fn := mod.NamedFunction("copy")
		body := fn.String()
		if got := strings.Count(body, "alloca [256 x i32]"); got != 1 {
			t.Fatalf("disjoint snapshots reserved %d slots, want 1:\n%s", got, body)
		}
		if strings.Contains(body, "AllocU") {
			t.Fatalf("sub-4KiB snapshots used heap AllocU:\n%s", body)
		}
		if got := strings.Count(body, "llvm.lifetime.start"); got != 8 {
			t.Fatalf("lifetime.start count = %d, want 8:\n%s", got, body)
		}
		if got := strings.Count(body, "llvm.lifetime.end"); got != 8 {
			t.Fatalf("lifetime.end count = %d, want 8:\n%s", got, body)
		}
		if !allocaInEntry(fn) {
			t.Fatalf("snapshot slot left the entry block:\n%s", body)
		}
	})

	t.Run("overlapping snapshots keep separate slots", func(t *testing.T) {
		mod := parseAggregateIR(t, `
declare void @mutate(ptr)
define void @copy(ptr %src1, ptr %src2, ptr %dst1, ptr %dst2) {
entry:
  %a = load [256 x i32], ptr %src1
  %b = load [256 x i32], ptr %src2
  call void @mutate(ptr %src1)
  store [256 x i32] %a, ptr %dst1
  store [256 x i32] %b, ptr %dst2
  ret void
}
`)
		if got := LowerAggregateCopies(td, mod, config); got != 2 {
			t.Fatalf("lowered %d copies, want 2:\n%s", got, mod.String())
		}
		if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
		}
		body := mod.NamedFunction("copy").String()
		if got := strings.Count(body, "alloca [256 x i32]"); got != 2 {
			t.Fatalf("overlapping snapshots reserved %d slots, want 2:\n%s", got, body)
		}
	})

	t.Run("sequential snapshots reuse one slot", func(t *testing.T) {
		mod := parseAggregateIR(t, `
declare void @mutate(ptr)
define void @copy(ptr %src, ptr %dst) {
entry:
  %a = load [256 x i32], ptr %src
  call void @mutate(ptr %src)
  store [256 x i32] %a, ptr %dst
  %b = load [256 x i32], ptr %src
  call void @mutate(ptr %src)
  store [256 x i32] %b, ptr %dst
  ret void
}
`)
		if got := LowerAggregateCopies(td, mod, config); got != 2 {
			t.Fatalf("lowered %d copies, want 2:\n%s", got, mod.String())
		}
		if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
		}
		body := mod.NamedFunction("copy").String()
		if got := strings.Count(body, "alloca [256 x i32]"); got != 1 {
			t.Fatalf("sequential snapshots reserved %d slots, want 1:\n%s", got, body)
		}
	})

	t.Run("loop keeps one entry alloca", func(t *testing.T) {
		mod := parseAggregateIR(t, `
declare void @mutate(ptr)
define void @copy(ptr %src, ptr %dst, i32 %n) {
entry:
  br label %loop
loop:
  %i = phi i32 [ 0, %entry ], [ %next, %loop ]
  %v = load [256 x i32], ptr %src
  call void @mutate(ptr %src)
  store [256 x i32] %v, ptr %dst
  %next = add i32 %i, 1
  %cmp = icmp slt i32 %next, %n
  br i1 %cmp, label %loop, label %done
done:
  ret void
}
`)
		if got := LowerAggregateCopies(td, mod, config); got != 1 {
			t.Fatalf("lowered %d copies, want 1:\n%s", got, mod.String())
		}
		if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
		}
		fn := mod.NamedFunction("copy")
		body := fn.String()
		if got := strings.Count(body, "alloca [256 x i32]"); got != 1 {
			t.Fatalf("loop snapshots reserved %d slots, want 1:\n%s", got, body)
		}
		if !allocaInEntry(fn) {
			t.Fatalf("loop snapshot left the entry block:\n%s", body)
		}
	})
}

func switchCopyIR(cases int) string {
	var b strings.Builder
	b.WriteString("declare void @mutate(ptr, i32)\n")
	b.WriteString("define void @copy(ptr %src, ptr %dst, i32 %id) {\nentry:\n")
	b.WriteString("  switch i32 %id, label %dflt [\n")
	for i := 0; i < cases; i++ {
		fmt.Fprintf(&b, "    i32 %d, label %%c%d\n", i, i)
	}
	b.WriteString("  ]\n")
	for i := 0; i < cases; i++ {
		fmt.Fprintf(&b, "c%d:\n  %%v%d = load [256 x i32], ptr %%src\n  call void @mutate(ptr %%src, i32 %d)\n  store [256 x i32] %%v%d, ptr %%dst\n  br label %%end\n", i, i, i, i)
	}
	b.WriteString("dflt:\n  br label %end\nend:\n  ret void\n}\n")
	return b.String()
}

func allocaInEntry(fn llvm.Value) bool {
	entry := fn.FirstBasicBlock()
	for instr := entry.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
		if !instr.IsAAllocaInst().IsNil() {
			return true
		}
	}
	return false
}

func parseAggregateIR(t *testing.T, source string) llvm.Module {
	t.Helper()
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	path := filepath.Join(t.TempDir(), "copy.ll")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
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

func TestLowerAggregateCopiesNestedConvergence(t *testing.T) {
	for _, depth := range []int{1, 8, 32} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			var ir strings.Builder
			ir.WriteString("%Nest0 = type [4096 x i8]\n")
			for i := 1; i <= depth; i++ {
				fmt.Fprintf(&ir, "%%Nest%d = type { %%Nest%d }\n", i, i-1)
			}
			ir.WriteString("define void @copy(ptr %src, ptr %dst) {\nentry:\n")
			fmt.Fprintf(&ir, "  %%v%d = load %%Nest%d, ptr %%src\n", depth, depth)
			for i := depth; i > 0; i-- {
				fmt.Fprintf(&ir, "  %%v%d = extractvalue %%Nest%d %%v%d, 0\n", i-1, i, i)
			}
			ir.WriteString("  store %Nest0 %v0, ptr %dst\n  ret void\n}\n")

			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "nested.ll")
			if err := os.WriteFile(path, []byte(ir.String()), 0o644); err != nil {
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
			defer mod.Dispose()
			td := llvm.NewTargetData("e-p:32:32-i64:64-n32:64-S128")
			defer td.Dispose()
			config := AggregateLoweringConfig{GoWordSize: 8, GCRoots: true, Wasm: true}
			if got := LowerAggregateCopies(td, mod, config); got != depth+1 {
				t.Fatalf("lowered %d copies, want %d", got, depth+1)
			}
			if got := LowerAggregateCopies(td, mod, config); got != 0 {
				t.Fatalf("second pass lowered %d copies, want 0", got)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
			}
		})
	}
}

func TestLowerLargeAggregates(t *testing.T) {
	const testIR = `
%Large = type [65537 x i8]
%Small = type [65536 x i8]
define %Large @callee(ptr nonnull %src) #1 {
entry:
  %value = load %Large, ptr %src, align 1
  %first = getelementptr inbounds %Large, ptr %src, i64 0, i64 0
  store i8 9, ptr %first, align 1
  ret %Large %value
}

define i8 @caller(ptr %src) {
entry:
  %value = call %Large @callee(ptr "llgo.reflect.methodbyname.name"="1" %src) #0
  %dst = alloca %Large, align 1
  store %Large %value, ptr %dst, align 1
  %first = getelementptr inbounds %Large, ptr %dst, i64 0, i64 0
  %result = load i8, ptr %first, align 1
  ret i8 %result
}

define %Large @wrapper(ptr %src) {
entry:
  %value = call %Large @callee(ptr %src)
  ret %Large %value
}

define %Large @indirect(ptr %fn, ptr %src) {
entry:
  %value = call %Large %fn(ptr %src)
  ret %Large %value
}

define %Large @self_copy(ptr %src) {
entry:
  %copy = load %Large, ptr %src, align 1
  store %Large %copy, ptr %src, align 1
  %result = load %Large, ptr %src, align 1
  ret %Large %result
}

define %Small @small(ptr %src) {
entry:
  %value = load %Small, ptr %src, align 1
  ret %Small %value
}

attributes #0 = { "llgo.reflect.methodbyname"="value" }
attributes #1 = { noinline }
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "large.ll")
	if err := os.WriteFile(path, []byte(testIR), 0o644); err != nil {
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
	defer mod.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()

	LowerLargeAggregates(td, mod, AggregateLoweringConfig{})

	callee := mod.NamedFunction("callee").String()
	if !strings.Contains(callee, "define void @callee(ptr sret([65537 x i8])") {
		t.Fatalf("large return was not lowered to sret:\n%s", callee)
	}
	if !strings.Contains(callee, "ptr nonnull %") || !strings.Contains(callee, "noinline") {
		t.Fatalf("callee attributes were not preserved:\n%s", callee)
	}
	copyAtLoad := strings.Index(callee, "call void @llvm.memcpy")
	mutation := strings.Index(callee, "store i8 9")
	if copyAtLoad < 0 || mutation < 0 || copyAtLoad >= mutation {
		t.Fatalf("return value was not copied before source mutation:\n%s", callee)
	}
	if strings.Contains(callee, "load [65537 x i8]") || strings.Contains(callee, "store [65537 x i8]") {
		t.Fatalf("callee retained a direct large aggregate copy:\n%s", callee)
	}

	caller := mod.NamedFunction("caller").String()
	for _, want := range []string{
		`call ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.AllocU"(i64 65537)`,
		"call void @callee(ptr sret([65537 x i8])",
		`ptr "llgo.reflect.methodbyname.name"="1"`,
		"call void @llvm.memcpy",
	} {
		if !strings.Contains(caller, want) {
			t.Fatalf("transformed caller missing %q:\n%s", want, caller)
		}
	}
	if strings.Contains(caller, "load [65537 x i8]") || strings.Contains(caller, "store [65537 x i8]") {
		t.Fatalf("caller reconstructed the large return as an SSA value:\n%s", caller)
	}
	if !strings.Contains(mod.String(), `"llgo.reflect.methodbyname"="value"`) {
		t.Fatalf("reflect MethodByName call marker was not preserved:\n%s", mod.String())
	}

	for _, name := range []string{"wrapper", "indirect", "self_copy"} {
		ir := mod.NamedFunction(name).String()
		if strings.Contains(ir, "load [65537 x i8]") || strings.Contains(ir, "store [65537 x i8]") {
			t.Fatalf("%s retained a direct large aggregate copy:\n%s", name, ir)
		}
	}
	if got := strings.Count(mod.NamedFunction("self_copy").String(), "call void @llvm.memcpy"); got != 1 {
		t.Fatalf("self_copy has %d memcpy calls, want 1:\n%s", got, mod.NamedFunction("self_copy").String())
	}

	small := mod.NamedFunction("small").String()
	if !strings.Contains(small, "define [65536 x i8] @small") || !strings.Contains(small, "load [65536 x i8]") {
		t.Fatalf("aggregate at the threshold was unexpectedly lowered:\n%s", small)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("transformed module is invalid: %v\n%s", err, mod.String())
	}
}

func TestLowerLargeAggregatePreservesClosureEnvAttribute(t *testing.T) {
	const testIR = `
%Large = type [65537 x i8]

define %Large @callee(ptr nest %env) {
entry:
  ret %Large zeroinitializer
}

define %Large @calleeSwift(ptr swiftself %env) {
entry:
  ret %Large zeroinitializer
}

define void @caller(ptr %env) {
entry:
  %unused = call %Large @callee(ptr nest %env)
  %unusedSwift = call %Large @calleeSwift(ptr swiftself %env)
  ret void
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "large_closure_env.ll")
	if err := os.WriteFile(path, []byte(testIR), 0o644); err != nil {
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
	defer mod.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()

	LowerLargeAggregates(td, mod, AggregateLoweringConfig{})

	nest := llvm.AttributeKindID("nest")
	callee := mod.NamedFunction("callee")
	if attr := callee.GetEnumAttributeAtIndex(2, nest); attr.IsNil() {
		t.Fatalf("large return lowering lost nest after inserting sret:\n%s", callee.String())
	}
	if attr := callee.GetEnumAttributeAtIndex(1, nest); !attr.IsNil() {
		t.Fatalf("large return lowering left nest on the new sret parameter:\n%s", callee.String())
	}
	swiftself := llvm.AttributeKindID("swiftself")
	calleeSwift := mod.NamedFunction("calleeSwift")
	if attr := calleeSwift.GetEnumAttributeAtIndex(2, swiftself); attr.IsNil() {
		t.Fatalf("large return lowering lost swiftself after inserting sret:\n%s", calleeSwift.String())
	}

	caller := mod.NamedFunction("caller")
	var nestedCall, swiftselfCall llvm.Value
	for block := caller.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			if call := instruction.IsACallInst(); !call.IsNil() {
				switch call.CalledValue().Name() {
				case "callee":
					nestedCall = call
				case "calleeSwift":
					swiftselfCall = call
				}
			}
		}
	}
	if nestedCall.IsNil() {
		t.Fatalf("large return lowering removed the callee call:\n%s", caller.String())
	}
	if attr := nestedCall.GetCallSiteEnumAttribute(2, nest); attr.IsNil() {
		t.Fatalf("large return call lowering lost nest after inserting sret:\n%s", caller.String())
	}
	if swiftselfCall.IsNil() || swiftselfCall.GetCallSiteEnumAttribute(2, swiftself).IsNil() {
		t.Fatalf("large return call lowering lost swiftself after inserting sret:\n%s", caller.String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("large-return closure-env module is invalid: %v\n%s", err, mod.String())
	}
}

func TestLowerLargeAggregateStoredLoads(t *testing.T) {
	const testIR = `
%Large = type [65537 x i8]
%Small = type [65536 x i8]
%WithPointer = type { ptr, %Large }
declare void @mutate(ptr)

define void @copy_volatile(ptr %src, ptr %dst) {
entry:
  store volatile %WithPointer zeroinitializer, ptr %dst, align 1
  %value = load volatile %Large, ptr %src, align 1
  store %Large %value, ptr %dst, align 1
  ret void
}

define ptr @project_volatile(ptr %src, ptr %dst) {
entry:
  %value = load volatile %WithPointer, ptr %src, align 1
  call void @mutate(ptr %src)
  store volatile %WithPointer %value, ptr %dst, align 1
  %pointer = extractvalue %WithPointer %value, 0
  ret ptr %pointer
}

define void @unsupported_value(ptr %src, ptr %dst) {
entry:
  %value = load %Large, ptr %src, align 1
  %changed = insertvalue %Large %value, i8 1, 0
  store %Large %changed, ptr %dst, align 1
  ret void
}


define void @copy_twice(ptr %src, ptr %dst1, ptr %dst2) {
entry:
  %value = load %Large, ptr %src, align 1
  %first = getelementptr inbounds %Large, ptr %src, i64 0, i64 0
  store i8 9, ptr %first, align 1
  store %Large %value, ptr %dst1, align 1
  store %Large %value, ptr %dst2, align 1
  ret void
}

define void @copy_once(ptr %src, ptr %dst) {
entry:
  %value = load %Large, ptr %src, align 1
  store %Large %value, ptr %dst, align 1
  ret void
}

define i8 @mixed_use(ptr %src, ptr %dst) {
entry:
  %value = load %Large, ptr %src, align 1
  store %Large %value, ptr %dst, align 1
  %first = extractvalue %Large %value, 0
  ret i8 %first
}

define void @small_copy(ptr %src, ptr %dst) {
entry:
  %value = load %Small, ptr %src, align 1
  store %Small %value, ptr %dst, align 1
  ret void
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "large_stored_loads.ll")
	if err := os.WriteFile(path, []byte(testIR), 0o644); err != nil {
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
	defer mod.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()

	LowerLargeAggregates(td, mod, AggregateLoweringConfig{})

	copyTwice := mod.NamedFunction("copy_twice").String()
	if got := strings.Count(copyTwice, "call void @llvm.memcpy"); got != 3 {
		t.Fatalf("copy_twice has %d memcpy calls, want snapshot plus two stores:\n%s", got, copyTwice)
	}
	snapshot := strings.Index(copyTwice, "call void @llvm.memcpy")
	mutation := strings.Index(copyTwice, "store i8 9")
	if snapshot < 0 || mutation < 0 || snapshot >= mutation {
		t.Fatalf("large value was not snapshotted before source mutation:\n%s", copyTwice)
	}
	if strings.Contains(copyTwice, "load [65537 x i8]") || strings.Contains(copyTwice, "store [65537 x i8]") {
		t.Fatalf("copy_twice retained a direct large aggregate copy:\n%s", copyTwice)
	}

	copyOnce := mod.NamedFunction("copy_once").String()
	if strings.Count(copyOnce, "call void @llvm.memmove") != 1 || strings.Contains(copyOnce, "AllocU") {
		t.Fatalf("adjacent large copy was not lowered directly to memmove:\n%s", copyOnce)
	}
	if strings.Contains(copyOnce, "load [65537 x i8]") || strings.Contains(copyOnce, "store [65537 x i8]") {
		t.Fatalf("copy_once retained a direct large aggregate copy:\n%s", copyOnce)
	}
	mixed := mod.NamedFunction("mixed_use").String()
	if strings.Contains(mixed, "load [65537 x i8]") || strings.Contains(mixed, "store [65537 x i8]") ||
		strings.Contains(mixed, "extractvalue") || !strings.Contains(mixed, "load i8") {
		t.Fatalf("mixed projection/store did not use the same snapshot:\n%s", mixed)
	}
	volatile := mod.NamedFunction("copy_volatile").String()
	if strings.Contains(volatile, "load volatile") || strings.Contains(volatile, "store volatile") ||
		strings.Count(volatile, "i1 true)") != 2 || !strings.Contains(volatile, "@llvm.memmove") || !strings.Contains(volatile, "@llvm.memset") {
		t.Fatalf("volatile adjacent copy lost its memory semantics:\n%s", volatile)
	}
	projected := mod.NamedFunction("project_volatile").String()
	if strings.Contains(projected, "load volatile") || strings.Contains(projected, "extractvalue") ||
		strings.Count(projected, "i1 true)") != 2 || !strings.Contains(projected, "load ptr") {
		t.Fatalf("volatile projected snapshot was not lowered:\n%s", projected)
	}
	if strings.Index(projected, "@llvm.memcpy") >= strings.Index(projected, "@mutate") {
		t.Fatalf("snapshot moved after source mutation:\n%s", projected)
	}
	unsupported := mod.NamedFunction("unsupported_value").String()
	if !strings.Contains(unsupported, "insertvalue") || !strings.Contains(unsupported, "load [65537 x i8]") {
		t.Fatalf("unsupported aggregate user was partially rewritten:\n%s", unsupported)
	}
	small := mod.NamedFunction("small_copy").String()
	if !strings.Contains(small, "load [65536 x i8]") || !strings.Contains(small, "store [65536 x i8]") {
		t.Fatalf("aggregate at the threshold was unexpectedly rewritten:\n%s", small)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("stored-load module is invalid: %v\n%s", err, mod.String())
	}
}

package abi

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWasmHeaderSnapshotForwarding(t *testing.T) {
	const checked = `
  %nil = icmp eq ptr %base, null
  br i1 %nil, label %panic, label %write
panic:
  call void @panic()
  unreachable
write:
  $write
  ret void
`
	for _, tc := range []struct {
		name, prefix, tail          string
		volatileSource, volatileDst bool
		forward                     bool
		stack, shared               bool
	}{
		{name: "checked destination", tail: checked, forward: true},
		{name: "private stack", stack: true, tail: checked, forward: true},
		{name: "volatile stack", stack: true, volatileSource: true, tail: checked},
		{name: "delayed stack", stack: true, tail: "call void @mutate(ptr %src)\n" + checked},
		{name: "shared stack", stack: true, shared: true, tail: "store %Headers zeroinitializer, ptr %slot\n" + checked},
		{name: "unused branch", forward: true, tail: `
  br i1 %condition, label %write, label %exit
write:
  $write
  br label %exit
exit:
  ret void
`},
		{name: "source mutation", tail: "store i8 1, ptr %src\n" + checked},
		{name: "intervening call", tail: "call void @mutate(ptr %src)\n" + checked},
		{name: "volatile source", volatileSource: true, tail: checked},
		{name: "volatile destination", volatileDst: true, tail: checked},
		{name: "published snapshot", prefix: "store ptr %slot, ptr %root", tail: checked},
		{name: "merged paths", tail: `
  br i1 %condition, label %left, label %right
left:
  br label %write
right:
  br label %write
write:
  $write
  ret void
`},
		{name: "repeated store", tail: `
  br label %write
write:
  $write
  br i1 %condition, label %write, label %exit
exit:
  ret void
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceVolatile, dstVolatile := "", ""
			if tc.volatileSource {
				sourceVolatile = "volatile"
			}
			if tc.volatileDst {
				dstVolatile = "volatile"
			}
			// The destination is deliberately defined after the original load.
			storeIR := "%dst = getelementptr i8, ptr %base, i32 8\n" +
				"store " + dstVolatile + " %Headers %value, ptr %dst, align 1"
			allocation := "call ptr @\"" + runtimeAllocU + "\"(i64 4032)"
			if tc.stack {
				allocation = "alloca %Headers, align 8"
			}
			mod := parseAggregateIR(t, strings.NewReplacer(
				"$alloc", runtimeAllocU, "$prefix", tc.prefix, "$volatile", sourceVolatile,
				"$storage", allocation,
				"$tail", strings.ReplaceAll(tc.tail, "$write", storeIR),
			).Replace(`
%Cell = type { [250 x i64], { ptr, i32 }, i64 }
%Headers = type [2 x %Cell]
declare ptr @"$alloc"(i64)
declare void @mutate(ptr)
declare void @panic()
define void @clear_headers(ptr %slot) {
  %high = getelementptr %Headers, ptr %slot, i32 0, i32 0, i32 1, i32 1
  store i32 0, ptr %high, align 1
  %second = getelementptr %Headers, ptr %slot, i32 0, i32 1, i32 1, i32 1
  store i32 0, ptr %second, align 1
  ret void
}
define void @copy(ptr %src, ptr %base, ptr %root, i1 %condition) {
entry:
  %slot = $storage
  $prefix
  %original = load $volatile %Headers, ptr %src, align 1
  store %Headers %original, ptr %slot, align 1
  call void @clear_headers(ptr %slot)
  %value = load %Headers, ptr %slot, align 1, !llgo.header.snapshot !0
  $tail
}
!0 = !{}
`))
			td := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
			defer td.Dispose()
			cfg := AggregateLoweringConfig{GoWordSize: 8, GCRoots: true, Wasm: true}
			LowerLargeAggregates(td, mod, cfg)
			LowerAggregateCopies(td, mod, cfg)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid forwarded snapshot: %v\n%s", err, mod.String())
			}
			fn := mod.NamedFunction("copy")
			var slot, clear, read, write llvm.Value
			allocs, copies := 0, 0
			for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if tc.stack && !inst.IsAAllocaInst().IsNil() && inst.Name() == "slot" {
						slot = inst
					}
					if inst.IsACallInst().IsNil() {
						continue
					}
					name := inst.CalledValue().Name()
					switch {
					case name == runtimeAllocU:
						allocs++
						slot = inst
					case name == "clear_headers":
						clear = inst
					case strings.HasPrefix(name, "llvm.memmove"), strings.HasPrefix(name, "llvm.memcpy"):
						copies++
						if inst.Operand(1) == fn.Param(0) {
							read = inst
						}
						if inst.Operand(0).Name() == "dst" {
							write = inst
						}
					}
				}
			}
			if read.IsNil() || write.IsNil() || clear.IsNil() {
				t.Fatalf("missing source read, destination write or header clear:\n%s", fn.String())
			}
			if tc.forward {
				if allocs != 0 || copies != 1 || read != write || write.InstructionParent().AsValue().Name() != "write" ||
					clear.Operand(0) != write.Operand(0) || llvm.NextInstruction(write) != clear {
					t.Fatalf("copy and clear must follow the destination check:\n%s", fn.String())
				}
			} else {
				wantAllocs, wantCopies := 1, 2
				if tc.stack {
					wantAllocs = 0
				}
				if tc.shared {
					wantCopies = 3 // Capture the value before mutable scratch is overwritten.
				}
				if allocs != wantAllocs || copies != wantCopies || read.Operand(0) != slot || clear.Operand(0) != slot ||
					(write.Operand(1) == slot) == tc.shared || (read.Operand(3).ZExtValue() != 0) != tc.volatileSource ||
					(write.Operand(3).ZExtValue() != 0) != tc.volatileDst {
					t.Fatalf("unsafe forwarding must retain one immutable snapshot:\n%s", fn.String())
				}
			}
		})
	}
}

func TestSnapshotLiveRegionLoopBoundary(t *testing.T) {
	mod := parseAggregateIR(t, `
define void @copy(ptr %src, ptr %dst, i1 %again) {
entry:
  br label %loop
loop:
  %v = load [5 x i64], ptr %src
  store [5 x i64] %v, ptr %dst
  br i1 %again, label %loop, label %exit
exit:
  ret void
}
define void @other() {
entry:
  ret void
}
`)
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()
	l := newLargeAggregateLowerer(td, AggregateLoweringConfig{GoWordSize: 8})
	fn := mod.NamedFunction("copy")
	l.ensureCFG(fn)
	loop := llvm.NextBasicBlock(fn.FirstBasicBlock())
	start := loop.FirstInstruction()
	store := llvm.NextInstruction(start)
	branch := loop.LastInstruction()

	// Repeated uses and the start boundary must not extend the old occupant
	// into the next loop iteration.
	r := snapshotRange{start: start, uses: []llvm.Value{store}}
	if l.liveAt(&r, llvm.Value{}) {
		t.Fatal("snapshot is live at a missing instruction")
	}
	if !l.liveAt(&r, store) || l.liveAt(&r, branch) {
		t.Fatal("snapshot live region extends past its final store")
	}
	if !l.isLastSnapshotUse(&r, store) {
		t.Fatal("store should end the current snapshot lifetime")
	}
	want := l.walkUses([]llvm.Value{store}, start)
	got := l.walkUses([]llvm.Value{{}, start, store, store}, start)
	if len(got) != len(want) {
		t.Fatalf("duplicate or boundary uses changed the region: got %d, want %d", len(got), len(want))
	}
	for v := range want {
		if _, ok := got[v]; !ok {
			t.Fatal("walkUses lost a live instruction")
		}
	}

	// A successor that starts a new occupant ends this occupant even when
	// the final use is the backedge itself.
	backedge := snapshotRange{start: start, uses: []llvm.Value{branch}}
	if !l.isLastSnapshotUse(&backedge, branch) {
		t.Fatal("loop re-entry kept the previous occupant alive")
	}
	if prev := l.prevInstructions(mod.NamedFunction("other").FirstBasicBlock().FirstInstruction()); len(prev) != 0 {
		t.Fatal("predecessor lookup crossed function boundaries")
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotPackedFieldAlignment(t *testing.T) {
	for _, tc := range []struct {
		name, typ, indices string
		align              int
	}{
		{"packed pointer", "<{ i8, [512 x ptr] }>", "1, 0", 1},
		{"nested packed pointer", "{ i64, <{ i8, [512 x ptr] }> }", "1, 1, 0", 1},
		{"aligned pointer", "{ i64, [512 x ptr] }", "1, 1", 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mod := parseAggregateIR(t, "declare void @mutate()\ndefine ptr @copy(ptr %src) {\n"+
				"%v = load "+tc.typ+", ptr %src\ncall void @mutate()\n"+
				"%p = extractvalue "+tc.typ+" %v, "+tc.indices+"\nret ptr %p\n}")
			td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
			defer td.Dispose()
			if got := LowerAggregateCopies(td, mod, AggregateLoweringConfig{GoWordSize: 8}); got != 1 {
				t.Fatalf("lowered %d loads, want 1", got)
			}
			fn := mod.NamedFunction("copy")
			load := fn.FirstBasicBlock().LastInstruction().Operand(0)
			if load.IsALoadInst().IsNil() || load.Alignment() != tc.align {
				t.Fatalf("projected load alignment = %d, want %d:\n%s", load.Alignment(), tc.align, fn.String())
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

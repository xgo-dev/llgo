package abi

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

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

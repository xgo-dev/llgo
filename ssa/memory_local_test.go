//go:build !llgo

package ssa

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLocalLoopAllocReservesOneSlotPerCall(t *testing.T) {
	for _, roots := range []bool{false, true} {
		prog := NewProgram(nil)
		pkg := prog.NewPackage("localalloc", "localalloc")
		fn := pkg.NewFunc("loop", NoArgsNoRet, InGo)
		b := fn.MakeBody(2)
		if roots {
			fn.NewGCRoots(1)
		}
		b.Jump(fn.Block(1))
		b.SetBlock(fn.Block(1))
		typ := prog.Type(types.NewArray(types.Typ[types.Byte], 96), InGo)
		b.Alloc(typ, false)
		b.Jump(fn.Block(1))
		b.EndBuild()
		entry := fn.impl.FirstBasicBlock()
		allocations := 0
		initializations := 0
		for block := entry; !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if call := inst.IsACallInst(); !call.IsNil() && strings.HasPrefix(call.CalledValue().Name(), "llvm.memset") {
					initializations++
					if block != fn.Block(1).first {
						t.Errorf("roots=%v: local is not zeroed on each loop iteration", roots)
					}
				}
				if inst.InstructionOpcode() != llvm.Alloca {
					continue
				}
				allocations++
				if block != entry {
					t.Errorf("roots=%v: local reserves stack space inside loop:\n%s", roots, pkg.String())
				}
			}
		}
		want := 1
		if roots {
			want++
		}
		if allocations != want {
			t.Errorf("roots=%v: got %d reservations, want %d", roots, allocations, want)
		}
		if initializations != 1 {
			t.Errorf("roots=%v: got %d zero-initializers, want 1", roots, initializations)
		}
		if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Error(err)
		}
		b.Dispose()
		prog.Dispose()
	}
}

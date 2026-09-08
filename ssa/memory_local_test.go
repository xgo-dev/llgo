//go:build !llgo

package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLocalLoopAllocReservesOneSlotPerCall(t *testing.T) {
	for _, roots := range []bool{false, true} {
		for _, debug := range []bool{false, true} {
			t.Run(fmt.Sprintf("roots=%v/debug=%v", roots, debug), func(t *testing.T) {
				prog := NewProgram(nil)
				pkg := prog.NewPackage("localalloc", "localalloc")
				if debug {
					pkg.InitDebug("localalloc", "localalloc", token.NewFileSet())
				}
				fn := pkg.NewFunc("loop", NoArgsNoRet, InGo)
				b := fn.MakeBody(2)
				if debug {
					pos := token.Position{Filename: "local.go", Line: 1, Column: 1}
					b.DebugFunction(fn, nil, pos, pos)
				}
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
								t.Errorf("local is not zeroed on each loop iteration")
							}
						}
						if inst.InstructionOpcode() != llvm.Alloca {
							continue
						}
						allocations++
						if block != entry {
							t.Errorf("local reserves stack space inside loop:\n%s", pkg.String())
						}
					}
				}
				want := 1
				if roots {
					want++
				}
				if allocations != want {
					t.Errorf("got %d reservations, want %d", allocations, want)
				}
				if initializations != 1 {
					t.Errorf("got %d zero-initializers, want 1", initializations)
				}
				if debug {
					pkg.FinalizeDebug()
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Error(err)
				}
				b.Dispose()
				prog.Dispose()
			})
		}
	}
}

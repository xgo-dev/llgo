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
				// Different sizes and alignments catch a zero-initializer aimed at
				// the wrong reservation, not just missing instruction counts.
				slots := make(map[llvm.Value]Type)
				for _, elem := range []types.Type{
					types.NewArray(types.Typ[types.Byte], 96),
					types.NewArray(types.Typ[types.Int64], 4),
				} {
					typ := prog.Type(elem, InGo)
					slot := b.Alloc(typ, false).impl
					if _, exists := slots[slot]; exists {
						t.Fatal("distinct locals share one reservation")
					}
					slots[slot] = typ
				}
				b.Jump(fn.Block(1))
				b.EndBuild()
				entry := fn.impl.FirstBasicBlock()
				allocations := 0
				initializations := make(map[llvm.Value]int)
				for block := entry; !block.IsNil(); block = llvm.NextBasicBlock(block) {
					for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
						if call := inst.IsACallInst(); !call.IsNil() && strings.HasPrefix(call.CalledValue().Name(), "llvm.memset") {
							slot := call.Operand(0)
							typ, exists := slots[slot]
							if !exists {
								t.Fatalf("initializer does not target a local reservation: %s", call.String())
							}
							initializations[slot]++
							if call.Operand(1).ZExtValue() != 0 || call.Operand(2).ZExtValue() != prog.SizeOf(typ) {
								t.Errorf("initializer must zero the complete local: %s", call.String())
							}
							if block != fn.Block(1).first {
								t.Errorf("local is not zeroed on each loop iteration")
							}
						}
						if inst.InstructionOpcode() != llvm.Alloca {
							continue
						}
						allocations++
						if typ, exists := slots[inst]; exists {
							if inst.AllocatedType() != typ.ll || inst.Alignment() < prog.TargetData().ABITypeAlignment(typ.ll) {
								t.Errorf("local reservation has the wrong type or alignment: %s", inst.String())
							}
						}
						if block != entry {
							t.Errorf("local reserves stack space inside loop:\n%s", pkg.String())
						}
					}
				}
				want := len(slots)
				if roots {
					want++
				}
				if allocations != want {
					t.Errorf("got %d reservations, want %d", allocations, want)
				}
				for slot := range slots {
					if got := initializations[slot]; got != 1 {
						t.Errorf("local %s has %d zero-initializers, want 1", slot.Name(), got)
					}
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

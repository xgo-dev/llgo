// Package callerlocation avoids redundant compiler-generated location records.
package callerlocation

import "github.com/xgo-dev/llvm"

// Location identifies a runtime update before target-specific ABI lowering.
type Location struct {
	Kind, Function, File string
	Line                 int
}

// Tracker follows append-only IR emission. Its zero value is ready to use.
// A different block, location or intervening call requires a fresh record.
type Tracker struct {
	block    llvm.BasicBlock
	cursor   llvm.Value
	location Location
	debug    llvm.DebugLoc
}

// Repeated checks only the instructions appended since the last checkpoint.
// It always advances the checkpoint to last, regardless of the result.
// Linear scanning requires a live checkpoint at or before last in this block;
// reset the tracker after IR rewrites. Reaching the block head returns false.
// Calls (including inline assembly) and terminators are conservative barriers.
func (t *Tracker) Repeated(block llvm.BasicBlock, last llvm.Value, location Location, debug llvm.DebugLoc) bool {
	previous := t.cursor
	same := !previous.IsNil() && t.block == block && t.location == location && t.debug == debug
	t.block, t.cursor, t.location, t.debug = block, last, location, debug
	if !same {
		return false
	}
	for inst := last; inst != previous; inst = llvm.PrevInstruction(inst) {
		if inst.IsNil() {
			return false
		}
		switch inst.InstructionOpcode() {
		case llvm.Call, llvm.Invoke, llvm.Ret, llvm.Br, llvm.Switch, llvm.IndirectBr,
			llvm.Unreachable, llvm.Resume, llvm.CleanupRet, llvm.CatchRet, llvm.CatchSwitch:
			return false
		}
	}
	return true
}

// Checkpoint includes the newly emitted record in the checked prefix.
func (t *Tracker) Checkpoint(last llvm.Value) {
	t.cursor = last
}

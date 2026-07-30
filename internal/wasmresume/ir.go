/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wasmresume

import (
	"fmt"

	"github.com/xgo-dev/llvm"
)

func splitBlockAfter(ctx llvm.Context, call llvm.Value, name string) (llvm.BasicBlock, error) {
	block := call.InstructionParent()
	if block.IsNil() || call.InstructionOpcode() != llvm.Call {
		return llvm.BasicBlock{}, fmt.Errorf("split point is not a call instruction")
	}
	firstMoved := llvm.NextInstruction(call)
	if firstMoved.IsNil() {
		return llvm.BasicBlock{}, fmt.Errorf("call has no continuation")
	}

	terminator := block.LastInstruction()
	successors := make([]llvm.BasicBlock, terminator.SuccessorsCount())
	for i := range successors {
		successors[i] = terminator.Successor(i)
	}

	fn := block.Parent()
	continuation := ctx.AddBasicBlock(fn, name)
	continuation.MoveAfter(block)
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(continuation)
	for instr := firstMoved; !instr.IsNil(); {
		next := llvm.NextInstruction(instr)
		instrName := instr.Name()
		instr.RemoveFromParentAsInstruction()
		if instrName == "" {
			builder.Insert(instr)
		} else {
			builder.InsertWithName(instr, instrName)
		}
		instr = next
	}
	builder.SetInsertPointAtEnd(block)
	builder.CreateBr(continuation)

	for _, successor := range successors {
		replacePhiPredecessor(ctx, successor, block, continuation)
	}
	return continuation, nil
}

func replacePhiPredecessor(ctx llvm.Context, block, old, replacement llvm.BasicBlock) {
	var phis []llvm.Value
	for phi := block.FirstInstruction(); !phi.IsNil() && phi.InstructionOpcode() == llvm.PHI; phi = llvm.NextInstruction(phi) {
		phis = append(phis, phi)
	}
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	for _, phi := range phis {
		values := make([]llvm.Value, phi.IncomingCount())
		blocks := make([]llvm.BasicBlock, len(values))
		changed := false
		for i := range values {
			values[i] = phi.IncomingValue(i)
			blocks[i] = phi.IncomingBlock(i)
			if blocks[i] == old {
				blocks[i] = replacement
				changed = true
			}
		}
		if !changed {
			continue
		}
		builder.SetInsertPointBefore(phi)
		next := builder.CreatePHI(phi.Type(), "")
		next.AddIncoming(values, blocks)
		next.InstructionSetDebugLoc(phi.InstructionDebugLoc())
		name := phi.Name()
		phi.SetName("")
		next.SetName(name)
		phi.ReplaceAllUsesWith(next)
		phi.EraseFromParentAsInstruction()
	}
}

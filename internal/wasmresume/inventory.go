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

// Package wasmresume plans the compiler half of LLGo's experimental
// WebAssembly resumable call ABI.
package wasmresume

import (
	"fmt"

	"github.com/xgo-dev/llvm"
)

const (
	FunctionAttribute    = "llgo.wasm.resume"
	CallMetadata         = "llgo.wasm.resume.call"
	SuspendSymbol        = "github.com/goplus/llgo/runtime/internal/wasmresume.SuspendCurrent"
	RegisterUnwindSymbol = "github.com/goplus/llgo/runtime/internal/wasmresume.RegisterUnwind"
	ClearUnwindSymbol    = "github.com/goplus/llgo/runtime/internal/wasmresume.ClearUnwind"
	MarkerVersion        = 1
	maxResumeID          = 1<<16 - 1
)

// Function describes the resumable calls in one generated Go function.
type Function struct {
	Name  string
	Calls []Call
}

// Call describes one generated Go call and its in-function resume ID.
type Call struct {
	ID       uint32
	Callee   string
	Indirect bool
}

// Inventory scans the actual LLVM calls produced by the frontend and assigns
// deterministic, function-local resume IDs.
func Inventory(mod llvm.Module) ([]Function, error) {
	ctx := mod.Context()
	kind := ctx.MDKindID(CallMetadata)
	var functions []Function
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		markedFunction := hasFunctionMarker(fn)
		var calls []Call
		for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				if !hasMetadata(instr, kind) {
					continue
				}
				if instr.InstructionOpcode() != llvm.Call {
					return nil, fmt.Errorf("%s: resumable marker is attached to a non-call instruction", fn.Name())
				}
				if !markedFunction {
					return nil, fmt.Errorf("%s: resumable call is in an unmarked function", fn.Name())
				}
				if err := validateCallMarker(instr.Metadata(kind)); err != nil {
					return nil, fmt.Errorf("%s: %w", fn.Name(), err)
				}
				if len(calls) == maxResumeID {
					return nil, fmt.Errorf("%s: too many resumable calls", fn.Name())
				}
				target := instr.CalledValue()
				callee := ""
				if !target.IsAFunction().IsNil() {
					callee = target.Name()
				}
				call := Call{
					ID:       uint32(len(calls) + 1),
					Callee:   callee,
					Indirect: callee == "",
				}
				calls = append(calls, call)
				setCallMarker(ctx, instr, kind, call.ID)
			}
		}
		if markedFunction {
			functions = append(functions, Function{Name: fn.Name(), Calls: calls})
		}
	}
	return functions, nil
}

func hasFunctionMarker(fn llvm.Value) bool {
	for _, attr := range fn.GetFunctionAttributes() {
		if attr.IsString() && attr.GetStringKind() == FunctionAttribute {
			return attr.GetStringValue() == "1"
		}
	}
	return false
}

func hasMetadata(instr llvm.Value, kind int) bool {
	return instr.HasMetadata() && !instr.Metadata(kind).IsNil()
}

func validateCallMarker(marker llvm.Value) error {
	fields := marker.MDNodeOperands()
	if len(fields) < 1 || len(fields) > 2 || fields[0].IsAConstantInt().IsNil() ||
		fields[0].ZExtValue() != MarkerVersion {
		return fmt.Errorf("invalid resumable call marker")
	}
	return nil
}

func setCallMarker(ctx llvm.Context, call llvm.Value, kind int, id uint32) {
	i32 := ctx.Int32Type()
	fields := []llvm.Metadata{
		llvm.ConstInt(i32, MarkerVersion, false).ConstantAsMetadata(),
		llvm.ConstInt(i32, uint64(id), false).ConstantAsMetadata(),
	}
	call.SetMetadata(kind, ctx.MDNode(fields))
}

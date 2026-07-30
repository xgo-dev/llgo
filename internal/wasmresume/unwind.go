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

func lowerUnwindMarkers(
	ctx llvm.Context, entry llvm.Value, plan framePlan, unwindField llvm.Value,
) error {
	var markers []llvm.Value
	for block := entry.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if instr.InstructionOpcode() != llvm.Call {
				continue
			}
			switch instr.CalledValue().Name() {
			case RegisterUnwindSymbol, ClearUnwindSymbol:
				markers = append(markers, instr)
			}
		}
	}
	if len(markers) == 0 {
		if plan.unwindSlot != 0 {
			return fmt.Errorf("unwind frame has no registration marker")
		}
		return nil
	}
	if plan.unwindSlot == 0 || unwindField.IsNil() {
		return fmt.Errorf("unwind marker has no frame slot")
	}

	builder := ctx.NewBuilder()
	defer builder.Dispose()
	unwindType := plan.slots[plan.unwindSlot-1].typ
	for _, marker := range markers {
		builder.SetInsertPointBefore(marker)
		value := llvm.ConstNull(unwindType)
		if marker.CalledValue().Name() == RegisterUnwindSymbol {
			if marker.OperandsCount() < 3 {
				return fmt.Errorf("invalid unwind registration marker")
			}
			value = marker.Operand(0)
		}
		builder.CreateStore(value, unwindField)
		marker.EraseFromParentAsInstruction()
	}
	return nil
}

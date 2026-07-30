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

func spillValue(ctx llvm.Context, value, field llvm.Value) error {
	if value.IsAInstruction().IsNil() {
		replaceValueUsesWithLoads(ctx, value, field, llvm.Value{})
		return nil
	}
	if !value.IsAAllocaInst().IsNil() {
		if _, dynamic := persistentSlotType(value, slotAlloca); dynamic {
			return fmt.Errorf("dynamic alloca %q requires separate frame storage", value.Name())
		}
		value.ReplaceAllUsesWith(field)
		value.EraseFromParentAsInstruction()
		return nil
	}
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	if value.InstructionOpcode() == llvm.PHI {
		next := value
		for !next.IsNil() && next.InstructionOpcode() == llvm.PHI {
			next = llvm.NextInstruction(next)
		}
		if next.IsNil() {
			return fmt.Errorf("phi %q has no insertion point", value.Name())
		}
		builder.SetInsertPointBefore(next)
	} else {
		next := llvm.NextInstruction(value)
		if next.IsNil() {
			return fmt.Errorf("value %q has no insertion point", value.Name())
		}
		builder.SetInsertPointBefore(next)
	}
	store := builder.CreateStore(value, field)
	store.InstructionSetDebugLoc(value.InstructionDebugLoc())
	replaceValueUsesWithLoads(ctx, value, field, store)
	return nil
}

func replaceValueUsesWithLoads(ctx llvm.Context, value, field, skip llvm.Value) {
	var users []llvm.Value
	seen := make(map[llvm.Value]struct{})
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		if user == skip {
			continue
		}
		if _, ok := seen[user]; !ok {
			seen[user] = struct{}{}
			users = append(users, user)
		}
	}

	builder := ctx.NewBuilder()
	defer builder.Dispose()
	for _, user := range users {
		if user.InstructionOpcode() == llvm.PHI {
			for i := 0; i < user.IncomingCount(); i++ {
				if user.IncomingValue(i) != value {
					continue
				}
				terminator := user.IncomingBlock(i).LastInstruction()
				builder.SetInsertPointBefore(terminator)
				load := builder.CreateLoad(value.Type(), field, value.Name()+".reload")
				load.InstructionSetDebugLoc(user.InstructionDebugLoc())
				user.SetOperand(i, load)
			}
			continue
		}
		builder.SetInsertPointBefore(user)
		load := builder.CreateLoad(value.Type(), field, value.Name()+".reload")
		load.InstructionSetDebugLoc(user.InstructionDebugLoc())
		for i := 0; i < user.OperandsCount(); i++ {
			if user.Operand(i) == value {
				user.SetOperand(i, load)
			}
		}
	}
}

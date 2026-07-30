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

import "github.com/xgo-dev/llvm"

type movedBlockAddress struct {
	value llvm.Value
	block llvm.BasicBlock
}

func collectMovedBlockAddresses(function llvm.Value, blocks []llvm.BasicBlock) []movedBlockAddress {
	found := make(map[llvm.Value]llvm.BasicBlock)
	seen := make(map[llvm.Value]struct{})
	var visit func(llvm.Value)
	visit = func(value llvm.Value) {
		if value.IsNil() {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		if value.IsAUser().IsNil() {
			return
		}
		if value.OperandsCount() == 2 &&
			value.Operand(0) == function &&
			value.Operand(1).IsBasicBlock() {
			found[value] = value.Operand(1).AsBasicBlock()
			return
		}
		if value.IsAConstant().IsNil() {
			return
		}
		for i := 0; i < value.OperandsCount(); i++ {
			visit(value.Operand(i))
		}
	}
	for _, block := range blocks {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			for i := 0; i < instruction.OperandsCount(); i++ {
				visit(instruction.Operand(i))
			}
		}
	}

	addresses := make([]movedBlockAddress, 0, len(found))
	for value, block := range found {
		addresses = append(addresses, movedBlockAddress{value: value, block: block})
	}
	return addresses
}

func remapMovedBlockAddresses(function llvm.Value, addresses []movedBlockAddress) {
	for _, address := range addresses {
		address.value.ReplaceAllUsesWith(llvm.BlockAddress(function, address.block))
	}
}

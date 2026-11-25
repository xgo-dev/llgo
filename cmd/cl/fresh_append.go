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

package cl

import (
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// freshNilAppendSource identifies backing storage that can become the append
// result itself. The destination must be nil and the source exclusively used
// by this append. Keep its make in place for checks and evaluation effects.
func freshNilAppendSource(call *ssa.Call) ssa.Value {
	builtin, ok := call.Call.Value.(*ssa.Builtin)
	if !ok || builtin.Name() != "append" || len(call.Call.Args) != 2 {
		return nil
	}
	destination := call.Call.Args[0]
	for {
		change, ok := destination.(*ssa.ChangeType)
		if !ok {
			break
		}
		destination = change.X
	}
	dst, ok := destination.(*ssa.Const)
	if !ok || !dst.IsNil() {
		return nil
	}
	source := call.Call.Args[1]
	current, parent := source, ssa.Instruction(call)
	for {
		if singleNonDebugReferrer(current) != parent {
			return nil
		}
		change, ok := current.(*ssa.ChangeType)
		if !ok {
			break
		}
		current, parent = change.X, change
	}
	switch src := current.(type) {
	case *ssa.MakeSlice:
		// A static single use can execute repeatedly without recreating its
		// source. Sharing a block proves that every append execution is
		// preceded by a new allocation, including when that block is a loop.
		if src.Block() != call.Block() {
			return nil
		}
		// Require SSA value identity: forward a full-capacity backing array,
		// rather than retaining capacity the copying append would discard.
		if src.Len == src.Cap {
			return source
		}
	case *ssa.Slice:
		alloc, ok := src.X.(*ssa.Alloc)
		if !ok || alloc.Comment != "makeslice" || src.Low != nil || src.Max != nil {
			return nil
		}
		if alloc.Block() != call.Block() {
			return nil
		}
		pointer, ok := alloc.Type().(*types.Pointer)
		if !ok {
			return nil
		}
		array, ok := pointer.Elem().Underlying().(*types.Array)
		if !ok {
			return nil
		}
		if src.High != nil {
			high, ok := src.High.(*ssa.Const)
			if !ok || high.Value == nil {
				return nil
			}
			if n, exact := constant.Int64Val(high.Value); !exact || n != array.Len() {
				return nil
			}
		}
		if singleNonDebugReferrer(alloc) == src {
			return source
		}
	}
	return nil
}

// The source used to be copied by append. Forwarding it instead requires its
// backing array to survive the append result; do not use a temporary stack
// buffer even when escape analysis sees only the original borrowing append.
func freshNilAppendAllocation(value ssa.Value) bool {
	// Most placement sites cannot be a fresh slice's backing allocation.
	// Reject them before inspecting referrers, including ordinary locals.
	switch allocation := value.(type) {
	case *ssa.MakeSlice:
		if allocation.Len != allocation.Cap {
			return false
		}
	case *ssa.Alloc:
		if allocation.Comment != "makeslice" {
			return false
		}
	default:
		return false
	}
	for {
		switch use := singleNonDebugReferrer(value).(type) {
		case *ssa.Slice:
			value = use
		case *ssa.ChangeType:
			value = use
		case *ssa.Call:
			return freshNilAppendSource(use) == value
		default:
			return false
		}
	}
}

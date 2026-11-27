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

// Package abipolicy defines pure-Go ABI size thresholds and layout policies
// shared between pure-Go analysis/optimization passes (such as ssaopt and
// goescape) and LLVM backend lowering (cmd/compiler/abi, llvmgen).
//
// This package intentionally has zero LLVM dependencies.
package abipolicy

import runtimeabi "github.com/xgo-dev/llgo/runtime/abi"

const (
	// SmallStackBufferSize bounds stack-allocated small slice/string buffers.
	SmallStackBufferSize = runtimeabi.SmallStackBufferSize

	// MaxEscapeStackBytes bounds storage newly promoted by Go escape analysis and
	// its lowering helpers in one function. Existing locals and later LLVM frame
	// layout remain independent; this is not a bound on the entire call stack.
	MaxEscapeStackBytes uint64 = 64 * 1024

	// MaxStackVarSize matches cmd/compile's default limit for explicitly
	// declared variables. Values larger than this must not live on LLGo's
	// fixed native stack.
	MaxStackVarSize uint64 = 128 * 1024

	// MaxImplicitStackVarSize matches cmd/compile's default limit for
	// compiler-generated temporaries.
	MaxImplicitStackVarSize uint64 = 64 * 1024

	// MinAggregateCopySize is the point at which aggregate loads and
	// stores are lowered to memory intrinsics to avoid LLVM scalarization.
	// Return types and the native stack/return ABI still use MaxImplicitStackVarSize.
	MinAggregateCopySize uint64 = 4 * 1024

	// MaxSSAValueWords matches cmd/compile's ssa.MaxStruct: a value may occupy
	// at most this many pointer-sized words and still be SSAable.
	MaxSSAValueWords = 4

	// MaxNativeRecoverStackValueSize is the upper bound for promoting a
	// recover-visible local aggregate to entry-frame stack storage on native
	// amd64 and arm64 targets.
	MaxNativeRecoverStackValueSize uint64 = 128
)

// MaxSSAValueSize is cmd/compile's CanSSA size limit (MaxStruct * PtrSize).
func MaxSSAValueSize(ptrSize int) uint64 {
	return uint64(MaxSSAValueWords * ptrSize)
}

// MaxRecoverStackValueSize returns the maximum size of a recover-visible local
// capture that may be promoted to entry-frame stack storage on goarch.
func MaxRecoverStackValueSize(goarch string, ptrSize int) uint64 {
	switch goarch {
	case "arm64", "amd64":
		return MaxNativeRecoverStackValueSize
	default:
		return MaxSSAValueSize(ptrSize)
	}
}

// ShouldLowerArrayCopy reports whether a multi-element array should be copied
// in memory instead of as a first-class LLVM value. Arrays larger than
// MaxSSAValueSize are not SSAable in cmd/compile; smaller ones stay in registers.
func ShouldLowerArrayCopy(length int, size uint64, ptrSize int) bool {
	return length > 1 && size > MaxSSAValueSize(ptrSize)
}

// ShouldSnapshotAggregateLoad reports whether ABI copy lowering may heap-allocate
// a snapshot of this load (AllocU), which the frontend must treat as a GC
// safepoint. Copies smaller than MinAggregateCopySize use a stack alloca
// instead, including CanSSA-sized arrays, so they are not safepoints.
func ShouldSnapshotAggregateLoad(isArray bool, arrayLen int, isStruct bool, size uint64, ptrSize int) bool {
	return (isArray || isStruct) && size >= MinAggregateCopySize
}

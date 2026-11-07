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
	"go/types"

	"github.com/xgo-dev/llgo/internal/goescape"
)

func signatureEscapeSummary(sig *types.Signature) goescape.Summary {
	n := sig.Params().Len()
	if sig.Recv() != nil {
		n++
	}
	s := goescape.Summary{Parameters: make([]goescape.Parameter, n)}
	for i := range s.Parameters {
		s.Parameters[i].Results = make([]goescape.Effect, sig.Results().Len())
	}
	return s
}

// instructionEscapeContract describes only the resolved lowering operations
// listed here. Unknown instructions keep the conservative external-call policy.
// Effect depths use the same storage/contents distinction as ordinary SSA
// loads and stores: atomics borrow their address but can publish stored values.
func instructionEscapeContract(kind int, sig *types.Signature) (goescape.Summary, bool) {
	s := signatureEscapeSummary(sig)
	n, results := len(s.Parameters), sig.Results().Len()
	switch kind {
	case llgoSkip:
		return s, true // lowering discards operands and returns zero values
	case llgoNoEscape:
		if noEscapeSignature(sig) {
			// Trusted runtime callers promise the identity result is borrowed
			// only for the lifetime of the original object. Hide the flow as
			// runtime.noescape/internal/abi.NoEscape do in cmd/compile.
			// The result may still be used to mutate data or invoke callbacks;
			// hiding retention must not establish a read-only/purity proof.
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Callee = 1
			return s, true
		}
	case llgoAtomicLoad:
		if n == 1 && results == 1 {
			s.Parameters[0].Results[0] = 2
			return s, true
		}
	case llgoAtomicStore, llgoAtomicStoreUnordered:
		if n == 2 && results == 0 {
			s.Parameters[0].Mutator = 1
			s.Parameters[1].Heap = 1
			return s, true
		}
	case llgoAtomicCmpXchg, llgoAtomicCmpXchgOK:
		if n == 3 && results >= 1 && results <= 2 {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Results[0] = 2
			s.Parameters[2].Heap = 1
			return s, true
		}
	case llgoAtomicAddReturnNew:
		if n == 2 && results == 1 {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Results[0] = 2
			s.Parameters[1].Heap = 1
			s.Parameters[1].Results[0] = 1
			return s, true
		}
	default:
		if kind >= llgoAtomicOpBase && kind <= llgoAtomicOpLast && n == 2 && results == 1 {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Results[0] = 2
			s.Parameters[1].Heap = 1
			return s, true
		}
	}
	return goescape.Summary{}, false
}

func noEscapeSignature(sig *types.Signature) bool {
	return sig.Recv() == nil && sig.Params().Len() == 1 && sig.Results().Len() == 1 &&
		types.Identical(sig.Params().At(0).Type(), types.Typ[types.UnsafePointer]) &&
		types.Identical(sig.Results().At(0).Type(), types.Typ[types.UnsafePointer])
}

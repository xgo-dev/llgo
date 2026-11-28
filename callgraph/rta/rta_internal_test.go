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

package rta

import (
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/callgraph"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestRTADefensiveGuards(t *testing.T) {
	prog := ssa.NewProgram(token.NewFileSet(), 0)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fn := prog.NewFunction("f", sig, "")

	r := &rta{
		result: &Result{
			CallGraph: callgraph.New(fn),
			Reachable: make(map[*ssa.Function]struct{ AddrTaken bool }),
		},
		prog: prog,
	}
	// Nil/empty guards.
	r.addEdge(fn, nil, nil, false)
	r.visitAddrTakenFunc(nil)
	r.visitDynCall(&ssa.Call{Call: ssa.CallCommon{Method: types.NewFunc(0, nil, "m", nil)}})
	r.addRuntimeType(nil)
	(&rta{result: r.result}).addInvokeEdge(&ssa.Call{}, types.Typ[types.Int])

	// Cached concreteTypeInfo lookup in interfaces().
	_ = r.interfaces(types.Typ[types.Int])
	_ = r.interfaces(types.Typ[types.Int])

	// Panic guards: nil caller in addEdge, signature with receiver, TypeParam, nil/unexpected type in forEachElement.
	assertPanics := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s did not panic", name)
			}
		}()
		f()
	}
	assertPanics("addEdge nil caller", func() {
		r.addEdge(nil, &ssa.Call{}, fn, false)
	})
	recvSig := types.NewSignatureType(types.NewVar(0, nil, "x", types.Typ[types.Int]), nil, nil, nil, nil, false)
	assertPanics("forEachElement sig with recv", func() {
		forEachElement(types.NewMethodSet, recvSig, func(types.Type, bool) bool { return false })
	})
	tp := types.NewTypeParam(types.NewTypeName(0, nil, "T", nil), types.Universe.Lookup("any").Type())
	assertPanics("forEachElement TypeParam", func() {
		forEachElement(types.NewMethodSet, tp, func(types.Type, bool) bool { return false })
	})
	assertPanics("forEachElement nil type", func() {
		forEachElement(types.NewMethodSet, nil, func(types.Type, bool) bool { return false })
	})
}

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

package rta_test

import (
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/callgraph/rta"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestRTAAnalyzeDirectClosureAndInterface(t *testing.T) {
	prog := ssa.NewProgram(token.NewFileSet(), 0)
	goPkg := types.NewPackage("example.com/rta", "rta")
	pkg := prog.CreatePackage(goPkg, nil, nil, true)

	// interface Speaker { Speak() }
	speakSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	speakMethod := types.NewFunc(0, goPkg, "Speak", speakSig)
	speakerIface := types.NewInterfaceType([]*types.Func{speakMethod}, nil)
	speakerIface.Complete()

	// type Dog struct{}; func (*Dog) Speak()
	dogName := types.NewTypeName(0, goPkg, "Dog", nil)
	dogNamed := types.NewNamed(dogName, types.NewStruct(nil, nil), nil)
	dogPtr := types.NewPointer(dogNamed)
	dogSpeakSig := types.NewSignatureType(types.NewVar(0, goPkg, "d", dogPtr), nil, nil, nil, nil, false)
	dogSpeakObj := types.NewFunc(0, goPkg, "Speak", dogSpeakSig)
	dogNamed.AddMethod(dogSpeakObj)
	dogSpeakFn := prog.NewFunction("Speak", dogSpeakSig, "")
	pkg.AddFunction(dogSpeakFn, dogSpeakObj)
	prog.RegisterMethod(dogPtr, goPkg, "Speak", dogSpeakFn)
	bSpeak := dogSpeakFn.NewBasicBlock("entry")
	bSpeak.Emit(&ssa.Return{}, nil, token.NoPos)

	// helper, closureTarget, deadFunc
	voidSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	helperFn := prog.NewFunction("helper", voidSig, "")
	pkg.AddFunction(helperFn, types.NewFunc(0, goPkg, "helper", voidSig))
	helperFn.NewBasicBlock("entry").Emit(&ssa.Return{}, nil, token.NoPos)

	cloFn := prog.NewFunction("clo", voidSig, "")
	pkg.AddFunction(cloFn, types.NewFunc(0, goPkg, "clo", voidSig))
	cloFn.NewBasicBlock("entry").Emit(&ssa.Return{}, nil, token.NoPos)

	deadFn := prog.NewFunction("dead", voidSig, "")
	pkg.AddFunction(deadFn, types.NewFunc(0, goPkg, "dead", voidSig))
	deadFn.NewBasicBlock("entry").Emit(&ssa.Return{}, nil, token.NoPos)

	// main(cb func()) calls helper(), takes clo, calls cb() dynamically, boxes *Dog into Speaker & invokes Speak().
	mainSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewParam(0, goPkg, "cb", voidSig)),
		nil, false,
	)
	mainFn := prog.NewFunction("main", mainSig, "")
	pkg.AddFunction(mainFn, types.NewFunc(0, goPkg, "main", mainSig))
	cbParam := mainFn.AddParam("cb", nil, voidSig)
	bMain := mainFn.NewBasicBlock("entry")
	bMain.Emit(&ssa.Call{Call: ssa.CallCommon{Value: helperFn}}, nil, token.Pos(1))
	mkClo := &ssa.MakeClosure{Fn: cloFn}
	bMain.Emit(mkClo, voidSig, token.Pos(2))
	bMain.Emit(&ssa.Call{Call: ssa.CallCommon{Value: cbParam}}, nil, token.Pos(3))
	allocDog := &ssa.Alloc{Heap: true}
	bMain.Emit(allocDog, dogPtr, token.Pos(4))
	mkIface := &ssa.MakeInterface{X: allocDog}
	bMain.Emit(mkIface, speakerIface, token.Pos(5))
	bMain.Emit(&ssa.Call{Call: ssa.CallCommon{Value: mkIface, Method: speakMethod}}, nil, token.Pos(6))
	bMain.Emit(&ssa.Return{}, nil, token.Pos(7))

	mainFn.Finish()
	helperFn.Finish()
	cloFn.Finish()
	dogSpeakFn.Finish()
	prog.Build()

	res := rta.Analyze([]*ssa.Function{mainFn}, true)
	if res.CallGraph == nil {
		t.Fatal("expected non-nil CallGraph")
	}
	for _, want := range []*ssa.Function{mainFn, helperFn, cloFn, dogSpeakFn} {
		if _, ok := res.Reachable[want]; !ok {
			t.Errorf("expected %s reachable", want)
		}
	}
	if _, ok := res.Reachable[deadFn]; ok {
		t.Errorf("expected dead function unreachable")
	}
	if !res.Reachable[cloFn].AddrTaken {
		t.Errorf("expected cloFn marked AddrTaken")
	}
	if res.RuntimeTypes.At(dogPtr) == nil {
		t.Errorf("expected *Dog in RuntimeTypes")
	}
}

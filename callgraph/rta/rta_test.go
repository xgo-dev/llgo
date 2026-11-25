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

func TestRTAEdgeCasesReflectAndRuntimeTypeTraversal(t *testing.T) {
	if got := rta.Analyze(nil, true); got != nil {
		t.Fatalf("Analyze(nil) = %v, want nil", got)
	}

	prog := ssa.NewProgram(token.NewFileSet(), 0)
	// Mock reflect package with Value.Call method to cover reflectValueCall edges.
	reflectPkg := types.NewPackage("reflect", "reflect")
	valName := types.NewTypeName(0, reflectPkg, "Value", nil)
	valNamed := types.NewNamed(valName, types.NewStruct(nil, nil), nil)
	reflectPkg.Scope().Insert(valName)
	valCallSig := types.NewSignatureType(types.NewVar(0, reflectPkg, "v", valNamed), nil, nil, nil, nil, false)
	valCallObj := types.NewFunc(0, reflectPkg, "Call", valCallSig)
	valNamed.AddMethod(valCallObj)
	ssaReflect := prog.CreatePackage(reflectPkg, nil, nil, true)
	valCallFn := prog.NewFunction("Call", valCallSig, "")
	ssaReflect.AddFunction(valCallFn, valCallObj)
	prog.RegisterMethod(valNamed, reflectPkg, "Call", valCallFn)
	valCallFn.NewBasicBlock("entry").Emit(&ssa.Return{}, nil, token.NoPos)
	valCallFn.Finish()

	goPkg := types.NewPackage("example.com/rta2", "rta2")
	pkg := prog.CreatePackage(goPkg, nil, nil, true)
	voidSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)

	// interface Runner { Run() }
	runMethod := types.NewFunc(0, goPkg, "Run", voidSig)
	runnerIface := types.NewInterfaceType([]*types.Func{runMethod}, nil)
	runnerIface.Complete()

	// Inner named type first reached as inaccessible (via method param), then as accessible (via struct field).
	innerName := types.NewTypeName(0, goPkg, "Inner", nil)
	innerNamed := types.NewNamed(innerName, types.NewStruct(nil, nil), nil)
	// Alias to float64 (unique base type) to exercise *types.Alias branch.
	aliasName := types.NewTypeName(0, goPkg, "MyFloat", nil)
	aliasType := types.NewAlias(aliasName, types.Typ[types.Float64])

	// Underlying struct reached first via Named.Underlying() (access=false), then directly via MakeInterface (access=true).
	sharedStruct := types.NewStruct([]*types.Var{types.NewField(0, goPkg, "V", types.Typ[types.Uint16], false)}, nil)
	wrapName := types.NewTypeName(0, goPkg, "Wrap", nil)
	wrapNamed := types.NewNamed(wrapName, sharedStruct, nil)

	// Rich struct covering slice, chan, map, array, func signature, alias, pointer, and inner struct.
	richStruct := types.NewStruct([]*types.Var{
		types.NewField(0, goPkg, "S", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(0, goPkg, "C", types.NewChan(types.SendRecv, types.Typ[types.Int]), false),
		types.NewField(0, goPkg, "M", types.NewMap(types.Typ[types.String], types.Typ[types.Bool]), false),
		types.NewField(0, goPkg, "A", types.NewArray(types.Typ[types.Byte], 4), false),
		types.NewField(0, goPkg, "F", voidSig, false),
		types.NewField(0, goPkg, "Al", aliasType, false),
		types.NewField(0, goPkg, "W", wrapNamed, false),
	}, nil)
	catName := types.NewTypeName(0, goPkg, "Cat", nil)
	catNamed := types.NewNamed(catName, richStruct, nil)
	catPtr := types.NewPointer(catNamed)

	// Exported Run(), unexported secret(), method taking Inner (inaccessible first), and generic method.
	catRunSig := types.NewSignatureType(types.NewVar(0, goPkg, "c", catPtr), nil, nil, nil, nil, false)
	catRunObj := types.NewFunc(0, goPkg, "Run", catRunSig)
	catNamed.AddMethod(catRunObj)
	catRunFn := prog.NewFunction("Run", catRunSig, "")
	pkg.AddFunction(catRunFn, catRunObj)
	prog.RegisterMethod(catPtr, goPkg, "Run", catRunFn)
	catRunFn.NewBasicBlock("entry").Emit(&ssa.Return{}, nil, token.NoPos)

	catPrivSig := types.NewSignatureType(types.NewVar(0, goPkg, "c", catPtr), nil, nil,
		types.NewTuple(types.NewParam(0, goPkg, "x", innerNamed)), nil, false)
	catPrivObj := types.NewFunc(0, goPkg, "secret", catPrivSig)
	catNamed.AddMethod(catPrivObj)

	tpObj := types.NewTypeName(0, goPkg, "T", nil)
	tp := types.NewTypeParam(tpObj, types.Universe.Lookup("any").Type())
	catGenSig := types.NewSignatureType(types.NewVar(0, goPkg, "c", catPtr), nil, []*types.TypeParam{tp}, nil, nil, false)
	catGenObj := types.NewFunc(0, goPkg, "Gen", catGenSig)
	catNamed.AddMethod(catGenObj)

	// Late address-taken function discovered AFTER dynamic call site is registered.
	lateFn := prog.NewFunction("late", voidSig, "")
	pkg.AddFunction(lateFn, types.NewFunc(0, goPkg, "late", voidSig))
	lateFn.NewBasicBlock("entry").Emit(&ssa.Return{}, nil, token.NoPos)

	// Helper discovered via invoke/edge that registers *Cat after Runner invoke site exists,
	// and takes address of lateFn after dynamic call site exists.
	initOrderSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewParam(0, goPkg, "cb", voidSig),
			types.NewParam(0, goPkg, "r", runnerIface),
		), nil, false)
	mainFn := prog.NewFunction("main", initOrderSig, "")
	pkg.AddFunction(mainFn, types.NewFunc(0, goPkg, "main", initOrderSig))
	cbParam := mainFn.AddParam("cb", nil, voidSig)
	rParam := mainFn.AddParam("r", nil, runnerIface)

	bMain := mainFn.NewBasicBlock("entry")
	// 1. Dynamic call BEFORE lateFn address is taken (hits visitAddrTakenFunc -> existing dynCallSites).
	bMain.Emit(&ssa.Call{Call: ssa.CallCommon{Value: cbParam}}, nil, token.Pos(1))
	// 2. Interface invoke BEFORE *Cat is boxed (hits addRuntimeType -> existing invokeSites & interfaces()).
	bMain.Emit(&ssa.Call{Call: ssa.CallCommon{Value: rParam, Method: runMethod}}, nil, token.Pos(2))
	// 3. Second invoke on same Runner interface (hits cached implementations()).
	bMain.Emit(&ssa.Call{Call: ssa.CallCommon{Value: rParam, Method: runMethod}}, nil, token.Pos(3))
	// 4. Take address of lateFn now.
	bMain.Emit(&ssa.MakeClosure{Fn: lateFn}, voidSig, token.Pos(4))
	// 5. Box *Cat into Runner, and box sharedStruct directly to upgrade access=false -> true.
	allocCat := &ssa.Alloc{Heap: true}
	bMain.Emit(allocCat, catPtr, token.Pos(5))
	bMain.Emit(&ssa.MakeInterface{X: allocCat}, runnerIface, token.Pos(6))
	allocStruct := &ssa.Alloc{Heap: true}
	bMain.Emit(allocStruct, types.NewPointer(sharedStruct), token.Pos(7))
	loadStruct := &ssa.UnOp{Op: token.MUL, X: allocStruct}
	bMain.Emit(loadStruct, sharedStruct, token.Pos(8))
	bMain.Emit(&ssa.MakeInterface{X: loadStruct}, types.Universe.Lookup("any").Type(), token.Pos(9))
	bMain.Emit(&ssa.Return{}, nil, token.Pos(10))

	catRunFn.Finish()
	lateFn.Finish()
	mainFn.Finish()
	prog.Build()

	// Include a nil root to exercise addReachable(nil).
	res := rta.Analyze([]*ssa.Function{mainFn, nil}, true)
	if res == nil {
		t.Fatal("Analyze returned nil")
	}
	for _, want := range []*ssa.Function{mainFn, lateFn, catRunFn} {
		if _, ok := res.Reachable[want]; !ok {
			t.Errorf("expected %s reachable", want)
		}
	}
	if !res.Reachable[lateFn].AddrTaken {
		t.Errorf("expected lateFn marked AddrTaken")
	}
	if inaccess, ok := res.RuntimeTypes.At(sharedStruct).(bool); !ok || inaccess {
		t.Errorf("expected sharedStruct runtime type upgraded to accessible, got ok=%v inaccess=%v", ok, inaccess)
	}
	if res.RuntimeTypes.At(types.Typ[types.String]) == nil || res.RuntimeTypes.At(types.Typ[types.Bool]) == nil {
		t.Errorf("expected element types from struct fields in RuntimeTypes")
	}
}

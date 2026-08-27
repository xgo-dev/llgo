/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package ssa

import (
	"go/token"
	"go/types"
	"testing"
)

func TestHasTypeParam(t *testing.T) {
	generic := newGenericNamedType("Box")
	instantiated, err := types.Instantiate(types.NewContext(), generic, []types.Type{types.Typ[types.String]}, true)
	if err != nil {
		t.Fatalf("Instantiate: %v", err)
	}
	partialArg := newTypeParam("PartialArg")
	partialInstance, err := types.Instantiate(types.NewContext(), generic, []types.Type{partialArg}, false)
	if err != nil {
		t.Fatalf("Instantiate partial: %v", err)
	}

	arrayType := func() types.Type {
		tp := newTypeParam("ArrayElem")
		return types.NewArray(tp, 3)
	}()

	chanType := types.NewChan(types.SendRecv, newTypeParam("ChanElem"))

	tupleType := func() types.Type {
		tp := newTypeParam("TupleElem")
		elem := types.NewVar(token.NoPos, nil, "v", tp)
		return types.NewTuple(elem)
	}()

	structWithParam := func() types.Type {
		tp := newTypeParam("StructElem")
		field := types.NewVar(token.NoPos, nil, "value", tp)
		return types.NewStruct([]*types.Var{field}, nil)
	}()

	signatureWithParam := func() types.Type {
		tp := newTypeParam("SigParam")
		params := types.NewTuple(types.NewVar(token.NoPos, nil, "x", tp))
		return types.NewSignatureType(nil, nil, []*types.TypeParam{tp}, params, types.NewTuple(), false)
	}()

	signatureWithResult := func() types.Type {
		tp := newTypeParam("SigResult")
		results := types.NewTuple(types.NewVar(token.NoPos, nil, "res", tp))
		return types.NewSignatureType(nil, nil, []*types.TypeParam{tp}, types.NewTuple(), results, false)
	}()
	signatureParamWithoutDecl := func() types.Type {
		tp := newTypeParam("SigParamExternal")
		params := types.NewTuple(types.NewVar(token.NoPos, nil, "x", tp))
		return types.NewSignatureType(nil, nil, nil, params, types.NewTuple(), false)
	}()
	signatureResultWithoutDecl := func() types.Type {
		tp := newTypeParam("SigResultExternal")
		results := types.NewTuple(types.NewVar(token.NoPos, nil, "res", tp))
		return types.NewSignatureType(nil, nil, nil, types.NewTuple(), results, false)
	}()

	interfaceWithParam := func() types.Type {
		tp := newTypeParam("IfaceParam")
		params := types.NewTuple(types.NewVar(token.NoPos, nil, "v", tp))
		method := types.NewFunc(token.NoPos, nil, "Do", types.NewSignatureType(nil, nil, []*types.TypeParam{tp}, params, types.NewTuple(), false))
		iface := types.NewInterfaceType([]*types.Func{method}, nil)
		iface.Complete()
		return iface
	}()

	interfaceWithEmbed := func() types.Type {
		base := interfaceWithParam
		tp := newTypeParam("EmbedParam")
		embedMethod := types.NewFunc(token.NoPos, nil, "Run", types.NewSignatureType(nil, nil, []*types.TypeParam{tp}, types.NewTuple(), types.NewTuple(), false))
		iface := types.NewInterfaceType([]*types.Func{embedMethod}, []types.Type{base})
		iface.Complete()
		return iface
	}()
	interfaceWithEmbeddedOnly := func() types.Type {
		embedded := interfaceWithParam
		iface := types.NewInterfaceType(nil, []types.Type{embedded})
		iface.Complete()
		return iface
	}()

	selfRecursive := func() types.Type {
		typeName := types.NewTypeName(token.NoPos, nil, "Node", nil)
		placeholder := types.NewStruct(nil, nil)
		named := types.NewNamed(typeName, placeholder, nil)
		field := types.NewVar(token.NoPos, nil, "next", types.NewPointer(named))
		structType := types.NewStruct([]*types.Var{field}, nil)
		named.SetUnderlying(structType)
		return named
	}()

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{"nilType", nil, false},
		{"basic", types.Typ[types.Int], false},
		{"typeParam", newTypeParam("T"), true},
		{"pointerToTypeParam", types.NewPointer(newTypeParam("PtrT")), true},
		{"sliceOfTypeParam", types.NewSlice(newTypeParam("SliceT")), true},
		{"arrayOfTypeParam", arrayType, true},
		{"mapWithTypeParam", types.NewMap(newTypeParam("MapKey"), types.Typ[types.String]), true},
		{"chanOfTypeParam", chanType, true},
		{"tupleWithTypeParam", tupleType, true},
		{"structWithTypeParam", structWithParam, true},
		{"signatureWithTypeParam", signatureWithParam, true},
		{"signatureWithResultTypeParam", signatureWithResult, true},
		{"signatureParamWithoutDecl", signatureParamWithoutDecl, true},
		{"signatureResultWithoutDecl", signatureResultWithoutDecl, true},
		{"interfaceWithTypeParam", interfaceWithParam, true},
		{"interfaceWithEmbeddedTypeParam", interfaceWithEmbed, true},
		{"interfaceWithEmbeddedOnlyTypeParam", interfaceWithEmbeddedOnly, true},
		{"namedGeneric", generic, true},
		{"pointerToNamedGeneric", types.NewPointer(generic), true},
		{"namedInstanceWithTypeParamArg", partialInstance, true},
		{"namedInstanceNoParam", instantiated, false},
		{"selfRecursiveStruct", selfRecursive, false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := hasTypeParam(tc.typ); got != tc.want {
				t.Fatalf("hasTypeParam(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func newTypeParam(name string) *types.TypeParam {
	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()
	return types.NewTypeParam(types.NewTypeName(token.NoPos, nil, name, nil), iface)
}

func newGenericNamedType(name string) *types.Named {
	tp := newTypeParam("T")
	field := types.NewVar(token.NoPos, nil, "value", tp)
	structType := types.NewStruct([]*types.Var{field}, nil)
	named := types.NewNamed(types.NewTypeName(token.NoPos, nil, name, nil), structType, nil)
	named.SetTypeParams([]*types.TypeParam{tp})
	return named
}

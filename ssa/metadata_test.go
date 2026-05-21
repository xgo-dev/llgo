//go:build !llgo
// +build !llgo

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

package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/goplus/llgo/internal/metadata"
)

func newMetadataTestBuilder(t *testing.T) (Builder, *metadata.Builder) {
	t.Helper()

	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("foo", "foo")
	mb := metadata.NewBuilder()
	pkg.MetaBuilder = mb
	fn := pkg.NewFunc("foo.main", NoArgsNoRet, InGo)
	return fn.MakeBody(1), mb
}

func TestRecordInterfaceUseSkipsInterfaceTypes(t *testing.T) {
	b, mb := newMetadataTestBuilder(t)

	typePkg := types.NewPackage("foo", "foo")
	concrete := types.NewNamed(types.NewTypeName(token.NoPos, typePkg, "T", nil), types.NewStruct(nil, nil), nil)
	iface := types.NewNamed(types.NewTypeName(token.NoPos, typePkg, "I", nil), types.NewInterfaceType(nil, nil).Complete(), nil)

	b.recordInterfaceUse(concrete)
	b.recordInterfaceUse(iface)

	meta := mb.Build()
	got := make(map[string][]string)
	meta.ForEachUseIface(func(owner metadata.Symbol, typs []metadata.Symbol) {
		for _, typ := range typs {
			got[meta.SymbolName(owner)] = append(got[meta.SymbolName(owner)], meta.SymbolName(typ))
		}
	})

	want := map[string][]string{
		"foo.main": {"_llgo_foo.T"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UseIface = %#v, want %#v", got, want)
	}
}

func TestRecordInterfaceUseSkipsWhenMetaBuilderDisabled(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("foo.main", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)

	b.recordInterfaceUse(types.Typ[types.Int])
}

func TestDirectTypeChildrenRecordsSemanticTypeEdges(t *testing.T) {
	b, _ := newMetadataTestBuilder(t)
	typePkg := types.NewPackage("foo", "foo")
	namedElem := types.NewNamed(types.NewTypeName(token.NoPos, typePkg, "Elem", nil), types.NewStruct(nil, nil), nil)
	methodSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	methodTypeName, _ := b.Prog.abi.TypeName(funcType(b.Prog, methodSig))

	cases := []struct {
		name string
		typ  types.Type
		want []string
	}{
		{
			name: "pointer",
			typ:  types.NewPointer(namedElem),
			want: []string{"_llgo_foo.Elem"},
		},
		{
			name: "chan",
			typ:  types.NewChan(types.SendRecv, types.Typ[types.Int]),
			want: []string{"_llgo_int"},
		},
		{
			name: "slice",
			typ:  types.NewSlice(types.Typ[types.String]),
			want: []string{"_llgo_string"},
		},
		{
			name: "array",
			typ:  types.NewArray(types.Typ[types.Int], 3),
			want: []string{"_llgo_int", "[]_llgo_int"},
		},
		{
			name: "map",
			typ:  types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
			want: []string{"_llgo_string", "_llgo_int"},
		},
		{
			name: "signature",
			typ: types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.String])),
				types.NewTuple(types.NewVar(token.NoPos, nil, "ok", types.Typ[types.Bool])),
				false),
			want: []string{"_llgo_string", "_llgo_bool"},
		},
		{
			name: "struct",
			typ: types.NewStruct([]*types.Var{
				types.NewVar(token.NoPos, nil, "S", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "E", namedElem),
			}, nil),
			want: []string{"_llgo_string", "_llgo_foo.Elem"},
		},
		{
			name: "interface",
			typ: types.NewInterfaceType([]*types.Func{
				types.NewFunc(token.NoPos, typePkg, "M", methodSig),
			}, nil).Complete(),
			want: []string{methodTypeName},
		},
		{
			name: "basic",
			typ:  types.Typ[types.Bool],
			want: []string{},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gotTypes := b.directTypeChildren(tt.typ)
			got := typeNames(t, b, gotTypes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("directTypeChildren(%s) = %#v, want %#v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRecordTypeChildrenStoresChildSymbols(t *testing.T) {
	b, mb := newMetadataTestBuilder(t)

	b.recordTypeChildren("_llgo_foo.S", types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "N", types.Typ[types.Int]),
		types.NewVar(token.NoPos, nil, "S", types.Typ[types.String]),
	}, nil))

	meta := mb.Build()
	got := make(map[string][]string)
	meta.ForEachTypeChild(func(parent metadata.Symbol, children []metadata.Symbol) {
		for _, child := range children {
			got[meta.SymbolName(parent)] = append(got[meta.SymbolName(parent)], meta.SymbolName(child))
		}
	})

	want := map[string][]string{
		"_llgo_foo.S": {"_llgo_int", "_llgo_string"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TypeChildren = %#v, want %#v", got, want)
	}
}

func TestRecordTypeChildrenSkipsWhenMetaBuilderDisabled(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("foo.main", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)

	b.recordTypeChildren("_llgo_foo.S", types.Typ[types.Int])
}

func TestDirectTypeChildrenDoesNotRecordMapBucket(t *testing.T) {
	b, _ := newMetadataTestBuilder(t)
	mapTyp := types.NewMap(types.Typ[types.String], types.Typ[types.Int])

	got := typeNames(t, b, b.directTypeChildren(mapTyp))
	want := []string{"_llgo_string", "_llgo_int"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directTypeChildren(map) = %#v, want only semantic key/elem edges %#v", got, want)
	}
}

func TestDirectTypeChildrenFallsBackForUnknownType(t *testing.T) {
	b, _ := newMetadataTestBuilder(t)

	if got := b.directTypeChildren(fakeType{}); got != nil {
		t.Fatalf("directTypeChildren(fakeType) = %#v, want nil", got)
	}
}

func TestAppendTupleTypeChildrenAllowsNilTuple(t *testing.T) {
	existing := []types.Type{types.Typ[types.Int]}

	got := appendTupleTypeChildren(existing, nil)
	if !reflect.DeepEqual(got, existing) {
		t.Fatalf("appendTupleTypeChildren(existing, nil) = %#v, want %#v", got, existing)
	}
}

func TestDirectTypeChildrenExpandsNamedUnderlyingType(t *testing.T) {
	b, _ := newMetadataTestBuilder(t)
	typePkg := types.NewPackage("foo", "foo")
	named := types.NewNamed(
		types.NewTypeName(token.NoPos, typePkg, "S", nil),
		types.NewStruct([]*types.Var{
			types.NewVar(token.NoPos, nil, "N", types.Typ[types.Int]),
		}, nil),
		nil,
	)

	got := typeNames(t, b, b.directTypeChildren(named))
	want := []string{"_llgo_int"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directTypeChildren(named) = %#v, want %#v", got, want)
	}
}

type fakeType struct{}

func (fakeType) Underlying() types.Type { return fakeType{} }
func (fakeType) String() string         { return "fake" }

func TestRecordInterfaceMethodCallRecordsDemandAndInterfaceInfo(t *testing.T) {
	b, mb := newMetadataTestBuilder(t)
	typePkg := types.NewPackage("foo", "foo")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	method := types.NewFunc(token.NoPos, typePkg, "M", sig)
	rawIntf := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	intfExpr := Expr{Type: b.Prog.Type(rawIntf, InGo)}

	b.recordInterfaceMethodCall(intfExpr, rawIntf, method)

	meta := mb.Build()
	gotDemands := make(map[string][]string)
	meta.ForEachUseIfaceMethod(func(owner metadata.Symbol, demands []metadata.IfaceMethodDemand) {
		for _, demand := range demands {
			gotDemands[meta.SymbolName(owner)] = append(gotDemands[meta.SymbolName(owner)],
				meta.SymbolName(demand.Target)+"."+meta.Name(demand.Sig.Name)+":"+meta.SymbolName(demand.Sig.MType))
		}
	})
	gotIfaceInfo := make(map[string][]string)
	meta.ForEachInterface(func(iface metadata.Symbol, methods []metadata.MethodSig) {
		for _, method := range methods {
			gotIfaceInfo[meta.SymbolName(iface)] = append(gotIfaceInfo[meta.SymbolName(iface)],
				meta.Name(method.Name)+":"+meta.SymbolName(method.MType))
		}
	})

	wantIface, _ := b.Prog.abi.TypeName(rawIntf)
	wantMType, _ := b.Prog.abi.TypeName(funcType(b.Prog, sig))
	wantDemands := map[string][]string{
		"foo.main": {wantIface + ".M:" + wantMType},
	}
	wantIfaceInfo := map[string][]string{
		wantIface: {"M:" + wantMType},
	}
	if !reflect.DeepEqual(gotDemands, wantDemands) {
		t.Fatalf("UseIfaceMethod = %#v, want %#v", gotDemands, wantDemands)
	}
	if !reflect.DeepEqual(gotIfaceInfo, wantIfaceInfo) {
		t.Fatalf("InterfaceInfo = %#v, want %#v", gotIfaceInfo, wantIfaceInfo)
	}
}

func TestRecordInterfaceMethodCallSkipsWhenMetaBuilderDisabled(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("foo.main", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)
	method := types.NewFunc(token.NoPos, nil, "M", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	rawIntf := types.NewInterfaceType([]*types.Func{method}, nil).Complete()

	b.recordInterfaceMethodCall(Expr{}, rawIntf, method)
}

func TestMethodInfoRecorderRecordsMethodSlots(t *testing.T) {
	b, mb := newMetadataTestBuilder(t)
	typePkg := types.NewPackage("foo", "foo")
	recvType := types.NewNamed(types.NewTypeName(token.NoPos, typePkg, "T", nil), types.NewStruct(nil, nil), nil)
	recv := types.NewVar(token.NoPos, typePkg, "", recvType)
	sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
	method := types.NewFunc(token.NoPos, typePkg, "m", sig)
	ftyp := funcType(b.Prog, sig)
	ifn := b.Pkg.NewFunc("foo.(*T).M", sig, InGo).impl
	tfn := b.Pkg.NewFunc("foo.T.M", sig, InGo).impl

	recorder := b.newMethodInfoRecorder(recvType, 1)
	recorder.add(method, ftyp, ifn, tfn)
	recorder.finish()

	meta := mb.Build()
	got := make(map[string][]string)
	meta.ForEachMethodInfo(func(typ metadata.Symbol, slots []metadata.MethodSlot) {
		for _, slot := range slots {
			got[meta.SymbolName(typ)] = append(got[meta.SymbolName(typ)],
				meta.Name(slot.Sig.Name)+":"+meta.SymbolName(slot.Sig.MType)+":"+meta.SymbolName(slot.IFn)+":"+meta.SymbolName(slot.TFn))
		}
	})

	wantMType, _ := b.Prog.abi.TypeName(ftyp)
	want := map[string][]string{
		"_llgo_foo.T": {"foo.m:" + wantMType + ":foo.(*T).M:foo.T.M"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MethodInfo = %#v, want %#v", got, want)
	}
}

func TestMethodInfoRecorderSkipsWhenMetaBuilderDisabled(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("foo.main", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)
	recorder := b.newMethodInfoRecorder(types.Typ[types.Int], 1)

	if recorder != nil {
		t.Fatalf("newMethodInfoRecorder without MetaBuilder = %#v, want nil", recorder)
	}
}

func typeNames(t *testing.T, b Builder, typs []types.Type) []string {
	t.Helper()

	names := make([]string, 0, len(typs))
	for _, typ := range typs {
		name, _ := b.Prog.abi.TypeName(typ)
		names = append(names, name)
	}
	return names
}

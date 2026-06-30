//go:build !llgo
// +build !llgo

package abi

import (
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	rabi "github.com/goplus/llgo/runtime/abi"
)

func newCoverageBuilder() *Builder {
	return New(unsafe.Sizeof(uintptr(0)), types.SizesFor("gc", runtime.GOARCH))
}

func TestUnderlyingKindAndChanDirCoverage(t *testing.T) {
	sig := types.NewSignature(nil, nil, nil, false)
	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()
	st := types.NewStruct(nil, nil)
	cases := []struct {
		typ  types.Type
		want rabi.Kind
	}{
		{types.Typ[types.Int], rabi.Int},
		{types.NewPointer(types.Typ[types.Int]), rabi.Pointer},
		{types.NewSlice(types.Typ[types.Int]), rabi.Slice},
		{sig, rabi.Func},
		{iface, rabi.Interface},
		{st, rabi.Struct},
		{types.NewMap(types.Typ[types.String], types.Typ[types.Int]), rabi.Map},
		{types.NewArray(types.Typ[types.Int], 2), rabi.Array},
		{types.NewChan(types.SendRecv, types.Typ[types.Int]), rabi.Chan},
	}
	for _, c := range cases {
		if got := UnderlyingKind(c.typ); got != c.want {
			t.Fatalf("UnderlyingKind(%T)=%v, want %v", c.typ, got, c.want)
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("UnderlyingKind should panic for unsupported type")
			}
		}()
		_ = UnderlyingKind(types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])))
	}()

	if d, s := ChanDir(types.SendRecv); d != rabi.BothDir || s != "chan" {
		t.Fatalf("ChanDir(SendRecv)=(%v,%q), want (%v,%q)", d, s, rabi.BothDir, "chan")
	}
	if d, s := ChanDir(types.SendOnly); d != rabi.SendDir || s != "chan<-" {
		t.Fatalf("ChanDir(SendOnly)=(%v,%q), want (%v,%q)", d, s, rabi.SendDir, "chan<-")
	}
	if d, s := ChanDir(types.RecvOnly); d != rabi.RecvDir || s != "<-chan" {
		t.Fatalf("ChanDir(RecvOnly)=(%v,%q), want (%v,%q)", d, s, rabi.RecvDir, "<-chan")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("ChanDir should panic on invalid direction")
			}
		}()
		_, _ = ChanDir(types.ChanDir(99))
	}()
}

func TestDataKindOfCoverage(t *testing.T) {
	assertDataKind := func(raw types.Type, lvl int, is32 bool, wantKind DataKind, wantType types.Type, wantLvl int) {
		t.Helper()
		gotKind, gotType, gotLvl := DataKindOf(raw, lvl, is32)
		if gotKind != wantKind || gotType != wantType || gotLvl != wantLvl {
			t.Fatalf("DataKindOf(%v,lvl=%d,is32=%v)=(%v,%v,%d), want (%v,%v,%d)",
				raw, lvl, is32, gotKind, gotType, gotLvl, wantKind, wantType, wantLvl)
		}
	}

	assertDataKind(types.Typ[types.Int], 0, false, Integer, types.Typ[types.Int], 0)
	assertDataKind(types.Typ[types.Int64], 0, true, Indirect, types.Typ[types.Int64], 0)
	assertDataKind(types.Typ[types.Float32], 0, false, BitCast, types.Typ[types.Float32], 0)
	assertDataKind(types.Typ[types.Float64], 0, true, Indirect, types.Typ[types.Float64], 0)
	assertDataKind(types.Typ[types.Float64], 0, false, BitCast, types.Typ[types.Float64], 0)
	assertDataKind(types.Typ[types.UnsafePointer], 0, false, Pointer, types.Typ[types.UnsafePointer], 0)
	ptrType := types.NewPointer(types.Typ[types.Int])
	assertDataKind(ptrType, 0, false, Pointer, ptrType, 0)
	sigType := types.NewSignature(nil, nil, nil, false)
	assertDataKind(sigType, 0, false, Pointer, sigType, 0)
	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	assertDataKind(mapType, 0, false, Pointer, mapType, 0)
	chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])
	assertDataKind(chanType, 0, false, Pointer, chanType, 0)

	singleFieldStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "X", types.Typ[types.Int], false),
	}, nil)
	assertDataKind(singleFieldStruct, 0, false, Integer, types.Typ[types.Int], 1)

	multiFieldStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "X", types.Typ[types.Int], false),
		types.NewField(token.NoPos, nil, "Y", types.Typ[types.Int], false),
	}, nil)
	assertDataKind(multiFieldStruct, 0, false, Indirect, multiFieldStruct, 0)

	arrayLen1 := types.NewArray(types.Typ[types.Float32], 1)
	assertDataKind(arrayLen1, 0, false, BitCast, types.Typ[types.Float32], 1)
	arrayLen2 := types.NewArray(types.Typ[types.Int], 2)
	assertDataKind(arrayLen2, 0, false, Indirect, arrayLen2, 0)

	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()
	assertDataKind(iface, 0, false, Indirect, iface, 0)
	slice := types.NewSlice(types.Typ[types.Int])
	assertDataKind(slice, 0, false, Indirect, slice, 0)

	defer func() {
		if recover() == nil {
			t.Fatalf("DataKindOf should panic on unsupported underlying type")
		}
	}()
	_, _, _ = DataKindOf(types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), 0, false)
}

func TestTypeNameAndHashingCoverage(t *testing.T) {
	b := newCoverageBuilder()
	pkg := types.NewPackage("example.com/p", "p")

	if got, pub := b.TypeName(types.Typ[types.Int]); got != "_llgo_int" || !pub {
		t.Fatalf("TypeName(int)=(%q,%v), want (%q,%v)", got, pub, "_llgo_int", true)
	}
	if got, pub := b.TypeName(types.NewPointer(types.Typ[types.Int])); got != "*_llgo_int" || !pub {
		t.Fatalf("TypeName(*int)=(%q,%v), want (%q,%v)", got, pub, "*_llgo_int", true)
	}
	if got, pub := b.TypeName(types.NewSlice(types.Typ[types.Int])); got != "[]_llgo_int" || !pub {
		t.Fatalf("TypeName([]int)=(%q,%v), want (%q,%v)", got, pub, "[]_llgo_int", true)
	}
	if got, pub := b.TypeName(types.NewArray(types.Typ[types.Int], 3)); got != "[3]_llgo_int" || !pub {
		t.Fatalf("TypeName([3]int)=(%q,%v), want (%q,%v)", got, pub, "[3]_llgo_int", true)
	}
	if got, pub := b.TypeName(types.NewMap(types.Typ[types.String], types.Typ[types.Int])); got != "map[_llgo_string]_llgo_int" || !pub {
		t.Fatalf("TypeName(map[string]int)=(%q,%v), want (%q,%v)", got, pub, "map[_llgo_string]_llgo_int", true)
	}
	if got, pub := b.TypeName(types.NewChan(types.SendOnly, types.Typ[types.Int])); got != "chan<- _llgo_int" || !pub {
		t.Fatalf("TypeName(chan<-int)=(%q,%v), want (%q,%v)", got, pub, "chan<- _llgo_int", true)
	}
	if got, pub := b.TypeName(types.NewChan(types.RecvOnly, types.Typ[types.Int])); got != "<-chan _llgo_int" || !pub {
		t.Fatalf("TypeName(<-chan int)=(%q,%v), want (%q,%v)", got, pub, "<-chan _llgo_int", true)
	}
	if got, pub := b.TypeName(types.NewChan(types.SendRecv, types.Typ[types.Int])); got != "chan _llgo_int" || !pub {
		t.Fatalf("TypeName(chan int)=(%q,%v), want (%q,%v)", got, pub, "chan _llgo_int", true)
	}

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	if got, pub := b.TypeName(emptyIface); got != "_llgo_any" || !pub {
		t.Fatalf("TypeName(any)=(%q,%v), want (%q,%v)", got, pub, "_llgo_any", true)
	}

	methodSig := types.NewSignature(nil, nil, nil, false)
	expIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "M", methodSig),
	}, nil)
	expIface.Complete()
	if got, pub := b.TypeName(expIface); !strings.HasPrefix(got, "_llgo_iface$") || !pub {
		t.Fatalf("TypeName(exported iface)=(%q,%v), want _llgo_iface$*,true", got, pub)
	}

	localStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "x", types.Typ[types.Int], false),
	}, nil)
	if got, pub := b.TypeName(localStruct); !strings.HasPrefix(got, pkg.Path()+".struct$") || pub {
		t.Fatalf("TypeName(local struct)=(%q,%v), want %q*,false", got, pub, pkg.Path()+".struct$")
	}

	obj := types.NewTypeName(token.NoPos, pkg, "T", nil)
	pkg.Scope().Insert(obj)
	named := types.NewNamed(obj, types.Typ[types.Int], nil)
	if got, pub := b.TypeName(named); !strings.HasPrefix(got, "_llgo_example.com/p.T") || !pub {
		t.Fatalf("TypeName(named exported)=(%q,%v), want _llgo_example.com/p.T*,true", got, pub)
	}

	obj2 := types.NewTypeName(token.NoPos, pkg, "t", nil)
	pkg.Scope().Insert(obj2)
	named2 := types.NewNamed(obj2, types.Typ[types.Int], nil)
	if got, pub := b.TypeName(named2); !strings.HasPrefix(got, "_llgo_example.com/p.t") || pub {
		t.Fatalf("TypeName(named unexported)=(%q,%v), want _llgo_example.com/p.t*,false", got, pub)
	}

	aliasObj := types.NewTypeName(token.NoPos, pkg, "AliasInt", nil)
	alias := types.NewAlias(aliasObj, types.Typ[types.Int])
	if got, pub := b.TypeName(alias); got != "_llgo_int" || !pub {
		t.Fatalf("TypeName(alias int)=(%q,%v), want (%q,%v)", got, pub, "_llgo_int", true)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("TypeName should panic on unsupported type")
			}
		}()
		_, _ = b.TypeName(types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])))
	}()

	sig1 := types.NewSignature(nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), nil, false)
	sig2 := types.NewSignature(nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Uint])), nil, false)
	name1, name2 := b.FuncName(sig1), b.FuncName(sig2)
	if !strings.HasPrefix(name1, "_llgo_func$") || !strings.HasPrefix(name2, "_llgo_func$") || name1 == name2 {
		t.Fatalf("FuncName mismatch: %q vs %q", name1, name2)
	}
}

func TestStructInterfaceClosureAndPathCoverage(t *testing.T) {
	b := newCoverageBuilder()
	pkg := types.NewPackage("example.com/q", "q")
	sig := types.NewSignature(nil, nil, nil, false)

	closureStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "$f", sig, false),
		types.NewField(token.NoPos, nil, "$data", types.Typ[types.UnsafePointer], false),
	}, nil)
	if got, pub := b.StructName(closureStruct); !strings.HasPrefix(got, "_llgo_closure$") || !pub {
		t.Fatalf("StructName(closure)=(%q,%v), want _llgo_closure$*,true", got, pub)
	}
	if !IsClosure(closureStruct) {
		t.Fatalf("IsClosure false, want true")
	}
	if !IsClosureFields([]*types.Var{
		types.NewVar(token.NoPos, nil, "$f", sig),
		types.NewVar(token.NoPos, nil, "$data", types.Typ[types.UnsafePointer]),
	}) {
		t.Fatalf("IsClosureFields false, want true")
	}
	if IsClosureFields([]*types.Var{
		types.NewVar(token.NoPos, nil, "$f", sig),
	}) {
		t.Fatalf("IsClosureFields true, want false")
	}

	pubStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", types.Typ[types.Int], false),
	}, nil)
	if got, pub := b.StructName(pubStruct); !strings.HasPrefix(got, "_llgo_struct$") || pub {
		t.Fatalf("StructName(public struct)=(%q,%v), want _llgo_struct$*,false", got, pub)
	}
	taggedStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", types.Typ[types.Int], false),
	}, []string{`json:"x"`})
	plainName, _ := b.StructName(pubStruct)
	taggedName, _ := b.StructName(taggedStruct)
	if plainName == taggedName {
		t.Fatalf("StructName should distinguish struct tags")
	}

	privStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "x", types.Typ[types.Int], false),
	}, nil)
	if got, pub := b.StructName(privStruct); !strings.HasPrefix(got, pkg.Path()+".struct$") || pub {
		t.Fatalf("StructName(private struct)=(%q,%v), want %q*,false", got, pub, pkg.Path()+".struct$")
	}

	embedded := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "", types.Typ[types.Int], true),
	}, nil)
	if got, _ := b.StructName(embedded); got == "" {
		t.Fatalf("StructName(embedded) should not be empty")
	}

	localObj := types.NewTypeName(token.NoPos, pkg, "local", nil)
	pkg.Scope().Insert(localObj)
	localNamed := types.NewNamed(localObj, types.Typ[types.Int], nil)
	structWithPrivateType := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", localNamed, false),
	}, nil)
	if got, pub := b.StructName(structWithPrivateType); !strings.HasPrefix(got, "_llgo_struct$") || pub {
		t.Fatalf("StructName(private type field)=(%q,%v), want %q*,false", got, pub, "llgo_struct$")
	}

	privIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "m", sig),
	}, nil)
	privIface.Complete()
	if got, pub := b.InterfaceName(privIface); !strings.HasPrefix(got, pkg.Path()+".iface$") || pub {
		t.Fatalf("InterfaceName(private iface)=(%q,%v), want %q*,false", got, pub, pkg.Path()+".iface$")
	}

	realReflectlite := types.NewPackage("internal/reflectlite", "reflectlite")
	patchReflectlite := types.NewPackage(PatchPathPrefix+"internal/reflectlite", "reflectlite")
	realIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, realReflectlite, "common", sig),
	}, nil)
	realIface.Complete()
	patchIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, patchReflectlite, "common", sig),
	}, nil)
	patchIface.Complete()
	realName, _ := b.InterfaceName(realIface)
	patchName, _ := b.InterfaceName(patchIface)
	if realName != patchName {
		t.Fatalf("InterfaceName should canonicalize patch package path: real=%q patch=%q", realName, patchName)
	}

	if got := PathOf(nil); got != "" {
		t.Fatalf("PathOf(nil)=%q, want empty", got)
	}
	patchPkg := types.NewPackage(PatchPathPrefix+"runtime", "runtime")
	if got := PathOf(patchPkg); got != "runtime" {
		t.Fatalf("PathOf(patchPkg)=%q, want %q", got, "runtime")
	}
	if got := FullName(nil, "X"); got != "X" {
		t.Fatalf("FullName(nil,\"X\")=%q, want %q", got, "X")
	}
	if got := FullName(patchPkg, "X"); got != "runtime.X" {
		t.Fatalf("FullName(patchPkg,\"X\")=%q, want %q", got, "runtime.X")
	}
}

func TestScopeIndicesAndTypeArgFallbackCoverage(t *testing.T) {
	pkg := types.NewPackage("example.com/scope", "scope")
	root := pkg.Scope()
	child := types.NewScope(root, token.NoPos, token.NoPos, "child")
	leaf := types.NewScope(child, token.NoPos, token.NoPos, "leaf")

	topObj := types.NewTypeName(token.NoPos, pkg, "Top", nil)
	root.Insert(topObj)
	if got := scopeIndices(topObj); got != "" {
		t.Fatalf("scopeIndices(top)=%q, want empty", got)
	}

	localObj := types.NewTypeName(token.NoPos, pkg, "Local", nil)
	leaf.Insert(localObj)
	if got := scopeIndices(localObj); !strings.HasPrefix(got, ".") {
		t.Fatalf("scopeIndices(local)=%q, want dot-prefix", got)
	}

	sig := types.NewSignature(nil, nil, nil, false)
	if got := TypeArgs([]types.Type{sig}); !strings.HasPrefix(got, "[func(") {
		t.Fatalf("TypeArgs(signature)=%q, want [func(...)]", got)
	}

	aliasPkg := types.NewPackage("example.com/alias", "alias")
	aliasObj := types.NewTypeName(token.NoPos, aliasPkg, "AliasInt", nil)
	aliasPkg.Scope().Insert(aliasObj)
	alias := types.NewAlias(aliasObj, types.Typ[types.Int])
	if got := TypeArgs([]types.Type{alias}); got != "[int]" {
		t.Fatalf("TypeArgs(alias)=%q, want %q", got, "[int]")
	}

	tpObj := types.NewTypeName(token.NoPos, aliasPkg, "T", nil)
	tp := types.NewTypeParam(tpObj, types.Universe.Lookup("any").Type().Underlying().(*types.Interface))
	genObj := types.NewTypeName(token.NoPos, aliasPkg, "Vec", nil)
	aliasPkg.Scope().Insert(genObj)
	genAlias := types.NewAlias(genObj, types.NewSlice(tp))
	genAlias.SetTypeParams([]*types.TypeParam{tp})
	inst, err := types.Instantiate(types.NewContext(), genAlias, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatalf("Instantiate generic alias failed: %v", err)
	}
	got := TypeArgs([]types.Type{inst})
	if got != "[[]int]" {
		t.Fatalf("TypeArgs(generic alias instance)=%q, want %q", got, "[[]int]")
	}
}

func TestNamedLikeTypeArgStringCoverage(t *testing.T) {
	// pkg=nil and targs=nil
	plainObj := types.NewTypeName(token.NoPos, nil, "Plain", nil)
	if got := namedLikeTypeArgString(plainObj, nil); got != "Plain" {
		t.Fatalf("namedLikeTypeArgString(pkg=nil,targs=nil)=%q, want %q", got, "Plain")
	}

	pkg := types.NewPackage("example.com/namedlike", "namedlike")
	root := pkg.Scope()
	child := types.NewScope(root, token.NoPos, token.NoPos, "child")
	localObj := types.NewTypeName(token.NoPos, pkg, "Local", nil)
	child.Insert(localObj)

	// pkg!=nil and targs=nil
	if got := namedLikeTypeArgString(localObj, nil); !strings.HasPrefix(got, "example.com/namedlike.Local.") {
		t.Fatalf("namedLikeTypeArgString(local,nil)=%q, want prefix %q", got, "example.com/namedlike.Local.")
	}

	// pkg!=nil and targs!=nil
	tpObj := types.NewTypeName(token.NoPos, pkg, "T", nil)
	tp := types.NewTypeParam(tpObj, types.Universe.Lookup("any").Type().Underlying().(*types.Interface))
	genObj := types.NewTypeName(token.NoPos, pkg, "Box", nil)
	root.Insert(genObj)
	gen := types.NewNamed(genObj, types.NewStruct(nil, nil), nil)
	gen.SetTypeParams([]*types.TypeParam{tp})
	inst, err := types.Instantiate(types.NewContext(), gen, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatalf("Instantiate(Box[int]) failed: %v", err)
	}
	got := namedLikeTypeArgString(localObj, inst.(*types.Named).TypeArgs())
	if !strings.HasPrefix(got, "example.com/namedlike.Local[int].") {
		t.Fatalf("namedLikeTypeArgString(local,[int])=%q, want prefix %q", got, "example.com/namedlike.Local[int].")
	}
}

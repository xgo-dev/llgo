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
)

func testLocalNamed() *types.Named {
	obj := types.NewTypeName(token.NoPos, nil, "Local", nil)
	return types.NewNamed(obj, types.Typ[types.Int], nil)
}

func testGenericNamed(pkg *types.Package, name string) *types.Named {
	tpObj := types.NewTypeName(token.NoPos, pkg, "T", nil)
	anyIface := types.Universe.Lookup("any").Type().Underlying().(*types.Interface)
	tp := types.NewTypeParam(tpObj, anyIface)
	obj := types.NewTypeName(token.NoPos, pkg, name, nil)
	n := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
	n.SetTypeParams([]*types.TypeParam{tp})
	return n
}

func TestTypeArgs_FormatsCommonKinds(t *testing.T) {
	local := testLocalNamed()
	pkg := types.NewPackage("example.com/p", "p")
	aliasObj := types.NewTypeName(token.NoPos, pkg, "AliasInt", nil)
	pkg.Scope().Insert(aliasObj)
	aliasInt := types.NewAlias(aliasObj, types.Typ[types.Int])
	got := TypeArgs([]types.Type{
		types.Universe.Lookup("byte").Type(),
		types.NewPointer(types.Typ[types.Int]),
		types.NewSlice(types.Typ[types.String]),
		types.NewArray(types.Typ[types.Int], 2),
		types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
		types.NewChan(types.SendOnly, types.Typ[types.Int]),
		types.NewChan(types.RecvOnly, types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.NewChan(types.RecvOnly, types.Typ[types.Int])),
		local,
		aliasInt,
	})
	const want = "[byte,*int,[]string,[2]int,map[string]int,chan int,chan<- int,<-chan int,chan (<-chan int),Local,int]"
	if got != want {
		t.Fatalf("TypeArgs = %q, want %q", got, want)
	}
}

func TestNamedName_UsesTypeArgString(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	generic := testGenericNamed(pkg, "Box")

	inst, err := types.Instantiate(types.NewContext(), generic, []types.Type{testLocalNamed()}, false)
	if err != nil {
		t.Fatalf("Instantiate failed: %v", err)
	}
	got := NamedName(inst.(*types.Named))
	if !strings.Contains(got, "Box[") || !strings.Contains(got, "Local") {
		t.Fatalf("NamedName(%s) = %q, want Box[...] with Local", inst.String(), got)
	}

	aliasObj := types.NewTypeName(token.NoPos, pkg, "AliasInt", nil)
	pkg.Scope().Insert(aliasObj)
	aliasInt := types.NewAlias(aliasObj, types.Typ[types.Int])
	instAlias, err := types.Instantiate(types.NewContext(), generic, []types.Type{aliasInt}, false)
	if err != nil {
		t.Fatalf("Instantiate with alias failed: %v", err)
	}
	gotAlias := NamedName(instAlias.(*types.Named))
	if !strings.Contains(gotAlias, "int") {
		t.Fatalf("NamedName(%s) = %q, want canonicalized alias arg", instAlias.String(), gotAlias)
	}
}

func TestStr_NamedTypeArgsUsePackageName(t *testing.T) {
	b := New(unsafe.Sizeof(uintptr(0)), types.SizesFor("gc", runtime.GOARCH))
	pkg := types.NewPackage("command-line-arguments", "main")
	fooObj := types.NewTypeName(token.NoPos, pkg, "foo", nil)
	foo := types.NewNamed(fooObj, types.Typ[types.Int], nil)
	generic := testGenericNamed(pkg, "F")

	otherPkg := types.NewPackage("example.com/other", "other")
	barObj := types.NewTypeName(token.NoPos, otherPkg, "Bar", nil)
	bar := types.NewNamed(barObj, types.Typ[types.Int], nil)
	fn := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "", foo)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", bar)),
		false)

	cases := []struct {
		name string
		arg  types.Type
		want string
	}{
		{"same package named", foo, "main.F[main.foo]"},
		{"different package named", bar, "main.F[example.com/other.Bar]"},
		{"unsafe pointer", types.Typ[types.UnsafePointer], "main.F[unsafe.Pointer]"},
		{"byte alias", types.Typ[types.Byte], "main.F[uint8]"},
		{"rune alias", types.Typ[types.Rune], "main.F[int32]"},
		{"any alias", types.Universe.Lookup("any").Type(), "main.F[interface {}]"},
		{"composite pointer", types.NewPointer(foo), "main.F[*main.foo]"},
		{"composite pointer pointer", types.NewPointer(types.NewPointer(foo)), "main.F[**main.foo]"},
		{"composite slice", types.NewSlice(types.Typ[types.String]), "main.F[[]string]"},
		{"composite array", types.NewArray(foo, 2), "main.F[[2]main.foo]"},
		{"composite map", types.NewMap(foo, bar), "main.F[map[main.foo]example.com/other.Bar]"},
		{"composite chan", types.NewChan(types.RecvOnly, foo), "main.F[<-chan main.foo]"},
		{"func fallback", fn, "main.F[func(main.foo) example.com/other.Bar]"},
		{"nil package named", testLocalNamed(), "main.F[Local]"},
	}
	for _, tc := range cases {
		inst, err := types.Instantiate(types.NewContext(), generic, []types.Type{tc.arg}, false)
		if err != nil {
			t.Fatalf("%s: Instantiate failed: %v", tc.name, err)
		}
		if got := b.Str(inst); got != tc.want {
			t.Fatalf("%s: Str(%s) = %q, want %q", tc.name, inst.String(), got, tc.want)
		}
	}
	if got := reflectTypeArgPkgPath(nil); got != "" {
		t.Fatalf("reflectTypeArgPkgPath(nil) = %q, want empty string", got)
	}
}

func TestTypeName_NamedWithoutPackage(t *testing.T) {
	b := New(unsafe.Sizeof(uintptr(0)), types.SizesFor("gc", runtime.GOARCH))
	obj := types.NewTypeName(token.NoPos, nil, "Anon", nil)
	named := types.NewNamed(obj, types.Typ[types.Int], nil)
	got, pub := b.TypeName(named)
	if got != "_llgo_Anon" {
		t.Fatalf("TypeName = %q, want %q", got, "_llgo_Anon")
	}
	if !pub {
		t.Fatalf("TypeName pub = false, want true for pkg=nil")
	}
}

func TestTypeName_DetachedLocalScopeUsesPosition(t *testing.T) {
	b := New(unsafe.Sizeof(uintptr(0)), types.SizesFor("gc", runtime.GOARCH))
	pkg := types.NewPackage("example.com/p", "p")
	local := types.NewScope(nil, token.Pos(100), token.Pos(200), "detached")
	obj := types.NewTypeName(token.Pos(123), pkg, "Local", nil)
	local.Insert(obj)
	named := types.NewNamed(obj, types.Typ[types.Int], nil)

	got, _ := b.TypeName(named)
	if !strings.Contains(got, "Local.p123") {
		t.Fatalf("TypeName = %q, want position suffix for detached local scope", got)
	}
}

func TestScopeIndexHandlesRootAndInvalidScopes(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	root := pkg.Scope()
	parent := types.NewScope(root, token.NoPos, token.NoPos, "parent")
	child := types.NewScope(parent, token.NoPos, token.NoPos, "child")

	if got, ok := scopeIndex(child, root, ""); !ok || got != ".0.0" {
		t.Fatalf("scopeIndex(child, root) = %q, %v, want .0.0, true", got, ok)
	}
	if _, ok := scopeIndex(nil, root, ""); ok {
		t.Fatal("scopeIndex(nil, root) succeeded")
	}
	if _, ok := scopeIndex(child, nil, ""); ok {
		t.Fatal("scopeIndex(child, nil) succeeded")
	}
	if _, ok := scopeIndex(root, root, ""); ok {
		t.Fatal("scopeIndex(root, root) succeeded")
	}
	if _, ok := scopeIndex(types.NewScope(nil, token.NoPos, token.NoPos, "detached"), root, ""); ok {
		t.Fatal("scopeIndex(detached, root) succeeded")
	}
}

func TestBasicName_RuneAlias(t *testing.T) {
	if got := BasicName(types.Typ[types.Rune]); got != "_llgo_int32" {
		t.Fatalf("BasicName(rune) = %q, want %q", got, "_llgo_int32")
	}
}

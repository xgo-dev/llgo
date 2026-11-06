package typesizes

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestMalformedNativeCallbackReferences(t *testing.T) {
	for _, ref := range []string{
		"__llgo_native_Cdecl_4096_4096_false",
		"__llgo_native_Cdecl_2000000000_2000000000_false",
		"__llgo_native_Cdecl_9223372036854775807_1_false",
		"__llgo_native_Cdecl_1_0_false",
		"__llgo_native_Cdecl_2_0_false[int]",
		"__llgo_native_Cdecl_0_0_true",
		"__llgo_native_Unknown_0_0_false",
	} {
		t.Run(ref, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "callback.go", "package p\n//llgo:cdecl\ntype Native struct { F "+ref+" }", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			s := LLGoSizesFor("amd64")
			s.Layout = new(LLGoLayout)
			pkg := types.NewPackage("p", "p")
			if err := s.Layout.ParseSyntax(fset, pkg, []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
			if len(s.Layout.nativeFuncs) != 0 {
				t.Fatal("malformed callback reference created a native signature")
			}
			conf := types.Config{Sizes: s}
			err = types.NewChecker(&conf, fset, pkg, nil).Files([]*ast.File{file})
			if err == nil || !strings.Contains(err.Error(), "callback.go:3:") || !strings.Contains(err.Error(), "undefined:") {
				t.Fatalf("expected positioned checker diagnostic, got %v", err)
			}
		})
	}
}

func TestLayoutAttributeCacheInvalidation(t *testing.T) {
	fset := token.NewFileSet()
	pkg := types.NewPackage("p", "p")
	layout := new(LLGoLayout)
	callback := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Callback", nil), types.NewSignatureType(nil, nil, nil, nil, nil, false), nil)
	pkg.Scope().Insert(callback.Obj())
	if got := layout.TypeAttrs(callback); got.Native {
		t.Fatal("ordinary function was native")
	}
	file, err := parser.ParseFile(fset, "callback.go", "package p\n//llgo:stdcall\ntype Callback func()", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if err := layout.ParseSyntax(fset, pkg, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if got := layout.TypeAttrs(callback); !got.Native || got.CallConv != "stdcall" {
			t.Fatalf("lookup %d after syntax registration: %+v", i, got)
		}
	}
	// An independent layout first sees an ordinary imported callback; metadata
	// exported later must replace that cached result when the importer records it.
	importerLayout := new(LLGoLayout)
	if got := importerLayout.TypeAttrs(callback); got.Native {
		t.Fatal("unrecorded export metadata was native")
	}
	layout.RecordPackage(pkg)
	importerLayout.RecordPackage(pkg)
	if got := importerLayout.TypeAttrs(callback); !got.Native || got.CallConv != "stdcall" {
		t.Fatalf("lookup after recording export metadata: %+v", got)
	}
}

func BenchmarkLayoutTypeAttrs(b *testing.B) {
	for _, native := range []bool{false, true} {
		b.Run(fmt.Sprintf("native=%t", native), func(b *testing.B) {
			pkg := types.NewPackage("p", "p")
			layout := new(LLGoLayout)
			typ := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "T", nil), types.NewStruct(nil, nil), nil)
			if native {
				layout.types = map[llgoDeclKey]llgoTypeDecl{{pkg, "T"}: {attrs: LLGoTypeAttrs{Native: true}}}
			}
			layout.TypeAttrs(typ)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if got := layout.TypeAttrs(typ); got.Native != native {
					b.Fatal(got)
				}
			}
		})
	}
}

func TestNativeCallbackContainers(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "callbacks.go", `package p
type Box[T any] struct { Value T }
type Pair[A, B any] struct { A A; B B }
//llgo:stdcall
type Native struct {
 Single Box[func()]
 Pair Pair[func(), func(int) int]
 Pointer *(func())
 Map map[*func()]func()
 Chan chan func()
 Variadic func(...func()) func()
}
//go:linkname linked stdcall.callback
var linked func()
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	// Reusing syntax must recreate the same canonical callbacks, including
	// stdcall and variadic signatures, in a fresh checker.
	for attempt := 0; attempt < 2; attempt++ {
		s := LLGoSizesFor("amd64")
		s.Layout = new(LLGoLayout)
		pkg := types.NewPackage("p", "p")
		if err := s.Layout.ParseSyntax(fset, pkg, []*ast.File{file}, nil); err != nil {
			t.Fatal(err)
		}
		conf := types.Config{Sizes: s}
		if err := types.NewChecker(&conf, fset, pkg, nil).Files([]*ast.File{file}); err != nil {
			t.Fatal(err)
		}
		var check func(types.Type)
		check = func(typ types.Type) {
			switch u := typ.Underlying().(type) {
			case *types.Struct:
				for i := 0; i < u.NumFields(); i++ {
					check(u.Field(i).Type())
				}
			case *types.Pointer:
				check(u.Elem())
			case *types.Slice:
				check(u.Elem())
			case *types.Map:
				check(u.Key())
				check(u.Elem())
			case *types.Chan:
				check(u.Elem())
			case *types.Signature:
				attrs := s.Layout.TypeAttrs(typ)
				if !attrs.Native || attrs.CallConv != "stdcall" || s.Sizeof(typ) != 8 {
					t.Errorf("callback %s: attrs=%+v, size=%d", typ, attrs, s.Sizeof(typ))
				}
				for _, tuple := range []*types.Tuple{u.Params(), u.Results()} {
					for i := 0; i < tuple.Len(); i++ {
						check(tuple.At(i).Type())
					}
				}
			}
		}
		st := pkg.Scope().Lookup("Native").Type().Underlying().(*types.Struct)
		check(st)
		check(pkg.Scope().Lookup("linked").Type())
		if sig := st.Field(5).Type().Underlying().(*types.Signature); !sig.Variadic() {
			t.Fatal("native callback lost variadic signature")
		}
	}
}

func TestLayoutDeclarationForms(t *testing.T) {
	s := LLGoSizesFor("amd64")
	pkg := checkSource(t, `package p
/* Block comments do not declare layout attributes. */
//llgo:stdcall
type (
 // single-spec documentation
 Callback func()
)
//llgo:type stdcall
type Legacy func()
//llgo:type unrelated
type Ordinary func()
//llgo:cdecl
type Generic[A, B any] func(A) B
type Derived (Generic[int, byte])
type Embedded[T any] struct { Value T }
type Outer struct { Embedded[int] }
`, s)
	for _, name := range []string{"Callback", "Legacy", "Derived"} {
		if got := s.Sizeof(pkg.Scope().Lookup(name).Type()); got != 8 {
			t.Errorf("%s size = %d, want 8", name, got)
		}
	}
	if got := s.Sizeof(pkg.Scope().Lookup("Ordinary").Type()); got != 16 {
		t.Fatalf("unrelated directive changed Go funcval size to %d", got)
	}
}

func TestLocalPaddingWarning(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "local.go", `package p
func f() {
 var value struct { X byte
 //llgo:padding
 E struct{}
 }
 _ = value
}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var warnings []error
	err = new(LLGoLayout).ParseSyntax(fset, types.NewPackage("p", "p"), []*ast.File{file}, func(err error) { warnings = append(warnings, err) })
	if err != nil || len(warnings) != 1 || !strings.Contains(warnings[0].Error(), "local.go:4:") {
		t.Fatalf("warnings=%v, error=%v", warnings, err)
	}
}

func TestPackedGenericInstances(t *testing.T) {
	for _, field := range []string{"byte", "*byte"} {
		t.Run(field, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "instance.go", `package p
//llgo:packed
type Packed[T any] struct { X T }
func f() { var _ Packed[`+field+`] }
`, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			s := LLGoSizesFor("amd64")
			s.Layout = new(LLGoLayout)
			pkg := types.NewPackage("p", "p")
			if err := s.Layout.ParseSyntax(fset, pkg, []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Instances: make(map[*ast.Ident]types.Instance)}
			conf := types.Config{Sizes: s}
			if err := types.NewChecker(&conf, fset, pkg, info).Files([]*ast.File{file}); err != nil {
				t.Fatal(err)
			}
			if err := s.ValidatePackage(pkg); err != nil {
				t.Fatalf("uninstantiated declaration: %v", err)
			}
			err = s.ValidateInstances(fset, info)
			if field == "byte" && err != nil {
				t.Fatal(err)
			}
			if field == "*byte" && (err == nil || !strings.Contains(err.Error(), "instance.go:4:") || !strings.Contains(err.Error(), "managed references")) {
				t.Fatalf("missing positioned instance diagnostic: %v", err)
			}
		})
	}
}

func TestNestedManagedPackingDiagnostics(t *testing.T) {
	s := LLGoSizesFor("amd64")
	pkg := checkSource(t, "package p\n//llgo:packed\ntype Bad struct { P *byte }", s)
	bad := pkg.Scope().Lookup("Bad").Type()
	field := types.NewField(token.NoPos, pkg, "Bad", bad, false)
	for _, typ := range []types.Type{
		types.NewStruct([]*types.Var{field}, nil),
		types.NewMap(bad, types.Typ[types.Int]),
		types.NewSignatureType(nil, nil, nil, types.NewTuple(field), nil, false),
	} {
		if err := s.Validate(typ); err == nil || !strings.Contains(err.Error(), "managed references") {
			t.Errorf("%s: %v", typ, err)
		}
	}
}

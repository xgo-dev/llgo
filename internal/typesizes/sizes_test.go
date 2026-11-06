package typesizes

import (
	"bytes"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/gcexportdata"
)

func checkSource(t *testing.T, source string, sizes *LLGoSizes) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "layout.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("example/layout", file.Name.Name)
	if sizes.Layout == nil {
		sizes.Layout = new(LLGoLayout)
	}
	if err := sizes.Layout.ParseSyntax(fset, pkg, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	conf := types.Config{Sizes: sizes, Importer: importer.Default()}
	if err := types.NewChecker(&conf, fset, pkg, nil).Files([]*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	sizes.Layout.RecordPackage(pkg)
	return pkg
}

func TestLayoutIdentityAndExportData(t *testing.T) {
	s := LLGoSizesFor("amd64")
	pkg := checkSource(t, `package layout
//llgo:cdecl
type Native struct { S struct { X byte; E struct{} } }
type Go struct { S struct { X byte; E struct{} } }
//llgo:cdecl
type Callback func(int32) int32
type CallbackCopy Callback
//llgo:cdecl
type Callbacks struct { F func(int32) int32 }
//llgo:packed
type Packed struct { B byte; X uint64 `+"`json:\"x\"`"+` }
type NoPad struct { X uint64
//llgo:no-padding
E struct{}
}
`, s)
	n := pkg.Scope().Lookup("Native").Type().Underlying().(*types.Struct).Field(0).Type()
	g := pkg.Scope().Lookup("Go").Type().Underlying().(*types.Struct).Field(0).Type()
	if types.Identical(n, g) {
		t.Fatal("different layouts have the same identity")
	}
	var data bytes.Buffer
	if err := gcexportdata.Write(&data, token.NewFileSet(), pkg); err != nil {
		t.Fatal(err)
	}
	imported, err := gcexportdata.Read(&data, token.NewFileSet(), make(map[string]*types.Package), pkg.Path())
	if err != nil {
		t.Fatal(err)
	}
	fresh := LLGoSizesFor("amd64")
	fresh.Layout = new(LLGoLayout)
	fresh.Layout.RecordPackage(imported)
	for _, name := range []string{"Native", "Go", "Packed", "NoPad", "Callback", "CallbackCopy", "Callbacks"} {
		want, got := pkg.Scope().Lookup(name).Type(), imported.Scope().Lookup(name).Type()
		if fresh.Sizeof(got) != s.Sizeof(want) || fresh.Alignof(got) != s.Alignof(want) {
			t.Errorf("%s lost layout in export data", name)
		}
		st, ok := got.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		fields := make([]*types.Var, st.NumFields())
		for i := range fields {
			fields[i] = st.Field(i)
		}
		if name == "Packed" && fresh.Offsetsof(fields)[1] != 1 {
			t.Fatal("imported packed offset")
		}
	}
	packed := imported.Scope().Lookup("Packed").Type().Underlying().(*types.Struct)
	if tag := LLGoFieldTag(packed.Tag(1)); tag != `json:"x"` {
		t.Fatalf("user tag = %q", tag)
	}
}

func TestConcurrentNativeFunctionChecks(t *testing.T) {
	s := LLGoSizesFor("amd64")
	s.Layout = new(LLGoLayout)
	var workers sync.WaitGroup
	for i := 0; i < 20; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			pkg := checkSource(t, `package p
import "unsafe"
//go:linkname f C.f
func f()
const Size = unsafe.Sizeof(f)
`, s)
			if size, _ := constant.Int64Val(pkg.Scope().Lookup("Size").(*types.Const).Val()); size != 16 {
				t.Errorf("native function size = %d", size)
			}
			if s.Sizeof(pkg.Scope().Lookup("f").Type()) != 16 {
				t.Error("linked function value is not a Go funcval")
			}
		}()
	}
	workers.Wait()
}

func TestLayoutSyntaxCanBeCheckedAgain(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "native.go", `package p
//llgo:cdecl
type Native[T any] struct { F func(T) int32; G func() }
type IntNative = Native[int]
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
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
		if got := s.Sizeof(pkg.Scope().Lookup("IntNative").Type()); got != 16 {
			t.Fatalf("iteration %d: size = %d", i, got)
		}
	}
}

func TestLayoutsAndUnsafeConstants(t *testing.T) {
	s := LLGoSizesFor("amd64")
	pkg := checkSource(t, `package layout
import "unsafe"
type Empty struct{}
type Empty2 struct{}
type GoFunc func()
//llgo:cdecl
type CFunc func()
//go:linkname native C.native
func native()
//go:linkname callback C.callback
var callback func()
const CallbackSize = unsafe.Sizeof(callback)
const NativeSize = unsafe.Sizeof(native)
const NativeAlign = unsafe.Alignof(native)
const GoSize = unsafe.Sizeof(GoFunc(nil))
type FuncTail struct { B byte; F GoFunc; E Empty }
//llgo:cdecl
type Native struct { F func(); P *byte; B byte }
type NativeCopy Native
type NativeAlias = Native
//llgo:packed
type Packed struct { B byte; F GoFunc; Z byte }
//llgo:pack 2
type Pack2 struct { B byte; X uint32; Y uint16 }
type Pad struct {
 //llgo:padding
 E Empty
 B byte
}
type NoPad struct {
 X uint64
 //llgo:no-padding
 E Empty
}
type Pair[A, B any] struct {
 //llgo:padding-if
 A A
 //llgo:padding-if
 B B
}
type Same = Pair[Empty, Empty]
type Different = Pair[Empty, Empty2]
type Nonempty = Pair[int, Empty]
type Nested struct { B byte; P [2]Packed; N [2]NoPad }
const TailOffset = unsafe.Offsetof(FuncTail{}.E)
const PackedSize = unsafe.Sizeof(Packed{})
const SameSize = unsafe.Sizeof(Same{})
`, s)
	for _, tc := range []struct {
		name        string
		size, align int64
		offsets     []int64
	}{
		{"GoFunc", 16, 8, nil}, {"CFunc", 8, 8, nil},
		{"FuncTail", 32, 8, []int64{0, 8, 24}},
		{"Native", 24, 8, []int64{0, 8, 16}},
		{"NativeCopy", 24, 8, []int64{0, 8, 16}},
		{"NativeAlias", 24, 8, []int64{0, 8, 16}},
		{"Packed", 18, 1, []int64{0, 1, 17}},
		{"Pack2", 8, 2, []int64{0, 2, 6}},
		{"Pad", 2, 1, []int64{0, 1}},
		{"NoPad", 8, 8, []int64{0, 8}},
		{"Same", 2, 1, []int64{0, 1}},
		{"Different", 0, 1, []int64{0, 0}},
		{"Nonempty", 8, 8, []int64{0, 8}},
		{"Nested", 56, 8, []int64{0, 1, 40}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			typ := pkg.Scope().Lookup(tc.name).Type()
			if size, align := s.Sizeof(typ), s.Alignof(typ); size != tc.size || align != tc.align {
				t.Fatalf("size/align = %d/%d, want %d/%d", size, align, tc.size, tc.align)
			}
			if tc.offsets != nil {
				st := typ.Underlying().(*types.Struct)
				fields := make([]*types.Var, st.NumFields())
				for i := range fields {
					fields[i] = st.Field(i)
				}
				if got := s.Offsetsof(fields); !reflect.DeepEqual(got, tc.offsets) {
					t.Fatalf("offsets = %v, want %v", got, tc.offsets)
				}
			}
		})
	}
	for name, want := range map[string]int64{"NativeSize": 16, "CallbackSize": 8, "NativeAlign": 8, "GoSize": 16, "TailOffset": 24, "PackedSize": 18, "SameSize": 2} {
		got, ok := constant.Int64Val(pkg.Scope().Lookup(name).(*types.Const).Val())
		if !ok || got != want {
			t.Errorf("constant %s = %d, want %d", name, got, want)
		}
	}
}

func TestOverflowAndZeroArrays(t *testing.T) {
	s := LLGoSizesFor("amd64")
	fn := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	tooBig := types.NewArray(fn, 1<<59)
	if got := s.Sizeof(tooBig); got != -1 {
		t.Fatalf("overflow size = %d", got)
	}
	if got := s.Sizeof(types.NewArray(tooBig, 0)); got != 0 {
		t.Fatalf("zero array size = %d", got)
	}
	fields := []*types.Var{types.NewField(0, nil, "A", tooBig, false), types.NewField(0, nil, "B", types.Typ[types.Byte], false)}
	if got := s.Offsetsof(fields); !reflect.DeepEqual(got, []int64{0, -1}) {
		t.Fatalf("overflow offsets = %v", got)
	}
	if got := s.Sizeof(types.NewStruct(fields, nil)); got != -1 {
		t.Fatalf("overflow struct = %d", got)
	}
}

func TestDirectiveDiagnostics(t *testing.T) {
	for _, source := range []string{
		"//llgo:padding\ntype T int",
		"type T struct {\n//llgo:no-padding\nA struct{}; B byte\n}",
		"type T[A any] struct {\n//llgo:no-padding\nA A\n}",
		"type T struct {\n//llgo:padding-if\nA struct{}\n}",
		"type T struct {\n//llgo:padding\n//llgo:no-padding\nA struct{}\n}",
		"//llgo:pack 3\ntype T struct { A int }",
		"//llgo:packed\ntype T int",
		"type (\n//llgo:cdecl\nT func()\nU int\n)",
		"//llgo:packed\n//llgo:pack 2\ntype T struct { X int }",
		"type T struct {\n//llgo:padding 1\nX struct{}\n}",
		"var X struct {\n//llgo:padding-if\nX struct{}\n}",
		"//llgo:cdecl argument\ntype T func()",
		"//llgo:stdcall argument\ntype T func()",
		"//llgo:cdecl\n//llgo:stdcall\ntype T func()",
		"//llgo:cdecl\ntype (T func(); U int)",
		"type (\n//llgo:cdecl argument\nT func()\nU int\n)",
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "bad.go", "package p\n"+source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		err = new(LLGoLayout).ParseSyntax(fset, types.NewPackage("p", "p"), []*ast.File{file}, nil)
		if err == nil || !strings.Contains(err.Error(), "bad.go:") {
			t.Errorf("missing diagnostic for %s: %v", source, err)
		}
	}
}

func TestManagedPackingDiagnostics(t *testing.T) {
	{
		s := LLGoSizesFor("wasm")
		pkg := checkSource(t, "package p\ntype Go struct { P *byte; X byte }\n//llgo:cdecl\ntype Native Go", s)
		if err := s.ValidatePackage(pkg); err == nil || !strings.Contains(err.Error(), "field identity") {
			t.Fatalf("native struct identity diagnostic: %v", err)
		}
	}
	for _, field := range []string{"*byte", "func()", "string", "[]byte", "interface{}", "map[int]int", "chan int", "[2]*byte"} {
		s := LLGoSizesFor("amd64")
		pkg := checkSource(t, "package p\n//llgo:packed\ntype T struct { X byte; V "+field+" }", s)
		if err := s.ValidatePackage(pkg); err == nil || !strings.Contains(err.Error(), "managed references") {
			t.Errorf("%s: %v", field, err)
		}
	}
	for _, source := range []string{
		"//llgo:packed\ntype T struct { X byte; V uint64 }",
		"//llgo:pack 8\ntype T struct { X byte; V *byte }",
		"//llgo:cdecl\n//llgo:packed\ntype T struct { X byte; V *byte }",
		"//llgo:cdecl\ntype Callback func()\n//llgo:packed\ntype T struct { X byte; V Callback }",
	} {
		s := LLGoSizesFor("amd64")
		pkg := checkSource(t, "package p\n"+source, s)
		if err := s.ValidatePackage(pkg); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGeneratedAdapterMatchesPatch(t *testing.T) {
	cmd := exec.Command("go", "run", "generate.go", "-check")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated adapters: %v\n%s", err, out)
	}
}

func TestPointerLayoutConversions(t *testing.T) {
	for _, tc := range []struct {
		expression string
		allowed    bool
	}{
		{"(*Plain)(p)", false},
		{"(*Plain)(unsafe.Pointer(p))", true},
		{"Plain(*p)", true},
		{"(*Same)(p)", true},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			s := LLGoSizesFor("amd64")
			s.Layout = new(LLGoLayout)
			fset := token.NewFileSet()
			source := `package p
import "unsafe"
var _ unsafe.Pointer
//llgo:packed
type Packed struct { B byte; X uint64 }
type Plain struct { B byte; X uint64 }
//llgo:packed
type Same struct { B byte; X uint64 }
func convert(p *Packed) { _ = ` + tc.expression + ` }
`
			file, err := parser.ParseFile(fset, "pointer.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := types.NewPackage("p", "p")
			files := []*ast.File{file}
			if err := s.Layout.ParseSyntax(fset, pkg, files, nil); err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
			conf := types.Config{Sizes: s, Importer: importer.Default()}
			if err := types.NewChecker(&conf, fset, pkg, info).Files(files); err != nil {
				t.Fatal(err)
			}
			s.Layout.RecordPackage(pkg)
			err = s.ValidateConversions(fset, files, info)
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, error=%v", tc.allowed, err)
			}
			if err != nil && !strings.Contains(err.Error(), "different LLGo layouts") {
				t.Fatal(err)
			}
		})
	}
}

func TestReservedLayoutDeclarations(t *testing.T) {
	for _, source := range []string{
		"const LLGoTypeLayout_T = 1",
		"type __llgo_native_Cdecl_0_0_false int",
		"func LLGoTypeLayout_T() {}",
		"//go:linkname f C.f\nvar f, g func()",
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "reserved.go", "package p\n"+source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if err := new(LLGoLayout).ParseSyntax(fset, types.NewPackage("p", "p"), []*ast.File{file}, nil); err == nil {
			t.Fatalf("missing diagnostic: %s", source)
		}
	}
}

func TestInvalidTypeSizeDuringErrorRecovery(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "invalid.go", `package p
import "unsafe"
type Broken struct { X Missing }
const N = unsafe.Sizeof(Broken{})
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics []string
	conf := types.Config{
		Sizes: LLGoSizesFor("amd64"), Importer: importer.Default(),
		Error: func(err error) { diagnostics = append(diagnostics, err.Error()) },
	}
	_, err = conf.Check("p", fset, []*ast.File{file}, nil)
	if err == nil || len(diagnostics) != 1 || !strings.Contains(diagnostics[0], "undefined: Missing") {
		t.Fatalf("diagnostics = %v, error = %v", diagnostics, err)
	}
}

func TestLayoutSyntaxTraversal(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "p.go", `package p
// declaration
var A = f(1, 2)
func f(a, b int) int { return a + b }
type T struct { X [2]*int; Y func(int) string }
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, prune := range []bool{false, true} {
		collect := func(out *[]ast.Node) func(ast.Node) bool {
			return func(node ast.Node) bool {
				if node != nil {
					*out = append(*out, node)
				}
				_, body := node.(*ast.BlockStmt)
				return !prune || !body
			}
		}
		var got, want []ast.Node
		ast.Inspect(file, collect(&want))
		llgoInspect(file, collect(&got))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("prune=%t: traversal order differs", prune)
		}
	}
}

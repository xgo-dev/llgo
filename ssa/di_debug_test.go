//go:build !llgo

package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestDebugRecursiveNamedTypesFinalize(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "recursive.go", `package p
type Link *Link
type Peano *Peano
func inspect() {
	if true {
		value := 1
		_ = value
	}
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	typesPkg := types.NewPackage("example.com/p", "p")
	info := &types.Info{Scopes: make(map[ast.Node]*types.Scope)}
	if err := types.NewChecker(&types.Config{}, fset, typesPkg, info).Files([]*ast.File{file}); err != nil {
		t.Fatal(err)
	}

	prog := NewProgram(&Target{OptLevel: optlevel.O0})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.InitDebug("p", "example.com/p", fset)
	fn := pkg.NewFunc("debugTypes", NoArgsNoRet, InGo)
	builder := fn.NewBuilder()
	defer builder.impl.Dispose()
	for _, name := range []string{"Link", "Peano"} {
		object := typesPkg.Scope().Lookup(name)
		pos := fset.Position(object.Pos())
		global := pkg.NewVar("example.com/p."+name, types.NewPointer(object.Type()), InGo)
		builder.DIGlobal(global.Expr, name, pos)
	}

	decl := file.Decls[2].(*ast.FuncDecl)
	object := typesPkg.Scope().Lookup("inspect").(*types.Func)
	function := pkg.NewFunc("example.com/p.inspect", object.Type().(*types.Signature), InGo)
	functionBuilder := function.MakeBody(1)
	defer functionBuilder.Dispose()
	functionBuilder.DebugFunction(
		function,
		object.Scope(),
		fset.Position(object.Pos()),
		fset.Position(decl.Body.Lbrace),
	)
	if got := functionBuilder.DIScope(function, nil); got != function {
		t.Fatal("nil scope did not resolve to the function")
	}
	if got := functionBuilder.DIScope(function, object.Scope()); got != function {
		t.Fatal("function scope did not resolve to the function")
	}
	if got := functionBuilder.DIScope(function, typesPkg.Scope()); got != function {
		t.Fatal("package scope did not resolve to the function")
	}
	innerBlock := decl.Body.List[0].(*ast.IfStmt).Body
	innerScope := info.Scopes[innerBlock]
	if innerScope == nil {
		t.Fatal("inner lexical scope not found")
	}
	lexical := functionBuilder.DIScope(function, innerScope)
	if lexical == function || functionBuilder.DIScope(function, innerScope) != lexical {
		t.Fatal("inner lexical scope was not created and cached")
	}
	functionBuilder.DISetCurrentDebugLocation(lexical, fset.Position(innerBlock.Lbrace))
	functionBuilder.Return()
	functionBuilder.EndBuild()

	pkg.FinalizeDebug()
	pkg.FinalizeDebug()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("recursive debug metadata is invalid: %v\n%s", err, pkg.Module().String())
	}
	ir := pkg.Module().String()
	for _, name := range []string{"Link", "Peano"} {
		if !strings.Contains(ir, `name: "`+name+`"`) {
			t.Fatalf("module is missing debug type %s:\n%s", name, ir)
		}
	}
	for _, want := range []string{"DILexicalBlock", "isOptimized: false"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("module is missing %q:\n%s", want, ir)
		}
	}
}

func TestDebugGoTypeEncodings(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "types.go", `package p
type Named int64
type Recursive struct { Next *Recursive }
type Shape struct {
	Complex complex128
	Text string
	Values []Named
	Lookup map[string]Named
	Queue chan Named
	Callback func(Named) (Named, error)
	Any any
	Recursive *Recursive
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	typesPkg, err := (&types.Config{}).Check("example.com/p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}

	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	prog.SetRuntime(newDebugRuntimePackage())
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.InitDebug("p", "example.com/p", fset)

	shape := typesPkg.Scope().Lookup("Shape").Type()
	global := pkg.NewVar("example.com/p.GlobalShape", types.NewPointer(shape), InGo)
	fn := pkg.NewFunc("debugTypes", NoArgsNoRet, InGo)
	builder := fn.NewBuilder()
	defer builder.impl.Dispose()
	builder.DIGlobal(global.Expr, "GlobalShape", fset.Position(typesPkg.Scope().Lookup("Shape").Pos()))

	fallback := token.Position{Filename: "fallback.go", Line: 7}
	noPos := types.NewNamed(types.NewTypeName(token.NoPos, typesPkg, "NoPos", nil), types.Typ[types.Int], nil)
	if got := pkg.di.typeDeclarationPosition(noPos, fallback); got != fallback {
		t.Fatalf("invalid declaration position = %v, want %v", got, fallback)
	}

	pkg.FinalizeDebug()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("Go type debug metadata is invalid: %v\n%s", err, pkg.Module().String())
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		"DW_LANG_C",
		"DW_ATE_complex_float",
		"!DISubroutineType",
		`name: "map[string]example.com/p.Named"`,
		`name: "chan example.com/p.Named"`,
		`name: "example.com/p.Recursive"`,
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("module is missing %q:\n%s", want, ir)
		}
	}
}

func TestDIGlobalIgnoresStorageLessFrontendVariable(t *testing.T) {
	var builder Builder
	builder.DIGlobal(pyVarExpr(Nil, "attribute"), "module.attribute", token.Position{})
}

func TestDeferInitBuilderInheritsDebugLocation(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "defer.go", `package p
func f() {}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	typesPkg, err := (&types.Config{}).Check("example.com/p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}

	prog := NewProgram(&Target{OptLevel: optlevel.O0})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.InitDebug("p", "example.com/p", fset)
	decl := file.Decls[0].(*ast.FuncDecl)
	object := typesPkg.Scope().Lookup("f").(*types.Func)
	fn := pkg.NewFunc("example.com/p.f", object.Type().(*types.Signature), InGo)
	builder := fn.MakeBody(1)
	defer builder.Dispose()
	bodyPos := fset.Position(decl.Body.Lbrace)
	builder.DebugFunction(fn, object.Scope(), fset.Position(object.Pos()), bodyPos)
	builder.DISetCurrentDebugLocation(fn, bodyPos)
	builder.Return()

	deferBuilder, next := fn.deferInitBuilder(builder)
	defer deferBuilder.Dispose()
	loc := deferBuilder.impl.GetCurrentDebugLocation()
	if loc.Line != uint(bodyPos.Line) || loc.Col != uint(bodyPos.Column) || loc.Scope != fn.diFunc.ll {
		t.Fatalf("defer debug location = %+v, want %s:%d:%d", loc, bodyPos.Filename, bodyPos.Line, bodyPos.Column)
	}
	deferBuilder.Jump(next)

	pkg.FinalizeDebug()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("defer debug metadata is invalid: %v\n%s", err, pkg.Module().String())
	}
}

func newDebugRuntimePackage() *types.Package {
	pkg := types.NewPackage(PkgRuntime, "runtime")
	unsafePointer := types.Typ[types.UnsafePointer]
	members := map[string][]*types.Var{
		"String": {
			types.NewField(token.NoPos, pkg, "data", unsafePointer, false),
			types.NewField(token.NoPos, pkg, "len", types.Typ[types.Uint], false),
		},
		"Slice": {
			types.NewField(token.NoPos, pkg, "data", unsafePointer, false),
			types.NewField(token.NoPos, pkg, "len", types.Typ[types.Uint], false),
			types.NewField(token.NoPos, pkg, "cap", types.Typ[types.Uint], false),
		},
		"Eface": {
			types.NewField(token.NoPos, pkg, "type", unsafePointer, false),
			types.NewField(token.NoPos, pkg, "data", unsafePointer, false),
		},
		"Iface": {
			types.NewField(token.NoPos, pkg, "type", unsafePointer, false),
			types.NewField(token.NoPos, pkg, "data", unsafePointer, false),
		},
		"Map": {
			types.NewField(token.NoPos, pkg, "count", types.Typ[types.Int], false),
		},
		"Chan": {
			types.NewField(token.NoPos, pkg, "count", types.Typ[types.Int], false),
		},
	}
	for name, fields := range members {
		obj := types.NewTypeName(token.NoPos, pkg, name, nil)
		types.NewNamed(obj, types.NewStruct(fields, nil), nil)
		pkg.Scope().Insert(obj)
	}
	return pkg
}

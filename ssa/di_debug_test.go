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

func TestWindowsDebugPointerParameter(t *testing.T) {
	const goarch = "amd64"
	for _, test := range []struct {
		goos        string
		wantDeclare bool
	}{
		{goos: "linux"},
		{goos: "windows", wantDeclare: true},
	} {
		t.Run(test.goos, func(t *testing.T) {
			fset := token.NewFileSet()
			file := fset.AddFile("param.go", -1, 100)
			pkgTypes := types.NewPackage("example.com/p", "p")
			param := types.NewParam(file.Pos(20), pkgTypes, "p", types.NewPointer(types.Typ[types.Int]))
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), nil, false)
			object := types.NewFunc(file.Pos(10), pkgTypes, "f", sig)

			prog := NewProgram(&Target{GOOS: test.goos, GOARCH: goarch, OptLevel: optlevel.O0})
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", goarch))
			pkg := prog.NewPackage("p", "example.com/p")
			pkg.InitDebug("p", "example.com/p", fset)
			fn := pkg.NewFunc("example.com/p.f", sig, InGo)
			builder := fn.MakeBody(1)
			defer builder.Dispose()
			pos := fset.Position(param.Pos())
			builder.DebugFunction(fn, object.Scope(), fset.Position(object.Pos()), pos)
			debugParam := builder.DIVarParam(fn, pos, param.Name(), prog.Type(param.Type(), InGo), 1)
			builder.DIParam(param, fn.Param(0), debugParam, fn, pos, fn.Block(0))
			for _, kind := range []types.BasicKind{types.Uint, types.Uintptr} {
				typ := types.Typ[kind]
				global := pkg.NewVar("example.com/p."+typ.Name(), types.NewPointer(typ), InGo)
				builder.DIGlobal(global.Expr, typ.Name(), pos)
			}
			builder.Return()
			pkg.FinalizeDebug()

			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("debug metadata is invalid: %v\n%s", err, pkg.Module().String())
			}
			ir := pkg.Module().String()
			if got := strings.Contains(ir, "#dbg_declare"); got != test.wantDeclare {
				t.Fatalf("dbg_declare presence = %v, want %v:\n%s", got, test.wantDeclare, ir)
			}
			if test.goos == "windows" {
				for _, want := range []string{
					`!DIBasicType(name: "int64"`,
					`!DIBasicType(name: "uint64"`,
					`DW_TAG_typedef, name: "int"`,
					`DW_TAG_typedef, name: "uint"`,
					`DW_TAG_typedef, name: "uintptr"`,
				} {
					if !strings.Contains(ir, want) {
						t.Errorf("Windows debug metadata is missing %q:\n%s", want, ir)
					}
				}
			}
		})
	}
}

func TestWindows386WideIntegerDebugValue(t *testing.T) {
	const goarch = "386"
	fset := token.NewFileSet()
	file := fset.AddFile("wide.go", -1, 100)
	pkgTypes := types.NewPackage("example.com/p", "p")
	variable := types.NewVar(file.Pos(20), pkgTypes, "value", types.Typ[types.Uint64])
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	object := types.NewFunc(file.Pos(10), pkgTypes, "f", sig)

	prog := NewProgram(&Target{GOOS: "windows", GOARCH: goarch, OptLevel: optlevel.O0})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", goarch))
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.InitDebug("p", "example.com/p", fset)
	fn := pkg.NewFunc("example.com/p.f", sig, InGo)
	builder := fn.MakeBody(1)
	defer builder.Dispose()
	pos := fset.Position(variable.Pos())
	builder.DebugFunction(fn, object.Scope(), fset.Position(object.Pos()), pos)
	debugVar := builder.DIVarAuto(fn, pos, variable.Name(), prog.Uint64())
	builder.DIValue(variable, prog.IntVal(1<<32|17, prog.Uint64()), debugVar, fn, pos, fn.Block(0))
	builder.Return()
	pkg.FinalizeDebug()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("debug metadata is invalid: %v\n%s", err, pkg.Module().String())
	}
	ir := pkg.Module().String()
	for _, want := range []string{"alloca i64", "store i64 4294967313", "#dbg_value(ptr", "DW_OP_deref"} {
		if !strings.Contains(ir, want) {
			t.Errorf("Windows/386 wide integer debug value is missing %q:\n%s", want, ir)
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
	// A cached declaration can be visited again while patched packages are
	// lowered. Initializing its source location must remain idempotent: the
	// function's DISubprogram is a scope, not an inlined-at DILocation.
	builder.DebugFunction(fn, object.Scope(), fset.Position(object.Pos()), bodyPos)
	loc := builder.impl.GetCurrentDebugLocation()
	if !loc.InlinedAt.IsNil() {
		t.Fatalf("function debug location has an inlined-at node: %+v", loc)
	}
	builder.DISetCurrentDebugLocation(fn, bodyPos)
	builder.Return()

	deferBuilder, next := fn.deferInitBuilder(builder)
	defer deferBuilder.Dispose()
	loc = deferBuilder.impl.GetCurrentDebugLocation()
	if loc.Line != uint(bodyPos.Line) || loc.Col != uint(bodyPos.Column) || loc.Scope != fn.diFunc.ll {
		t.Fatalf("defer debug location = %+v, want %s:%d:%d", loc, bodyPos.Filename, bodyPos.Line, bodyPos.Column)
	}
	deferBuilder.Jump(next)

	pkg.FinalizeDebug()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("defer debug metadata is invalid: %v\n%s", err, pkg.Module().String())
	}
}

func TestSyntheticBuilderDebugLocation(t *testing.T) {
	for _, mode := range []string{"before debug init", "after debug init", "without debug"} {
		t.Run(mode, func(t *testing.T) {
			prog := NewProgram(&Target{OptLevel: optlevel.O0})
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
			pkg := prog.NewPackage("p", "example.com/p")
			fn := pkg.NewFunc("example.com/p.f", NoArgsNoRet, InGo)
			body := fn.MakeBody(1)
			defer body.Dispose()
			body.Return()

			var synthetic Builder
			if mode != "after debug init" {
				synthetic = fn.NewBuilder()
			}
			debug := mode != "without debug"
			if debug {
				pkg.InitDebug("p", "example.com/p", token.NewFileSet())
				pos := token.Position{Filename: "defer.go", Line: 2, Column: 1}
				body.DebugFunction(fn, nil, pos, pos)
			}
			if synthetic == nil {
				synthetic = fn.NewBuilder()
				if synthetic.diLocation.Scope != fn.diFunc.ll {
					t.Fatal("new synthetic builder has no function debug scope")
				}
			}
			defer synthetic.Dispose()

			init, next := fn.deferInitBuilder(synthetic)
			defer init.Dispose()
			if debug {
				loc := init.impl.GetCurrentDebugLocation()
				if loc.Scope != fn.diFunc.ll || loc.Line != 0 || loc.Col != 0 || !loc.InlinedAt.IsNil() {
					t.Fatalf("synthetic location = %+v, want line zero in the function scope", loc)
				}
			} else if !init.diLocation.Scope.IsNil() {
				t.Fatal("non-debug function acquired debug info")
			}
			// A recursive call is inlinable and forces LLVM's !dbg verifier to
			// exercise the same rule as calls in generated defer paths.
			call := init.impl.CreateCall(fn.impl.GlobalValueType(), fn.impl, nil, "")
			if debug && call.InstructionDebugLoc().IsNil() {
				t.Fatal("synthetic call has no debug location")
			}
			init.Jump(next)
			if debug {
				pkg.FinalizeDebug()
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid synthetic debug metadata: %v\n%s", err, pkg.Module().String())
			}
		})
	}
}

func TestSetBlockRestoresTrackedDebugLocation(t *testing.T) {
	prog := NewProgram(&Target{OptLevel: optlevel.O0})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.InitDebug("p", "example.com/p", token.NewFileSet())
	fn := pkg.NewFunc("example.com/p.f", NoArgsNoRet, InGo)
	builder := fn.MakeBody(2)
	defer builder.Dispose()
	pos := token.Position{Filename: "blocks.go", Line: 3, Column: 2}
	builder.DebugFunction(fn, nil, pos, pos)
	builder.Jump(fn.Block(1))

	// Model LLVM losing its current location during a synthetic CFG rewrite.
	// SetBlock must restore the Go-side shadow before emitting in the new block.
	builder.impl.SetCurrentDebugLocation(0, 0, llvm.Metadata{}, llvm.Metadata{})
	builder.SetBlock(fn.Block(1))
	call := builder.impl.CreateCall(fn.impl.GlobalValueType(), fn.impl, nil, "")
	loc := call.InstructionDebugLoc()
	if loc.IsNil() || loc.LocationLine() != uint(pos.Line) || loc.LocationColumn() != uint(pos.Column) {
		t.Fatalf("restored call location = %+v, want %s:%d:%d", loc, pos.Filename, pos.Line, pos.Column)
	}
	builder.Return()

	pkg.FinalizeDebug()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid restored debug metadata: %v\n%s", err, pkg.Module().String())
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

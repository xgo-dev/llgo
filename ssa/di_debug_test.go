package ssa

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

// The synthetic map/channel DWARF follows these runtime fields by name and
// replaces their pointer types. Check the real declarations, not only the
// reduced runtime package used by the metadata unit tests below.
func TestDebugRuntimeContainerFieldContract(t *testing.T) {
	for _, test := range []struct {
		file  string
		types map[string]map[string]string
	}{
		{"../runtime/internal/runtime/z_chan.go", map[string]map[string]string{
			"Chan":      {"sendq": "chanWaitq", "recvq": "chanWaitq"},
			"chanWaitq": {"first": "*chanWaiter", "last": "*chanWaiter"},
			"chanWaiter": {
				"prev": "*chanWaiter", "next": "*chanWaiter", "all": "*chanWaiter",
				"ch": "*Chan", "elem": "unsafe.Pointer",
			},
		}},
		{"../runtime/internal/runtime/map.go", map[string]map[string]string{
			"hmap": {"buckets": "unsafe.Pointer", "oldbuckets": "unsafe.Pointer"},
		}},
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, test.file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for name, want := range test.types {
			t.Run(name, func(t *testing.T) {
				object := file.Scope.Lookup(name)
				if object == nil {
					t.Fatalf("debugger runtime type %s missing in %s", name, test.file)
				}
				structure, ok := object.Decl.(*ast.TypeSpec).Type.(*ast.StructType)
				if !ok {
					t.Fatalf("debugger runtime type %s is no longer a struct", name)
				}
				fields := make(map[string]string)
				for _, field := range structure.Fields.List {
					var spelling bytes.Buffer
					if err := format.Node(&spelling, fset, field.Type); err != nil {
						t.Fatal(err)
					}
					for _, fieldName := range field.Names {
						fields[fieldName.Name] = spelling.String()
					}
				}
				for field, typ := range want {
					if got := fields[field]; got != typ {
						t.Errorf("debugger field %s.%s = %q, want %q; update the synthetic DWARF contract", name, field, got, typ)
					}
				}
			})
		}
	}
}

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
	fn := pkg.NewFunc("debugTypes", NoArgsNoRet, GoDecl)
	builder := fn.NewBuilder()
	defer builder.impl.Dispose()
	for _, name := range []string{"Link", "Peano"} {
		object := typesPkg.Scope().Lookup(name)
		pos := fset.Position(object.Pos())
		global := pkg.NewVar("example.com/p."+name, types.NewPointer(object.Type()), GoDecl)
		builder.DIGlobal(global.Expr, name, pos)
	}

	decl := file.Decls[2].(*ast.FuncDecl)
	object := typesPkg.Scope().Lookup("inspect").(*types.Func)
	function := pkg.NewFunc("example.com/p.inspect", object.Type().(*types.Signature), GoDecl)
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

func TestDebugParameterHomes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		opt       optlevel.Level
		wantHomes bool
	}{
		{"O0", optlevel.O0, true},
		{"O2", optlevel.O2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "params.go", `package p
func inspect(first, second int) {}
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			typesPkg, err := (&types.Config{}).Check("example.com/p", fset, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			object := typesPkg.Scope().Lookup("inspect").(*types.Func)
			signature := object.Type().(*types.Signature)

			prog := NewProgram(&Target{OptLevel: tc.opt})
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
			pkg := prog.NewPackage("p", "example.com/p")
			pkg.InitDebug("p", "example.com/p", fset)
			function := pkg.NewFunc("example.com/p.inspect", signature, GoDecl)
			builder := function.MakeBody(1)
			defer builder.Dispose()
			decl := file.Decls[0].(*ast.FuncDecl)
			builder.DebugFunction(
				function,
				object.Scope(),
				fset.Position(object.Pos()),
				fset.Position(decl.Body.Lbrace),
			)

			first := signature.Params().At(0)
			firstPos := fset.Position(first.Pos())
			firstVar := builder.DIVarParam(function, firstPos, first.Name(), prog.Int(), 1)
			home := builder.DIParamWithHome(first, function.Param(0), firstVar, function, firstPos, function.Block(0))
			if got := !home.IsNil(); got != tc.wantHomes {
				t.Fatalf("stable parameter home: %v, want %v", got, tc.wantHomes)
			}
			if !home.IsNil() {
				builder.DIStore(home, function.Param(0))
			}

			second := signature.Params().At(1)
			secondPos := fset.Position(second.Pos())
			secondVar := builder.DIVarParam(function, secondPos, second.Name(), prog.Int(), 2)
			builder.DIParam(second, function.Param(1), secondVar, function, secondPos, function.Block(0))
			builder.Return()
			builder.EndBuild()
			pkg.FinalizeDebug()

			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("parameter debug metadata is invalid: %v\n%s", err, pkg.Module().String())
			}
			ir := pkg.Module().String()
			hasDeclare := strings.Contains(ir, "#dbg_declare")
			hasValue := strings.Contains(ir, "#dbg_value")
			if hasDeclare != tc.wantHomes || hasValue == tc.wantHomes {
				t.Fatalf("debug records: declare=%v value=%v, want homes=%v\n%s",
					hasDeclare, hasValue, tc.wantHomes, ir)
			}
			if tc.wantHomes {
				stores := 0
				for _, line := range strings.Split(ir, "\n") {
					if strings.Contains(line, " load ") {
						t.Fatalf("debug home emits an unused load: %s", line)
					}
					if strings.Contains(line, "store ") {
						stores++
						if strings.Contains(line, "!dbg") {
							t.Fatalf("debug home store has a source location: %s", line)
						}
					}
				}
				if stores < 3 {
					t.Fatalf("found %d debug home stores, want at least 3\n%s", stores, ir)
				}
			}
		})
	}
}

func TestDebugGoTypeEncodings(t *testing.T) {
	for _, packageName := range []string{"p", "main"} {
		t.Run(packageName, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "types.go", `package `+packageName+`
type Named int64
type Large [129]byte
type Recursive struct { Next *Recursive }
type Shape struct {
	Complex complex128
	Text string
	Values []Named
	Lookup map[string]Named
	LargeLookup map[Large]Large
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
			pkg := prog.NewPackage(packageName, "example.com/p")
			pkg.InitDebug(packageName, "example.com/p", fset)

			shape := typesPkg.Scope().Lookup("Shape").Type()
			global := pkg.NewVar("example.com/p.GlobalShape", types.NewPointer(shape), GoDecl)
			fn := pkg.NewFunc("debugTypes", NoArgsNoRet, GoDecl)
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
				`name: "hash<string,example.com/p.Named>"`,
				`name: "bucket<string,example.com/p.Named>"`,
				`name: "indirectkeys"`,
				`name: "indirectvalues"`,
				`name: "chan example.com/p.Named"`,
				`name: "hchan<example.com/p.Named>"`,
				`name: "waitq<example.com/p.Named>"`,
				`name: "sudog<example.com/p.Named>"`,
				`name: "example.com/p.Recursive"`,
			} {
				if packageName == "main" {
					want = strings.ReplaceAll(want, "example.com/p", "main")
				}
				if !strings.Contains(ir, want) {
					t.Errorf("module is missing %q:\n%s", want, ir)
				}
			}
		})
	}
}

func TestWindowsDebugPointerParameter(t *testing.T) {
	const goarch = "amd64"
	for _, test := range []struct {
		goos        string
		wantDeclare bool
	}{
		{goos: "linux", wantDeclare: true},
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
			fn := pkg.NewFunc("example.com/p.f", sig, GoDecl)
			builder := fn.MakeBody(1)
			defer builder.Dispose()
			pos := fset.Position(param.Pos())
			builder.DebugFunction(fn, object.Scope(), fset.Position(object.Pos()), pos)
			debugParam := builder.DIVarParam(fn, pos, param.Name(), prog.Type(param.Type(), GoDecl), 1)
			builder.DIParam(param, fn.Param(0), debugParam, fn, pos, fn.Block(0))
			for _, kind := range []types.BasicKind{types.Uint, types.Uintptr} {
				typ := types.Typ[kind]
				global := pkg.NewVar("example.com/p."+typ.Name(), types.NewPointer(typ), GoDecl)
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
	fn := pkg.NewFunc("example.com/p.f", sig, GoDecl)
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
	fn := pkg.NewFunc("example.com/p.f", object.Type().(*types.Signature), GoDecl)
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
			fn := pkg.NewFunc("example.com/p.f", NoArgsNoRet, GoDecl)
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
	fn := pkg.NewFunc("example.com/p.f", NoArgsNoRet, GoDecl)
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
	}
	for name, fields := range members {
		obj := types.NewTypeName(token.NoPos, pkg, name, nil)
		types.NewNamed(obj, types.NewStruct(fields, nil), nil)
		pkg.Scope().Insert(obj)
	}
	mapObj := types.NewTypeName(token.NoPos, pkg, "Map", nil)
	types.NewNamed(mapObj, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "count", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "flags", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "B", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "noverflow", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "hash0", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "buckets", unsafePointer, false),
		types.NewField(token.NoPos, pkg, "oldbuckets", unsafePointer, false),
		types.NewField(token.NoPos, pkg, "nevacuate", types.Typ[types.Uintptr], false),
		types.NewField(token.NoPos, pkg, "extra", unsafePointer, false),
	}, nil), nil)
	pkg.Scope().Insert(mapObj)

	waiterObj := types.NewTypeName(token.NoPos, pkg, "chanWaiter", nil)
	waiter := types.NewNamed(waiterObj, nil, nil)
	queueObj := types.NewTypeName(token.NoPos, pkg, "chanWaitq", nil)
	queue := types.NewNamed(queueObj, nil, nil)
	chanObj := types.NewTypeName(token.NoPos, pkg, "Chan", nil)
	channel := types.NewNamed(chanObj, nil, nil)
	waiterPtr := types.NewPointer(waiter)
	waiter.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "prev", waiterPtr, false),
		types.NewField(token.NoPos, pkg, "next", waiterPtr, false),
		types.NewField(token.NoPos, pkg, "all", waiterPtr, false),
		types.NewField(token.NoPos, pkg, "ch", types.NewPointer(channel), false),
		types.NewField(token.NoPos, pkg, "elem", unsafePointer, false),
	}, nil))
	queue.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "first", waiterPtr, false),
		types.NewField(token.NoPos, pkg, "last", waiterPtr, false),
	}, nil))
	channel.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "qcount", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "dataqsiz", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "buf", unsafePointer, false),
		types.NewField(token.NoPos, pkg, "elemsize", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "closed", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "recvx", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "sendx", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "sendq", queue, false),
		types.NewField(token.NoPos, pkg, "recvq", queue, false),
	}, nil))
	pkg.Scope().Insert(waiterObj)
	pkg.Scope().Insert(queueObj)
	pkg.Scope().Insert(chanObj)
	return pkg
}

func TestDebugMapSnapshotUsesMapValueStorage(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux-gnu"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			prog.SetRuntime(newDebugRuntimePackage())
			pkg := prog.NewPackage("mapsnapshot", "mapsnapshot")
			pkg.InitDebug("mapsnapshot", "mapsnapshot", token.NewFileSet())
			mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
			mapVar := types.NewVar(token.NoPos, nil, "mapping", mapType)
			chanVar := types.NewVar(token.NoPos, nil, "queue", types.NewChan(types.SendRecv, types.Typ[types.Int]))
			sig := types.NewSignatureType(nil, nil, nil,
				types.NewTuple(mapVar, chanVar), nil, false)
			fn := pkg.NewFunc("snapshot", sig, GoDecl)
			b := fn.MakeBody(2)
			defer b.Dispose()
			pos := token.Position{Filename: "snapshot.go", Line: 1, Column: 1}
			b.DebugFunction(fn, nil, pos, pos)
			b.Jump(fn.Block(1))
			b.SetBlock(fn.Block(1))
			value := fn.Param(0)
			home, store := b.constructDebugAddrWithStore(value)
			// A Go map value holds a header pointer. Its debug snapshot must
			// reserve that pointer's physical storage, not the Map header itself.
			// Wasm32 uses an eight-byte Go slot despite four-byte host pointers.
			want := prog.storageType(value.Type)
			if got := home.impl.AllocatedType(); got != want {
				t.Fatalf("map snapshot reserves %s, want map-value storage %s", got.String(), want.String())
			}
			if home.impl.InstructionParent() != fn.impl.EntryBasicBlock() {
				t.Fatal("map snapshot reserves storage inside the loop")
			}
			if store.impl.InstructionParent() != fn.Block(1).first {
				t.Fatal("map snapshot does not update at the source location")
			}
			// Local map/channel values use their pointer SSA values directly.
			// Do not route them through a frame-index plus DW_OP_deref: Wasm's
			// backend can discard that expression after promoting the slot.
			for i, variable := range []*types.Var{mapVar, chanVar} {
				value := fn.Param(i)
				dv := b.DIVarAuto(fn, pos, variable.Name(), value.Type)
				b.DIValue(variable, value, dv, fn, pos, fn.Block(1))
			}
			b.Return()
			b.EndBuild()
			pkg.FinalizeDebug()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("map snapshot module is invalid: %v\n%s", err, pkg.String())
			}
			ir := fn.impl.String()
			for i := range 2 {
				if !strings.Contains(ir, "#dbg_value("+fn.Param(i).impl.String()+",") {
					t.Fatalf("pointer value %d has no direct debug location:\n%s", i, ir)
				}
			}
			if strings.Contains(ir, "DW_OP_deref") {
				t.Fatalf("pointer debug values use an indirect snapshot:\n%s", ir)
			}
		})
	}
}

func TestInlineAsmNoDebugPreservesBuilderLocation(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "asm.go", `package p
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
	fn := pkg.NewFunc("example.com/p.f", object.Type().(*types.Signature), GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()
	pos := fset.Position(decl.Body.Lbrace)
	b.DebugFunction(fn, object.Scope(), fset.Position(object.Pos()), pos)
	b.DISetCurrentDebugLocation(fn, pos)
	b.InlineAsmNoDebug("nop")
	b.Return()
	b.EndBuild()
	pkg.FinalizeDebug()

	asm := fn.impl.EntryBasicBlock().FirstInstruction()
	if !asm.InstructionDebugLoc().IsNil() {
		t.Fatal("inline assembly retained the current debug location")
	}
	ret := llvm.NextInstruction(asm)
	if ret.IsNil() || ret.InstructionDebugLoc().IsNil() {
		t.Fatal("inline assembly cleared the builder debug location")
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("inline assembly debug metadata is invalid: %v\n%s", err, pkg.Module().String())
	}
}

func TestDebugTupleUsesStorageAlignment(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	// A self-hosted compiler gets strict LLGoSizes from go/types, which
	// correctly rejects Tuple as a source-language storage type.
	prog.TypeSizes(prog.sourceSizes)
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.InitDebug("p", "example.com/p", token.NewFileSet())
	results := types.NewTuple(
		types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int64]),
		types.NewVar(token.NoPos, nil, "y", types.Typ[types.Int64]),
	)
	fn := pkg.NewFunc("example.com/p.pair", types.NewSignatureType(nil, nil, nil, nil, results, false), GoDecl)
	body := fn.MakeBody(1)
	defer body.Dispose()
	pos := token.Position{Filename: "pair.go", Line: 1, Column: 1}
	body.DebugFunction(fn, nil, pos, pos)
	body.Return(prog.Zero(prog.Int64()), prog.Zero(prog.Int64()))
	body.EndBuild()
	pkg.FinalizeDebug()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

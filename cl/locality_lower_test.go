package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/locality"
	localitylayout "github.com/xgo-dev/llgo/internal/locality/layout"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
)

func localitySSAGlobal(t *testing.T, path string) (*types.Package, *ssa.Global) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", "package locality\nvar Value int\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	info := newLocalityTypeInfo()
	pkg, err := (&types.Config{}).Check(path, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, []*ast.File{file}, info, true)
	global, ok := ssaPkg.Members["Value"].(*ssa.Global)
	if !ok {
		t.Fatalf("Value SSA member = %T", ssaPkg.Members["Value"])
	}
	return pkg, global
}

func TestLocalityLoweringResolution(t *testing.T) {
	typesPkg, global := localitySSAGlobal(t, "example.com/lowering")
	prog := ssatest.NewProgram(t, nil)
	llvmPkg := prog.NewPackage(typesPkg.Name(), typesPkg.Path())
	ctx := &context{prog: prog, pkg: llvmPkg, goTyps: typesPkg}

	if variable, ok, err := ctx.localVariableFor(llvmPkg, global, true); err != nil || ok || variable != nil {
		t.Fatalf("ordinary localVariableFor = %+v, %v, %v", variable, ok, err)
	}
	name := llssa.FullName(typesPkg, global.Name())
	prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
	prog.SetLocalStorage(name, llssa.LocalStorageNativeTLS)
	variable, ok, err := ctx.localVariableFor(llvmPkg, global, true)
	if err != nil || !ok || variable.planned.Storage != localitylayout.StorageNativeTLS {
		t.Fatalf("local localVariableFor = %+v, %v, %v", variable, ok, err)
	}
	ctx.locality.variables = map[*ssa.Global]*localVariable{global: variable}
	if !ctx.localityAllowsGlobalDebug(global) {
		t.Fatal("native TLS was hidden from global debug info")
	}
	variable.planned.Storage = localitylayout.StoragePackage
	if ctx.localityAllowsGlobalDebug(global) {
		t.Fatal("package storage was exposed as a fixed debug global")
	}
	if !ctx.localityAllowsGlobalDebug(new(ssa.Global)) {
		t.Fatal("ordinary global was hidden from debug info")
	}
	if owner, err := ctx.localPackageFor(typesPkg, llvmPkg, true); err != nil || owner != variable.owner {
		t.Fatalf("cached localPackageFor = %p, %v; want %p", owner, err, variable.owner)
	}

	loaded := types.NewPackage("example.com/loaded", "loaded")
	loaded.Scope().Insert(types.NewVar(token.NoPos, loaded, "Value", types.Typ[types.Int]))
	ctx.loaded = map[*types.Package]*pkgInfo{loaded: {kind: PkgDeclOnly}}
	if got := ctx.localTypesPackage("example.com/loaded.Value"); got != loaded {
		t.Fatalf("loaded localTypesPackage = %v, want %v", got, loaded)
	}
	if got := (&context{goProg: global.Pkg.Prog}).localTypesPackage(name); got != typesPkg {
		t.Fatalf("SSA-program localTypesPackage = %v, want %v", got, typesPkg)
	}
	if got := (&context{}).localTypesPackage("example.com/missing.Value"); got != nil {
		t.Fatalf("missing localTypesPackage = %v", got)
	}
	if owner, err := ctx.localPackageFor(nil, llvmPkg, false); err != nil || owner != nil {
		t.Fatalf("nil localPackageFor = %v, %v", owner, err)
	}
	empty := types.NewPackage("example.com/empty", "empty")
	if owner, err := ctx.localPackageFor(empty, llvmPkg, false); err != nil || owner != nil {
		t.Fatalf("empty localPackageFor = %v, %v", owner, err)
	}
}

func TestLocalityLoweringDeclarationOnlyState(t *testing.T) {
	typesPkg, global := localitySSAGlobal(t, "example.com/declaration")
	name := llssa.FullName(typesPkg, global.Name())
	prog := ssatest.NewProgram(t, nil)
	prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
	prog.SetLocalStorage(name, llssa.LocalStorageNativeTLS)
	llvmPkg := prog.NewPackage(typesPkg.Name(), typesPkg.Path())
	ctx := &context{
		prog:   prog,
		pkg:    llvmPkg,
		goTyps: typesPkg,
		locality: localityLowering{
			variables: make(map[*ssa.Global]*localVariable),
		},
	}

	addr := ctx.localVariableAddr(nil, global, llssa.VariableLocality{Info: llssa.LocalityInfo{Locality: llssa.ThreadLocal}}, name)
	if addr != ctx.locality.variables[global].owner.direct[name].Expr {
		t.Fatal("localVariableAddr did not return declaration-only TLS storage")
	}

	owner := &localPackage{plan: localitylayout.Package{Path: typesPkg.Path()}}
	initializer := ctx.buildLocalInitializer(llvmPkg, owner, locality.Thread, []localitylayout.Initializer{{Name: typesPkg.Path() + ".initLocal", Order: 1}}, false)
	if initializer.dispatch.HasBody() || initializer.ensure.HasBody() {
		t.Fatal("declaration-only initializer unexpectedly defined a body")
	}
}

func TestLocalityLoweringDiagnostics(t *testing.T) {
	typesPkg, global := localitySSAGlobal(t, "example.com/diagnostic")
	name := llssa.FullName(typesPkg, global.Name())
	newProgram := func() llssa.Program {
		prog := ssatest.NewProgram(t, nil)
		prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
		return prog
	}

	t.Run("missing types package", func(t *testing.T) {
		ctx := &context{prog: newProgram()}
		if _, _, err := ctx.localVariableFor(nil, global, false); err == nil || !strings.Contains(err.Error(), "missing types package") {
			t.Fatalf("localVariableFor error = %v", err)
		}
	})

	t.Run("invalid plan", func(t *testing.T) {
		prog := newProgram()
		prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal, HasInitializer: true})
		ctx := &context{prog: prog, goTyps: typesPkg}
		if _, err := ctx.localPackageFor(typesPkg, nil, false); err == nil || !strings.Contains(err.Error(), "inconsistent initializer metadata") {
			t.Fatalf("localPackageFor error = %v", err)
		}
		if _, _, err := ctx.localVariableFor(nil, global, false); err == nil || !strings.Contains(err.Error(), "inconsistent initializer metadata") {
			t.Fatalf("localVariableFor error = %v", err)
		}
	})

	t.Run("stale plan", func(t *testing.T) {
		prog := newProgram()
		ctx := &context{prog: prog, goTyps: typesPkg}
		ctx.locality.packages = map[string]*localPackage{
			typesPkg.Path(): {plan: localitylayout.Package{Path: typesPkg.Path()}},
		}
		if _, _, err := ctx.localVariableFor(nil, global, false); err == nil || !strings.Contains(err.Error(), "missing locality layout") {
			t.Fatalf("localVariableFor error = %v", err)
		}
	})

	t.Run("local linkname", func(t *testing.T) {
		prog := newProgram()
		other := typesPkg.Path() + ".Other"
		prog.SetLinkname(name, other)
		prog.SetLinkname(other, name)
		ctx := &context{prog: prog, goTyps: typesPkg}
		if _, _, err := ctx.localVariableFor(nil, global, false); err == nil || !strings.Contains(err.Error(), "cannot use go:linkname") {
			t.Fatalf("localVariableFor error = %v", err)
		}
		assertLocalityPanic(t, "resolveLocality", func() { ctx.resolveLocality(typesPkg, name) })
		assertLocalityPanic(t, "prepareLocalVariables", func() { ctx.prepareLocalVariables(nil, nil) })
		assertLocalityPanic(t, "localVariableAddr", func() {
			ctx.localVariableAddr(nil, global, llssa.VariableLocality{Info: llssa.LocalityInfo{Locality: llssa.ThreadLocal}}, name)
		})
	})

	t.Run("imported metadata error", func(t *testing.T) {
		current := types.NewPackage("example.com/current", "current")
		prog := newProgram()
		other := typesPkg.Path() + ".Other"
		prog.SetLinkname(name, other)
		prog.SetLinkname(other, name)
		ctx := &context{prog: prog, goTyps: current}
		assertLocalityPanic(t, "prepare imported local", func() { ctx.prepareLocalVariables(nil, []*ssa.Global{global}) })
	})

	t.Run("missing prepared layout", func(t *testing.T) {
		ctx := &context{prog: newProgram()}
		assertLocalityPanic(t, "localityGlobalStorage", func() {
			ctx.localityGlobalStorage(nil, global, name, global.Type(), llssa.InGo)
		})
	})

	t.Run("missing address metadata", func(t *testing.T) {
		ctx := &context{prog: ssatest.NewProgram(t, nil)}
		assertLocalityPanic(t, "localVariableAddr", func() {
			ctx.localVariableAddr(nil, global, llssa.VariableLocality{Info: llssa.LocalityInfo{Locality: llssa.ThreadLocal}}, name)
		})
	})

	t.Run("missing native storage", func(t *testing.T) {
		ctx := &context{locality: localityLowering{variables: map[*ssa.Global]*localVariable{
			global: {
				planned: localitylayout.Variable{Declaration: localitylayout.Declaration{Name: name}, Storage: localitylayout.StorageNativeTLS},
				owner:   &localPackage{direct: map[string]llssa.Global{}, init: map[locality.Kind]*localInitializer{}},
			},
		}}}
		assertLocalityPanic(t, "localVariableAddr", func() {
			ctx.localVariableAddr(nil, global, llssa.VariableLocality{Info: llssa.LocalityInfo{Locality: llssa.ThreadLocal}}, name)
		})
	})
}

func assertLocalityPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s did not panic", name)
		}
	}()
	fn()
}

package cl

import (
	"go/ast"
	"go/parser"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llgo/internal/llvmgen/abi"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/typepatch"
	"github.com/xgo-dev/llvm"
	gossa "golang.org/x/tools/go/ssa"
)

func TestCrossPackageInlineMethods(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
type Number uint64
func (n Number) Add(x uint64) uint64 { return uint64(n) + x }
//go:noinline
func (n Number) Kept(x uint64) uint64 { return uint64(n) + x }
type Box struct { X uint64 }
func (b Box) Value() uint64 { return b.X }
func (b *Box) Read() uint64 { if b == nil { return 7 }; return b.X }
func (b *Box) Address() *uint64 { return &b.X }
`, "example.com/root", `package root
import "example.com/dep"
func Add(n dep.Number, x uint64) uint64 { return n.Add(x) }
func Kept(n dep.Number, x uint64) uint64 { return n.Kept(x) }
func Value(b dep.Box) uint64 { return b.Value() }
func Read(b *dep.Box) uint64 { return b.Read() }
func Address(b *dep.Box) *uint64 { return b.Address() }
`)
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		for _, level := range []optlevel.Level{optlevel.O0, optlevel.Os, optlevel.O3} {
			t.Run(target.GOOS+"/"+target.GOARCH+"/"+level.String(), func(t *testing.T) {
				target.OptLevel = level
				prog := newLLSSAProgForTarget(t, &target)
				defer prog.Dispose()
				tracking := NewCallerTracking()
				tracking.Precompute([]*gossa.Package{dep, root})
				owner, _, err := NewPackageExWithEmbed(prog, tracking, nil, nil, dep, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer owner.Module().Dispose()
				pkg, _, err := NewPackageExWithEmbed(prog, tracking, nil, nil, root, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				mod := pkg.Module()
				defer mod.Dispose()
				for _, name := range []string{"Number.Add", "Box.Value", "(*Box).Read", "(*Box).Address"} {
					symbol := "example.com/dep." + name
					fn := mustNamedFunction(t, mod, symbol)
					if level == optlevel.O0 {
						if !fn.IsDeclaration() {
							t.Fatalf("method imported at O0: %s", fn.String())
						}
					} else if fn.IsDeclaration() || fn.Linkage() != llvm.AvailableExternallyLinkage {
						t.Fatalf("missing optimization-only method body: %s", fn.String())
					}
					if fn := mustNamedFunction(t, owner.Module(), symbol); fn.IsDeclaration() || fn.Linkage() != llvm.ExternalLinkage {
						t.Fatalf("method lost its defining package's entry: %s", fn.String())
					}
				}
				kept := mustNamedFunction(t, mod, "example.com/dep.Number.Kept")
				if !kept.IsDeclaration() || kept.GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() {
					t.Fatalf("noinline method changed: %s", kept.String())
				}
				if level != optlevel.O0 {
					address := mustNamedFunction(t, mod, "example.com/dep.(*Box).Address")
					if !strings.Contains(address.String(), "AssertNilDeref") {
						t.Fatalf("imported method lost field-address nil check: %s", address.String())
					}
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if level == optlevel.O0 {
					return
				}
				runModulePasses(t, prog, mod, "default<"+level.String()+">")
				for _, name := range []string{"Add", "Value", "Read"} {
					if ir := mustNamedFunction(t, mod, "example.com/root."+name).String(); strings.Contains(ir, "call ") {
						t.Fatalf("method call survived optimization:\n%s", ir)
					}
				}
				buf, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				buf.Dispose()
			})
		}
	}
}

func TestCrossPackageInlineMethodPolicy(t *testing.T) {
	pkg, _, _ := buildGoSSAPkg(t, `package policy
type T int
func (x T) Small() int { return int(x) }
//go:noinline
func (x T) Kept() int { return int(x) }
func (x T) Recursive() int { return x.Recursive() }
func (x T) Loop() int { for x > 0 { x-- }; return int(x) }
func (x T) Variadic(v ...int) int { return len(v) }
func (x T) Closure() func() int { return func() int { return int(x) } }
func (x T) Recover() any { return recover() }
type Generic[T any] struct { X T }
func (x Generic[T]) Value() T { return x.X }
func Use(x Generic[int]) int { return x.Value() }
type Outer struct { T }
`)
	facts := newInlineFacts()
	for _, typeName := range []string{"T", "Generic"} {
		named := pkg.Type(typeName).Type().(*types.Named)
		for method := range named.Methods() {
			fn := pkg.Prog.FuncValue(method)
			if got, want := facts.cost(fn) <= inlineBodyBudget, fn.Name() == "Small"; got != want {
				t.Errorf("%s eligible = %v, want %v", fn, got, want)
			}
		}
	}
	for _, block := range pkg.Func("Use").Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*gossa.Call); ok && facts.cost(call.Call.StaticCallee()) <= inlineBodyBudget {
				t.Fatal("generic method instance accepted")
			}
		}
	}
	outer := pkg.Type("Outer").Type()
	selection := pkg.Prog.MethodSets.MethodSet(outer).Lookup(pkg.Pkg, "Small")
	if fn := pkg.Prog.MethodValue(selection); fn.Synthetic == "" || facts.cost(fn) <= inlineBodyBudget {
		t.Fatal("promoted-method wrapper accepted")
	}
}

func TestCrossPackageInlineMethodCallerFrame(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
import "runtime"
type T int
func (T) Caller() uintptr { pc, _, _, _ := runtime.Caller(0); return pc }
`, "example.com/root", `package root
import "example.com/dep"
func Caller(x dep.T) uintptr { return x.Caller() }
`)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	tracking := NewCallerTracking()
	tracking.Precompute([]*gossa.Package{dep, root})
	pkg, _, err := NewPackageExWithEmbed(prog, tracking, nil, nil, root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pkg.Module().Dispose()
	if fn := mustNamedFunction(t, pkg.Module(), "example.com/dep.T.Caller"); !fn.IsDeclaration() {
		t.Fatalf("method needing its own caller frame was imported: %s", fn.String())
	}
}

func buildInlinePatch(t *testing.T, original, replacement, caller string) (*gossa.Package, *gossa.Package, Patch) {
	t.Helper()
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", original, "example.com/root", caller)
	fset := dep.Prog.Fset
	file, err := parser.ParseFile(fset, "replacement.go", replacement, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Instances:  map[*ast.Ident]types.Instance{},
	}
	conf := types.Config{}
	typesPkg, err := conf.Check(abi.PatchPathPrefix+dep.Pkg.Path(), fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	alt := dep.Prog.CreatePackage(typesPkg, []*ast.File{file}, info, true)
	alt.Build()
	return dep, root, Patch{Alt: alt, Types: typepatch.Clone(typesPkg)}
}

func TestCrossPackageInlinePatchedMethods(t *testing.T) {
	for _, rebound := range []bool{false, true} {
		name := "original package identity"
		if rebound {
			name = "merged package identity"
		}
		t.Run(name, func(t *testing.T) {
			dep, root, patch := buildInlinePatch(t, `package dep
type T struct { X int64 }
func (t T) Value() int64 { return t.X + 1 }
func (t *T) Pointer() int64 { if t == nil { return 2 }; return t.X + 3 }
type U int64
func (u U) Value() int64 { return int64(u) + 4 }
`, `package dep
type T struct { X int64 }
func (t T) value() int64 { return t.X + 41 }
func (t T) Value() int64 { return t.value() }
func (t *T) Pointer() int64 { if t == nil { return 43 }; return t.X + 44 }
type U int64
func (u U) Value() int64 { return int64(u) + 83 }
func Value() int64 { return -1 }
`, `package root
import "example.com/dep"
func Value() int64 { return (dep.T{X: 1}).Value() }
func Nil() int64 { return (*dep.T)(nil).Pointer() }
func Other() int64 { return dep.U(1).Value() }
`)
			typepatch.Merge(patch.Types, dep.Pkg, nil, false)
			if rebound {
				// Package lowering rebinds Pkg to the merged type scope. SSA
				// members still carry the objects registered with the program.
				dep.Pkg, patch.Alt.Pkg = patch.Types, patch.Types
			}
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			tracking := NewCallerTracking()
			tracking.Precompute([]*gossa.Package{dep, patch.Alt, root})
			pkg, _, err := NewPackageExWithEmbed(prog, tracking, Patches{"example.com/dep": patch}, nil, root, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			defer mod.Dispose()
			for _, method := range []string{"T.Value", "T.value", "(*T).Pointer", "U.Value"} {
				fn := mustNamedFunction(t, mod, "example.com/dep."+method)
				if fn.IsDeclaration() || fn.Linkage() != llvm.AvailableExternallyLinkage {
					t.Fatalf("replacement method not imported: %s", fn.String())
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			runModulePasses(t, prog, mod, "default<O2>")
			for name, want := range map[string]string{"Value": "ret i64 42", "Nil": "ret i64 43", "Other": "ret i64 84"} {
				ir := mustNamedFunction(t, mod, "example.com/root."+name).String()
				if !strings.Contains(ir, want) {
					t.Fatalf("%s did not use the replacement body (want %s):\n%s", name, want, ir)
				}
			}
		})
	}
}

func TestCrossPackageInlinePatchedReceiverLayout(t *testing.T) {
	dep, root, patch := buildInlinePatch(t, `package dep
type T struct { opaque struct{} }
func (T) Value() int64 { return -1 }
func (*T) Pointer() int64 { return -1 }
`, `package dep
type T struct { Padding, X int64 }
func (t T) Value() int64 { return t.X }
func (t *T) Pointer() int64 { return t.X }
`, `package root
import "example.com/dep"
func Value(t dep.T) int64 { return t.Value() }
func Pointer(t *dep.T) int64 { return t.Pointer() }
`)
	typepatch.Merge(patch.Types, dep.Pkg, nil, false)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, _, err := NewPackageEx(prog, Patches{dep.Pkg.Path(): patch}, nil, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	defer mod.Dispose()
	for _, method := range []string{"T.Value", "(*T).Pointer"} {
		fn := mustNamedFunction(t, mod, "example.com/dep."+method)
		if fn.IsDeclaration() || fn.Linkage() != llvm.AvailableExternallyLinkage {
			t.Fatalf("replacement method not imported: %s", fn.String())
		}
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	runModulePasses(t, prog, mod, "default<O2>")
	for name, field := range map[string]string{"Value": ", 1", "Pointer": "i64 8"} {
		ir := mustNamedFunction(t, mod, "example.com/root."+name).String()
		if strings.Contains(ir, "call ") || !strings.Contains(ir, field) {
			t.Fatalf("%s did not inline the replacement's second field:\n%s", name, ir)
		}
	}
}

func TestCrossPackageInlinePatchedMethodGuards(t *testing.T) {
	const original = "type T int64\nfunc (x T) Value() int64 { return int64(x) + 1 }"
	const replacement = "type T int64\nfunc (x T) Value() int64 { return int64(x) + 2 }"
	for _, test := range []struct {
		name, original, replacement string
		unmapped, wantImport        bool
	}{
		{name: "value receiver", replacement: replacement, wantImport: true},
		{name: "original declaration only", original: "type T int64\nfunc (T) Value() int64", replacement: replacement, wantImport: true},
		{name: "original body not eligible", original: "type T int64\nfunc (x T) Value() int64 { for x > 0 { x-- }; return int64(x) }", replacement: replacement, wantImport: true},
		{name: "pointer receiver", original: "type T int64\nfunc (x *T) Value() int64 { return 1 }", replacement: "type T int64\nfunc (x *T) Value() int64 { return 2 }", wantImport: true},
		{name: "replacement receiver alias", replacement: "type T int64\ntype Alias = T\nfunc (x Alias) Value() int64 { return int64(x) + 2 }", wantImport: true},
		{name: "original receiver alias", original: "type T int64\ntype Alias = T\nfunc (x Alias) Value() int64 { return int64(x) + 1 }", replacement: replacement},
		{name: "original pointer receiver alias", original: "type T int64\ntype Alias = T\nfunc (x *Alias) Value() int64 { return 1 }", replacement: "type T int64\nfunc (x *T) Value() int64 { return 2 }"},
		{name: "missing type", replacement: "func Value() int64 { return 2 }"},
		{name: "missing method", replacement: "type T int64\nfunc Value() int64 { return 2 }"},
		{name: "promoted method", replacement: "type U int64\nfunc (U) Value() int64 { return 2 }\ntype T struct { U }"},
		{name: "different receiver form", replacement: "type T int64\nfunc (*T) Value() int64 { return 2 }"},
		{name: "different parameters", replacement: "type T int64\nfunc (T) Value(int64) int64 { return 2 }"},
		{name: "different result", replacement: "type T int64\nfunc (T) Value() uint64 { return 2 }"},
		{name: "missing body", replacement: "type T int64\nfunc (T) Value() int64"},
		{name: "declaration package", original: "const LLGoPackage = \"decl\"\n" + original, replacement: replacement},
		{name: "declaration replacement", replacement: "const LLGoPackage = \"decl\"\n" + replacement},
		{name: "original noinline", original: strings.Replace(original, "func", "//go:noinline\nfunc", 1), replacement: replacement},
		{name: "replacement noinline", replacement: strings.Replace(replacement, "func", "//go:noinline\nfunc", 1)},
		{name: "generic original", original: "type T[A any] int64\nfunc (x T[A]) Value() int64 { return int64(x) }", replacement: replacement},
		{name: "generic replacement", replacement: "type T[A any] int64\nfunc (x T[A]) Value() int64 { return int64(x) }"},
		{name: "receiver not mapped", replacement: replacement, unmapped: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.original == "" {
				test.original = original
			}
			dep, root, patch := buildInlinePatch(t, "package dep\n"+test.original, "package dep\n"+test.replacement,
				"package root\nimport _ \"example.com/dep\"")
			if !test.unmapped {
				typepatch.Merge(patch.Types, dep.Pkg, nil, false)
			}
			patch.Alt.Pkg = patch.Types
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg := prog.NewPackage("root", "example.com/root")
			defer pkg.Module().Dispose()
			ctx := &context{
				prog: prog, pkg: pkg, goPkg: root, goTyps: root.Pkg, goProg: root.Prog,
				patches:     Patches{dep.Pkg.Path(): patch},
				inlineFacts: newInlineFacts(),
				loaded:      map[*types.Package]*pkgInfo{dep.Pkg: {kind: PkgNormal}, patch.Types: {kind: PkgNormal}},
			}
			named := dep.Type("T").Type().(*types.Named)
			ctx.importInlineBody(dep.Prog.FuncValue(named.Method(0)))
			if got := len(ctx.inits) != 0; got != test.wantImport {
				t.Fatalf("imported = %v, want %v", got, test.wantImport)
			}
		})
	}
}

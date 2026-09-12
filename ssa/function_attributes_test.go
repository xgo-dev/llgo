package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
)

// Read the real declarations: the test must not maintain a second symbol table
// of contracts. Target-specific panic entries have function-only attributes.
func loadRuntimeSourceAttributes(t *testing.T, prog Program) {
	t.Helper()
	files, err := filepath.Glob("../runtime/internal/runtime/*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	count := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			attrs, err := funcattrs.Parse(fset, fn)
			if err != nil {
				t.Fatal(err)
			}
			if len(attrs) == 0 {
				continue
			}
			count++
			name := PkgRuntime + "." + fn.Name.Name
			if err = prog.SetFunctionAttributes(name, attrs); err != nil {
				t.Fatal(err)
			}
			for _, item := range directive.ParseGroup(fn.Doc) {
				if item.Name == "go:linkname" {
					parts := strings.Fields(item.Args)
					if len(parts) == 2 {
						prog.SetLinkname(name, parts[1])
					}
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("no annotated runtime declarations found")
	}
}

func TestSourceAttributesGCRootPolicyMatchesImportedDeclarations(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.EnableGCRoots(true)
	prog.EnableCooperativeSafepoints(true)
	loadRuntimeSourceAttributes(t, prog)
	p := prog.NewPackage("roots", "roots")
	sig := runtimeContractSignature([]types.Type{types.Typ[types.UnsafePointer]}, types.Typ[types.Int])
	fn := p.NewFunc(PkgRuntime+".MapLen", sig, InGo)
	// A caller's imported declaration must already have the same conservative
	// effects before a definition and its root plan exist.
	imported := prog.NewPackage("caller", "caller").NewFunc(PkgRuntime+".MapLen", sig, InGo)
	for _, function := range []Function{fn, imported} {
		for _, name := range []string{"memory", "nofree", "nosync", "nounwind", "willreturn"} {
			if !function.impl.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(name)).IsNil() {
				t.Errorf("declaration retained %s in instrumented target mode", name)
			}
		}
		for _, name := range []string{"readonly", "captures"} {
			if !function.impl.GetEnumAttributeAtIndex(1, llvm.AttributeKindID(name)).IsNil() {
				t.Errorf("declaration retained parameter %s in instrumented target mode", name)
			}
		}
		if function.impl.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("range")).IsNil() {
			t.Error("instrumentation dropped the independent result range")
		}
	}
	fn.MakeBody(1)
	fn.NewGCRoots(1)
	fn.CheckAttributeInstrumentation("cooperative safepoints", "memory", "capture")
}

func TestSourceAttributesHiddenEnvironmentAndGenericOrigin(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "env.go", `package p
//llgo:attr param(p) returned
//llgo:attr result(0) nonnull
func F(p *int) *int { return p }
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	a, err := funcattrs.Parse(fset, f.Decls[0].(*ast.FuncDecl))
	if err != nil {
		t.Fatal(err)
	}
	if err = prog.SetFunctionAttributes("p.F", a); err != nil {
		t.Fatal(err)
	}
	prog.SetFunctionAttributeOrigin("p.F[int]", "p.F")
	ptr := types.NewPointer(types.Typ[types.Int])
	sig := runtimeContractSignature([]types.Type{ptr}, ptr)
	for _, name := range []string{"p.F", "p.F[int]"} {
		p := prog.NewPackage(name, name)
		env := types.NewParam(token.NoPos, nil, "$env", types.Typ[types.UnsafePointer])
		fn := p.NewEnvFunc(name, sig, InGo, env, false)
		if !fn.impl.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("returned")).IsNil() {
			t.Fatal("contract moved onto hidden environment")
		}
		if fn.impl.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("returned")).IsNil() {
			t.Fatal("source parameter contract lost")
		}
		if fn.impl.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("nonnull")).IsNil() {
			t.Fatal("result contract lost")
		}
		if err = llvm.VerifyModule(p.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceNoAliasInstrumentation(t *testing.T) {
	for _, mode := range []string{"none", "roots", "safepoints"} {
		t.Run(mode, func(t *testing.T) {
			prog := NewProgram(nil)
			defer prog.Dispose()
			prog.EnableGCRoots(mode == "roots")
			prog.EnableCooperativeSafepoints(mode == "safepoints")
			attr := funcattrs.Attribute{Target: funcattrs.Target{Scope: funcattrs.Parameter, Index: 0}, Name: "noalias"}
			if err := prog.SetFunctionAttributes("p.F", []funcattrs.Attribute{attr}); err != nil {
				t.Fatal(err)
			}
			ptr := types.NewPointer(types.Typ[types.Int])
			sig := runtimeContractSignature([]types.Type{ptr}, ptr)
			for _, owner := range []string{"p", "caller"} {
				fn := prog.NewPackage(owner, owner).NewFunc("p.F", sig, InGo)
				retained := !fn.impl.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("noalias")).IsNil()
				if retained != (mode == "none") {
					t.Fatalf("%s: unexpected noalias policy in %s", owner, mode)
				}
				if mode == "none" {
					func() {
						defer func() {
							failure := recover()
							err, ok := failure.(error)
							if !ok || !strings.Contains(err.Error(), "cooperative safepoints") {
								t.Fatalf("unexpected instrumentation diagnostic: %v", failure)
							}
						}()
						fn.CheckAttributeInstrumentation("cooperative safepoints")
					}()
				} else {
					fn.CheckAttributeInstrumentation("cooperative safepoints")
				}
			}
		})
	}
}

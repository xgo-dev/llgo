package ssa

import (
	"fmt"
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

func TestSourceAttributesRejectGCRootPublication(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	loadRuntimeSourceAttributes(t, prog)
	p := prog.NewPackage("roots", "roots")
	sig := runtimeContractSignature([]types.Type{types.Typ[types.UnsafePointer]}, types.Typ[types.Int])
	fn := p.NewFunc(PkgRuntime+".MapLen", sig, InGo)
	defer func() {
		err := recover()
		if err == nil || !strings.Contains(fmt.Sprint(err), "compiler-generated GC root publication") || !strings.Contains(fmt.Sprint(err), "z_map.go:") {
			t.Fatalf("error = %v", err)
		}
	}()
	fn.NewGCRoots(1)
}

func TestSourceAttributesHiddenEnvironmentAndGenericOrigin(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "env.go", `package p
//llgo:attribute param(p) returned
//llgo:attribute result(0) nonnull
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

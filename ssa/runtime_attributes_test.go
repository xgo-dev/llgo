package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestRuntimeAttributes(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	rt, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	for _, roots := range []bool{false, true} {
		prog := NewProgram(nil)
		prog.SetRuntime(rt)
		defer prog.Dispose()
		prog.EnableGCRoots(roots)
		for _, module := range []string{PkgRuntime, "example.com/caller"} {
			pkg := prog.NewPackage("p", module)
			for _, tc := range []struct {
				name  string
				index int
				attr  string
				want  bool
			}{
				{"AllocU", 0, "nonnull", true}, {"AllocZ", -1, "allocsize", true},
				{"AllocRoot", 0, "noalias", false}, {"CStrDup", 0, "nonnull", true},
				{"StringEqual", -1, "memory", !roots}, {"StringLess", -1, "willreturn", !roots},
				{"CStrCopy", 1, "returned", !roots}, {"StringFrom", 1, "readonly", !roots},
				{"Panic", -1, "noreturn", true}, {"PanicIndex", -1, "noreturn", true},
				{"PanicSIMDUnimplemented", -1, "noreturn", true},
				{"ChanCap", -1, "memory", !roots}, {"MapLen", -1, "memory", !roots},
				{"ChanLen", -1, "memory", false}, {"PanicWrapNilPointer", -1, "noreturn", false},
				{"AssertNilDeref", -1, "noreturn", false}, {"Rethrow", -1, "noreturn", false},
				{"EfaceEqual", -1, "memory", false},
			} {
				obj := rt.Scope().Lookup(tc.name)
				if obj == nil {
					t.Fatalf("missing runtime function %s", tc.name)
				}
				fn := pkg.NewFunc(PkgRuntime+"."+tc.name, obj.Type().(*types.Signature), InGo)
				got := !fn.impl.GetEnumAttributeAtIndex(tc.index, llvm.AttributeKindID(tc.attr)).IsNil()
				if got != tc.want {
					t.Errorf("roots=%v %s: %s index %d = %v, want %v", roots, tc.name, tc.attr, tc.index, got, tc.want)
				}
			}
			// An identically named user helper must not inherit a runtime contract.
			fn := pkg.NewFunc("example.com/runtime.AllocZ", prog.tyMalloc(), InGo)
			if !fn.impl.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("nonnull")).IsNil() {
				t.Fatal("runtime contract leaked to user function")
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// Check the audited list against declarations, including target-specific files.
// This catches stale names and commented-out upstream runtime implementations.
func TestRuntimeAttributeModelSymbols(t *testing.T) {
	files, err := filepath.Glob("../runtime/internal/runtime/*.go")
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
				names[fn.Name.Name] = true
			}
		}
	}
	for name := range runtimeFunctionModels {
		if !names[name] {
			t.Errorf("modeled runtime function %s does not exist", name)
		}
	}
}

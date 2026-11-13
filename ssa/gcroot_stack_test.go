package ssa

import (
	"github.com/goplus/gogen/packages"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestTypedStackRootStorageLayouts(t *testing.T) {
	for _, tc := range []struct {
		name, triple, profile string
		word                  uint64
	}{
		{"J32", "wasm32-unknown-emscripten", "j32", 8},
		{"J64", "wasm64-unknown-emscripten", "j64", 8},
		{"W32", "wasm32-unknown-wasip1", "w32", 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			goos := "js"
			if tc.profile == "w32" {
				goos = "wasip1"
			}
			p := NewProgram(&Target{GOOS: goos, GOARCH: "wasm", LLVMTarget: tc.triple, WasmProfile: tc.profile})
			defer p.Dispose()
			p.EnableGCRoots(true)
			p.SetRuntime(func() *types.Package {
				imp := packages.NewImporter(token.NewFileSet())
				pkg, err := imp.Import(PkgRuntime)
				if err != nil {
					t.Fatal(err)
				}
				return pkg
			})
			pkg := p.NewPackage("roots", "roots")
			fn := pkg.NewFunc("roots.fields", NoArgsNoRet, GoDecl)
			b := fn.MakeBody(1)
			typ := p.Struct(p.Uintptr(), p.VoidPtr(), p.Type(types.NewSlice(types.Typ[types.Int]), GoDecl), p.Type(types.NewInterfaceType(nil, nil).Complete(), GoDecl))
			var offsets []uint64
			fn.stackRootOffsets(typ, 0, func(offset uint64) { offsets = append(offsets, offset) })
			if want := []uint64{tc.word, 2 * tc.word, 6 * tc.word}; !reflect.DeepEqual(offsets, want) {
				t.Fatalf("offsets=%v want %v", offsets, want)
			}
			root := fn.NewGCRoots(1)[0]
			object := b.Alloc(typ, false)
			b.SetGCRoot(root, object)
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%v\n%s", err, pkg.String())
			}
			ir := pkg.String()
			if !strings.Contains(ir, "__llgo_gcroot_stack_pointer_") || !strings.Contains(ir, "i32 4, i32 4") {
				t.Fatalf("missing typed map:\n%s", ir)
			}
			// A managed alloca and all field initialization must dominate the
			// first source block. No root may reference conditional storage.
			if !strings.Contains(fn.impl.EntryBasicBlock().FirstInstruction().String(), "alloca") {
				t.Fatal("missing entry storage")
			}
			pbo := llvm.NewPassBuilderOptions()
			defer pbo.Dispose()
			pkg.mod.SetDataLayout(p.DataLayout())
			pkg.mod.SetTarget(p.Target().Spec().Triple)
			if err := pkg.mod.RunPasses("default<O2>", p.TargetMachine(), pbo); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(pkg.mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

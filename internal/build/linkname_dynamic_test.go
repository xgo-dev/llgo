package build

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestWindowsDynamicImportFromGoSource(t *testing.T) {
	const source = `package dynslot
import _ "unsafe"
//go:cgo_import_dynamic dynslot.Default CloseHandle%1 "kernel32.dll"
var Default *byte
//go:cgo_import_dynamic bare_import GetCurrentProcess%0 "kernel32.dll"
//go:linkname Renamed bare_import
var Renamed *byte
`
	for _, arch := range []string{"386", "amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "imports.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			files := []*ast.File{file}
			imp := importer.Default()
			ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset,
				types.NewPackage("dynslot", "dynslot"), files, ssa.SanityCheckFunctions)
			if err != nil {
				t.Fatal(err)
			}
			prog := ssatest.NewProgramEx(t, &llssa.Target{GOOS: "windows", GOARCH: arch}, imp)
			prog.TypeSizes(types.SizesFor("gc", arch))
			pkg, _, err := cl.NewPackageEx(prog, nil, nil, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			for _, name := range []string{"dynslot.Default", "bare_import"} {
				v := mod.NamedGlobal(name)
				if v.IsNil() || v.GlobalValueType().TypeKind() != llvm.PointerTypeKind || !v.Initializer().IsNil() {
					t.Fatalf("%s must reach import lowering as an external pointer: %s", name, mod.String())
				}
			}
			if err := lowerWindowsCgoImportPointers("windows", arch, "dynslot", files, mod); err != nil {
				t.Fatal(err)
			}
			for local, remote := range map[string]string{"dynslot.Default": "CloseHandle", "bare_import": "GetCurrentProcess"} {
				fn := mod.NamedFunction(remote)
				if fn.IsNil() || fn.DLLStorageClass() != llvm.DLLImportStorageClass || mod.NamedGlobal(local).Initializer() != fn {
					t.Fatalf("%s does not contain the imported address: %s", local, mod.String())
				}
				want := llvm.CCallConv
				if arch == "386" {
					want = llvm.X86StdcallCallConv
				}
				if fn.FunctionCallConv() != want {
					t.Fatalf("%s has wrong call convention", remote)
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWindowsDynamicImportPreservesStaticDefinition(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("static-import")
	defer mod.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	global := llvm.AddGlobal(mod, ptr, "dynslot.Value")
	initial := llvm.ConstIntToPtr(llvm.ConstInt(ctx.Int64Type(), 42, false), ptr)
	global.SetInitializer(initial)
	file, err := parser.ParseFile(token.NewFileSet(), "imports.go", "package dynslot\n//go:cgo_import_dynamic dynslot.Value CloseHandle%1 \"kernel32.dll\"\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if err := lowerWindowsCgoImportPointers("windows", "amd64", "dynslot", []*ast.File{file}, mod); err != nil {
		t.Fatal(err)
	}
	if global.Initializer() != initial || !mod.NamedFunction("CloseHandle").IsNil() {
		t.Fatal("dynamic import replaced a static definition")
	}
}

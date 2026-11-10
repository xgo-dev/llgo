package build

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestLinknameNativeExportSnapshotAndWrapper(t *testing.T) {
	for _, arch := range []string{"386", "amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			files, ssaPkg := nativeExportSource(t, `package entries
import _ "unsafe"
//go:linkname Entry stdcall.native_entry
func Entry(v int32) int32 { return v+7 }
`)
			pkg, _, err := cl.NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, ssaPkg, files, nil, false, cl.Options{CExportWrappers: true})
			if err != nil {
				t.Fatal(err)
			}
			if !pkg.Module().NamedFunction("native_entry").IsNil() {
				t.Fatal("package emitted a final-link wrapper")
			}
			exports := packageCExportDeclarations(pkg)
			if len(exports) != 1 || exports[0].callConv != llssa.Stdcall || exports[0].goName != "entries.Entry" || exports[0].sig.Results().At(0).Type() != types.Typ[types.Int32] {
				t.Fatalf("incorrect export snapshot: %+v", exports)
			}
			mainPkg := prog.NewPackage("entrypoint", "entrypoint")
			ensure := declareNoArgFunc(mainPkg, "ensureInit")
			defineCExportWrappers(mainPkg, exports, ensure)
			wrapper := mainPkg.Module().NamedFunction("native_entry")
			want := llvm.CCallConv
			if arch == "386" {
				want = llvm.X86StdcallCallConv
			}
			if wrapper.IsNil() || wrapper.FunctionCallConv() != want {
				t.Fatal("final-link wrapper lost its native ABI")
			}
			for _, name := range []string{"ensureInit", "entries.Entry", "EnterForeignThread", "ExitForeignThread"} {
				if !strings.Contains(wrapper.String(), name) {
					t.Errorf("wrapper lacks %s", name)
				}
			}
			if err := llvm.VerifyModule(mainPkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func nativeExportSource(t *testing.T, source string) ([]*ast.File, *ssa.Package) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "native.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer.Default()}, fset, types.NewPackage("entries", "entries"), files, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	return files, pkg
}

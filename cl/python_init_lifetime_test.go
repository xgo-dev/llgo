package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	gpackages "github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llgo/internal/typepatch"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestPythonDeferredModuleInitialization(t *testing.T) {
	const source = `package pyconsumer

import (
	"github.com/xgo-dev/llgo/cl/_testpy/callpy/builtins"
	"github.com/xgo-dev/llgo/cl/_testpy/math"
)

var Marker int

func init() { Marker++ }

func laterBool() *builtins.Object { return builtins.Bool() }

func laterSqrt() func(*math.Object) *math.Object {
	return func(value *math.Object) *math.Object { return math.Sqrt(value) }
}
`
	for _, test := range []struct {
		name    string
		debug   bool
		patch   bool
		skipOld bool
	}{
		{name: "plain"},
		{name: "debug", debug: true},
		{name: "patch/plain", patch: true},
		{name: "patch/debug", patch: true, debug: true},
		{name: "patch_skip_old/plain", patch: true, skipOld: true},
		{name: "patch_skip_old/debug", patch: true, skipOld: true, debug: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			originalSource := source
			if test.patch {
				originalSource = `package pyconsumer
var OriginalMarker = oldMarker()
func oldMarker() int { return 7 }
`
			}
			file, err := parser.ParseFile(fset, "python_init.go", originalSource, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			importer := gpackages.NewImporter(fset)
			mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
			if test.debug {
				mode |= gossa.GlobalDebug
			}
			files := []*ast.File{file}
			pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer}, fset,
				types.NewPackage("pyconsumer", "pyconsumer"), files, mode)
			if err != nil {
				t.Fatal(err)
			}
			var patches Patches
			if test.patch {
				patchSource := source
				if test.skipOld {
					patchSource += "\n//llgo:skip init\nconst SkipOldInit = true\n"
				}
				patchFile, err := parser.ParseFile(fset, "python_patch.go", patchSource, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				patchTypes := types.NewPackage("pyconsumer", "pyconsumer")
				patchInfo := newLocalityTypeInfo()
				patchInfo.FileVersions = make(map[*ast.File]string)
				patchFiles := []*ast.File{patchFile}
				checker := types.NewChecker(&types.Config{Importer: importer}, fset, patchTypes, patchInfo)
				if err := checker.Files(patchFiles); err != nil {
					t.Fatal(err)
				}
				var createImports func(*types.Package)
				createImports = func(dependency *types.Package) {
					if pkg.Prog.Package(dependency) != nil {
						return
					}
					pkg.Prog.CreatePackage(dependency, nil, nil, true)
					for _, imported := range dependency.Imports() {
						createImports(imported)
					}
				}
				for _, dependency := range patchTypes.Imports() {
					createImports(dependency)
				}
				patchSSA := pkg.Prog.CreatePackage(patchTypes, patchFiles, patchInfo, false)
				patchSSA.Build()
				patches = Patches{"pyconsumer": {Alt: patchSSA, Types: typepatch.Clone(patchTypes)}}
				files = append(files, patchFile)
			}
			prog := ssatest.NewProgramEx(t, nil, importer)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
			prog.SetPython(func() *types.Package {
				py := types.NewPackage(llssa.PkgPython, "py")
				object := types.NewTypeName(token.NoPos, py, "Object", nil)
				types.NewNamed(object, types.NewStruct(nil, nil), nil)
				py.Scope().Insert(object)
				py.MarkComplete()
				return py
			})
			compiled, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, patches, nil,
				pkg, files, nil, false, Options{Debug: test.debug, DebugSymbols: test.debug})
			if err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal("deferred Python initialization produced invalid LLVM IR", err)
			}

			// These Python references are discovered while compiling later
			// ordinary/nested function bodies, after init's original builder
			// has finished. Their loads still belong between dependency init
			// calls and this package's user init, not at function-body teardown.
			init := compiled.Module().NamedFunction("pyconsumer.init")
			if init.IsNil() {
				t.Fatal("missing package init")
			}
			var calls []string
			for _, block := range init.BasicBlocks() {
				for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if inst.InstructionOpcode() == llvm.Call {
						calls = append(calls, inst.CalledValue().Name())
					}
				}
			}
			loads, dependencies := 0, 0
			userInit := false
			for _, call := range calls {
				switch {
				case strings.HasSuffix(call, "/builtins.init"), strings.HasSuffix(call, "/math.init"):
					if loads != 0 {
						t.Fatal("Python loads preceded dependency initialization", calls)
					}
					dependencies++
				case call == "llgoLoadPyModSyms":
					if dependencies != 2 || userInit {
						t.Fatal("Python loads are outside the post-dependency init position", calls)
					}
					loads++
				case strings.HasPrefix(call, "pyconsumer.init#"):
					if loads != 2 {
						t.Fatal("user init ran before both late-discovered Python modules", calls)
					}
					userInit = true
				}
			}
			if dependencies != 2 || loads != 2 || !userInit {
				t.Fatal("incomplete deferred initialization", calls)
			}
			if test.patch {
				oldInit := compiled.Module().NamedFunction("pyconsumer.init$hasPatch")
				oldCalls := 0
				for _, call := range calls {
					if call == "pyconsumer.init$hasPatch" {
						oldCalls++
					}
				}
				if test.skipOld {
					if oldCalls != 0 || !oldInit.IsNil() && len(oldInit.BasicBlocks()) != 0 {
						t.Fatal("skipped original initializer was retained", calls)
					}
				} else if oldCalls != 1 || oldInit.IsNil() || len(oldInit.BasicBlocks()) == 0 {
					t.Fatal("missing original initializer body or unique call", calls)
				}
			}
			object, err := prog.TargetMachine().EmitToMemoryBuffer(compiled.Module(), llvm.ObjectFile)
			if err != nil {
				t.Fatal("compile deferred initialization to an actual native object", err)
			}
			defer object.Dispose()
			if len(object.Bytes()) == 0 {
				t.Fatal("deferred initialization produced an empty object")
			}
		})
	}
}

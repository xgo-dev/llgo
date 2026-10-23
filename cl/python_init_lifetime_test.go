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
	for _, debug := range []bool{false, true} {
		name := "plain"
		if debug {
			name = "debug"
		}
		t.Run(name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "python_init.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			importer := gpackages.NewImporter(fset)
			mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
			if debug {
				mode |= gossa.GlobalDebug
			}
			files := []*ast.File{file}
			pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer}, fset,
				types.NewPackage("pyconsumer", "pyconsumer"), files, mode)
			if err != nil {
				t.Fatal(err)
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
			compiled, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil,
				pkg, files, nil, false, Options{Debug: debug, DebugSymbols: debug})
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

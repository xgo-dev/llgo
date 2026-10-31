package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	gpackages "github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llgo/internal/goembed"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func TestLinknameNativeEntriesIR(t *testing.T) {
	const source = `package native
import _ "unsafe"
//go:linkname Entry C.native_entry
func Entry(v int32) int32 { return v + 7 }
//go:linkname CallbackEntry C.native_callback_entry
func CallbackEntry(fn func(int32) int32, v int32) int32 { return fn(v)+1 }
//go:linkname StdEntry stdcall.std_entry
func StdEntry(v int32) int32 { return v + 8 }
//go:linkname Callback stdcall.native_callback
var Callback func(int32) int32
//go:linkname Scalar stdcall.native_scalar
var Scalar int32
func Use(v int32) int32 { return Callback(v) + Scalar }
func AsGo() func(int32) int32 { return Callback }
func Reset() { Callback = nil }
//go:linkname Twice stdcall.native_twice
func Twice(int32) int32
func Restore() { Callback = Twice }
//go:linkname CCallback C.c_callback
var CCallback func(int32) int32
func Capture() func() { old := CCallback; return func() { CCallback = old } }
//llgo:link Placeholder C.external_entry
func Placeholder(int32) int32 { panic("native placeholder body") }
//llgo:link GoPlaceholder plain_go_entry
func GoPlaceholder(int32) int32 { panic("Go placeholder body") }
func Calls(v int32) int32 { return Placeholder(v) + GoPlaceholder(v) }
`
	for _, triple := range []string{
		"i686-pc-windows-msvc", "x86_64-pc-windows-msvc", "aarch64-pc-windows-msvc",
		"i686-w64-windows-gnu", "x86_64-w64-windows-gnu", "aarch64-w64-windows-gnu",
	} {
		arch := map[string]string{"i686": "386", "x86_64": "amd64", "aarch64": "arm64"}[strings.Split(triple, "-")[0]]
		for _, preloaded := range []bool{false, true} {
			name := triple + "/direct"
			if preloaded {
				name = triple + "/preloaded"
			}
			t.Run(name, func(t *testing.T) {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "native.go", source, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				files := []*ast.File{file}
				imp := gpackages.NewImporter(fset)
				prog := ssatest.NewProgramEx(t, &llssa.Target{GOOS: "windows", GOARCH: arch, LLVMTarget: triple}, imp)
				prog.TypeSizes(types.SizesFor("gc", arch))
				pkg := types.NewPackage("native", "native")
				if preloaded {
					// The build preloader runs before type checking.
					if err := ParsePkgSyntax(prog, fset, pkg, files); err != nil {
						t.Fatal(err)
					}
				}
				info := newLocalityTypeInfo()
				if err := types.NewChecker(&types.Config{Importer: imp}, fset, pkg, info).Files(files); err != nil {
					t.Fatal(err)
				}
				goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
				goProg.CreatePackage(types.Unsafe, nil, nil, true)
				ssaPkg := goProg.CreatePackage(pkg, files, info, true)
				ssaPkg.Build()
				compiled, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, ssaPkg, files, goembed.VarMap{}, false, Options{PreloadedSyntax: preloaded})
				if err != nil {
					t.Fatal(err)
				}
				mod := compiled.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				cc := llvm.CCallConv
				if arch == "386" {
					cc = llvm.X86StdcallCallConv
				}
				for name, conv := range map[string]llvm.CallConv{"native_entry": llvm.CCallConv, "std_entry": cc} {
					fn := mod.NamedFunction(name)
					if fn.IsNil() || fn.BasicBlocksCount() == 0 || fn.FunctionCallConv() != conv {
						t.Fatalf("wrong native entry %s:\n%s", name, mod.String())
					}
					ir := fn.String()
					for _, boundary := range []string{"EnterForeignThread", "ExitForeignThread"} {
						if !strings.Contains(ir, boundary) {
							t.Errorf("entry %s lacks %s:\n%s", name, boundary, ir)
						}
					}
				}
				for _, name := range []string{"external_entry", "plain_go_entry"} {
					fn := mod.NamedFunction(name)
					if fn.IsNil() || fn.BasicBlocksCount() != 0 {
						t.Fatalf("placeholder emitted a definition: %s", name)
					}
				}
				if strings.Contains(mod.String(), "placeholder body") || len(compiled.ExportFuncs()) != 3 {
					t.Fatal("llgo:link participated in export/body emission")
				}
				for _, name := range []string{"native_callback", "native_scalar"} {
					g := mod.NamedGlobal(name)
					if g.IsNil() || !g.Initializer().IsNil() {
						t.Fatalf("%s must be external native storage", name)
					}
				}
				if arch == "386" && !strings.Contains(mod.NamedFunction("native.Use").String(), "call x86_stdcallcc i32") {
					t.Fatal("loaded callback lost stdcall")
				}
				// Restoring a captured Go funcval into a C function-pointer slot
				// must store only its code pointer, never the two-word funcval.
				restore := mod.NamedFunction("native.Capture$1").String()
				if !strings.Contains(restore, "store ptr") || strings.Contains(restore, "store { ptr, ptr }") {
					t.Fatalf("callback restore overwrites adjacent native storage:\n%s", restore)
				}
				assembly, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				defer assembly.Dispose()
				if arch == "386" && !strings.Contains(string(assembly.Bytes()), "_std_entry@4:") {
					t.Fatal("stdcall entry lost its 386 symbol decoration")
				}
			})
		}
	}
}

func TestLinknameNativeSyntaxDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"bodyless", "//llgo:link F C.f\nfunc F()", "requires a placeholder function body"},
		{"bodyless-go", "//llgo:link F bare\nfunc F()", "requires a placeholder function body"},
		{"C-initializer", "//go:linkname V C.v\nvar V int = 0", "cannot have a Go initializer"},
		{"stdcall-initializer", "//go:linkname V stdcall.v\nvar V func() = nil", "cannot have a Go initializer"},
		{"export-placeholder", "//export F\n//llgo:link F C.f\nfunc F() {}", "cannot combine //export"},
		{"placeholder-export", "//llgo:link F C.f\n//export F\nfunc F() {}", "cannot combine //export"},
		{"export-native", "//export F\n//go:linkname F C.f\nfunc F() {}", "cannot combine //export"},
		{"empty-prototype", "//go:linkname F C.\nfunc F()", "requires a symbol name"},
		{"implementation-collision", "//go:linkname F C.invalid.F\nfunc F() {}", "conflicts with its Go implementation"},
		{"empty", "//go:linkname V C.\nvar V int", "requires a symbol name"},
		{"intrinsic-variable", "//go:linkname V llgo.foo\nvar V int", "applies only to functions"},
		{"generic-export", "//go:linkname F C.f\nfunc F[T any]() {}", "unsupported signature"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "invalid.go", "package invalid\n"+tc.source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			prog := ssatest.NewProgramEx(t, nil, importer.Default())
			err = ParsePkgSyntax(prog, fset, types.NewPackage("invalid", "invalid"), []*ast.File{file})
			if tc.name == "generic-export" {
				if _, ok := prog.PackageExport("invalid.F"); ok {
					t.Error("rejected generic registered a package export")
				}
				if _, ok := prog.NativeExport("invalid.F"); ok {
					t.Error("rejected generic registered a native export")
				}
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
		})
	}
}

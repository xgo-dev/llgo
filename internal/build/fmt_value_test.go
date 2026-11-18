package build

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	"golang.org/x/tools/go/ssa"
)

const typedFormatTestRuntime = `package llgofmt
func AppendInt(b []byte, value int64) []byte { return b }
func AppendUint(b []byte, value uint64) []byte { return b }
func AppendFloat(b []byte, value float64, bits int) []byte { return b }
func AppendComplex(b []byte, value complex128, bits int) []byte { return b }
func AppendPointer(b []byte, value uintptr) []byte { return b }
func Compare[T any](a,b T) int { return 0 }
func AppendMap[K comparable,V any](b []byte, m map[K]V, cmp func(K,K) int, k func([]byte,K) []byte, v func([]byte,V) []byte) []byte { return b }
`

func TestSpecializeValueFormats(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lang   string
		decl   string
		main   string
		local  string
		expr   string
		asm    string
		open   bool
		native bool
		typed  bool
	}{
		{name: "string-int map", typed: true},
		{name: "go1.16 map", lang: "go1.16", typed: true},
		{name: "go1.16 imported private field", lang: "go1.16", expr: "fmt.NewRecord()", typed: true},
		{name: "named map", expr: "plainMap(nil)", typed: true},
		{name: "compound map", expr: "map[[2]bool][]uint16(nil)", typed: true},
		{name: "struct array and slice", expr: "struct{ X [2][]int }{}", typed: true},
		{name: "float", expr: "float32(4.25)", typed: true},
		{name: "float64", expr: "4.25", typed: true},
		{name: "complex64", expr: "complex64(4+2i)", typed: true},
		{name: "complex128", expr: "4+2i", typed: true},
		{name: "int", expr: "17", typed: true},
		{name: "uint", expr: "uint(17)", typed: true},
		{name: "bool", expr: "true", typed: true},
		{name: "string", expr: "string(118)", typed: true},
		{name: "blank field", expr: "struct{ _ int; I int }{I: 1}", typed: true},
		{name: "struct key", expr: "map[struct{ X float64; _ int; P *int }]uint(nil)", typed: true},
		{name: "channel key", expr: "map[chan int]bool(nil)", typed: true},
		{name: "channel", expr: "(chan int)(nil)", typed: true},
		{name: "pointer to map", expr: "new(map[int]string)", typed: true},
		{name: "local named type", local: "type local struct{ a int }; ", expr: "local{7}", typed: true},
		{name: "recursive slice", expr: "recursive(nil)"},
		{name: "interface field", expr: "struct{ X any }{}"},
		{name: "nested method", expr: "[]namedMap(nil)"},
		{name: "general formatting", main: "extra()"},
		{name: "assembly calls Go", asm: "main.extra"},
		{name: "C export", decl: "//export extra\n"},
		{name: "unknown host export", decl: "//go:wasmexport extra\n"},
		{name: "user linkname", decl: "//go:linkname extra external_extra\n"},
		{name: "unknown native roots", open: true},
		{name: "native target", native: true},
		{name: "custom method", expr: "namedMap(nil)"},
		{name: "interface operand", expr: "dynamicMap()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
			prog.CreatePackage(types.Unsafe, nil, nil, true)
			runtime := buildFormatTestPackage(t, prog, fmtspecialize.RuntimePath, typedFormatTestRuntime, nil)
			fmtPkg := buildFormatTestPackage(t, prog, "fmt", `package fmt
type record struct { hidden int }
func NewRecord() record { return record{7} }
type pp struct{}
func (*pp) printArg(a any) string { return "generic" }
func Sprintf(_ string, args ...any) string { return (&pp{}).printArg(args[0]) }
`, nil)
			expr := tc.expr
			if expr == "" {
				expr = "map[string]int(nil)"
			}
			src := `package main
import "fmt"
type plainMap map[string]int
type recursive []recursive
type namedMap map[string]int
func (namedMap) String() string { return "named" }
func dynamicMap() interface{} { return map[string]int(nil) }
func render() string { ` + tc.local + `return fmt.Sprintf("%v", ` + expr + `) }
` + tc.decl + `func extra() { println(fmt.Sprintf("%#v", []int{1, 2})) }
func main() { println(render()); ` + tc.main + ` }
`
			entry := checkFormatTestPackage(t, prog.Fset, "main", src, types.Config{
				Importer: literalFormatImporter{fmtPkg.Pkg}, GoVersion: tc.lang,
			})
			ctx := &context{
				progSSA: prog, initial: []*packages.Package{entry},
				pkgByID: map[string]*aPackage{"main": {Package: entry}},
				buildConf: &Config{
					OptLevel: optlevel.Oz, Goarch: "wasm", BuildMode: BuildModeExe, Verbose: true,
				},
				crossCompile: crosscompile.Export{
					WasmProvider: crosscompile.WasmProviderWASI, WasmProfile: crosscompile.WasmProfileW32,
				},
				wasmClosedProgram: !tc.open,
			}
			entry.Imports = map[string]*packages.Package{"fmt": {PkgPath: "fmt", Types: fmtPkg.Pkg}}
			ctx.dedup = packages.NewDeduper()
			runtimePackage := &packages.Package{ID: fmtspecialize.RuntimePath, PkgPath: fmtspecialize.RuntimePath, Types: runtime.Pkg}
			if !tc.native {
				ctx.formatRuntime = &packages.Cached{Package: runtimePackage, Types: runtime.Pkg}
			}
			dep, err := ctx.prepareValueFormatSource(entry)
			if err != nil {
				t.Fatal(err)
			}
			if entry.Types.GoVersion() != tc.lang {
				t.Fatalf("package language version changed to %s", entry.Types.GoVersion())
			}
			if tc.name == "string-int map" && dep != runtimePackage {
				t.Errorf("formatter helper dependency = %v, want %v", dep, runtimePackage)
			}
			if (tc.native || tc.name == "bool") && dep != nil {
				t.Errorf("unexpected formatter helper dependency %v", dep)
			}
			mainPkg := prog.CreatePackage(entry.Types, entry.Syntax, entry.TypesInfo, true)
			prog.Build()
			if err := recordPackageSSAInstructions(ctx); err != nil {
				t.Fatal(err)
			}
			in := &wasiSizePruningInputs{}
			if tc.asm != "" {
				in.roots = []string{tc.asm}
			}
			configureValueFormats(ctx, in)
			found := false
			for _, inst := range mainPkg.Func("render").Blocks[0].Instrs {
				if tc.typed {
					switch ir := inst.(type) {
					case *ssa.MakeInterface:
						t.Error("typed formatter retained interface boxing")
					case *ssa.Alloc:
						if ir.Comment == "varargs" {
							t.Error("typed formatter retained the variadic array")
						}
					}
				}
				if call, ok := inst.(*ssa.Call); ok {
					common := call.Common()
					callee := common.StaticCallee()
					if callee == nil || callee != fmtPkg.Func("Sprintf") && !strings.HasPrefix(callee.Name(), "__llgo_fmt_") {
						continue
					}
					found = true
					gotTyped := callee.Pkg == mainPkg && len(common.Args) == 1 &&
						!types.IsInterface(common.Args[0].Type()) &&
						types.Identical(common.Args[0].Type(), callee.Signature.Params().At(0).Type())
					if gotTyped != tc.typed {
						t.Errorf("render calls %s with args %v, want typed=%v", callee, common.Args, tc.typed)
					}
				}
			}
			if !found {
				t.Fatal("missing render call")
			}
			if got := ctx.valueFormatPackages[mainPkg.Pkg]; got != tc.typed {
				t.Errorf("cache key specialized = %v, want %v", got, tc.typed)
			}
		})
	}
}

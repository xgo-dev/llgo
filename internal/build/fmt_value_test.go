package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	"golang.org/x/tools/go/ssa"
)

func TestSpecializeValueFormats(t *testing.T) {
	for _, tc := range []struct {
		name   string
		decl   string
		main   string
		expr   string
		asm    string
		open   bool
		native bool
		helper string
	}{
		{name: "string-int map", helper: "llgoSprintfValueWithMaps"},
		{name: "named map without methods", expr: "plainMap(nil)", helper: "llgoSprintfValueWithMaps"},
		{name: "compound map", expr: "map[[2]bool][]uint16(nil)", helper: "llgoSprintfValueWithMaps"},
		{name: "struct array and slice", expr: "struct{ X [2][]int }{}", helper: "llgoSprintfValue"},
		{name: "float", expr: "float32(4.25)", helper: "llgoSprintfValue"},
		{name: "pointer to map", expr: "new(map[int]string)", helper: "llgoSprintfValueWithMaps"},
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
		{name: "named map", expr: "namedMap(nil)"},
		{name: "interface operand", expr: "dynamicMap()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			fmtPkg := buildFormatTestPackage(t, prog, "fmt", `package fmt
type pp struct{}
func (*pp) printArg(a any) string { return "generic" }
func Sprintf(_ string, args ...any) string { return (&pp{}).printArg(args[0]) }
func llgoSprintfValue(_ string, args ...any) string { return "specialized" }
func llgoSprintfValueWithMaps(_ string, args ...any) string { return "specialized with maps" }
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
func dynamicMap() any { return map[string]int(nil) }
func render() string { return fmt.Sprintf("%v", ` + expr + `) }
` + tc.decl + `func extra() { println(fmt.Sprintf("%#v", []int{1, 2})) }
func main() { println(render()); ` + tc.main + ` }
`
			mainPkg := buildFormatTestPackage(t, prog, "main", src, fmtPkg.Pkg)
			prog.Build()
			file, err := parser.ParseFile(token.NewFileSet(), "main.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			entry := &packages.Package{PkgPath: "main", Types: mainPkg.Pkg, Syntax: []*ast.File{file}}
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
			if tc.native {
				ctx.buildConf.Goarch = "arm64"
				ctx.crossCompile = crosscompile.Export{}
			}
			if err := recordPackageSSAInstructions(ctx); err != nil {
				t.Fatal(err)
			}
			in := &wasiSizePruningInputs{}
			if tc.asm != "" {
				in.roots = []string{tc.asm}
			}
			configureValueFormats(ctx, in)
			want := tc.helper
			if want == "" {
				want = "Sprintf"
			}
			found := false
			for _, inst := range mainPkg.Func("render").Blocks[0].Instrs {
				if call, ok := inst.(*ssa.Call); ok && call.Common().StaticCallee().Pkg == fmtPkg {
					found = true
					if got := call.Common().StaticCallee().Name(); got != want {
						t.Errorf("render calls %s, want %s", got, want)
					}
				}
			}
			if !found {
				t.Fatal("missing render call")
			}
			if got, want := ctx.valueFormatPackages[mainPkg.Pkg], tc.helper != ""; got != want {
				t.Errorf("cache key specialized = %v, want %v", got, want)
			}
		})
	}
}

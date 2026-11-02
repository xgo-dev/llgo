package build

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"golang.org/x/tools/go/ssa"
)

func TestLiteralFormat(t *testing.T) {
	for _, format := range []string{"", "hello", "你好\x00世界", "%%", "a%%b%%%%c", "\xff%%"} {
		got, ok := literalFormat(format)
		if !ok || got != strings.ReplaceAll(format, "%%", "%") {
			t.Fatalf("literalFormat(%q) = %q, %v", format, got, ok)
		}
	}
	for _, format := range []string{"%", "%%%", "%s", "%[1]s", "%*s", "%.2f", "%5%", "%q", "%!"} {
		if _, ok := literalFormat(format); ok {
			t.Fatalf("accepted nonliteral format %q", format)
		}
	}
}

type literalFormatImporter struct{ pkg *types.Package }

func (i literalFormatImporter) Import(string) (*types.Package, error) { return i.pkg, nil }

func literalFormatProgram(t *testing.T, helpers bool) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	build := func(path, source string, imported *types.Package) *ssa.Package {
		file, err := parser.ParseFile(fset, path+".go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Scopes: make(map[ast.Node]*types.Scope)}
		conf := &types.Config{Importer: literalFormatImporter{imported}}
		pkg, err := conf.Check(path, fset, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
		return prog.CreatePackage(pkg, []*ast.File{file}, info, true)
	}
	fmtSource := `package fmt
func Printf(s string, a ...any) (int, error) { return 0, nil }
func Fprintf(w any, s string, a ...any) (int, error) { return 0, nil }
func Sprintf(s string, a ...any) string { return s }
func Sprint(a ...any) string { return "" }
`
	if helpers {
		fmtSource += `
func llgoPrintfLiteral(s string) (int, error) { return 0, nil }
func llgoFprintfLiteral(w any, s string) (int, error) { return 0, nil }
func llgoSprintfLiteral(s string) string { return s }
`
	}
	fmtPkg := build("fmt", fmtSource, nil)
	main := build("main", `package main
import "fmt"
func literal() { fmt.Printf("hello%%") }
func writer(w any) { fmt.Fprintf(w, "") }
func sprint() string { return fmt.Sprintf("你好\x00%%") }
func dynamic(s string) { fmt.Printf(s) }
func argument() { fmt.Printf("literal", 42) }
func missing() { fmt.Printf("%s") }
func spread(a []any) { fmt.Printf("literal", a...) }
func nilSpread() { fmt.Printf("literal", []any(nil)...) }
func deferred() { defer fmt.Printf("literal") }
func goroutine() { go fmt.Printf("literal") }
func other() { _ = fmt.Sprint("literal") }
func ordinary() {}
func main() { ordinary(); println("ok") }
`, fmtPkg.Pkg)
	prog.Build()
	return main
}

func TestSpecializeLiteralFormats(t *testing.T) {
	for _, helpers := range []bool{false, true} {
		pkg := literalFormatProgram(t, helpers)
		recordPackageSSAInstructions(&context{progSSA: pkg.Prog,
			buildConf: &Config{Goarch: "wasm", OptLevel: optlevel.O2}})
		for name, want := range map[string]string{
			"literal": "llgoPrintfLiteral", "writer": "Fprintf", "sprint": "llgoSprintfLiteral",
			"nilSpread": "llgoPrintfLiteral", "deferred": "llgoPrintfLiteral", "goroutine": "llgoPrintfLiteral",
			"dynamic": "Printf", "argument": "Printf", "missing": "Printf", "spread": "Printf", "other": "Sprint",
		} {
			if !helpers {
				want = strings.TrimSuffix(strings.TrimPrefix(want, "llgo"), "Literal")
			}
			found := false
			for _, block := range pkg.Func(name).Blocks {
				for _, instruction := range block.Instrs {
					call, ok := instruction.(ssa.CallInstruction)
					if !ok || call.Common().StaticCallee() == nil {
						continue
					}
					common := call.Common()
					found = true
					if got := common.StaticCallee().Name(); got != want {
						t.Fatalf("%s helpers=%v: call %s, want %s", name, helpers, got, want)
					}
					if helpers && name == "literal" {
						if len(common.Args) != 1 || constant.StringVal(common.Args[0].(*ssa.Const).Value) != "hello%" {
							t.Fatalf("literal arguments = %v", common.Args)
						}
					}
				}
			}
			if !found {
				t.Fatalf("no call in %s", name)
			}
		}
	}
}

func TestSpecializeLiteralFormatsTargetPolicy(t *testing.T) {
	for _, test := range []struct {
		arch     string
		opt      optlevel.Level
		mode     Mode
		debugger bool
		want     bool
	}{
		{"amd64", optlevel.O2, ModeBuild, false, true},
		{"arm64", optlevel.O2, ModeBuild, false, true},
		{"riscv32", optlevel.O2, ModeBuild, false, true},
		{"amd64", optlevel.O0, ModeBuild, false, false},
		{"amd64", optlevel.O2, ModeBuild, true, false},
		{"wasm", optlevel.O0, ModeBuild, false, false},
		{"wasm", optlevel.O2, ModeGen, false, false},
		{"wasm", optlevel.O2, ModeBuild, false, true},
		{"wasm", optlevel.O2, ModeBuild, true, false},
	} {
		pkg := literalFormatProgram(t, true)
		ctx := &context{progSSA: pkg.Prog, mode: test.mode, buildConf: &Config{Goarch: test.arch, OptLevel: test.opt}}
		ctx.frontendOptions.DebuggerFrames = test.debugger
		recordPackageSSAInstructions(ctx)
		call := pkg.Func("literal").Blocks[0].Instrs[0].(*ssa.Call)
		want := "Printf"
		if test.want {
			want = "llgoPrintfLiteral"
		}
		if got := call.Common().StaticCallee().Name(); got != want {
			t.Fatalf("%+v: call %s, want %s", test, got, want)
		}
	}
}

func TestLiteralWriterFrames(t *testing.T) {
	for _, arch := range []string{"amd64", "wasm"} {
		for _, opt := range []optlevel.Level{optlevel.O2, optlevel.Oz} {
			pkg := literalFormatProgram(t, true)
			recordPackageSSAInstructions(&context{progSSA: pkg.Prog,
				buildConf: &Config{Goarch: arch, OptLevel: opt}})
			call := pkg.Func("writer").Blocks[0].Instrs[0].(*ssa.Call)
			if got := call.Common().StaticCallee().Name(); got != "Fprintf" {
				t.Fatalf("%s %v: Writer must retain Fprintf for caller inspection, got %s", arch, opt, got)
			}
		}
	}
}

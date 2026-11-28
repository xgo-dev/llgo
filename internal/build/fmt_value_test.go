package build

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	buildenv "github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
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
	t.Setenv("LDFLAGS", "")
	for _, tc := range []struct {
		name        string
		lang        string
		decl        string
		main        string
		local       string
		expr        string
		asm         string
		alt         string
		rtInit      bool
		dot         bool
		extra       string
		open        bool
		noRuntime   bool
		foreignLink bool
		opaqueAsm   bool
		native      bool
		typed       bool
	}{
		{name: "string-int map", typed: true},
		{name: "go1.16 map", lang: "go1.16", typed: true},
		{name: "go1.16 imported private field", lang: "go1.16", expr: "fmt.NewRecord()", typed: true},
		{name: "named map", expr: "plainMap(nil)", typed: true},
		{name: "complex keys", expr: "map[complex128]int(nil)", typed: true},
		{name: "unsafe pointer keys", expr: "map[unsafe.Pointer]int(nil)", typed: true},
		{name: "unsafe pointer", expr: "unsafe.Pointer(nil)", typed: true},
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
		{name: "plain pointer", expr: "new(int)", typed: true},
		{name: "dot import", dot: true, typed: true},
		{name: "spread arguments", main: "extra()", extra: "println(fmt.Sprintf(\"%v\", []any{1, 2}...))"},
		{name: "pointer to map", expr: "new(map[int]string)", typed: true},
		{name: "local named type", local: "type local struct{ a int }; ", expr: "local{7}", typed: true},
		{name: "recursive slice", expr: "recursive(nil)"},
		{name: "interface field", expr: "struct{ X any }{}"},
		{name: "nested method", expr: "[]namedMap(nil)", typed: true},
		{name: "general formatting", main: "extra()"},
		{name: "general formatter via a function value", main: "extra()",
			decl: "var indirect = fmt.Sprintf\n", extra: `println(indirect("%d", 7))`},
		{name: "assembly calls Go", asm: "main.extra"},
		{name: "unresolved assembly", opaqueAsm: true},
		{name: "native assembly symbol", asm: "native_symbol", typed: true},
		{name: "unknown translated symbol", asm: "main.missing", typed: true},
		{name: "runtime initializer calls fmt", rtInit: true},
		{name: "alternate package host export", alt: "//go:wasmexport launch"},
		{name: "unknown method export", decl: "type Listener int\n//export call\nfunc (Listener) call() {}\n"},
		{name: "shadowed builtin", decl: "var len int\n"},
		{name: "C export", decl: "//export extra\n"},
		{name: "unknown host export", decl: "//go:wasmexport extra\n"},
		{name: "user linkname", decl: "//go:linkname extra external_extra\n"},
		{name: "unknown native roots", open: true},
		{name: "missing runtime helper", noRuntime: true},
		{name: "native typed formatter", native: true, typed: true},
		{name: "native generic fallback", native: true, main: "extra()"},
		{name: "native opaque linker flags", native: true, foreignLink: true},
		{name: "custom method", expr: "namedMap(nil)", typed: true},
		{name: "safe field and package variables", expr: "label{}", typed: true,
			decl: "type label struct { text string }; var suffix string; func (l label) String() string { return l.text + suffix + fmt.Prefix }\n"},
		{name: "safe error method", expr: "label(0)", typed: true,
			decl: "type label int; func (label) Error() string { return \"error\" }\n"},
		{name: "error overrides string", expr: "label(0)", typed: true,
			decl: "type label int; func (label) String() string { panic(0) }; func (label) Error() string { return \"error\" }\n"},
		{name: "format overrides string", expr: "label(0)",
			decl: "type label int; func (label) String() string { return \"text\" }; func (label) Format(int, rune) {}\n"},
		{name: "method with pointer receiver", expr: "new(label)",
			decl: "type label int; func (*label) String() string { return \"text\" }\n"},
		{name: "promoted method", expr: "struct { label }{}",
			decl: "type label int; func (label) String() string { return \"text\" }\n"},
		{name: "method reads field through pointer", expr: "label{}",
			decl: "type child struct { text string }; type label struct { p *child }; func (l label) String() string { return l.p.text }\n"},
		{name: "method indexes slice", expr: "label(nil)",
			decl: "type label []string; func (l label) String() string { return l[0] }\n"},
		{name: "method with unexpected signature", expr: "label(0)",
			decl: "type label int; func (label) String(int) string { return \"text\" }\n"},
		{name: "interface operand", expr: "dynamicMap()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
			prog.CreatePackage(types.Unsafe, nil, nil, true)
			runtime := buildFormatTestPackage(t, prog, fmtspecialize.RuntimePath, typedFormatTestRuntime, nil)
			fmtPkg := buildFormatTestPackage(t, prog, "fmt", `package fmt
type record struct { hidden int }
func NewRecord() record { return record{7} }
var Prefix string
type pp struct{}
func (*pp) printArg(a any) string { return "generic" }
func Sprintf(_ string, args ...any) string { return (&pp{}).printArg(args[0]) }
`, nil)
			expr := tc.expr
			if expr == "" {
				expr = "map[string]int(nil)"
			}
			extraImport := ""
			if strings.Contains(expr, "unsafe.Pointer") {
				extraImport = "import \"unsafe\"\n"
			}
			extra := tc.extra
			if extra == "" {
				extra = `println(fmt.Sprintf("%#v", []int{1, 2}))`
			}
			src := `package main
import "fmt"
` + extraImport + `type plainMap map[string]int
type recursive []recursive
type namedMap map[string]int
func (namedMap) String() string { return "named" }
func dynamicMap() interface{} { return map[string]int(nil) }
func render() string { ` + tc.local + `return fmt.Sprintf("%v", ` + expr + `) }
` + tc.decl + `func extra() { ` + extra + ` }
func main() { println(render()); ` + tc.main + ` }
`
			if tc.dot {
				src = strings.Replace(src, "import \"fmt\"", "import . \"fmt\"", 1)
				src = strings.ReplaceAll(src, "fmt.", "")
			}
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
			if tc.rtInit {
				buildFormatTestPackage(t, prog, llssa.PkgRuntime, `package runtime
import "fmt"
func init() { println(fmt.Sprintf("%#v", []int{1})) }
`, fmtPkg.Pkg)
			}
			if tc.alt != "" {
				alt := checkFormatTestPackage(t, prog.Fset, "alternate", "package alternate\n"+tc.alt+"\nfunc launch() {}\n", types.Config{})
				prog.CreatePackage(alt.Types, alt.Syntax, alt.TypesInfo, true)
				ctx.pkgByID["main"].AltPkg = &packages.Cached{Package: alt}
			}
			entry.Imports = map[string]*packages.Package{"fmt": {PkgPath: "fmt", Types: fmtPkg.Pkg}}
			ctx.dedup = packages.NewDeduper()
			runtimePackage := &packages.Package{ID: fmtspecialize.RuntimePath, PkgPath: fmtspecialize.RuntimePath, Types: runtime.Pkg}
			if tc.native {
				ctx.buildConf.Goarch = "arm64"
				ctx.buildConf.Goos = "darwin"
				ctx.buildConf.Mode = ModeBuild
				ctx.buildConf.LTO = lto.Full
				ctx.buildConf.DeadcodeDrop = true
				ctx.buildConf.DisableGoGlobalDCE = true
				ctx.wasmClosedProgram = false
				if tc.foreignLink {
					ctx.commands.environ = []string{"LDFLAGS=-Wl,--undefined=foreign"}
				}
			}
			if !tc.noRuntime {
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
			if (tc.noRuntime || tc.name == "bool") && dep != nil {
				t.Errorf("unexpected formatter helper dependency %v", dep)
			}
			mainPkg := prog.CreatePackage(entry.Types, entry.Syntax, entry.TypesInfo, true)
			prog.Build()
			if err := recordPackageSSAInstructions(ctx); err != nil {
				t.Fatal(err)
			}
			in := &linkPruningInputs{}
			if tc.opaqueAsm {
				in.opaqueAsmPkg = "example.com/native"
			}
			if tc.asm != "" {
				in.roots = []string{tc.asm}
			}
			switch {
			case tc.native && buildenv.Dev:
				configureFormattingAndInits(ctx, "", nil)
				if ctx.nativeClosedProgram != !tc.foreignLink {
					t.Fatalf("native metadata pruning enabled = %v", ctx.nativeClosedProgram)
				}
			case tc.foreignLink:
				if opaque := opaqueGlobalInitPackage(ctx, make(map[string]bool), &linkPruningInputs{nativeFormat: true}); opaque == "" {
					t.Error("external linker input must make hosted inputs opaque")
				}
				configureValueFormats(ctx, nil)
			default:
				configureValueFormats(ctx, in)
			}
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

func TestRegisterFormatSupportErrors(t *testing.T) {
	prog := ssa.NewProgram(token.NewFileSet(), ssa.InstantiateGenerics)
	prog.CreatePackage(types.Unsafe, nil, nil, true)
	fmtPkg := buildFormatTestPackage(t, prog, "fmt", `package fmt
func Sprintf(_ string, args ...any) string { return "" }
`, nil)
	runtime := buildFormatTestPackage(t, prog, fmtspecialize.RuntimePath, "package llgofmt", nil)
	entry := checkFormatTestPackage(t, prog.Fset, "main", `package main
import "fmt"
func main() { println(fmt.Sprintf("%v", 1)) }
`, types.Config{Importer: literalFormatImporter{fmtPkg.Pkg}})
	entry.Imports = map[string]*packages.Package{"fmt": {ID: "fmt", PkgPath: "fmt", Types: fmtPkg.Pkg}}
	other := checkFormatTestPackage(t, prog.Fset, "other", "package other", types.Config{})
	ctx := &context{
		progSSA: prog, buildConf: &Config{}, dedup: packages.NewDeduper(),
		pkgByID:       map[string]*aPackage{"fmt": {}},
		formatRuntime: &packages.Cached{Types: runtime.Pkg},
	}
	all, entries, err := registerSSAPkgs(ctx, []*packages.Package{entry, other}, false)
	if err == nil || !strings.Contains(err.Error(), "AppendInt") || len(all) != 0 || len(entries) != 0 {
		t.Fatalf("register returned %d packages, %d SSA entries, %v", len(all), len(entries), err)
	}
}

func TestTypedJSONSupport(t *testing.T) {
	conf := Config{Goos: "linux", Goarch: "amd64", Mode: ModeBuild, BuildMode: BuildModeExe,
		OptLevel: optlevel.Oz, LTO: lto.Full, DeadcodeDrop: true, DisableGoGlobalDCE: true}
	if got := typedJSONSupport(&conf, crosscompile.Export{}, cl.Options{}); got != buildenv.Dev {
		t.Fatalf("Linux typed decoder enabled = %v", got)
	}
	conf.LinkOptions.ExternalLinkerFlags = "foreign.o"
	if typedJSONSupport(&conf, crosscompile.Export{}, cl.Options{}) {
		t.Fatal("external link input enabled the typed decoder")
	}
	conf.LinkOptions.ExternalLinkerFlags = ""
	conf.Goos, conf.Goarch = "darwin", "arm64"
	if got := typedJSONSupport(&conf, crosscompile.Export{}, cl.Options{}); got != buildenv.Dev {
		t.Fatalf("Darwin typed decoder enabled = %v", got)
	}
}

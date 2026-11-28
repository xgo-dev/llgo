package build

import (
	"go/token"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

func TestCanOmitWASIArgs(t *testing.T) {
	const initArgs = "Args = runtime_args()"
	for _, tc := range []struct {
		name, main, init, assembly, alias string
		external, rawRead, want           bool
	}{
		{name: "initialization only", want: true},
		{name: "argument read", main: "println(len(os.Args))"},
		{name: "argument address", main: "println(&os.Args)"},
		{name: "method visible through reflection", main: "println(os.Box())"},
		{name: "interface dispatch", main: "println(os.Box().(interface{ Count() int }).Count())"},
		{name: "another initializer read", init: initArgs + "; println(len(Args))"},
		{name: "another use of returned slice", init: "v := runtime_args(); Args = v; println(len(v))"},
		{name: "different store", init: "other = runtime_args()"},
		{name: "assembly reference", assembly: wasiArgvSymbol},
		{name: "assembly reference to Args", assembly: wasiArgsGlobal},
		{name: "linknamed Args", alias: "go:" + wasiArgsGlobal},
		{name: "linknamed runtime args", alias: wasiRuntimeArgsSymbol},
		{name: "linknamed argc", alias: "C." + wasiArgcSymbol},
		{name: "linknamed argv", alias: "C." + wasiArgvSymbol},
		{name: "runtime provider alias", alias: wasiArgsProviderSymbol},
		{name: "direct raw read", rawRead: true},
		{name: "external PCLN", external: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions|ssa.GlobalDebug)
			clite := buildFormatTestPackage(t, prog, wasiClitePath, "package clite\nvar Argc int\nvar Argv *byte\nfunc Observe() int { return Argc }", nil)
			rt := buildFormatTestPackage(t, prog, wasiRuntimePath, `package runtime
import "github.com/xgo-dev/llgo/runtime/internal/clite"
func os_runtime_args() []string { return make([]string, clite.Argc) }
`, clite.Pkg)
			init := tc.init
			if init == "" {
				init = initArgs
			}
			os := buildFormatTestPackage(t, prog, "os", `package os
var Args, other []string
func runtime_args() []string
type reader struct{}
func (reader) Count() int { return len(Args) }
func Box() any { return reader{} }
func init() { `+init+" }", nil)
			body := tc.main
			if body == "" {
				body = "println(42)"
			}
			entry := buildFormatTestPackage(t, prog, "main", "package main\nimport \"os\"\nvar _ = os.Box\nfunc main() {"+body+"}", os.Pkg)
			prog.Build()
			backend := llssa.NewProgram(nil)
			defer backend.Dispose()
			backend.SetLinkname(wasiArgsProviderSymbol, wasiRuntimeArgsSymbol)
			backend.SetLinkname(wasiClitePath+".Argc", "C."+wasiArgcSymbol)
			backend.SetLinkname(wasiClitePath+".Argv", "C."+wasiArgvSymbol)
			if tc.alias != "" {
				backend.SetLinkname("example.com/other.(*T).Observe", tc.alias)
			}
			var assembly []string
			if tc.assembly != "" {
				assembly = append(assembly, tc.assembly)
			}
			roots := []*ssa.Function{entry.Func("init"), entry.Func("main"), rt.Func(wasiArgsProvider)}
			if tc.rawRead {
				roots = append(roots, clite.Func("Observe"))
			}
			ctx := &context{prog: backend, progSSA: prog, buildConf: &Config{}}
			if tc.external {
				ctx.buildConf.PCLNMode = PCLNExternal
			}
			result := rta.Analyze(roots, false)
			if got := canOmitWASIArgs(ctx, result, assembly, func(symbol string) bool { return symbol == tc.assembly }); got != tc.want {
				t.Fatalf("omit WASI args = %v, want %v", got, tc.want)
			}
		})
	}
}

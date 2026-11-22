package build

import (
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestWASIStartupLegacyGodebugRegistration(t *testing.T) {
	for _, test := range []struct {
		name, initBody string
		otherProvider  bool
		want           bool
	}{
		{name: "stock older Go registration", initBody: "setUpdate(update); setNewIncNonDefault(newIncNonDefault)", want: true},
		{name: "unrecognized callback", initBody: "setUpdate(other); setNewIncNonDefault(newIncNonDefault)"},
		{name: "non-call before factory", initBody: "observed = true; setNewIncNonDefault(newIncNonDefault)"},
		{name: "unrecognized operation", initBody: "println(1); setUpdate(update); setNewIncNonDefault(newIncNonDefault)"},
		{name: "unexpected provider", initBody: "setUpdate(update); setNewIncNonDefault(newIncNonDefault)", otherProvider: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			settings := buildFormatTestPackage(t, prog, "internal/godebug", `package godebug
func update() {}
func newIncNonDefault() {}
func other() {}
var observed bool
func setUpdate(func())
func setNewIncNonDefault(func())
func init() { `+test.initBody+` }
`, nil)
			providerPath := wasiRuntimePath
			symbol := "runtime.setUpdate"
			if test.otherProvider {
				providerPath = "example.test/runtime"
				symbol = providerPath + ".setUpdate"
			}
			provider := buildFormatTestPackage(t, prog, providerPath, `package runtime
var callback func()
func setUpdate(f func()) { callback = f }
`, nil)
			prog.Build()
			backend := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer backend.Dispose()
			backend.SetLinkname(symbol, "internal/godebug.setUpdate")
			ctx := &context{buildConf: &Config{}, progSSA: prog, prog: backend}
			effects := startupEffects{
				trusted: map[*types.Package]bool{settings.Pkg: true, provider.Pkg: true},
				passed:  make(map[*ssa.Function]bool),
			}
			allowWASIGodebugSetup(ctx, &effects)
			if got := effects.passed[settings.Func("init#1")]; got != test.want {
				t.Fatalf("passive registration = %t, want %t", got, test.want)
			}
		})
	}
}

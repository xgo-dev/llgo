package cl

import (
	"fmt"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestInterfaceUseLowering(t *testing.T) {
	const src = `package domain
//llgo:link used llgo.interfaceKindUsed
func used(kind int) bool { panic("placeholder") }
type Alias = float64
func Basic(x any) (float64, bool) { v, ok := x.(float64); return v, ok }
func Aliased(x any) (Alias, bool) { v, ok := x.(Alias); return v, ok }
func Composite(x any) (struct{}, bool) { v, ok := x.(struct{}); return v, ok }
`
	for _, tc := range []struct {
		arch    string
		enabled bool
	}{{"wasm", false}, {"amd64", true}, {"wasm", true}} {
		t.Run(fmt.Sprintf("%s/%v", tc.arch, tc.enabled), func(t *testing.T) {
			source := src
			for kind := types.Float32; kind <= types.Complex128; kind++ {
				source += fmt.Sprintf("func Used%d() bool { return used(%d) }\n", kind, kind)
			}
			ssapkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOARCH: tc.arch})
			defer prog.Dispose()
			prog.EnableDeadcodeDrop(tc.enabled)
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			wantPredicate := tc.arch == "wasm" && tc.enabled
			for _, name := range []string{"Basic", "Aliased", "Composite"} {
				ir := mustNamedFunction(t, mod, "domain."+name).String()
				want := wantPredicate && name != "Composite"
				if strings.Contains(ir, "@_llgo_iface_use.") != want || strings.Contains(ir, "and i1") != want {
					t.Errorf("%s: unexpected interface visibility guard:\n%s", name, ir)
				}
				if !strings.Contains(ir, "icmp eq ptr") {
					t.Errorf("%s: missing dynamic type comparison:\n%s", name, ir)
				}
			}
			for kind := types.Float32; kind <= types.Complex128; kind++ {
				ir := mustNamedFunction(t, mod, fmt.Sprintf("domain.Used%d", kind)).String()
				if wantPredicate {
					if !strings.Contains(ir, "@_llgo_iface_use._llgo_"+types.Typ[kind].Name()+"(") {
						t.Errorf("missing kind predicate:\n%s", ir)
					}
				} else if !strings.Contains(ir, "ret i1 true") {
					t.Errorf("ungated target must preserve the kind:\n%s", ir)
				}
			}
		})
	}
}

func TestInterfaceKindUsedInvalidArguments(t *testing.T) {
	for _, tc := range []struct{ name, decl, call, want string }{
		{"arity", "func used() bool", "used()", "invalid arguments"},
		{"dynamic", "func used(kind int) bool", "used(kind)", "expected a constant floating-point kind"},
		{"integer", "func used(kind int) bool", fmt.Sprintf("used(%d)", types.Int), "expected a constant floating-point kind"},
		{"string", "func used(kind int) bool", fmt.Sprintf("used(%d)", types.String), "expected a constant floating-point kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package domain\n//llgo:link used llgo.interfaceKindUsed\n" + tc.decl + " { panic(\"placeholder\") }\nfunc bad(kind int) bool { return " + tc.call + " }"
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOARCH: "wasm"})
			defer prog.Dispose()
			defer func() {
				if got := fmt.Sprint(recover()); !strings.Contains(got, tc.want) {
					t.Fatalf("diagnostic = %q, want %q", got, tc.want)
				}
			}()
			if _, err := NewPackage(prog, ssapkg, files); err != nil {
				panic(err)
			}
		})
	}
}

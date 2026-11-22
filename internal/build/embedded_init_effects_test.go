package build

import (
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestEmbeddedSizePruningConfigStartupEffects(t *testing.T) {
	for _, test := range []struct {
		name, source string
		pass         bool
	}{
		{"private state", "var table = [2]int{1, 2}; var first = table[0]", true},
		{"literal map", "var table = map[string]int{\"one\": 1}", true},
		{"dynamic map reservation", "var n int; var table = make(map[string]int, n)", false},
		{"unused closure", "var f = func() { println(\"later\") }", true},
		{"observable output", "func init() { println(\"init\") }", false},
		{"explicit panic", "func init() { panic(\"init\") }", false},
		{"pointer not proved", "var target *int; var v = *target", false},
		{"dynamic division", "var d int; var v = 5 / d", false},
		{"indirect call", "var f func() int; var v = f()", false},
		{"loop", "var n int; func init(){ for n > 0 { n-- } }", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			pkg := buildFormatTestPackage(t, prog, "example.test/effects", "package effects\n"+test.source, nil)
			pkg.Build()
			effects := startupEffects{
				trusted: map[*types.Package]bool{pkg.Pkg: true},
				active:  make(map[*ssa.Function]bool),
				passed:  make(map[*ssa.Function]bool),
			}
			if err := effects.check(pkg.Func("init")); (err == nil) != test.pass {
				t.Fatalf("check startup got %v, want passive %t", err, test.pass)
			}
		})
	}
}

func TestEmbeddedSizePruningConfigStartupObservers(t *testing.T) {
	for _, test := range []struct {
		name, observer string
		alias          string
		safe           bool
	}{
		{name: "passive copy", safe: true},
		{name: "reads copied state", observer: `import . "example.test/copied"
func init() { println(Derived) }`},
		{name: "indirect reader", observer: `import . "example.test/copied"
func read() { println(Derived) }
var observed = read
func init() { observed() }`},
		{name: "alias can read private state", alias: "example.test/data.Value"},
		{name: "effect independent of dropped state", observer: `import _ "example.test/copied"
func init() { println(2) }`, safe: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			data := buildFormatTestPackage(t, prog, "example.test/data",
				`package data
var Value = newValue()
func newValue() int { return 42 }`, nil)
			copyOfData := buildFormatTestPackage(t, prog, "example.test/copied",
				`package copied
import "example.test/data"
var Derived = data.Value`, data.Pkg)
			trusted := map[*types.Package]bool{data.Pkg: true, copyOfData.Pkg: true}
			if test.observer != "" {
				reader := buildFormatTestPackage(t, prog, "example.test/reader",
					"package reader\n"+test.observer, copyOfData.Pkg)
				trusted[reader.Pkg] = true
			}
			prog.Build()
			backend := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "arm"})
			defer backend.Dispose()
			if test.alias != "" {
				backend.SetLinkname("example.test/other.Value", test.alias)
			}
			ctx := &context{buildConf: &Config{}, progSSA: prog, prog: backend}
			effects := startupEffects{trusted: trusted, active: make(map[*ssa.Function]bool), passed: make(map[*ssa.Function]bool)}
			if ok := checkStartupReads(ctx, &effects, map[*ssa.Package]bool{data: true}); ok != test.safe {
				t.Fatalf("startup proof = %t, want %t", ok, test.safe)
			}
		})
	}
}

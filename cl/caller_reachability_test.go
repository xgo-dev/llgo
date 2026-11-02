package cl

import "testing"

func TestRuntimeCallerRecursiveReachability(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "static calls",
			src: `
func a() { b(); runtime.Caller(0) }
func b() { c() }
func c() { a() }
`,
			want: []string{"a", "b", "c"},
		},
		{
			name: "function argument",
			src: `
func a() { invoke(b); runtime.Caller(0) }
func b() { a() }
func invoke(fn func()) { fn() }
`,
			want: []string{"a", "b", "invoke"},
		},
		{
			name: "interface method",
			src: `
type I interface { Step() }
type T struct{}
func (T) Step() { b() }
func a() { var i I = T{}; i.Step(); runtime.Caller(0) }
func b() { a() }
`,
			want: []string{"a", "b"},
		},
		{
			name: "closure",
			src: `
func a() { func() { b() }(); runtime.Caller(0) }
func b() { a() }
`,
			want: []string{"a", "b"},
		},
		{
			name: "unresolved dispatch",
			src: `
var sink func()
func a() { b(); sink() }
func b() { a() }
func consumer() { runtime.Caller(0) }
`,
			want: []string{"a", "b", "consumer"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pkg, _ := buildCallerFrameSSAPackage(t, "example.com/cycle", `package cycle
import "runtime"
func plainA() { plainB() }
func plainB() { plainA() }
`+tt.src)
			for _, start := range append(append([]string(nil), tt.want...), "plainA") {
				t.Run("first="+start, func(t *testing.T) {
					analysis := newRuntimeCallerAnalysis(pkg)
					if got := analysis.fnMayReachRuntimeCaller(pkg.Func(start)); got != (start != "plainA") {
						t.Fatalf("reachability of %s = %v", start, got)
					}
					for _, name := range tt.want {
						if !analysis.fnMayReachRuntimeCaller(pkg.Func(name)) {
							t.Errorf("%s does not reach runtime.Caller after visiting %s", name, start)
						}
					}
					for _, name := range []string{"plainA", "plainB"} {
						if analysis.fnMayReachRuntimeCaller(pkg.Func(name)) {
							t.Errorf("disconnected cycle %s reaches runtime.Caller", name)
						}
					}
					base := computeRuntimeCallerBaseSetFromAnalysis(analysis)
					for _, name := range tt.want {
						if !base[pkg.Func(name)] {
							t.Errorf("caller base set omits %s", name)
						}
					}
					// Methods and closures are graph members too, even though
					// they are not available through Package.Func.
					for fn := range analysis.funcs {
						if fn.Name() == "Step" || fn.Parent() == pkg.Func("a") {
							if !base[fn] {
								t.Errorf("caller base set omits %s", fn)
							}
						}
					}
				})
			}
		})
	}
}

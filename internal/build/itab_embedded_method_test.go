//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestNewItabSlowReleasesMethodIndex(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, tags := range []string{"", "nogc"} {
		name := "gc"
		if tags != "" {
			name = tags
		}
		t.Run(name, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Tags = tags
			pkgs, err := Build(Invocation{Args: []string{llssa.PkgRuntime}, Config: conf})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 || pkgs[0].LPkg == nil {
				t.Fatal("missing compiled runtime package")
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			mod := pkgs[0].LPkg.Module()
			idxFn := mod.NamedFunction(llssa.PkgRuntime + ".buildItabMethodIndex")
			if idxFn.IsNil() || !strings.Contains(idxFn.String(), "runtime.AllocRoot") {
				t.Fatalf("buildItabMethodIndex missing AllocRoot:\n%s", idxFn.String())
			}
			slowFn := mod.NamedFunction(llssa.PkgRuntime + ".newItabSlow")
			if slowFn.IsNil() || !strings.Contains(slowFn.String(), "runtime.FreeRoot") {
				t.Fatalf("newItabSlow missing FreeRoot release for scratch index:\n%s", slowFn.String())
			}
		})
	}
}

func TestItabEmbeddedUnexportedMethodInMain(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs an embedded unexported method itab fixture")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module aaa.example/testitab\n\ngo 1.24\n",
		"foreign/foreign.go": `package foreign

type I interface {
	hidden() int
}

type Missing interface {
	hidden() int
	absent()
}

type Embedded struct{}

func (Embedded) hidden() int { return 42 }

func CallHidden(v I) int { return v.hidden() }
`,
		"pkga/a.go": `package pkga

type Even interface {
	m00() int
	m02() int
	m04() int
	m06() int
}

type A struct{}

func (A) m00() int { return 1 }
func (A) m02() int { return 3 }
func (A) m04() int { return 5 }
func (A) m06() int { return 7 }

func SumEven(e Even) int { return e.m00() + e.m02() + e.m04() + e.m06() }
`,
		"pkgb/b.go": `package pkgb

type Odd interface {
	m01() int
	m03() int
	m05() int
	m07() int
}

type B struct{}

func (B) m01() int { return 2 }
func (B) m03() int { return 4 }
func (B) m05() int { return 6 }
func (B) m07() int { return 8 }

func SumOdd(o Odd) int { return o.m01() + o.m03() + o.m05() + o.m07() }
`,
		"main.go": `package main

import (
	"reflect"

	"aaa.example/testitab/foreign"
	"aaa.example/testitab/pkga"
	"aaa.example/testitab/pkgb"
)

type T struct {
	foreign.Embedded
}

func (T) local() int { return 7 }

type both interface {
	foreign.I
	local() int
}

type alternating struct {
	pkga.A
	pkgb.B
}

type alternatingWithOther struct {
	pkga.A
	pkgb.B
}

func (alternatingWithOther) other() int { return 99 }

type alternatingIface interface {
	pkga.Even
	pkgb.Odd
}

type alternatingMissing interface {
	pkga.Even
	pkgb.Odd
	extra() int
}

type noHidden struct{}

func (noHidden) local() int { return 9 }

//go:noinline
func assertForeign(v any) (foreign.I, bool) {
	got, ok := v.(foreign.I)
	return got, ok
}

//go:noinline
func assertBoth(v any) (both, bool) {
	got, ok := v.(both)
	return got, ok
}

//go:noinline
func assertMissing(v any) bool {
	_, ok := v.(foreign.Missing)
	return ok
}

//go:noinline
func assertAlternating(v any) (alternatingIface, bool) {
	got, ok := v.(alternatingIface)
	return got, ok
}

//go:noinline
func assertAlternatingMissing(v any) bool {
	_, ok := v.(alternatingMissing)
	return ok
}

func main() {
	for i := 0; i < 2; i++ {
		if !reflect.TypeFor[T]().Implements(reflect.TypeFor[foreign.I]()) {
			panic("reflect.Implements(foreign.I) returned false")
		}
		if !reflect.TypeFor[T]().Implements(reflect.TypeFor[both]()) {
			panic("reflect.Implements(both) returned false")
		}
		if reflect.TypeFor[T]().Implements(reflect.TypeFor[foreign.Missing]()) {
			panic("reflect.Implements(foreign.Missing) returned true")
		}
		if reflect.TypeFor[noHidden]().Implements(reflect.TypeFor[foreign.I]()) {
			panic("noHidden unexpectedly implements foreign.I")
		}
		if !reflect.TypeFor[alternating]().Implements(reflect.TypeFor[alternatingIface]()) {
			panic("reflect.Implements(alternatingIface) returned false")
		}
		if reflect.TypeFor[alternating]().Implements(reflect.TypeFor[alternatingMissing]()) {
			panic("reflect.Implements(alternatingMissing) returned true")
		}
		fi, ok := assertForeign(T{})
		if !ok || foreign.CallHidden(fi) != 42 {
			panic("type assert to foreign.I failed")
		}
		bi, ok := assertBoth(T{})
		if !ok || foreign.CallHidden(bi) != 42 || bi.local() != 7 {
			panic("type assert to both failed")
		}
		if assertMissing(T{}) {
			panic("type assert to foreign.Missing unexpectedly succeeded")
		}
		if _, ok := assertForeign(noHidden{}); ok {
			panic("noHidden type assert to foreign.I unexpectedly succeeded")
		}
		ai, ok := assertAlternating(alternating{})
		if !ok || pkga.SumEven(ai) != 16 || pkgb.SumOdd(ai) != 20 {
			panic("type assert to alternatingIface failed")
		}
		if assertAlternatingMissing(alternating{}) {
			panic("type assert to alternatingMissing unexpectedly succeeded")
		}
		if reflect.TypeFor[alternatingWithOther]().Implements(reflect.TypeFor[alternatingMissing]()) {
			panic("alternatingWithOther unexpectedly implements alternatingMissing")
		}
		if assertAlternatingMissing(alternatingWithOther{}) {
			panic("alternatingWithOther type assert to alternatingMissing unexpectedly succeeded")
		}
	}
	println("itab-embedded-ok")
}
`,
	}
	for rel, content := range files {
		dest := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		name string
		tags string
	}{
		{name: "gc", tags: ""},
		{name: "nogc", tags: "nogc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.Tags = tc.tags
			conf.OutFile = filepath.Join(dir, "testitab-"+tc.name)
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "itab-embedded-ok" {
				t.Fatalf("testitab (%s) execution: %v\n%s", tc.name, err, out)
			}
		})
	}
}

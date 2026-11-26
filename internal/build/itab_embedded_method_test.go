//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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
		"main.go": `package main

import (
	"reflect"

	"aaa.example/testitab/foreign"
)

type T struct {
	foreign.Embedded
}

func (T) local() int { return 7 }

type both interface {
	foreign.I
	local() int
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

	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "testitab")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
		t.Fatal(err)
	}
	out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "itab-embedded-ok" {
		t.Fatalf("testitab execution: %v\n%s", err, out)
	}
}

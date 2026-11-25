package cl_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestDirectSinglePtrClosureEnvExecution(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	const source = `package main

import (
	"runtime"
	"unsafe"
)

type box struct {
	v int
}

func (b *box) add(delta int) int {
	if b == nil {
		return -delta
	}
	b.v += delta
	return b.v
}

//go:noinline
func makeImmutablePtrClosure(b *box) func(int) int {
	return func(d int) int {
		if b == nil {
			return -d
		}
		return b.v + d
	}
}

//go:noinline
func makeBoundMethod(b *box) func(int) int {
	return b.add
}

//go:noinline
func makeMutableCellClosure(start int) func() int {
	n := start
	return func() int {
		n++
		return n
	}
}

//go:noinline
func makeNestedPtrClosure(b *box) func() func(int) int {
	return func() func(int) int {
		return func(d int) int {
			if b == nil {
				return -d
			}
			return b.v + d
		}
	}
}

//go:noinline
func makeMapClosure(m map[int]int) func(int) int {
	return func(k int) int {
		if m == nil {
			return -1
		}
		return m[k]
	}
}

//go:noinline
func makeChanClosure(ch chan int) func() int {
	return func() int {
		if ch == nil {
			return -2
		}
		return <-ch
	}
}

//go:noinline
func makeUnsafePtrClosure(p unsafe.Pointer) func() uintptr {
	return func() uintptr {
		return uintptr(p)
	}
}

var sink func(int) int
var sinkNest func() func(int) int
var sinkMap func(int) int
var sinkChan func() int
var sinkUnsafe func() uintptr

func main() {
	b := &box{v: 10}
	if got := makeImmutablePtrClosure(b)(5); got != 15 {
		panic("immutable ptr closure")
	}
	if got := makeImmutablePtrClosure(nil)(5); got != -5 {
		panic("nil immutable ptr closure")
	}
	if got := makeBoundMethod(b)(3); got != 13 || b.v != 13 {
		panic("bound method")
	}
	if got := makeBoundMethod(nil)(4); got != -4 {
		panic("nil bound method")
	}
	inc := makeMutableCellClosure(20)
	if inc() != 21 || inc() != 22 {
		panic("mutable cell closure")
	}
	if got := makeNestedPtrClosure(b)()(7); got != 20 {
		panic("nested ptr closure")
	}
	if got := makeNestedPtrClosure(nil)()(7); got != -7 {
		panic("nil nested ptr closure")
	}
	m := map[int]int{3: 99}
	if got := makeMapClosure(m)(3); got != 99 {
		panic("map closure")
	}
	if got := makeMapClosure(nil)(3); got != -1 {
		panic("nil map closure")
	}
	ch := make(chan int, 1)
	ch <- 42
	if got := makeChanClosure(ch)(); got != 42 {
		panic("chan closure")
	}
	if got := makeChanClosure(nil)(); got != -2 {
		panic("nil chan closure")
	}
	if got := makeUnsafePtrClosure(nil)(); got != 0 {
		panic("nil unsafe.Pointer closure")
	}
	if got := makeUnsafePtrClosure(unsafe.Pointer(b))(); got != uintptr(unsafe.Pointer(b)) {
		panic("unsafe.Pointer closure")
	}

	// Verify steady-state zero allocations for escaped single-pointer closures
	// and bound pointer-receiver methods on native hidden-context targets.
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 256; i++ {
		sink = makeImmutablePtrClosure(b)
		if sink(1) != 14 {
			panic("sink immutable")
		}
		sink = makeBoundMethod(b)
		if sink(0) != 13 {
			panic("sink bound")
		}
		sinkNest = makeNestedPtrClosure(b)
		if sinkNest()(2) != 15 {
			panic("sink nested")
		}
		sinkMap = makeMapClosure(m)
		if sinkMap(3) != 99 {
			panic("sink map")
		}
		sinkChan = makeChanClosure(nil)
		if sinkChan() != -2 {
			panic("sink chan")
		}
		sinkUnsafe = makeUnsafePtrClosure(unsafe.Pointer(b))
		if sinkUnsafe() == 0 {
			panic("sink unsafe")
		}
	}
	runtime.ReadMemStats(&after)
	if after.Mallocs != before.Mallocs {
		panic("unexpected allocation for direct single-pointer closure")
	}
	println("direct-closure-ok")
}
`
	for name, text := range map[string]string{
		"go.mod": "module directclosureexec\n\ngo 1.27.0\n", "main.go": source,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, opt := range []optlevel.Level{optlevel.O0, optlevel.O2} {
		t.Run(opt.String(), func(t *testing.T) {
			conf := build.NewDefaultConf(build.ModeBuild)
			conf.OptLevel = opt
			conf.OutFile = filepath.Join(dir, "directclosure-"+opt.String())
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := build.Build(build.Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "direct-closure-ok" {
				t.Fatalf("direct closure execution (%s): %v\n%s", opt, err, out)
			}
		})
	}
}

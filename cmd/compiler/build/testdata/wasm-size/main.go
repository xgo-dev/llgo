package main

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"unicode"
)

var effects int
var unused = map[int]int{sideEffect(): sideEffect()}
var alias = unused
var kept = map[string]int{"a": 4}

func sideEffect() int { effects++; return effects }

func main() {
	checkStaticClosures()
	checkMethodMetadata()
	if effects != 2 || kept["a"] != 4 {
		panic("initializer effects lost")
	}
	kept["a"]++
	if kept["a"] != 5 || !unicode.Is(unicode.Scripts["Han"], '中') {
		panic("live map lost")
	}
	capture()
	if len(os.Args) > 1 && os.Args[1] == "panic" {
		panicSite()
	}
	fmt.Printf("effects=%d map=%d value=%08x float=%.2f text=%q\n", effects, kept["a"], 42, 1.25, "中")
}

// This key needs its structural metadata for hashing and equality, but never
// exposes its method set through an interface or reflection.
type privateKey struct{ N int }

func (k privateKey) Number() int { return k.N }

type observed struct{}

func (observed) Number(n int) int { return n }
func (observed) hidden()          {}

func checkMethodMetadata() {
	k := privateKey{42}
	m := map[privateKey]int{k: 7}
	if m[privateKey{42}] != 7 || k.Number() != 42 {
		panic("structural metadata or direct method lost")
	}
	// Reach a method-bearing type through a reflected parent's field. Method
	// names/signatures are observable even before the method is invoked.
	typ := reflect.TypeFor[struct{ Value observed }]().Field(0).Type
	iface := reflect.TypeFor[interface{ Number(int) int }]()
	if typ.NumMethod() != 1 || !typ.Implements(iface) {
		panic("reflected method set lost")
	}
	method := typ.Method(0)
	if method.Name != "Number" || method.Type.NumIn() != 2 || method.Type.NumOut() != 1 {
		panic("reflected method signature lost")
	}
	values := method.Func.Call([]reflect.Value{reflect.ValueOf(observed{}), reflect.ValueOf(42)})
	if values[0].Int() != 42 {
		panic("reflected method implementation lost")
	}
}

//go:noinline
func capture() {
	_, file, line, ok := runtime.Caller(0)
	if !ok || !strings.HasSuffix(file, "/main.go") || line <= 0 {
		panic("Caller location lost")
	}
	pcs := make([]uintptr, 8)
	n := runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()
	if !strings.HasSuffix(frame.Function, ".capture") || frame.File != file || frame.Line <= 0 {
		panic("CallersFrames location lost")
	}
}

//go:noinline
func panicSite() { panic("size-probe-panic") }

//go:noinline
func unusedFunctionMetadata() int { return 42 }

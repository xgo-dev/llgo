package build

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestRuntimeAtomicARM64FixtureHasCompleteCalls(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", runtimeAtomicProgram(t), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	uses := make(map[string]int)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "main" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok && strings.HasPrefix(identifier.Name, "native") {
				uses[strings.TrimPrefix(identifier.Name, "native")]++
			}
			return true
		})
	}
	if len(uses) != 49 {
		t.Fatal("the numerical driver does not call the complete actual SDK inventory", uses)
	}
	for _, row := range runtimeAtomicRows() {
		if uses[row.name] != 1 {
			t.Fatal("the driver duplicated or omitted a real SDK wrapper", row.name, uses[row.name])
		}
	}
}

func runtimeAtomicProgram(t *testing.T) string {
	t.Helper()
	var declarations, calls strings.Builder
	for _, row := range runtimeAtomicRows() {
		fmt.Fprintf(&declarations, "\n//go:linkname native%s internal/runtime/atomic.%s\n%s\n",
			row.name, row.name, strings.Replace(row.declaration(), "func "+row.name, "func native"+row.name, 1))
		entry := "atomicEntry(native" + row.name + ")"
		switch row.family {
		case "load-pointer":
			fmt.Fprintf(&calls, "checkPointerLoad(%s, a, b)\n", entry)
		case "store-pointer":
			fmt.Fprintf(&calls, "checkPointerStore(%s, a, b)\n", entry)
		case "cas-pointer":
			fmt.Fprintf(&calls, "checkPointerCAS(%s, a, b)\n", entry)
		case "and-void", "or-void":
			fmt.Fprintf(&calls, "checkBitwiseVoid(%q, %s, %s(input), %t)\n",
				row.name, entry, row.value, row.family == "and-void")
		case "and", "or", "xchg":
			fmt.Fprintf(&calls, "checkRMW(%q, %s, %s(input), %q)\n", row.name, entry, row.value, row.family)
		case "add":
			fmt.Fprintf(&calls, "checkAdd(%q, %s, %s(input), %s(input ^ 0xa5137f9b))\n",
				row.name, entry, row.value, row.delta)
		default:
			fmt.Fprintf(&calls, "check%s(%q, %s, %s(input))\n",
				strings.ToUpper(row.family[:1])+row.family[1:], row.name, entry, row.value)
		}
	}
	source := runtimeAtomicDriver + declarations.String() + `
func main() {
	inputs := [...]uint64{
		0, 1, 0x80, 0x8000, 0x80000000, 0x8000000000000000,
		^uint64(0), 0x76543210fedcba98, 0xa513cf402e97b681,
	}
	for _, input := range inputs {
		first, second := new(uint64), new(uint64)
		*first, *second = input, input ^ 0x5a
		a, b := unsafe.Pointer(first), unsafe.Pointer(second)
` + calls.String() + `
	}
	checkConcurrent()
	fmt.Println("atomic wrappers 49 checks", checks)
}
`
	formatted, err := format.Source([]byte(source))
	if err != nil {
		t.Fatal("format the bounded complete atomic fixture", err)
	}
	return string(formatted)
}

// All 49 function values call the actual SDK entry, rather than substituting
// sync/atomic. Only private operations inside the future wrappers are intrinsics.
// Both compilers execute this same compact table, values and canary logic.
const runtimeAtomicDriver = `package main

import (
	"fmt"
	"runtime"
	"unsafe"
)

type number interface {
	~uint8 | ~uint32 | ~uint64 | ~uint | ~uintptr | ~int32 | ~int64
}

var checks int

//go:noinline
func atomicEntry[F any](function F) F {
	return function
}

func require(name string, condition bool) {
	if !condition {
		panic(name)
	}
	checks++
}

func checkLoad[T number](name string, call func(*T) T, value T) {
	slots := [3]T{0x5a, value, 0x33}
	before := slots
	got := call(&slots[1])
	require(name, got == value && slots == before)
}

func checkStore[T number](name string, call func(*T, T), value T) {
	slots := [3]T{0x5a, value, 0x33}
	next := value ^ T(0x55)
	call(&slots[1], next)
	require(name, slots == [3]T{0x5a, next, 0x33})
}

func checkCas[T number](name string, call func(*T, T, T) bool, value T) {
	slots := [3]T{0x5a, value, 0x33}
	next := value ^ T(0x55)
	passed := call(&slots[1], value, next)
	failed := call(&slots[1], value, value)
	require(name, passed && !failed && slots == [3]T{0x5a, next, 0x33})
}

func checkAdd[T number, D number](name string, call func(*T, D) T, value T, delta D) {
	slots := [3]T{0x5a, value, 0x33}
	want := value + T(delta)
	got := call(&slots[1], delta)
	require(name, got == want && slots == [3]T{0x5a, want, 0x33})
}

func checkRMW[T number](name string, call func(*T, T) T, value T, operation string) {
	slots := [3]T{0x5a, value, 0x33}
	operand := value ^ T(0x55)
	want := operand
	if operation == "and" {
		want = value & operand
	} else if operation == "or" {
		want = value | operand
	}
	old := call(&slots[1], operand)
	require(name, old == value && slots == [3]T{0x5a, want, 0x33})
}

func checkBitwiseVoid[T number](name string, call func(*T, T), value T, and bool) {
	slots := [3]T{0x5a, value, 0x33}
	operand := value ^ T(0x55)
	want := value | operand
	if and {
		want = value & operand
	}
	call(&slots[1], operand)
	require(name, slots == [3]T{0x5a, want, 0x33})
}

func checkPointerLoad(call func(unsafe.Pointer) unsafe.Pointer, a, b unsafe.Pointer) {
	slots := [3]unsafe.Pointer{a, b, a}
	got := call(unsafe.Pointer(&slots[1]))
	require("Loadp", got == b && slots == [3]unsafe.Pointer{a, b, a})
}

func checkPointerStore(call func(unsafe.Pointer, unsafe.Pointer), a, b unsafe.Pointer) {
	slots := [3]unsafe.Pointer{a, a, a}
	call(unsafe.Pointer(&slots[1]), b)
	require("StorepNoWB", slots == [3]unsafe.Pointer{a, b, a})
}

func checkPointerCAS(call func(*unsafe.Pointer, unsafe.Pointer, unsafe.Pointer) bool, a, b unsafe.Pointer) {
	slots := [3]unsafe.Pointer{a, a, a}
	passed := call(&slots[1], a, b)
	failed := call(&slots[1], a, a)
	require("Casp1", a != b && passed && !failed && slots == [3]unsafe.Pointer{a, b, a})
}

func checkConcurrent() {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(4))
	var counter uint32
	done := make(chan bool)
	for worker := 0; worker < 4; worker++ {
		go func() {
			add := atomicEntry(nativeXadd)
			for step := 0; step < 256; step++ {
				add(&counter, 1)
			}
			done <- true
		}()
	}
	for worker := 0; worker < 4; worker++ {
		<-done
	}
	if atomicEntry(nativeLoad)(&counter) != 1024 {
		panic("contended Xadd lost an update")
	}
	var ready uint32
	var payload uint64
	go func() {
		payload = 0x76543210fedcba98
		atomicEntry(nativeStoreRel)(&ready, 1)
		done <- true
	}()
	for atomicEntry(nativeLoadAcq)(&ready) == 0 {
		runtime.Gosched()
	}
	if payload != 0x76543210fedcba98 {
		panic("release/acquire publication lost payload")
	}
	<-done
}
`

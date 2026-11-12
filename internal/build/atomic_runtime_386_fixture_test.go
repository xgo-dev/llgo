//go:build go1.27

package build

import (
	"fmt"
	"go/ast"
	"go/format"
	"strings"
	"testing"
)

// The five 64-bit operations and their signed aliases require an alignment
// panic on 386. And64/Or64 have a different original SDK contract.
func runtimeAtomic386AlignmentRows() []runtimeAtomicRow {
	return []runtimeAtomicRow{
		{"Load64", "load", "uint64", ""},
		{"Loadint64", "load", "int64", ""},
		{"Store64", "store", "uint64", ""},
		{"Storeint64", "store", "int64", ""},
		{"Cas64", "cas", "uint64", ""},
		{"Casint64", "cas", "int64", ""},
		{"Xadd64", "add", "uint64", "int64"},
		{"Xaddint64", "add", "int64", "int64"},
		{"Xchg64", "xchg", "uint64", ""},
		{"Xchgint64", "xchg", "int64", ""},
	}
}

func TestRuntimeAtomic386FixtureHasCompleteCalls(t *testing.T) {
	functions := runtimeAtomicFunctions(t, []byte(runtimeAtomic386Program(t)))
	main := functions["main"]
	if main == nil || main.Body == nil {
		t.Fatal("the 386 execution fixture lacks its main driver")
	}
	entries, uses := make(map[string]int), make(map[string]int)
	ast.Inspect(main.Body, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && strings.HasPrefix(identifier.Name, "native") {
			uses[strings.TrimPrefix(identifier.Name, "native")]++
		}
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		function, ok := call.Fun.(*ast.Ident)
		if !ok || function.Name != "atomicEntry" {
			return true
		}
		entry, ok := call.Args[0].(*ast.Ident)
		if ok && strings.HasPrefix(entry.Name, "native") {
			entries[strings.TrimPrefix(entry.Name, "native")]++
		}
		return true
	})
	if len(entries) != 10 || len(uses) != 10 {
		t.Fatal("the 386 fixture omitted an actual SDK function value", entries, uses)
	}
	for _, row := range runtimeAtomic386AlignmentRows() {
		function := functions["native"+row.name]
		if function == nil || function.Body != nil || function.Doc == nil ||
			len(function.Doc.List) != 1 ||
			function.Doc.List[0].Text != "//go:linkname native"+row.name+" internal/runtime/atomic."+row.name {
			t.Fatal("the fixture does not select the real SDK entry", row.name)
		}
		original := runtimeAtomicFunctions(t, []byte("package atomic\n"+row.declaration()))[row.name]
		if runtimeAtomicSignature(t, function) != runtimeAtomicSignature(t, original) ||
			entries[row.name] != 1 || uses[row.name] != 1 {
			t.Fatal("the fixture changed, omitted or directly called an SDK entry", row.name)
		}
	}
	entry := functions["atomicEntry"]
	if entry == nil || entry.Doc == nil || len(entry.Doc.List) != 1 ||
		entry.Doc.List[0].Text != "//go:noinline" {
		t.Fatal("the real function-value barrier lost its noinline contract")
	}
}

func runtimeAtomic386Program(t *testing.T) string {
	t.Helper()
	var declarations, calls strings.Builder
	for _, row := range runtimeAtomic386AlignmentRows() {
		fmt.Fprintf(&declarations, "\n//go:linkname native%s internal/runtime/atomic.%s\n%s\n",
			row.name, row.name, strings.Replace(row.declaration(), "func "+row.name, "func native"+row.name, 1))
		fmt.Fprintf(&calls, "check%s(%q, atomicEntry(native%s), %s(input))\n",
			strings.ToUpper(row.family[:1])+row.family[1:], row.name, row.name, row.value)
	}
	source := runtimeAtomic386Driver + declarations.String() + `
func main() {
	if unsafe.Sizeof(uintptr(0)) != 4 {
		panic("the 386 atomic fixture must execute as a real 32-bit program")
	}
	inputs := [...]uint64{
		0, 1, 0xffffffff, 0x100000000, 0x8000000000000000,
		^uint64(0), 0x76543210fedcba98, 0xa513cf402e97b681,
	}
	for _, input := range inputs {
` + calls.String() + `
	}
	fmt.Println("atomic 386 wrappers 10 aligned", alignedChecks, "unaligned", unalignedChecks)
}
`
	formatted, err := format.Source([]byte(source))
	if err != nil {
		t.Fatal("format the actual 386 atomic execution fixture", err)
	}
	return string(formatted)
}

// Both compilers call the same real runtime entries through function values.
// No sync/atomic public intrinsic can replace the operation under examination.
const runtimeAtomic386Driver = `package main

import (
	"fmt"
	"unsafe"
)

type number interface { ~int64 | ~uint64 }

var alignedChecks, unalignedChecks int

//go:noinline
func atomicEntry[F any](function F) F { return function }

func expectUnaligned(name string, call func()) {
	defer func() {
		message, ok := recover().(string)
		if !ok || message != "unaligned 64-bit atomic operation" {
			panic(name + ": missing original alignment panic")
		}
		unalignedChecks++
	}()
	call()
}

func checkMemory[T number](name string, value T, operation func(*T) (T, bool)) {
	var memory [48]byte
	for index := range memory {
		memory[index] = byte(index*37 + 0x5a)
	}
	// An explicit interior alignment keeps canaries on both sides, even when
	// the native 32-bit stack or the enclosing allocation is only word aligned.
	offset := int((8 - (uintptr(unsafe.Pointer(&memory[0])) & 7)) & 7) + 8
	pointer := (*T)(unsafe.Pointer(&memory[offset]))
	*pointer = value
	want := memory
	next, correct := operation(pointer)
	for index := 0; index < 8; index++ {
		want[offset+index] = byte(uint64(next) >> (index*8))
	}
	if !correct || memory != want {
		panic(name + ": incorrect aligned value or overwritten canary")
	}
	alignedChecks++
	for misalignment := 1; misalignment <= 7; misalignment++ {
		before := memory
		expectUnaligned(name, func() {
			operation((*T)(unsafe.Pointer(&memory[offset+misalignment])))
		})
		if memory != before {
			panic(name + ": unaligned operation modified memory before its panic")
		}
	}
}

func checkLoad[T number](name string, call func(*T) T, value T) {
	checkMemory(name, value, func(pointer *T) (T, bool) {
		return value, call(pointer) == value
	})
}

func checkStore[T number](name string, call func(*T, T), value T) {
	checkMemory(name, value, func(pointer *T) (T, bool) {
		next := value ^ T(0x514f73a2)
		call(pointer, next)
		return next, true
	})
}

func checkCas[T number](name string, call func(*T, T, T) bool, value T) {
	checkMemory(name, value, func(pointer *T) (T, bool) {
		next := value ^ T(0x514f73a2)
		passed := call(pointer, value, next)
		failed := call(pointer, value, value)
		return next, passed && !failed
	})
}

func checkAdd[T number](name string, call func(*T, int64) T, value T) {
	checkMemory(name, value, func(pointer *T) (T, bool) {
		delta := int64(-0x102030405)
		next := value + T(delta)
		return next, call(pointer, delta) == next
	})
}

func checkXchg[T number](name string, call func(*T, T) T, value T) {
	checkMemory(name, value, func(pointer *T) (T, bool) {
		next := value ^ T(0x514f73a2)
		return next, call(pointer, next) == value
	})
}
`

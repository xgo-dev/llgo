package gotest

import (
	"reflect"
	"runtime"
	"testing"
	"unsafe"
)

// Wasm stores an indirect table index in the descriptor. LLGo normalizes
// that entry to its runtime PC namespace for reflection and FuncForPC.
func functionTokenCodePC(entry uintptr) uintptr {
	if runtime.Compiler == "llgo" {
		return functionTokenRuntimePC(unsafe.Pointer(entry))
	}
	return entry
}

//go:linkname functionTokenRuntimePC github.com/xgo-dev/llgo/runtime/internal/runtime.FunctionPC
func functionTokenRuntimePC(unsafe.Pointer) uintptr

func TestFunctionTokenOneWordStorage(t *testing.T) {
	var fn func(int) int
	if got, want := unsafe.Sizeof(fn), unsafe.Sizeof(uintptr(0)); got != want {
		t.Fatalf("function storage = %d bytes, want one Go word (%d)", got, want)
	}
	var item struct {
		prefix byte
		fn     func(int) int
		suffix byte
	}
	if got, want := unsafe.Offsetof(item.suffix)-unsafe.Offsetof(item.fn), unsafe.Sizeof(uintptr(0)); got != want {
		t.Fatalf("function struct field occupies %d bytes, want %d", got, want)
	}
}

func TestFunctionTokenDescriptorEntry(t *testing.T) {
	token := functionTokenFromInterface(functionTokenTwice)
	got := functionTokenCodePC(*(*uintptr)(unsafe.Pointer(token)))
	if want := reflect.ValueOf(functionTokenTwice).Pointer(); got != want {
		t.Fatalf("descriptor entry = %#x, want code PC %#x", got, want)
	}
}

func TestFunctionTokenCapturedTypedIdentity(t *testing.T) {
	t.Cleanup(func() { functionTokenTestRoots = nil })
	value := 35
	fn := func(n int) int {
		value += n
		return value
	}
	functionTokenTestRoots = append(functionTokenTestRoots, fn)
	typed := functionTokenFromTyped(fn)
	alias := fn
	boxed := functionTokenFromInterface(alias)
	if typed == 0 || typed != functionTokenFromTyped(alias) || typed != boxed {
		t.Fatal("copies of one captured function lost descriptor identity")
	}
	runtime.GC()
	if got := functionTokenToIntFunc(typed)(7); got != 42 {
		t.Fatalf("captured typed call = %d, want 42", got)
	}
	if got := alias(1); got != 43 || value != 43 {
		t.Fatalf("original captured cell = %d, want 43", got)
	}
	runtime.GC()
	if got := functionTokenToStructIntFunc(boxed)(-1); got != 42 || functionTokenTestRoots[0](0) != 42 {
		t.Fatalf("captured descriptor after GC = %d, want 42", got)
	}
}

func TestFunctionTokenReflectMakeFunc(t *testing.T) {
	t.Cleanup(func() { functionTokenTestRoots = nil })
	var signature func(int) int
	delta := 7
	fn := reflect.MakeFunc(reflect.TypeOf(signature), func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(int(args[0].Int()) + delta)}
	}).Interface().(func(int) int)
	functionTokenTestRoots = append(functionTokenTestRoots, fn)
	token := functionTokenFromTyped(fn)
	if token != functionTokenFromInterface(fn) || token != functionTokenFromTyped(fn) {
		t.Fatal("runtime-created function lost its descriptor identity")
	}
	runtime.GC()
	if got := functionTokenToIntFunc(token)(35); got != 42 {
		t.Fatalf("runtime-created function call = %d, want 42", got)
	}
	if got := functionTokenCodePC(*(*uintptr)(unsafe.Pointer(token))); got != reflect.ValueOf(fn).Pointer() {
		t.Fatal("runtime-created descriptor did not retain its actual code entry")
	}
	if functionTokenTestRoots[0](35) != 42 {
		t.Fatal("runtime-created function's typed root changed")
	}
}

func TestFunctionTokenStorageCopies(t *testing.T) {
	fn := functionTokenDynamicDeclaration()
	array := [1]func(int) int{fn}
	slice := []func(int) int{array[0]}
	values := map[string]func(int) int{"fn": slice[0]}
	var boxed any = values["fn"]
	copy := boxed.(func(int) int)
	want := functionTokenFromTyped(fn)
	if got := functionTokenFromTyped(copy); got != want || copy(21) != 42 {
		t.Fatalf("function copies changed identity: %#x, want %#x", got, want)
	}
	alias := &slice[0]
	*alias = copy
	if got := functionTokenFromTyped(*alias); got != want {
		t.Fatalf("function slot alias changed identity: %#x, want %#x", got, want)
	}
}

// Both paths expose the same gc function descriptor: interface payload data
// and the single word stored in a function-valued temporary.
func functionTokenFromInterface(fn any) uintptr {
	type words [2]uintptr
	return (*words)(unsafe.Pointer(&fn))[1]
}

func functionTokenFromTyped[T any](fn T) uintptr {
	return *(*uintptr)(unsafe.Pointer(&struct{ fn T }{fn}))
}

func functionTokenToIntFunc(token uintptr) func(int) int {
	return *(*func(int) int)(unsafe.Pointer(&struct{ uintptr }{token}))
}

func functionTokenToStructIntFunc(token uintptr) func(int) int {
	return (*struct{ fn func(int) int })(unsafe.Pointer(&struct{ uintptr }{token})).fn
}

func functionTokenTwice(n int) int { return n * 2 }

//go:noinline
func functionTokenDynamicDeclaration() func(int) int { return functionTokenTwice }

//go:noinline
func functionTokenDynamicTyped(fn func(int) int) uintptr {
	return *(*uintptr)(unsafe.Pointer(&struct{ fn func(int) int }{fn}))
}

func TestFunctionTokenDynamicNoEnvironment(t *testing.T) {
	first := functionTokenDynamicTyped(functionTokenDynamicDeclaration())
	second := functionTokenDynamicTyped(functionTokenDynamicDeclaration())
	if first == 0 || first != second {
		t.Fatalf("dynamic noncaptured function tokens differ: %#x, %#x", first, second)
	}
	if got := functionTokenToIntFunc(first)(21); got != 42 {
		t.Fatalf("dynamic noncaptured function token call = %d, want 42", got)
	}
}

// Retain original Go function values, not the implementation's token boxes.
// A uintptr by itself does not keep a gc function value alive or prevent a
// stack-resident closure from moving when the goroutine stack grows.
var functionTokenTestRoots []func(int) int

func TestFunctionTokenInterfaceRoundTrip(t *testing.T) {
	token := functionTokenFromInterface(functionTokenTwice)
	if token == 0 {
		t.Fatal("non-nil function has a zero interface token")
	}
	if got := functionTokenToIntFunc(token)(21); got != 42 {
		t.Fatalf("interface token call = %d, want 42", got)
	}
	if got := functionTokenToStructIntFunc(token)(21); got != 42 {
		t.Fatalf("struct-view token call = %d, want 42", got)
	}
}

func TestFunctionTokenTypedRoundTrip(t *testing.T) {
	token := functionTokenFromTyped(functionTokenTwice)
	if token == 0 {
		t.Fatal("non-nil function has a zero typed token")
	}
	if got := functionTokenToIntFunc(token)(21); got != 42 {
		t.Fatalf("typed token call = %d, want 42", got)
	}

	type named func(int) int
	var fn named = functionTokenTwice
	namedToken := *(*uintptr)(unsafe.Pointer(&struct{ fn named }{fn}))
	if got := (*(*named)(unsafe.Pointer(&struct{ uintptr }{namedToken})))(21); got != 42 {
		t.Fatalf("named typed token call = %d, want 42", got)
	}
}

func capturedInterfaceFunctionToken(delta int) (uintptr, func(int) int) {
	fn := func(n int) int { return n + delta }
	functionTokenTestRoots = append(functionTokenTestRoots, fn)
	return functionTokenFromInterface(fn), fn
}

func TestFunctionTokenCapturesAndGC(t *testing.T) {
	t.Cleanup(func() { functionTokenTestRoots = nil })
	t.Run("Interface", func(t *testing.T) {
		checkFunctionTokenCapturesAndGC(t)
	})
}

func checkFunctionTokenCapturesAndGC(t *testing.T) {
	t.Helper()
	firstToken, first := capturedInterfaceFunctionToken(7)
	secondToken, second := capturedInterfaceFunctionToken(9)
	firstCopy := firstToken
	for attempt := 0; attempt < 5; attempt++ {
		runtime.GC()
		if got := functionTokenToIntFunc(firstToken)(35); got != 42 {
			t.Fatalf("first environment call = %d, want 42", got)
		}
		if got := functionTokenToStructIntFunc(secondToken)(33); got != 42 {
			t.Fatalf("second environment call = %d, want 42", got)
		}
		if got := functionTokenToIntFunc(firstCopy)(35); got != first(35) {
			t.Fatalf("copied token call = %d, original call = %d", got, first(35))
		}
		if firstCopy != firstToken || second(33) != 42 {
			t.Fatal("token copy or second original closure changed")
		}
	}

	value := 35
	shared := func(n int) int {
		value += n
		return value
	}
	functionTokenTestRoots = append(functionTokenTestRoots, shared)
	sharedToken := functionTokenFromInterface(shared)
	recovered := functionTokenToIntFunc(sharedToken)
	if got := recovered(7); got != 42 || shared(1) != 43 {
		t.Fatalf("recovered closure lost the original captured cell: %d, cell %d", got, value)
	}
	runtime.GC()
	if got := recovered(-1); got != 42 || value != 42 {
		t.Fatalf("shared captured cell after GC = %d, cell %d", got, value)
	}
	if got := functionTokenTestRoots[len(functionTokenTestRoots)-1](0); got != 42 {
		t.Fatalf("typed root lost its shared captured cell: %d", got)
	}
}

func TestFunctionTokenNil(t *testing.T) {
	var fn func(int) int
	if token := functionTokenFromTyped(fn); token != 0 {
		t.Fatalf("nil typed function token = %#x, want 0", token)
	}
	if functionTokenToIntFunc(0) != nil || functionTokenToStructIntFunc(0) != nil {
		t.Fatal("zero token did not decode to a nil function")
	}
	if functionTokenToIntFunc(functionTokenFromInterface(fn)) != nil {
		t.Fatal("boxed nil function did not decode to nil")
	}
}

func TestFunctionTokenStableIdentity(t *testing.T) {
	t.Cleanup(func() { functionTokenTestRoots = nil })
	t.Run("Static", func(t *testing.T) {
		first := functionTokenFromTyped(functionTokenTwice)
		second := functionTokenFromTyped(functionTokenTwice)
		interfaced := functionTokenFromInterface(functionTokenTwice)
		if first != second || first != interfaced {
			t.Fatalf("static function tokens differ: %#x, %#x, %#x", first, second, interfaced)
		}
	})
}

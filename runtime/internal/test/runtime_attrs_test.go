//go:build llgo

package test

import (
	"testing"
	"unsafe"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

//go:linkname runtimeMemequal github.com/xgo-dev/llgo/runtime/internal/runtime.memequal
func runtimeMemequal(a, b unsafe.Pointer, size uintptr) bool

//go:linkname reflectTypedmemmove reflect.typedmemmove
func reflectTypedmemmove(typ *rt.Type, dst, src unsafe.Pointer)

func TestRuntimeCheckedPointer(t *testing.T) {
	x := 42
	p := unsafe.Pointer(&x)
	if rt.AssertNilDerefPtr(p) != p {
		t.Fatal("checked pointer identity changed")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("nil pointer check did not raise a recoverable panic")
		}
	}()
	rt.AssertNilDerefPtr(nil)
	t.Fatal("nil pointer check returned")
}

func TestRuntimeMemoryContracts(t *testing.T) {
	if !runtimeMemequal(nil, nil, 0) {
		t.Fatal("zero-byte equality must accept nil")
	}
	a, b := [4]byte{1, 2, 3, 4}, [4]byte{1, 2, 3, 4}
	if !runtimeMemequal(unsafe.Pointer(&a), unsafe.Pointer(&b), 4) {
		t.Fatal("equal bytes differ")
	}
	b[2] = 9
	if runtimeMemequal(unsafe.Pointer(&a), unsafe.Pointer(&b), 4) {
		t.Fatal("equality did not observe an intervening write")
	}
	// memmove permits overlap in both directions. The reflect linkname entry
	// must retain the same contract as the compiler's runtime entry.
	for _, move := range []func(*rt.Type, unsafe.Pointer, unsafe.Pointer){rt.Typedmemmove, reflectTypedmemmove} {
		x := [5]byte{1, 2, 3, 4, 5}
		typ := rt.Type{Size_: 4}
		move(&typ, unsafe.Pointer(&x[1]), unsafe.Pointer(&x[0]))
		if x != [5]byte{1, 1, 2, 3, 4} {
			t.Fatalf("forward overlap: %v", x)
		}
		move(&typ, unsafe.Pointer(&x[0]), unsafe.Pointer(&x[1]))
		if x != [5]byte{1, 2, 3, 4, 4} {
			t.Fatalf("backward overlap: %v", x)
		}
		move(nil, nil, nil) // equality fast path does not inspect the type
	}
	rt.Typedmemclr(&rt.Type{Size_: 4}, unsafe.Pointer(&a))
	if a != [4]byte{} {
		t.Fatalf("clear: %v", a)
	}
	buf := [8]byte{7, 7, 7, 7, 7, 7, 7, 7}
	for _, s := range []string{"", "abc"} {
		result := rt.CStrCopy(unsafe.Pointer(&buf[0]), *(*rt.String)(unsafe.Pointer(&s)))
		if unsafe.Pointer(result) != unsafe.Pointer(&buf[0]) || buf[len(s)] != 0 || string(buf[:len(s)]) != s {
			t.Fatalf("C string copy %q: %v", s, buf)
		}
	}
}

func TestRuntimeReadContracts(t *testing.T) {
	if rt.MapLen(nil) != 0 || rt.ChanCap(nil) != 0 {
		t.Fatal("nil length/capacity must be zero")
	}
	m := map[int]int{1: 10}
	mp := (*rt.Map)(*(*unsafe.Pointer)(unsafe.Pointer(&m)))
	if rt.MapLen(mp) != 1 {
		t.Fatal("map length")
	}
	m[2] = 20
	if rt.MapLen(mp) != 2 {
		t.Fatal("map length did not observe insertion")
	}
	ch := make(chan int, 3)
	cp := (*rt.Chan)(*(*unsafe.Pointer)(unsafe.Pointer(&ch)))
	if rt.ChanCap(cp) != 3 {
		t.Fatal("channel capacity")
	}
	for _, pair := range []struct {
		a, b        string
		equal, less bool
	}{
		{"", "", true, false}, {"", "x", false, true},
		{"ab", "abc", false, true}, {"abc", "abd", false, true},
		{"abc", "abc", true, false}, {"abd", "abc", false, false},
	} {
		ra, rb := *(*rt.String)(unsafe.Pointer(&pair.a)), *(*rt.String)(unsafe.Pointer(&pair.b))
		if rt.StringEqual(ra, rb) != pair.equal || rt.StringLess(ra, rb) != pair.less {
			t.Fatalf("string comparison: %q %q", pair.a, pair.b)
		}
	}
}

func TestRuntimePanicContracts(t *testing.T) {
	for _, raise := range []func(){
		func() { rt.Panic("test") },
		func() { rt.Panic(nil) },
		func() { rt.PanicErrorString("test") },
		func() { rt.PanicIndex(3, 2) },
		func() { rt.PanicIndexU(3, 2) },
		func() { rt.PanicSliceConvert(3, 2) },
		func() { rt.PanicTypeAssertionError("test") },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("panic was not recoverable")
				}
			}()
			raise()
			t.Fatal("panic entry returned normally")
		}()
	}
	// Rethrow is also used with no active panic and must return normally.
	rt.Rethrow(nil)
}

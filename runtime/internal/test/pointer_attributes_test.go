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
}

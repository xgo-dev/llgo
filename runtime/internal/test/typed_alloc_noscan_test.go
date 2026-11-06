//go:build llgo && !nogc && !baremetal && !wasm

package test

import (
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func TestTypedAllocationCollectorKinds(t *testing.T) {
	atomic := rt.AllocAtomicU(32)
	scanned := rt.AllocU(32)
	atomicKind := gcGetKindAndSize(atomic, nil)
	scannedKind := gcGetKindAndSize(scanned, nil)
	if atomicKind == scannedKind {
		t.Fatal("collector kinds are indistinguishable")
	}
	bytes := make([]byte, 7, 17)
	runes := append(make([]rune, 0, 1), 1, 2, 3)
	floats := []float64{1, 2, 3}
	strs := append(make([]string, 0, 1), "first", "second")
	ptrs := make([]*scalarObject, 3)
	iface := make([]any, 3)
	obj := new(scalarObject)
	boxed := any(scalarObject{Value: 42})
	// An empty interface is (type, data); the data points to the boxed object.
	boxPtr := (*[2]unsafe.Pointer)(unsafe.Pointer(&boxed))[1]
	rbyte := reflect.New(reflect.TypeOf([]byte{})).Elem()
	rbyte.Set(reflect.MakeSlice(rbyte.Type(), 3, 7))
	rstr := reflect.New(reflect.TypeOf([]string{})).Elem()
	rstr.Set(reflect.MakeSlice(rstr.Type(), 3, 7))
	rbyte.Grow(100)
	rstr.Grow(100)
	rnew := reflect.New(reflect.TypeOf(scalarObject{}))
	rptr := reflect.New(reflect.TypeOf(struct{ S string }{}))
	// Small slice buffers must preserve the allocation kind when capacity
	// exceeds the caller's buffer or no buffer is available.
	var buffer [4]byte
	byteFallback := rt.MakeSliceBuffer(3, 7, 1, 1, unsafe.Pointer(&buffer[0]), len(buffer), false)
	bytesFromBuffer := *(*[]byte)(unsafe.Pointer(&byteFallback))
	stringFallback := rt.MakeSliceBuffer(3, 7, int(unsafe.Sizeof("")), int(unsafe.Alignof("")), nil, 0, true)
	stringsFromBuffer := *(*[]string)(unsafe.Pointer(&stringFallback))
	for _, tc := range []struct {
		name   string
		ptr    unsafe.Pointer
		noscan bool
	}{
		{"make bytes", unsafe.Pointer(unsafe.SliceData(bytes)), true},
		{"grow runes", unsafe.Pointer(unsafe.SliceData(runes)), true},
		{"literal floats", unsafe.Pointer(unsafe.SliceData(floats)), true},
		{"grow strings", unsafe.Pointer(unsafe.SliceData(strs)), false},
		{"make pointers", unsafe.Pointer(unsafe.SliceData(ptrs)), false},
		{"make interfaces", unsafe.Pointer(unsafe.SliceData(iface)), false},
		{"new scalar", unsafe.Pointer(obj), true},
		{"box scalar", boxPtr, true},
		{"reflect grow bytes", rbyte.UnsafePointer(), true},
		{"reflect grow strings", rstr.UnsafePointer(), false},
		{"reflect new scalar", rnew.UnsafePointer(), true},
		{"reflect new string header", rptr.UnsafePointer(), false},
		{"byte buffer fallback", unsafe.Pointer(unsafe.SliceData(bytesFromBuffer)), true},
		{"string buffer fallback", unsafe.Pointer(unsafe.SliceData(stringsFromBuffer)), false},
		{"aligned atomic", rt.AllocAtomicZAligned(37, 64), true},
	} {
		base := bdwgc.Base(tc.ptr)
		if base == nil {
			t.Fatalf("%s: missing GC allocation", tc.name)
		}
		want := scannedKind
		if tc.noscan {
			want = atomicKind
		}
		if got := gcGetKindAndSize(base, nil); got != want {
			t.Errorf("%s kind = %d, want %d", tc.name, got, want)
		}
		runtime.KeepAlive(tc.ptr)
	}
	runtime.KeepAlive(boxed)
	runtime.KeepAlive(atomic)
	runtime.KeepAlive(scanned)
}

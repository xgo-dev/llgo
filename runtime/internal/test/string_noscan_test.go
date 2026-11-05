//go:build llgo && !nogc && !baremetal && !wasm

package test

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

//go:linkname gcGetKindAndSize C.GC_get_kind_and_size
func gcGetKindAndSize(ptr unsafe.Pointer, size *uintptr) int32

func TestStringAllocationsNoScan(t *testing.T) {
	atomic := bdwgc.MallocAtomic(32)
	wantKind := gcGetKindAndSize(atomic, nil)
	scanned := rt.AllocU(32)
	if gcGetKindAndSize(scanned, nil) == wantKind {
		t.Fatal("scanned and atomic allocations unexpectedly share a collector kind")
	}
	text := "hello\x00界🙂"
	data := []byte(text)
	from := runtimeStringValue(rt.StringFrom(unsafe.Pointer(unsafe.SliceData(data)), len(data)))
	joined := runtimeStringValue(rt.StringCat(stringRuntimeValue(text), stringRuntimeValue("tail")))
	for _, tc := range []struct {
		name string
		ptr  unsafe.Pointer
	}{
		{"StringFrom", unsafe.Pointer(unsafe.StringData(from))},
		{"StringCat", unsafe.Pointer(unsafe.StringData(joined))},
		{"CStrDup", unsafe.Pointer(rt.CStrDup(stringRuntimeValue(text)))},
		{"CStrDupEmpty", unsafe.Pointer(rt.CStrDup(rt.String{}))},
		{"AllocNoScanU", rt.AllocNoScanU(32)},
	} {
		base := bdwgc.Base(tc.ptr)
		if base == nil {
			t.Fatalf("%s did not return collector-managed storage", tc.name)
		}
		if got := gcGetKindAndSize(base, nil); got != wantKind {
			t.Errorf("%s collector kind = %d, want pointer-free kind %d", tc.name, got, wantKind)
		}
		runtime.KeepAlive(tc.ptr)
	}
	runtime.KeepAlive(from)
	runtime.KeepAlive(joined)
	runtime.KeepAlive(atomic)
	runtime.KeepAlive(scanned)
}

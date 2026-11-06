package gotest

import (
	"reflect"
	"testing"
	"unsafe"
)

// A trailing zero-sized field reserves an address byte in ordinary Go storage.
type trailingZeroStorage struct {
	Count int32
	End   [0]int32
}

func TestTrailingZeroStorageReflection(t *testing.T) {
	items := [4]trailingZeroStorage{{Count: 10}, {Count: 20}, {Count: 30}, {Count: 40}}
	stride := uintptr(unsafe.Pointer(&items[1])) - uintptr(unsafe.Pointer(&items[0]))
	if stride != 8 || reflect.TypeOf(items[0]).Size() != stride {
		t.Fatalf("physical stride=%d, reflected size=%d, want 8", stride, reflect.TypeOf(items[0]).Size())
	}
	v := reflect.ValueOf(&items).Elem()
	if got := v.Index(1).Field(0).Int(); got != 20 {
		t.Fatalf("reflect Index read %d, want 20", got)
	}
	v.Index(1).Field(0).SetInt(77)
	if items[1].Count != 77 || items[2].Count != 30 {
		t.Fatalf("reflect Set changed the wrong element: %v", items)
	}

	// Larger backing arrays keep a bad copy inside allocated memory while
	// making writes outside the destination slice visible through the guards.
	src := [8]trailingZeroStorage{{Count: 10}, {Count: 20}, {Count: 30}, {Count: 40}, {Count: 50}, {Count: 60}, {Count: 70}, {Count: 80}}
	dst := [8]trailingZeroStorage{{}, {}, {}, {}, {Count: 101}, {Count: 102}, {Count: 103}, {Count: 104}}
	n := reflect.Copy(reflect.ValueOf(dst[:4]), reflect.ValueOf(src[:4]))
	if n != 4 || dst[3].Count != 40 || dst[4].Count != 101 || dst[7].Count != 104 {
		t.Fatalf("reflect Copy returned %d, values/guards=%v", n, dst)
	}

	outer := struct {
		Head        trailingZeroStorage
		Next, Guard int32
	}{Head: trailingZeroStorage{Count: 10}, Next: 20, Guard: 30}
	r := reflect.ValueOf(&outer).Elem()
	if got := r.Field(1).Int(); got != 20 {
		t.Fatalf("nested reflect read=%d, want 20", got)
	}
	r.Field(1).SetInt(77)
	if outer.Next != 77 || outer.Guard != 30 {
		t.Fatalf("nested reflect write corrupted adjacent fields: %v", outer)
	}
}

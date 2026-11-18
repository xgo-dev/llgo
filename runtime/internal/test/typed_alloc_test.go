//go:build llgo

package test

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

type scalarObject struct {
	Bytes [17]byte
	Runes [3]rune
	Value float64
}

// Keep allocations observable across calls, including under LTO.
var typedAllocSink any

func TestPointerFreeZeroValues(t *testing.T) {
	for i := 0; i < 64; i++ {
		bytes := make([]byte, 17, 257)
		runes := make([]rune, 3, 129)
		floats := make([]float64, 5, 65)
		objects := make([]scalarObject, 1, 9)
		obj := new(scalarObject)
		for _, v := range []any{bytes, runes, floats, objects, obj} {
			typedAllocSink = v
		}
		for j, v := range bytes[:cap(bytes)] {
			if v != 0 {
				t.Fatalf("byte %d is not zero", j)
			}
			bytes[:cap(bytes)][j] = 0xa5
		}
		for j, v := range runes[:cap(runes)] {
			if v != 0 {
				t.Fatal("rune capacity is not zero")
			}
			runes[:cap(runes)][j] = -1
		}
		for j, v := range floats[:cap(floats)] {
			if v != 0 {
				t.Fatal("float capacity is not zero")
			}
			floats[:cap(floats)][j] = 123.5
		}
		for j, v := range objects[:cap(objects)] {
			if v != (scalarObject{}) {
				t.Fatal("struct capacity is not zero")
			}
			objects[:cap(objects)][j].Value = 123.5
		}
		if *obj != (scalarObject{}) {
			t.Fatal("new object is not zero")
		}
		obj.Value = 123.5
	}
}

func TestPointerFreeSliceGrowth(t *testing.T) {
	var bytes []byte
	var runes []rune
	var floats []float64
	for i := 0; i < 1000; i++ {
		bytes = append(bytes, byte(i))
		runes = append(runes, rune(i))
		floats = append(floats, float64(i))
	}
	bytes = append(bytes, "tail"...)
	for i := 0; i < 1000; i++ {
		if bytes[i] != byte(i) || runes[i] != rune(i) || floats[i] != float64(i) {
			t.Fatalf("append lost element %d", i)
		}
	}
	if string(bytes[1000:]) != "tail" {
		t.Fatal("append string lost data")
	}
	for _, v := range bytes[len(bytes):cap(bytes)] {
		if v != 0 {
			t.Fatal("grown byte capacity is not zero")
		}
	}
	v := reflect.ValueOf(&runes).Elem()
	v.Grow(2000)
	for _, n := range runes[len(runes):cap(runes)] {
		if n != 0 {
			t.Fatal("reflect.Grow capacity is not zero")
		}
	}
	for _, n := range []int{0, 1, 1024} {
		s := make([]struct{}, n)
		s = append(s, struct{}{})
		if len(s) != n+1 || unsafe.SliceData(s) == nil {
			t.Fatal("zero-sized slice growth")
		}
	}
}

func TestTypedPointersSurviveGC(t *testing.T) {
	type node struct {
		Text  string
		Child *scalarObject
	}
	values := make([]node, 0)
	stringsOnly := make([]string, 0)
	pointers := make([]*scalarObject, 0)
	reflected := reflect.MakeSlice(reflect.TypeOf([]*node{}), 0, 1)
	for i := 0; i < 32; i++ {
		obj := &scalarObject{Value: float64(i)}
		text := strings.Repeat(string(rune('a'+i)), 1024)
		n := &node{Text: text, Child: obj}
		values = append(values, *n)
		stringsOnly = append(stringsOnly, text)
		pointers = append(pointers, obj)
		reflected = reflect.Append(reflected, reflect.ValueOf(n))
	}
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	for i, n := range values {
		want := strings.Repeat(string(rune('a'+i)), 1024)
		if n.Text != want || stringsOnly[i] != want || n.Child.Value != float64(i) || pointers[i].Value != float64(i) {
			t.Fatalf("typed allocation %d lost data after GC", i)
		}
		if reflected.Index(i).Interface().(*node).Text != want {
			t.Fatal("reflect pointer lost after GC")
		}
	}
	runtime.KeepAlive(values)
	runtime.KeepAlive(stringsOnly)
	runtime.KeepAlive(pointers)
	runtime.KeepAlive(reflected)
}

func TestNoScanAlignedAllocations(t *testing.T) {
	for _, size := range []uintptr{0, 1, 257} {
		for _, align := range []uintptr{1, 16, 64, 8192} {
			p := rt.AllocAtomicZAligned(size, align)
			if p == nil || uintptr(p)%align != 0 {
				t.Fatalf("size %d align %d: %p", size, align, p)
			}
			for _, v := range unsafe.Slice((*byte)(p), size) {
				if v != 0 {
					t.Fatal("aligned storage is not zero")
				}
			}
			runtime.KeepAlive(p)
		}
	}
}

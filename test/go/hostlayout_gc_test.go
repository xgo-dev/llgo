//go:build llgo && !baremetal && !wasm && !nogc

package gotest

import (
	"reflect"
	"runtime"
	"structs"
	"testing"
	"unsafe"
)

//llgo:cdecl
type hostLayoutCallback func(int32) int32

type hostLayoutRecord struct {
	_      structs.HostLayout
	Native hostLayoutCallback
	Go     func(int32) int32
	Value  *int32
}

func hostLayoutIncrement(x int32) int32 { return x + 1 }

//go:noinline
func makeHostLayoutRecords() []hostLayoutRecord {
	value := int32(40)
	record := hostLayoutRecord{Native: hostLayoutIncrement, Go: func(x int32) int32 { return x + value }, Value: &value}
	var items []hostLayoutRecord
	for i := 0; i < 20; i++ {
		items = append(items, record)
	}
	return items
}

func TestHostLayoutReflectionAndGC(t *testing.T) {
	items := makeHostLayoutRecords()
	runtime.GC()
	for i := range items {
		item := &items[i]
		if item.Native(41) != 42 || item.Go(2) != 42 || *item.Value != 40 {
			t.Fatal("host record call or GC")
		}
	}
	r := &items[0]
	typ := reflect.TypeOf(*r)
	if typ.NumField() != 4 || typ.Field(3).Offset != unsafe.Offsetof(r.Value) || typ.Size() != unsafe.Sizeof(*r) {
		t.Fatal("host record reflection layout")
	}
	if unsafe.Sizeof(*r) != 4*unsafe.Sizeof(uintptr(0)) || unsafe.Offsetof(r.Value) != 3*unsafe.Sizeof(uintptr(0)) {
		t.Fatal("host record storage layout")
	}
	reflect.ValueOf(r).Elem().FieldByName("Value").Elem().SetInt(41)
	if r.Go(1) != 42 {
		t.Fatal("host record reflected pointer access")
	}
	runtime.KeepAlive(items)
}

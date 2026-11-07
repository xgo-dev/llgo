// LITTEST
// Scope: common
package main

import (
	"structs"
	"unsafe"
)

//llgo:cdecl
type Callback func(int32) int32

type Marker structs.HostLayout

type Record struct {
	_      Marker
	Native Callback
	Go     func(int32) int32
	Value  *int32
}

func increment(x int32) int32 { return x + 1 }

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	value := int32(40)
	r := &Record{Native: increment, Go: func(x int32) int32 { return x + value }, Value: &value}
	if unsafe.Sizeof(*r) != 4*unsafe.Sizeof(uintptr(0)) || unsafe.Offsetof(r.Value) != 3*unsafe.Sizeof(uintptr(0)) {
		panic("host record layout")
	}
	items := []Record{*r}
	for i := 0; i < 20; i++ {
		items = append(items, *r)
	}
	for i := range items {
		item := &items[i]
		if item.Native(41) != 42 || item.Go(2) != 42 || *item.Value != 40 {
			panic("host record call or pointer")
		}
	}
	*r.Value = 41
	if r.Go(1) != 42 {
		panic("host record pointer access")
	}
	println("host layout ok")
}

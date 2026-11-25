// LITTEST
// Scope: common
package main

import (
	"reflect"
	"unsafe"

	"github.com/xgo-dev/llgo/cmd/cl/_testrt/ptrtothislazy/dep"
)

type Local struct {
	N int
}

type PlainNew struct {
	N int
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	pointerTo := reflect.PointerTo

	// No static *Local type is named by the program, so this exercises the
	// runtime synthesis path rather than typelist canonicalization.
	local := reflect.TypeOf(Local{})
	localPtr := pointerTo(local)
	if localPtr.Kind() != reflect.Pointer || localPtr.Elem() != local {
		panic("failed to synthesize methodless pointer type")
	}

	// Keep the exact New/NewAt/Addr canonicalization shape from the historical
	// ptrtothislazynew regression in this broader owner.
	plainNew := reflect.TypeOf(PlainNew{})
	wantNew := reflect.TypeOf((*PlainNew)(nil))
	if reflect.New(plainNew).Type() != wantNew {
		panic("New returned a non-canonical local pointer type")
	}
	var localValue PlainNew
	if reflect.NewAt(plainNew, unsafe.Pointer(&localValue)).Type() != wantNew {
		panic("NewAt returned a non-canonical local pointer type")
	}
	if reflect.ValueOf(&localValue).Elem().Addr().Type() != wantNew {
		panic("Addr returned a non-canonical local pointer type")
	}

	// Plain comes from an imported package while *Plain is independently
	// referenced. The lookup path must still return the canonical static type.
	plain := reflect.TypeOf(dep.Plain{})
	plainPtr := reflect.TypeOf((*dep.Plain)(nil))
	if pointerTo(plain) != plainPtr {
		panic("PointerTo returned a non-canonical pointer type")
	}
	if reflect.New(plain).Type() != plainPtr {
		panic("New returned a non-canonical pointer type")
	}
	var value dep.Plain
	if reflect.NewAt(plain, unsafe.Pointer(&value)).Type() != plainPtr {
		panic("NewAt returned a non-canonical pointer type")
	}
	if reflect.ValueOf(&value).Elem().Addr().Type() != plainPtr {
		panic("Addr returned a non-canonical pointer type")
	}

	// A pointer descriptor with methods must remain static: synthesized pointer
	// descriptors do not contain uncommon method metadata.
	withMethod := reflect.TypeOf(dep.WithMethod{})
	withMethodPtr := reflect.TypeOf((*dep.WithMethod)(nil))
	if pointerTo(withMethod) != withMethodPtr {
		panic("method-bearing pointer type is not canonical")
	}
	if _, ok := withMethodPtr.MethodByName("Value"); !ok {
		panic("promoted value-receiver method is missing")
	}
	if _, ok := withMethodPtr.MethodByName("Pointer"); !ok {
		panic("pointer-receiver method is missing")
	}
	result := reflect.New(withMethod).MethodByName("Pointer").Call(nil)
	if len(result) != 1 || result[0].Int() != 42 {
		panic("pointer-receiver method call failed")
	}

	println("ok")
}

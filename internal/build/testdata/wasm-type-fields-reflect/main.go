package main

import "reflect"

// Keep this descriptor live as a map key, without a conversion to interface.
var values = map[struct{ A, B uint64 }]int{{1, 2}: 42}

type callback func(int) int

func (f callback) Apply(x int) int { return f(x) }

func checkCallback() {
	base := 40
	f := callback(func(x int) int { return base + x })
	value := reflect.ValueOf(f)
	typ := value.Type()
	method, ok := typ.MethodByName("Apply")
	if !ok || method.Type.In(0) != typ || reflect.PointerTo(typ).Elem() != typ {
		panic("named function method metadata")
	}
	if got := method.Func.Call([]reflect.Value{value, reflect.ValueOf(2)}); got[0].Int() != 42 {
		panic("named function method expression")
	}
}

func main() {
	word := reflect.TypeOf(uint64(0))
	typ := reflect.StructOf([]reflect.StructField{{Name: "A", Type: word}, {Name: "B", Type: word}})
	value := reflect.New(typ).Elem()
	value.Field(0).SetUint(1)
	value.Field(1).SetUint(2)
	key, ok := value.Interface().(struct{ A, B uint64 })
	if !ok || values[key] != 42 {
		panic("dynamic struct identity")
	}
	if typ.NumField() != 2 || typ.Field(1).Name != "B" || typ.Field(1).Offset != 8 {
		panic("dynamic struct fields")
	}
	// StructOf must reuse the static type so assertions retain pointer identity.
	if reflect.TypeOf(value.Interface()) != typ {
		panic("dynamic type identity")
	}
	checkCallback()
	println("wasm type fields reflection passed")
}

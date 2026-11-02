package main

import "reflect"

type Number float64
type Record struct{ Value float64 }
type Alias = float64

var global any = Number(1.25)

func main() {
	values := []any{
		reflect.New(reflect.TypeFor[float64]()).Elem().Interface(),
		reflect.ValueOf(Record{1.25}).Field(0).Interface(),
		reflect.ValueOf(global).Convert(reflect.TypeFor[float64]()).Interface(),
	}
	for _, value := range values {
		if _, ok := value.(Alias); !ok {
			panic("reflected basic type lost")
		}
		if _, ok := value.(complex128); ok {
			panic("impossible assertion succeeded")
		}
	}
	if global.(Number) != 1.25 {
		panic("named global interface lost")
	}
	// A dynamically constructed unnamed type need not appear in MakeInterface.
	empty := reflect.New(reflect.StructOf(nil)).Elem().Interface()
	if _, ok := empty.(struct{}); !ok {
		panic("dynamic composite assertion lost")
	}
	ptr := reflect.New(reflect.TypeFor[Record]()).Interface()
	if _, ok := ptr.(*Record); !ok {
		panic("derived pointer assertion lost")
	}
	checkFailingAssertion(values[0])
	println("wasm interface domain passed")
}

func checkFailingAssertion(value any) {
	defer func() {
		if recover() == nil {
			panic("failed assertion did not panic")
		}
	}()
	_ = value.(complex128)
}

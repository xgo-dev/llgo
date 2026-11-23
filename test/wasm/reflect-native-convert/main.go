package main

import (
	"reflect"

	"github.com/xgo-dev/llgo/test/wasm/reflect-native-convert/callback"
)

func addSeven(x int32) int32 { return x + 7 }

func main() {
	// Only Convert can request a typed adapter in this standalone program.
	v := reflect.ValueOf(callback.Native(addSeven)).Convert(reflect.TypeFor[func(int32) int32]())
	f := v.Interface().(func(int32) int32)
	if got := f(5); got != 12 {
		panic(got)
	}
	println("wasm named native conversion passed")
}

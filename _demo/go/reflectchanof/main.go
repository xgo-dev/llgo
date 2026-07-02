package main

import (
	"reflect"
)

type T struct{}

func main() {
	ch := reflect.ChanOf(reflect.BothDir, reflect.TypeOf(T{}))
	ptr := reflect.PointerTo(ch)
	if ptr.Elem() != ch {
		panic("error " + ptr.String())
	}
}

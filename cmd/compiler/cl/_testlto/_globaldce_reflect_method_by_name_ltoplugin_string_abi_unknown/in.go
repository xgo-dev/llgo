package main

import (
	"os"
	"reflect"
)

type Unknown struct{}

//go:noinline
func (Unknown) UnknownA() {}

//go:noinline
func (Unknown) UnknownB() {}

func unknownName() string {
	if len(os.Args) == 0 {
		return "UnknownA"
	}
	return os.Args[0]
}

func main() {
	_, _ = reflect.TypeOf(Unknown{}).MethodByName(unknownName())
}

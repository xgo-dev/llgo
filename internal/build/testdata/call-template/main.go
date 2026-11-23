package main

import "github.com/xgo-dev/llgo/internal/build/testdata/call-template/codec"

var dynamic any = "value"

//go:noinline
func throughInterface(v any) string {
	return codec.Stringify(v, 2)
}

func main() {
	println(codec.Stringify(1, 2))
	println(codec.Stringify(1, dynamic))
	println(throughInterface("generic"))
	println(codec.Equal([]string{"same"}, []string{"same"}))
	println(codec.Check(true, 7), codec.Check(false, 9))
}

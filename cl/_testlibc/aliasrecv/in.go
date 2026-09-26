package main

import "github.com/xgo-dev/llgo/cl/_testlibc/aliasrecv/lib"

func main() {
	text := &lib.Text{Bytes: [6]byte{'h', 'e', 'l', 'l', 'o'}}
	var str lib.String = text
	println(str.Compare(&text.Bytes[0]), str.CompareChain(&text.Bytes[0]), lib.Int(-42).Abs())
	compare := str.Compare
	println(compare(&text.Bytes[0]), lib.String.Compare(str, &text.Bytes[0]))
}

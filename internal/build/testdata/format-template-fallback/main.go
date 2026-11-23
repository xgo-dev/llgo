package main

import (
	"bytes"
	"fmt"
)

type named int

func (named) String() string { return "named" }

var format = "%s=%d"

func main() {
	var b bytes.Buffer
	n, err := fmt.Fprintf(&b, "n=%02d", 42)
	if err != nil || n != 4 || b.String() != "n=42" {
		panic("custom writer changed")
	}
	println(b.String())
	println(fmt.Sprintf("%03v", named(42)))
	println(fmt.Sprintf(format, "dynamic", 42))
	println(fmt.Sprintf("%02d", 7))
}

package main

import (
	"fmt"

	"github.com/goplus/llgo/internal/build/testdata/unexportedmethodidentity/b"
)

func main() {
	fmt.Println(b.F1(b.T{}))
	fmt.Println(b.F2(b.T{}))
}

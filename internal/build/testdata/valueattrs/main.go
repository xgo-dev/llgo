package main

import "github.com/xgo-dev/llgo/internal/build/testdata/valueattrs/dep"

func main() {
	x := 42
	if dep.Checked(&x) != &x {
		panic("identity")
	}
	defer func() {
		if recover() != "nil" {
			panic("missing panic")
		}
	}()
	dep.Checked(nil)
}

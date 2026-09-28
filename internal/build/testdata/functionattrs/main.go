package main

import "github.com/xgo-dev/llgo/internal/build/testdata/functionattrs/dep"

func recoverCall(f func()) {
	defer func() {
		if recover() != "stop" {
			panic("lost panic or returned normally")
		}
	}()
	f()
	panic("returned")
}

func main() {
	recoverCall(dep.Stop)
	recoverCall(func() { dep.Generic(1) })
	recoverCall(dep.T{}.Stop)
	if dep.Deferred != 3 {
		panic("lost deferred calls")
	}
}

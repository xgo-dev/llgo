// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdrop/reflect_callback_foreign/names"
	"github.com/xgo-dev/llgo/cl/_testdrop/reflect_callback_foreign/writer"
)

// SYMBOL-DAG: main{{.*}}Model{{.*}}Foreign

type Model struct{}

//go:noinline
func (Model) Original() string { panic("stale callback initializer") }

//go:noinline
func (Model) Foreign() string { return "foreign" }

func main() {
	writer.Replace()
	println(names.Invoke(Model{}))
}

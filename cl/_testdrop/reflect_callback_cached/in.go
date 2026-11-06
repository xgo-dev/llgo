// LITTEST
package main

import "github.com/xgo-dev/llgo/cl/_testdrop/reflect_callback_cached/hooks"

// SYMBOL-DAG: main{{.*}}Model{{.*}}BeforeCreate
// SYMBOL-DAG: main{{.*}}Model{{.*}}AfterFind
// SYMBOL-NOT: main{{.*}}Model{{.*}}Unused

type Model struct{}

//go:noinline
func (Model) BeforeCreate() string { return "before-create" }

//go:noinline
func (Model) AfterFind() string { return "after-find" }

//go:noinline
func (Model) Unused() string { panic("unreachable method") }

func main() { hooks.Invoke(Model{}) }

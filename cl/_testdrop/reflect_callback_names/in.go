// LITTEST
package main

import "reflect"

// SYMBOL-NOT: main{{.*}}Model{{.*}}Unused
// SYMBOL-DAG: main{{.*}}Model{{.*}}BeforeCreate
// SYMBOL-DAG: main{{.*}}Model{{.*}}AfterFind
// SYMBOL-NOT: main{{.*}}Model{{.*}}Unused

type callbackType string

var callbackTypes = []callbackType{
	"BeforeCreate", "BeforeUpdate", "AfterCreate", "AfterUpdate",
	"BeforeSave", "AfterSave", "BeforeDelete", "AfterDelete", "AfterFind",
}

type Model struct{}

//go:noinline
func (Model) BeforeCreate() string { return "before-create" }

//go:noinline
func (Model) AfterFind() string { return "after-find" }

//go:noinline
func (Model) Unused() string { panic("unreachable method") }

func main() {
	v := reflect.ValueOf(Model{})
	for _, name := range callbackTypes {
		m := v.MethodByName(string(name))
		if m.IsValid() {
			if m.Type().NumIn() != 0 || m.Type().NumOut() != 1 {
				panic("unexpected hook signature")
			}
			println(m.Call(nil)[0].String())
		}
	}
}

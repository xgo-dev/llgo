package main

import (
	"encoding/json"
	"reflect"
)

type callbackType string

// Match Gorm's schema callback discovery: the loop variable is not a literal
// MethodByName argument, but every possible name comes from this readonly list.
var callbackTypes = []callbackType{
	"BeforeCreate", "BeforeUpdate", "AfterCreate", "AfterUpdate",
	"BeforeSave", "AfterSave", "BeforeDelete", "AfterDelete", "AfterFind",
}

type model struct{}

//go:noinline
func (model) BeforeCreate() string { return "before-create" }

//go:noinline
func (model) AfterFind() string { return "after-find" }

//go:noinline
func (model) UnrelatedJSON() string {
	b, err := json.Marshal(map[string]int{"unreachable": 42})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func main() {
	v := reflect.ValueOf(model{})
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

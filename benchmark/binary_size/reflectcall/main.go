package main

import (
	"encoding/json"
	"reflect"
)

type payload struct{}

//go:noinline
func (payload) Decode() string {
	b, err := json.Marshal(map[string]int{"answer": 42})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func main() {
	t := reflect.TypeOf(payload{})
	m := t.Method(0)
	result := m.Func.Call([]reflect.Value{reflect.ValueOf(payload{})})
	println(result[0].String())
}

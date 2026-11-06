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
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		println(m.Name, m.Type.NumIn(), m.Type.NumOut())
	}
}

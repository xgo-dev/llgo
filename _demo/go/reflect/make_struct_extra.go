package main

import (
	"reflect"
)

func add(a int, b int) int {
	return a + b
}

func sub(a int, b int) int {
	return a - b
}

func testReflectStructOf() {
	typ := reflect.StructOf([]reflect.StructField{
		{
			Name:    "Add",
			PkgPath: "",
			Type:    reflect.TypeOf(add),
		},
		{
			Name:    "Sub",
			PkgPath: "",
			Type:    reflect.TypeOf(sub),
		},
	})
	st := reflect.New(typ).Elem()
	st.Field(0).Set(reflect.ValueOf(add))
	st.Field(1).Set(reflect.ValueOf(sub))
	r := st.Field(0).Call([]reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)})
	if len(r) != 1 || r[0].Interface() != 3 {
		panic("st.Add(1,2) error")
	}
	r = st.Field(1).Call([]reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)})
	if len(r) != 1 || r[0].Interface() != -1 {
		panic("st.Sub(1,2) error")
	}
	testPromotedFieldSearch()
}

func testPromotedFieldSearch() {
	type shared struct{ Token int }
	type left struct{ shared }
	type right struct{ shared }
	ambiguous := reflect.TypeOf(struct {
		left
		right
	}{})
	var seen string
	if _, ok := ambiguous.FieldByNameFunc(func(name string) bool {
		seen += "/" + name
		return name == "Token"
	}); ok || seen != "/left/right/shared/shared/Token" {
		panic("promoted field ambiguity or callback order")
	}

	type a struct{}
	type b struct{}
	type c struct{}
	type d struct{}
	type e struct{}
	type f struct{}
	type cycle struct {
		*cycle
		Value int
	}
	wide := reflect.ValueOf(struct {
		a
		b
		c
		d
		e
		f
		*cycle
	}{cycle: &cycle{Value: 42}})
	if got := wide.FieldByName("Value"); !got.IsValid() || got.Int() != 42 {
		panic("promoted field after queue growth")
	}
	if _, found := wide.Type().FieldByName("Missing"); found {
		panic("promoted field cycle")
	}
}

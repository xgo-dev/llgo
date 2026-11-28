// LITTEST
// Scope: common
package main

import "reflect"

// This is the LTO-plugin owner for finite names loaded from global aggregate
// storage, including a named-string slice.
// SYMBOL-NOT: main{{.*}}S{{.*}}Drop
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepValue
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepValueAlt
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepType
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepTypeAlt
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepGlobalSliceA
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepGlobalSliceB
// SYMBOL-NOT: main{{.*}}S{{.*}}Drop

type S struct{}

//go:noinline
func (S) KeepValue() string {
	return "keep-value"
}

//go:noinline
func (S) KeepValueAlt() string {
	return "keep-value-alt"
}

//go:noinline
func (S) KeepType() string {
	return "keep-type"
}

//go:noinline
func (S) KeepTypeAlt() string {
	return "keep-type-alt"
}

//go:noinline
func (S) KeepGlobalSliceA() string { return "global-slice-a" }

//go:noinline
func (S) KeepGlobalSliceB() string { return "global-slice-b" }

//go:noinline
func (S) Drop() string {
	panic("Drop should be unreachable")
}

type methodNames struct {
	value  [2]string
	nested nestedMethodNames
}

type nestedMethodNames struct {
	typ [2]string
}

var names = methodNames{
	value: [2]string{"KeepValue", "KeepValueAlt"},
	nested: nestedMethodNames{
		typ: [2]string{"KeepType", "KeepTypeAlt"},
	},
}

type callbackType string

var callbackTypes = []callbackType{"KeepGlobalSliceA", "KeepGlobalSliceB"}

func main() {
	v := reflect.ValueOf(S{})
	out := v.MethodByName(names.value[0]).Call(nil)
	println(out[0].String())

	out = v.MethodByName(names.value[1]).Call(nil)
	println(out[0].String())

	t := reflect.TypeOf(S{})
	m, ok := t.MethodByName(names.nested.typ[0])
	if !ok {
		panic("missing method")
	}
	out = m.Func.Call([]reflect.Value{v})
	println(out[0].String())

	m, ok = t.MethodByName(names.nested.typ[1])
	if !ok {
		panic("missing method")
	}
	out = m.Func.Call([]reflect.Value{v})
	println(out[0].String())

	for _, name := range callbackTypes {
		out = v.MethodByName(string(name)).Call(nil)
		println(out[0].String())
	}
}

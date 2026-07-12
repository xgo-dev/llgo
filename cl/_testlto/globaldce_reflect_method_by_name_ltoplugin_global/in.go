// LITTEST
package main

import "reflect"

// SYMBOL-NOT: globaldce_reflect_method_by_name_ltoplugin_global{{.*}}S{{.*}}Drop
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_global{{.*}}S{{.*}}KeepValue
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_global{{.*}}S{{.*}}KeepValueAlt
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_global{{.*}}S{{.*}}KeepType
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_global{{.*}}S{{.*}}KeepTypeAlt
// SYMBOL-NOT: globaldce_reflect_method_by_name_ltoplugin_global{{.*}}S{{.*}}Drop

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
}

// LITTEST
package main

import (
	"os"
	"reflect"
)

// SYMBOL-NOT: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}Drop
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepValue
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepValueAlt
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepType
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepTypeAlt
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepMultiValue
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepMultiAltValue
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepMultiType
// SYMBOL-DAG: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}KeepMultiAltType
// SYMBOL-NOT: globaldce_reflect_method_by_name_ltoplugin_concat{{.*}}S{{.*}}Drop

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
func (S) KeepMultiValue() string {
	return "keep-multi-value"
}

//go:noinline
func (S) KeepMultiAltValue() string {
	return "keep-multi-alt-value"
}

//go:noinline
func (S) KeepMultiType() string {
	return "keep-multi-type"
}

//go:noinline
func (S) KeepMultiAltType() string {
	return "keep-multi-alt-type"
}

//go:noinline
func (S) Drop() string {
	panic("Drop should be unreachable")
}

//go:noinline
func selector() int {
	if len(os.Args) == 0 {
		return 1
	}
	return 0
}

func valueSuffix() string {
	if selector() == 0 {
		return "Value"
	}
	return "ValueAlt"
}

func typeSuffix() string {
	if selector() == 0 {
		return "Type"
	}
	return "TypeAlt"
}

func valueName() string {
	return "Keep" + valueSuffix()
}

func typeName() string {
	return "Keep" + typeSuffix()
}

var multiPrefixParts = [2]string{"Ke", "ep"}
var multiChoices = [2]string{"Multi", "MultiAlt"}
var multiValueSuffixParts = [2]string{"Va", "lue"}
var multiTypeSuffixParts = [2]string{"Ty", "pe"}

func multiPrefix() string {
	return multiPrefixParts[0] + multiPrefixParts[1]
}

func multiMiddle() string {
	if selector() == 0 {
		return multiChoices[0]
	}
	return multiChoices[1]
}

func multiValueSuffix() string {
	return multiValueSuffixParts[0] + multiValueSuffixParts[1]
}

func multiTypeSuffix() string {
	return multiTypeSuffixParts[0] + multiTypeSuffixParts[1]
}

func multiValueName() string {
	return multiPrefix() + multiMiddle() + multiValueSuffix()
}

func multiTypeName() string {
	return multiPrefix() + multiMiddle() + multiTypeSuffix()
}

func main() {
	out := reflect.ValueOf(S{}).MethodByName(valueName()).Call(nil)
	println(out[0].String())

	m, ok := reflect.TypeOf(S{}).MethodByName(typeName())
	if !ok {
		panic("missing method")
	}
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	out = reflect.ValueOf(S{}).MethodByName(multiValueName()).Call(nil)
	println(out[0].String())

	m, ok = reflect.TypeOf(S{}).MethodByName(multiTypeName())
	if !ok {
		panic("missing multi method")
	}
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())
}

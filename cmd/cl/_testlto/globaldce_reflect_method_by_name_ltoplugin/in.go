// LITTEST
// Scope: common
package main

import (
	"os"
	"reflect"
)

// This is the LTO-plugin owner for finite names that cross either one
// noinline helper or a chain of helper parameters.
// SYMBOL-NOT: main{{.*}}S{{.*}}Drop
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepA
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepB
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepParamValueA
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepParamValueB
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepParamTypeA
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepParamTypeB
// SYMBOL-NOT: main{{.*}}S{{.*}}Drop

type S struct{}

//go:noinline
func (S) KeepA() string {
	return "keep-a"
}

//go:noinline
func (S) KeepB() string {
	return "keep-b"
}

//go:noinline
func (S) KeepParamValueA() string {
	return "param-value-a"
}

//go:noinline
func (S) KeepParamValueB() string {
	return "param-value-b"
}

//go:noinline
func (S) KeepParamTypeA() string {
	return "param-type-a"
}

//go:noinline
func (S) KeepParamTypeB() string {
	return "param-type-b"
}

//go:noinline
func (S) Drop() string {
	panic("Drop should be unreachable")
}

// Keep this helper call visible so the LTO plugin, rather than the pre-link
// inliner, must recover its finite return set.
//
//go:noinline
func methodName() string {
	name := "KeepA"
	if os.Args[0] == "" {
		name = "KeepB"
	}
	return name
}

var typeNames = [2]string{"KeepParamTypeA", "KeepParamTypeB"}

//go:noinline
func callValueByName(name string) string {
	out := reflect.ValueOf(S{}).MethodByName(name).Call(nil)
	return out[0].String()
}

//go:noinline
func forwardValueSuffix(suffix string) string {
	return callValueByName("Keep" + suffix)
}

//go:noinline
func callTypeByName(name string) string {
	m, ok := reflect.TypeOf(S{}).MethodByName(name)
	if !ok {
		panic("missing method")
	}
	out := m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	return out[0].String()
}

//go:noinline
func forwardTypeName(name string) string {
	return callTypeByName(name)
}

func main() {
	out := reflect.ValueOf(S{}).MethodByName(methodName()).Call(nil)
	println(out[0].String())

	m, ok := reflect.TypeOf(S{}).MethodByName(methodName())
	if !ok {
		panic("missing method")
	}
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	println(forwardValueSuffix("__ParamValueA__"[2:13]))
	println(forwardValueSuffix("__ParamValueB__"[2:13]))
	for _, name := range typeNames {
		println(forwardTypeName(name))
	}
}

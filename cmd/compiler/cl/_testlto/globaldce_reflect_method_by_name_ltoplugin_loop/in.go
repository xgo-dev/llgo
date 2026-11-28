// LITTEST
// Scope: common
package main

import (
	"os"
	"reflect"
)

// This is the LTO-plugin owner for finite-name propagation through loop,
// range-literal, and switch control flow.
// SYMBOL-NOT: main{{.*}}S{{.*}}Drop
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepLoopA
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepLoopB
// SYMBOL-DAG: main{{.*}}S{{.*}}Query
// SYMBOL-DAG: main{{.*}}S{{.*}}Mutation
// SYMBOL-DAG: main{{.*}}S{{.*}}Subscription
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepSwitchA
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepSwitchB
// SYMBOL-DAG: main{{.*}}S{{.*}}KeepSwitchC
// SYMBOL-NOT: main{{.*}}S{{.*}}Drop

type S struct{}

//go:noinline
func (S) KeepLoopA() string {
	return "loop-a"
}

//go:noinline
func (S) KeepLoopB() string {
	return "loop-b"
}

//go:noinline
func (S) Query() string { return "query" }

//go:noinline
func (S) Mutation() string { return "mutation" }

//go:noinline
func (S) Subscription() string { return "subscription" }

//go:noinline
func (S) KeepSwitchA() string { return "switch-a" }

//go:noinline
func (S) KeepSwitchB() string { return "switch-b" }

//go:noinline
func (S) KeepSwitchC() string { return "switch-c" }

//go:noinline
func (S) Drop() string {
	panic("Drop should be unreachable")
}

var loopNames = [2]string{"KeepLoopA", "KeepLoopB"}

const (
	queryName        = "Query"
	mutationName     = "Mutation"
	subscriptionName = "Subscription"
)

//go:noinline
func selector() int {
	return len(os.Args)
}

func switchMethodName() string {
	switch selector() {
	case 1:
		return "KeepSwitchA"
	case 2:
		return "KeepSwitchB"
	default:
		return "KeepSwitchC"
	}
}

func main() {
	v := reflect.ValueOf(S{})
	t := reflect.TypeOf(S{})
	for _, name := range loopNames {
		out := v.MethodByName(name).Call(nil)
		println(out[0].String())

		m, ok := t.MethodByName(name)
		if !ok {
			panic("missing method")
		}
		out = m.Func.Call([]reflect.Value{v})
		println(out[0].String())
	}

	for _, name := range [...]string{queryName, mutationName, subscriptionName} {
		m := v.MethodByName(name)
		if !m.IsValid() {
			panic("missing range-literal method")
		}
		out := m.Call(nil)
		println(out[0].String())
	}

	name := switchMethodName()
	out := v.MethodByName(name).Call(nil)
	println(out[0].String())

	m, ok := t.MethodByName(name)
	if !ok {
		panic("missing switch method")
	}
	out = m.Func.Call([]reflect.Value{v})
	println(out[0].String())
}

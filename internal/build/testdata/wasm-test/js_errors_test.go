//go:build js && wasm

package wasmtest

import (
	"syscall/js"
	"testing"
)

func TestJSCallPreservesThrownValue(t *testing.T) {
	constructor := js.Global().Get("Function")
	jsErr := js.Global().Get("Error").New("host failure")
	jsErr.Set("code", "ENOENT")
	// Only the host's actual exit signal is runtime control flow. An ordinary
	// user object with similarly named properties must still be recoverable.
	lookalikeExit := js.ValueOf(map[string]any{"name": "ExitStatus", "status": 7})
	for _, value := range []struct {
		name  string
		value js.Value
	}{
		{"error", jsErr},
		{"exit-lookalike", lookalikeExit},
		{"null", js.Null()},
		{"undefined", js.Undefined()},
		{"number", js.ValueOf(42)},
		{"string", js.ValueOf("host failure")},
	} {
		fn := constructor.New("value", "return function() { throw value; }").Invoke(value.value)
		object := js.Global().Get("Object").New()
		object.Set("fail", fn)
		for _, call := range []struct {
			name   string
			invoke func()
		}{
			{"Call", func() { object.Call("fail") }},
			{"Invoke", func() { fn.Invoke() }},
			{"New", func() { fn.New() }},
		} {
			t.Run(value.name+"/"+call.name, func(t *testing.T) {
				defer func() {
					got, ok := recover().(js.Error)
					if !ok {
						t.Fatal("JavaScript throw did not become a syscall/js.Error panic")
					}
					if !got.Value.Equal(value.value) {
						t.Fatal("syscall/js.Error lost the original thrown value")
					}
				}()
				call.invoke()
			})
		}
	}
}

func TestJSCallReceiverAndArguments(t *testing.T) {
	constructor := js.Global().Get("Function")
	fn := constructor.New("a", "b", "'use strict'; return [this, a, b];")
	object := js.Global().Get("Object").New()
	object.Set("method", fn)
	argument := js.Global().Get("Object").New()
	check := func(result, receiver js.Value) {
		t.Helper()
		if result.Length() != 3 || !result.Index(0).Equal(receiver) ||
			!result.Index(1).Equal(argument) || result.Index(2).Int() != 42 {
			t.Fatal("JavaScript call changed its receiver or arguments")
		}
	}
	check(object.Call("method", argument, 42), object)
	check(fn.Invoke(argument, 42), js.Undefined())
	result := fn.New(argument, 42)
	if !result.Index(0).InstanceOf(fn) {
		t.Fatal("New did not construct a fresh receiver")
	}
	check(result, result.Index(0))
}

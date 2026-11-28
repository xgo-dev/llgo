//go:build js && wasm

package wasmtest

import (
	"math"
	"syscall/js"
	"testing"
)

// Keep these assertions provider-neutral and run them with LLGo's named
// Emscripten provider, raw GoJS provider, and the official Go toolchain.
func TestJSHostValueOperations(t *testing.T) {
	for _, text := range []string{"", "hello", "a\x00b", "你好 🌍"} {
		value := js.ValueOf(text)
		if value.Type() != js.TypeString || value.String() != text || !value.Equal(js.ValueOf(text)) {
			t.Fatalf("string round trip %q: %v", text, value)
		}
	}
	for _, number := range []float64{0, math.Copysign(0, -1), 42.5, -5, math.Inf(1), math.Inf(-1)} {
		value := js.ValueOf(number)
		if value.Float() != number || !value.Equal(js.ValueOf(number)) {
			t.Fatalf("number round trip %v: %v", number, value)
		}
	}
	nan := js.ValueOf(math.NaN())
	if !nan.IsNaN() || nan.Equal(nan) || nan.Truthy() {
		t.Fatal("NaN semantics differ from Go")
	}

	object := js.Global().Get("Object").New()
	object.Set("self", object)
	if !object.Equal(object.Get("self")) || object.Equal(js.Global().Get("Object").New()) {
		t.Fatal("object identity was not preserved")
	}
	object.Set("a\x00b", 17)
	property := object.Get("a\x00b")
	if property.Type() != js.TypeNumber || property.Int() != 17 {
		t.Fatalf("property name was truncated: %v", property)
	}
	object.Delete("a\x00b")
	if !object.Get("a\x00b").IsUndefined() {
		t.Fatal("property deletion failed")
	}

	array := js.Global().Get("Array").New(2)
	array.SetIndex(0, "first")
	array.SetIndex(1, 42)
	first, second, length := array.Index(0), array.Index(1), array.Length()
	if length != 2 || first.Type() != js.TypeString || first.String() != "first" || second.Type() != js.TypeNumber || second.Int() != 42 {
		t.Fatalf("array operations returned %v, %v, length %d", first, second, length)
	}
	for _, length := range []any{js.Undefined(), math.NaN(), math.Inf(1), "not-a-number"} {
		object.Set("length", length)
		if got := object.Length(); got != 0 {
			t.Fatalf("non-numeric length %v = %d", length, got)
		}
	}
	object.Set("length", -3)
	if got := object.Length(); got != -3 {
		t.Fatalf("signed length = %d", got)
	}
	if !array.InstanceOf(js.Global().Get("Array")) {
		t.Fatal("Array instanceof Array failed")
	}
}

func TestJSHostByteCopies(t *testing.T) {
	for _, name := range []string{"Uint8Array", "Uint8ClampedArray"} {
		array := js.Global().Get(name).New(3)
		if n := js.CopyBytesToJS(array, []byte{1, 2, 255, 4}); n != 3 {
			t.Fatalf("%s copied %d bytes", name, n)
		}
		dst := make([]byte, 5)
		if n := js.CopyBytesToGo(dst, array); n != 3 || dst[2] != 255 || dst[3] != 0 {
			t.Fatalf("%s result: %d %v", name, n, dst)
		}
	}
	for _, name := range []string{"Array", "Uint16Array", "DataView"} {
		var array js.Value
		if name == "DataView" {
			array = js.Global().Get(name).New(js.Global().Get("ArrayBuffer").New(4))
		} else {
			array = js.Global().Get(name).New(4)
		}
		for _, toGo := range []bool{true, false} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("accepted %s for byte copy", name)
					}
				}()
				if toGo {
					js.CopyBytesToGo(make([]byte, 2), array)
				} else {
					js.CopyBytesToJS(array, []byte{1})
				}
			}()
		}
	}
}

// Based on Go's syscall/js TestInterleavedFunctions. It exercises an external
// callback that blocks while another synchronous callback enters Go.
func TestJSHostInterleavedFunctions(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		entered <- struct{}{}
		<-release
		return nil
	})
	defer callback.Release()
	js.Global().Get("setTimeout").Invoke(callback, 0)
	<-entered
	close(release)

	synchronous := js.FuncOf(func(js.Value, []js.Value) any { return 42 })
	defer synchronous.Release()
	if got := synchronous.Invoke().Int(); got != 42 {
		t.Fatalf("interleaved callback returned %d", got)
	}
}

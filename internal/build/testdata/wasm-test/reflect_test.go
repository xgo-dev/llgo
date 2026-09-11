package wasmtest

import (
	"reflect"
	"runtime"
	"testing"
)

type reflectRecord struct {
	Label string
	Value int64
	Next  *int
}

func (r *reflectRecord) Add(value int64) int64 {
	if r == nil {
		return -1
	}
	runtime.GC()
	return r.Value + value
}

func TestReflectCallAndMethod(t *testing.T) {
	next := 42
	captured := &reflectRecord{Label: "kept", Value: 17, Next: &next}
	callable := func(value reflectRecord, transform func(int) int) (reflectRecord, func() int) {
		runtime.GC()
		runtime.Gosched()
		value.Value += int64(transform(*value.Next))
		return value, func() int { return *captured.Next }
	}
	results := reflect.ValueOf(callable).Call([]reflect.Value{
		reflect.ValueOf(*captured),
		reflect.ValueOf(func(value int) int { return value + 1 }),
	})
	if got := results[0].Interface().(reflectRecord); got.Label != "kept" || got.Value != 60 || *got.Next != 42 {
		t.Fatalf("aggregate result = %+v", got)
	}
	if got := results[1].Interface().(func() int)(); got != 42 {
		t.Fatalf("closure result = %d, want 42", got)
	}

	method, ok := reflect.TypeOf(captured).MethodByName("Add")
	if !ok {
		t.Fatal("missing reflected method")
	}
	got := method.Func.Call([]reflect.Value{reflect.ValueOf(captured), reflect.ValueOf(int64(5))})
	if len(got) != 1 || got[0].Int() != 22 {
		t.Fatalf("method expression result = %v, want 22", got)
	}
	methodValue := reflect.ValueOf(captured).MethodByName("Add")
	if got := methodValue.Call([]reflect.Value{reflect.ValueOf(int64(6))}); len(got) != 1 || got[0].Int() != 23 {
		t.Fatalf("method value result = %v, want 23", got)
	}
	if got := methodValue.Interface().(func(int64) int64)(7); got != 24 {
		t.Fatalf("typed method value result = %d, want 24", got)
	}
}

func TestReflectMakeFunc(t *testing.T) {
	type signature func(reflectRecord, func(int) int) (reflectRecord, func() int)
	var escaped []reflect.Value
	value := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		escaped = in
		runtime.GC()
		runtime.Gosched()
		record := in[0].Interface().(reflectRecord)
		record.Value += int64(in[1].Interface().(func(int) int)(*record.Next))
		return []reflect.Value{reflect.ValueOf(record), reflect.ValueOf(func() int { return *record.Next })}
	})
	callable := value.Interface().(signature)
	next := 42
	input := reflectRecord{Label: "copy", Value: 7, Next: &next}
	got, result := callable(input, func(value int) int { return value + 1 })
	if got.Label != "copy" || got.Value != 50 || result() != 42 {
		t.Fatalf("MakeFunc results = (%+v, %d)", got, result())
	}
	input.Label = "changed"
	runtime.GC()
	if got := escaped[0].Interface().(reflectRecord); got.Label != "copy" || *got.Next != 42 {
		t.Fatalf("escaped argument = %+v", got)
	}

	reflected := value.Call([]reflect.Value{
		reflect.ValueOf(input), reflect.ValueOf(func(value int) int { return value }),
	})
	if got := reflected[0].Interface().(reflectRecord); got.Value != 49 {
		t.Fatalf("reflected MakeFunc result = %+v", got)
	}
}

func TestReflectDynamicMakeFunc(t *testing.T) {
	record := reflect.StructOf([]reflect.StructField{{Name: "Dynamic", Type: reflect.TypeOf(0)}})
	typ := reflect.FuncOf([]reflect.Type{record}, []reflect.Type{record}, false)
	callable := reflect.MakeFunc(typ, func(in []reflect.Value) []reflect.Value {
		runtime.GC()
		return in
	})
	argument := reflect.New(record).Elem()
	argument.Field(0).SetInt(91)
	got := callable.Call([]reflect.Value{argument})
	if len(got) != 1 || got[0].Field(0).Int() != 91 {
		t.Fatalf("dynamic MakeFunc result = %v", got)
	}
}

func TestReflectFuncOfStaticConversion(t *testing.T) {
	typ := reflect.FuncOf([]reflect.Type{reflect.TypeOf(0)}, []reflect.Type{reflect.TypeOf(0)}, false)
	value := reflect.MakeFunc(typ, func(in []reflect.Value) []reflect.Value {
		// Exercise an Asyncify unwind while libffi's closure trampoline owns
		// temporary argument and result buffers.
		runtime.Gosched()
		return []reflect.Value{reflect.ValueOf(int(in[0].Int()) + 1)}
	})
	callable := value.Interface().(func(int) int)
	if got := callable(41); got != 42 {
		t.Fatalf("FuncOf MakeFunc result = %d, want 42", got)
	}
}

func TestReflectVariadicMakeFunc(t *testing.T) {
	type signature func(int, ...int) int
	value := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		total := in[0].Int()
		for i := 0; i < in[1].Len(); i++ {
			total += in[1].Index(i).Int()
		}
		runtime.GC()
		return []reflect.Value{reflect.ValueOf(int(total))}
	})
	callable := value.Interface().(signature)
	if got := callable(3, 5, 7); got != 15 {
		t.Fatalf("typed variadic result = %d, want 15", got)
	}
	got := value.Call([]reflect.Value{reflect.ValueOf(3), reflect.ValueOf(5), reflect.ValueOf(7)})
	if len(got) != 1 || got[0].Int() != 15 {
		t.Fatalf("reflected variadic result = %v, want 15", got)
	}
	got = value.CallSlice([]reflect.Value{reflect.ValueOf(3), reflect.ValueOf([]int{5, 7})})
	if len(got) != 1 || got[0].Int() != 15 {
		t.Fatalf("reflected variadic slice result = %v, want 15", got)
	}
}

func TestReflectCallPanicRecover(t *testing.T) {
	check := func(name string, call func()) {
		t.Helper()
		deferred := false
		func() {
			defer func() {
				deferred = true
				if got := recover(); got != name {
					t.Fatalf("recover() = %v, want %q", got, name)
				}
			}()
			call()
		}()
		if !deferred {
			t.Fatal("panic did not run deferred recovery")
		}
	}
	check("Call", func() {
		reflect.ValueOf(func() { panic("Call") }).Call(nil)
	})
	typ := reflect.TypeOf((func())(nil))
	value := reflect.MakeFunc(typ, func([]reflect.Value) []reflect.Value { panic("MakeFunc") })
	check("MakeFunc", value.Interface().(func()))
}

func TestReflectABIKinds(t *testing.T) {
	type signature func([]int, any, map[string]int, chan int, complex128, [2]float32) (int, string, complex128)
	implementation := func(values []int, label any, lookup map[string]int, ready chan int, number complex128, pair [2]float32) (int, string, complex128) {
		runtime.GC()
		ready <- len(values)
		return lookup[label.(string)] + <-ready, label.(string), number + complex(float64(pair[0]), float64(pair[1]))
	}
	arguments := []reflect.Value{
		reflect.ValueOf([]int{1, 2, 3}),
		reflect.ValueOf("answer"),
		reflect.ValueOf(map[string]int{"answer": 39}),
		reflect.ValueOf(make(chan int, 1)),
		reflect.ValueOf(complex(20.0, 19.0)),
		reflect.ValueOf([2]float32{2, 3}),
	}
	check := func(name string, results []reflect.Value) {
		t.Helper()
		if len(results) != 3 || results[0].Int() != 42 || results[1].String() != "answer" || results[2].Complex() != complex(22, 22) {
			t.Fatalf("%s results = %v, want [42 answer (22+22i)]", name, results)
		}
	}
	check("Call", reflect.ValueOf(signature(implementation)).Call(arguments))

	value := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		result, label, number := implementation(
			in[0].Interface().([]int), in[1].Interface(), in[2].Interface().(map[string]int),
			in[3].Interface().(chan int), in[4].Complex(), in[5].Interface().([2]float32),
		)
		return []reflect.Value{reflect.ValueOf(result), reflect.ValueOf(label), reflect.ValueOf(number)}
	})
	check("MakeFunc", value.Call(arguments))
}

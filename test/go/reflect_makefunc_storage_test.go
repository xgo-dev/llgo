package gotest

import (
	"reflect"
	"runtime"
	"testing"
)

// The function result must have its full Go storage even when its machine
// pointer is narrower; copying it must not overwrite the adjacent record.
type reflectMakeFuncStorageRecord struct {
	Label string
	Value int64
	Next  *int
}

func TestReflectMakeFuncAggregateAndFuncResult(t *testing.T) {
	type signature func(reflectMakeFuncStorageRecord, func(int) int) (reflectMakeFuncStorageRecord, func() int)
	var escaped []reflect.Value
	value := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		escaped = in
		runtime.GC()
		runtime.Gosched()
		record := in[0].Interface().(reflectMakeFuncStorageRecord)
		record.Value += int64(in[1].Interface().(func(int) int)(*record.Next))
		return []reflect.Value{reflect.ValueOf(record), reflect.ValueOf(func() int { return *record.Next })}
	})
	callable := value.Interface().(signature)
	next := 42
	input := reflectMakeFuncStorageRecord{Label: "copy", Value: 7, Next: &next}
	got, result := callable(input, func(value int) int { return value + 1 })
	if got.Label != "copy" || got.Value != 50 || result() != 42 {
		t.Fatalf("MakeFunc results = (%+v, %d)", got, result())
	}
	input.Label = "changed"
	runtime.GC()
	if got := escaped[0].Interface().(reflectMakeFuncStorageRecord); got.Label != "copy" || *got.Next != 42 {
		t.Fatalf("escaped argument = %+v", got)
	}

	reflected := value.Call([]reflect.Value{
		reflect.ValueOf(input), reflect.ValueOf(func(value int) int { return value }),
	})
	if got := reflected[0].Interface().(reflectMakeFuncStorageRecord); got.Value != 49 {
		t.Fatalf("reflected MakeFunc result = %+v", got)
	}
}

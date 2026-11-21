package gotest

import (
	"reflect"
	"testing"
)

func TestReflectClosureParamTypeString(t *testing.T) {
	fn := func(func(int) bool) {}
	typ := reflect.TypeOf(fn)
	if got, want := typ.String(), "func(func(int) bool)"; got != want {
		t.Fatalf("function parameter type = %q, want %q", got, want)
	}
	if got, want := typ.In(0).Kind(), reflect.Func; got != want {
		t.Fatalf("function parameter kind = %v, want %v", got, want)
	}
}

type reflectClosureFieldHolder struct {
	F     func(int) bool
	Named reflectNamedFunc
}

type reflectNamedFunc func(int) bool

func TestReflectClosureNestedTypeString(t *testing.T) {
	field := reflect.TypeOf(reflectClosureFieldHolder{}).Field(0).Type
	if got, want := field.String(), "func(int) bool"; got != want {
		t.Fatalf("struct field type = %q, want %q", got, want)
	}
	if got, want := field.Kind(), reflect.Func; got != want {
		t.Fatalf("struct field kind = %v, want %v", got, want)
	}
	elem := reflect.TypeOf([]func(int) bool{}).Elem()
	if got, want := elem.String(), "func(int) bool"; got != want {
		t.Fatalf("slice element type = %q, want %q", got, want)
	}
	if got, want := elem.Kind(), reflect.Func; got != want {
		t.Fatalf("slice element kind = %v, want %v", got, want)
	}
	mapElem := reflect.TypeOf(map[int]func(int) bool{}).Elem()
	if got, want := mapElem.String(), "func(int) bool"; got != want {
		t.Fatalf("map element type = %q, want %q", got, want)
	}
	if got, want := mapElem.Kind(), reflect.Func; got != want {
		t.Fatalf("map element kind = %v, want %v", got, want)
	}

	captured := 42
	functions := map[int]func() int{0: func() int { return captured }}
	value := reflect.ValueOf(functions).MapIndex(reflect.ValueOf(0))
	if got, want := value.Kind(), reflect.Func; got != want {
		t.Fatalf("map value kind = %v, want %v", got, want)
	}
	if got := value.Interface().(func() int)(); got != captured {
		t.Fatalf("map value returned %d, want %d", got, captured)
	}

	dynamicCaptured := "dynamic"
	dynamicFunction := func(s string) string { return dynamicCaptured + s }
	dynamicType := reflect.MapOf(reflect.TypeOf(0), reflect.TypeOf(dynamicFunction))
	staticType := reflect.TypeOf(map[int]func(string) string{})
	if dynamicType != staticType {
		t.Fatalf("reflect.MapOf type = %v (%p), want canonical %v (%p)", dynamicType, dynamicType, staticType, staticType)
	}
	dynamic := reflect.MakeMap(dynamicType)
	dynamic.SetMapIndex(reflect.ValueOf(0), reflect.ValueOf(dynamicFunction))
	dynamicValue := dynamic.MapIndex(reflect.ValueOf(0))
	if got, want := dynamicValue.Kind(), reflect.Func; got != want {
		t.Fatalf("dynamic map value kind = %v, want %v", got, want)
	}
	if got, want := dynamicValue.Interface().(func(string) string)(" map"), "dynamic map"; got != want {
		t.Fatalf("dynamic map value returned %q, want %q", got, want)
	}
}

func TestReflectClosurePointerTypeIdentity(t *testing.T) {
	holder := &reflectClosureFieldHolder{}
	holderType := reflect.TypeOf(*holder)
	holderValue := reflect.ValueOf(holder).Elem()

	plainField := holderType.Field(0).Type
	plainPointer := reflect.PointerTo(plainField)
	plainAddr := holderValue.Field(0).Addr().Type()
	if plainAddr != plainPointer {
		t.Fatalf("function field address type = %v, want %v", plainAddr, plainPointer)
	}

	namedParam := reflect.TypeOf(func(*reflectNamedFunc) {}).In(0)
	namedField := holderType.Field(1).Type
	namedPointer := reflect.PointerTo(namedField)
	namedAddr := holderValue.Field(1).Addr().Type()
	if namedParam != namedPointer || namedAddr != namedPointer {
		t.Fatalf("named function pointer types differ: parameter=%v field=%v address=%v", namedParam, namedPointer, namedAddr)
	}
	types := map[reflect.Type]bool{namedParam: true}
	if !types[namedPointer] || !types[namedAddr] {
		t.Fatalf("canonical named function pointer type was not found as a reflect.Type map key")
	}
}

func TestReflectUnnamedFuncConcurrentPublication(t *testing.T) {
	input := reflect.StructOf([]reflect.StructField{{Name: "ColdClosureArg", Type: reflect.TypeOf(0)}})
	fn := reflect.FuncOf([]reflect.Type{input}, nil, false)
	const workers = 32
	start := make(chan struct{})
	results := make(chan reflect.Type, workers)
	for i := 0; i < workers; i++ {
		go func() {
			<-start
			v := reflect.New(fn)
			if v.Elem().Type() != fn || !v.Elem().IsNil() {
				results <- nil
				return
			}
			results <- v.Type()
		}()
	}
	close(start)
	want := <-results
	if want == nil {
		t.Fatal("reflect.New did not create a zero function of the synthesized type")
	}
	for i := 1; i < workers; i++ {
		if got := <-results; got != want {
			t.Fatalf("concurrent function pointer = %v, want canonical %v", got, want)
		}
	}
	if got := reflect.PointerTo(fn); got != want {
		t.Fatalf("reflect.PointerTo = %v, want canonical %v", got, want)
	}
}

func TestReflectClosureIndexSet(t *testing.T) {
	left, right := 1, 2
	var arr [2]func() int
	reflect.ValueOf(&arr).Elem().Index(0).Set(reflect.ValueOf(func() int { return left }))
	reflect.ValueOf(&arr).Elem().Index(1).Set(reflect.ValueOf(func() int { return right }))
	if got, want := arr[0](), left; got != want {
		t.Fatalf("array[0] = %d, want %d", got, want)
	}
	if got, want := arr[1](), right; got != want {
		t.Fatalf("array[1] = %d, want %d", got, want)
	}

	slice := make([]func() int, 2)
	reflect.ValueOf(slice).Index(0).Set(reflect.ValueOf(func() int { return left + 10 }))
	reflect.ValueOf(slice).Index(1).Set(reflect.ValueOf(func() int { return right + 10 }))
	if got, want := slice[0](), left+10; got != want {
		t.Fatalf("slice[0] = %d, want %d", got, want)
	}
	if got, want := slice[1](), right+10; got != want {
		t.Fatalf("slice[1] = %d, want %d", got, want)
	}

	var local struct {
		Callbacks [2]func() int
		Value     int
	}
	reflect.ValueOf(&local).Elem().Field(0).Index(0).Set(reflect.ValueOf(func() int { return left + 20 }))
	reflect.ValueOf(&local).Elem().Field(0).Index(1).Set(reflect.ValueOf(func() int { return right + 20 }))
	if got, want := local.Callbacks[0](), left+20; got != want {
		t.Fatalf("struct array[0] = %d, want %d", got, want)
	}
	if got, want := local.Callbacks[1](), right+20; got != want {
		t.Fatalf("struct array[1] = %d, want %d", got, want)
	}
}

func TestReflectClosureArrayOfSliceOf(t *testing.T) {
	fnType := reflect.TypeOf((func() int)(nil))
	if got, want := reflect.ArrayOf(2, fnType).Size(), reflect.TypeOf([2]func() int{}).Size(); got != want {
		t.Fatalf("ArrayOf size = %d, want %d", got, want)
	}
	if got, want := reflect.ArrayOf(2, fnType).Elem().Kind(), reflect.Func; got != want {
		t.Fatalf("ArrayOf elem kind = %v, want %v", got, want)
	}

	n := 7
	made := reflect.MakeSlice(reflect.SliceOf(fnType), 2, 2)
	made.Index(0).Set(reflect.ValueOf(func() int { return n }))
	made.Index(1).Set(reflect.ValueOf(func() int { return n + 1 }))
	got := made.Interface().([]func() int)
	if got[0]() != n || got[1]() != n+1 {
		t.Fatalf("MakeSlice values = %d, %d; want %d, %d", got[0](), got[1](), n, n+1)
	}
}

func TestReflectClosureSliceClear(t *testing.T) {
	left, right := 3, 4
	slice := make([]func() int, 2)
	slice[0] = func() int { return left }
	slice[1] = func() int { return right }

	call := func(v reflect.Value) int {
		t.Helper()
		return v.Index(0).Interface().(func() int)()
	}
	tail := reflect.ValueOf(slice).Slice(1, 2)
	if tail.Len() != 1 {
		t.Fatalf("Slice(1, 2) len = %d, want 1", tail.Len())
	}
	if got, want := call(tail), right; got != want {
		t.Fatalf("Slice(1, 2)[0] = %d, want %d", got, want)
	}

	mid := reflect.ValueOf(slice).Slice3(1, 2, 2)
	if got, want := call(mid), right; got != want {
		t.Fatalf("Slice3(1, 2, 2)[0] = %d, want %d", got, want)
	}

	var arr [2]func() int
	arr[0] = func() int { return left }
	arr[1] = func() int { return right }
	sliced := reflect.ValueOf(&arr).Elem().Slice(1, 2)
	if got, want := call(sliced), right; got != want {
		t.Fatalf("array Slice(1, 2)[0] = %d, want %d", got, want)
	}

	reflect.ValueOf(slice).Clear()
	if slice[0] != nil || slice[1] != nil {
		t.Fatal("Clear left a stale function value")
	}
}

func TestReflectCallCapturedClosure(t *testing.T) {
	captured := new(int)
	*captured = 7
	fn := func(a, b, c, d, e, f, g, h, i int) int {
		*captured += a + b + c + d + e + f + g + h + i
		return *captured
	}
	args := make([]reflect.Value, 9)
	for i := range args {
		args[i] = reflect.ValueOf(i + 1)
	}
	for _, want := range []int64{52, 97} {
		result := reflect.ValueOf(fn).Call(args)
		if got := result[0].Int(); got != want || int64(*captured) != want {
			t.Fatalf("captured closure returned %d, captured %d; want %d", got, *captured, want)
		}
	}
}

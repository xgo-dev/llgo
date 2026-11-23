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

func TestReflectClosureStructFields(t *testing.T) {
	type fields = struct {
		F     func(int) bool
		Named reflectNamedFunc
	}
	offset := 3
	original := fields{F: func(n int) bool { return n > offset }}
	typ := reflect.TypeOf(original)
	dynamic := reflect.StructOf([]reflect.StructField{typ.Field(0), typ.Field(1)})
	if dynamic != typ {
		t.Fatalf("StructOf = %v, want canonical %v", dynamic, typ)
	}

	type promoted struct{ *fields }
	value := reflect.ValueOf(promoted{&original})
	field := value.FieldByName("F")
	if field.Type() != typ.Field(0).Type || !field.Interface().(func(int) bool)(offset+1) {
		t.Fatalf("promoted function field = %v", field)
	}
	named := reflectNamedFunc(func(n int) bool { return n == offset })
	field, err := value.FieldByIndexErr([]int{0, 1})
	if err != nil {
		t.Fatal(err)
	}
	field.Set(reflect.ValueOf(named))
	if !original.Named(offset) || original.Named(offset+1) {
		t.Fatal("setting the named function field lost the closure")
	}
	zero := reflect.ValueOf(promoted{})
	if field, err := zero.FieldByIndexErr([]int{0, 0}); err == nil || field.IsValid() {
		t.Fatal("nil embedded pointer did not return an invalid value and an error")
	}
	defer func() {
		if recover() == nil {
			t.Error("FieldByName through a nil embedded pointer did not panic")
		}
	}()
	zero.FieldByName("F")
}

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

type reflectCallbackArg struct{ Words [7]uint32 }

//llgo:cdecl
type reflectCallbackAlias = func(reflectCallbackArg) int64

func reflectCallbackWord(arg reflectCallbackArg) int64 { return int64(arg.Words[0]) }

func TestReflectArrayOfNativeCallback(t *testing.T) {
	src := reflect.ValueOf([2]reflectCallbackAlias{nil, reflectCallbackWord})
	typ := reflect.ArrayOf(src.Len(), src.Type().Elem())
	if typ != src.Type() {
		t.Errorf("ArrayOf returned a different descriptor for %v", src.Type())
	}
	dst := reflect.New(typ).Elem()
	dst.Set(src)
	got, ok := dst.Interface().([2]reflectCallbackAlias)
	if !ok {
		t.Fatal("rebuilt array failed its original type assertion")
	}
	arg := reflectCallbackArg{Words: [7]uint32{42}}
	if got[0] != nil || got[1] == nil || got[1](arg) != 42 {
		t.Fatal("rebuilt array lost its callback")
	}
}

func TestReflectCallbackContainerElements(t *testing.T) {
	// Only the native containers are declared. Looking up the ordinary
	// signature must still produce storage that can hold a capturing closure.
	n := int64(42)
	fn := reflect.ValueOf(func(reflectCallbackArg) int64 { return n })
	key := reflect.ValueOf(1)
	check := func(native reflect.Type, build func(reflect.Type) reflect.Type) reflect.Type {
		t.Helper()
		if build(native.Elem()) != native {
			t.Errorf("rebuilding %v lost its type identity", native)
		}
		typ := build(fn.Type())
		if typ.Elem() != fn.Type() {
			t.Fatalf("%v selected the native callback element for a Go closure", typ)
		}
		return typ
	}
	array := check(reflect.TypeOf([2]reflectCallbackAlias{}), func(e reflect.Type) reflect.Type { return reflect.ArrayOf(2, e) })
	slice := check(reflect.TypeOf([]reflectCallbackAlias{}), reflect.SliceOf)
	mapType := check(reflect.TypeOf(map[int]reflectCallbackAlias{}), func(e reflect.Type) reflect.Type { return reflect.MapOf(key.Type(), e) })
	a := reflect.New(array).Elem()
	s := reflect.MakeSlice(slice, 2, 2)
	m := reflect.MakeMap(mapType)
	a.Index(1).Set(fn)
	s.Index(1).Set(fn)
	m.SetMapIndex(key, fn)
	for _, v := range []reflect.Value{a.Index(1), s.Index(1), m.MapIndex(key)} {
		if got := v.Interface().(func(reflectCallbackArg) int64)(reflectCallbackArg{}); got != n {
			t.Fatalf("stored closure returned %d, want %d", got, n)
		}
	}
}

func TestReflectHigherOrderClosureContainers(t *testing.T) {
	captured := int64(41)
	arg := reflectCallbackArg{Words: [7]uint32{42}}
	parameter := func(callback reflectCallbackAlias) int64 { return captured + callback(arg) }
	result := func() reflectCallbackAlias {
		if captured == 42 {
			return reflectCallbackWord
		}
		return nil
	}
	captured++
	t.Run("parameter", func(t *testing.T) {
		checkReflectHigherOrderContainers(t, parameter, func(_ *testing.T, fn func(reflectCallbackAlias) int64) int64 {
			return fn(reflectCallbackWord)
		}, 84)
	})
	t.Run("result", func(t *testing.T) {
		checkReflectHigherOrderContainers(t, result, func(t *testing.T, fn func() reflectCallbackAlias) int64 {
			callback := fn()
			if callback == nil {
				t.Fatal("stored closure returned a nil callback")
			}
			return callback(arg)
		}, 42)
	})
}

func checkReflectHigherOrderContainers[F any](t *testing.T, fn F, call func(*testing.T, F) int64, want int64) {
	t.Helper()
	var zero F
	for _, value := range []struct {
		name string
		fn   F
	}{{"nil", zero}, {"captured", fn}} {
		t.Run(value.name, func(t *testing.T) {
			for _, container := range []struct {
				name   string
				source any
				build  func(reflect.Type) reflect.Type
				read   func(any) F
			}{
				{"array", [2]F{zero, value.fn}, func(e reflect.Type) reflect.Type { return reflect.ArrayOf(2, e) },
					func(v any) F { return v.([2]F)[1] }},
				{"slice", []F{zero, value.fn}, reflect.SliceOf,
					func(v any) F { return v.([]F)[1] }},
				{"map", map[int]F{1: value.fn}, func(e reflect.Type) reflect.Type { return reflect.MapOf(reflect.TypeOf(0), e) },
					func(v any) F { return v.(map[int]F)[1] }},
			} {
				t.Run(container.name, func(t *testing.T) {
					source := reflect.ValueOf(container.source)
					for _, element := range []struct {
						name string
						typ  reflect.Type
					}{{"TypeOf", reflect.TypeOf(value.fn)}, {"Elem", source.Type().Elem()}} {
						t.Run(element.name, func(t *testing.T) {
							typ := container.build(element.typ)
							if typ != source.Type() {
								t.Fatalf("rebuilt %v lost its canonical type identity", source.Type())
							}
							dst := reflect.New(typ).Elem()
							dst.Set(source)
							if dst.Kind() == reflect.Map {
								dst.SetMapIndex(reflect.ValueOf(1), reflect.ValueOf(value.fn))
							} else {
								dst.Index(1).Set(reflect.ValueOf(value.fn))
							}
							stored := container.read(dst.Interface())
							if value.name == "nil" {
								if !reflect.ValueOf(stored).IsNil() {
									t.Fatal("rebuilt container lost its nil function")
								}
							} else if got := call(t, stored); got != want {
								t.Fatalf("stored closure returned %d, want %d", got, want)
							}
						})
					}
				})
			}
		})
	}
}

func TestReflectHigherOrderContainerTypeSelection(t *testing.T) {
	// Keep this signature distinct so its public containers do not become
	// exact candidates for the fallback cases above.
	fnType := reflect.TypeOf((func(reflectCallbackAlias, byte) int64)(nil))
	offset := int64(41)
	public := reflect.ValueOf(func(f func(reflectCallbackArg) int64, b byte) int64 {
		return offset + f(reflectCallbackArg{Words: [7]uint32{uint32(b)}})
	})
	for _, tc := range []struct {
		name              string
		qualified, public reflect.Type
		build             func(reflect.Type) reflect.Type
	}{
		{"array", reflect.TypeOf([2]func(reflectCallbackAlias, byte) int64{}),
			reflect.TypeOf([2]func(func(reflectCallbackArg) int64, byte) int64{}),
			func(e reflect.Type) reflect.Type { return reflect.ArrayOf(2, e) }},
		{"slice", reflect.TypeOf([]func(reflectCallbackAlias, byte) int64{}),
			reflect.TypeOf([]func(func(reflectCallbackArg) int64, byte) int64{}), reflect.SliceOf},
		{"map", reflect.TypeOf(map[int]func(reflectCallbackAlias, byte) int64{}),
			reflect.TypeOf(map[int]func(func(reflectCallbackArg) int64, byte) int64{}),
			func(e reflect.Type) reflect.Type { return reflect.MapOf(reflect.TypeOf(0), e) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elem := tc.qualified.Elem()
			if !fnType.AssignableTo(elem) || !elem.AssignableTo(fnType) {
				t.Errorf("function and %s element must be mutually assignable", tc.name)
			}
			slot := reflect.New(elem).Elem()
			slot.Set(public)
			fn := slot.Interface().(func(reflectCallbackAlias, byte) int64)
			if got := fn(reflectCallbackWord, 1); got != 42 {
				t.Fatalf("assigned function returned %d, want 42", got)
			}
			for _, element := range []struct {
				name      string
				typ, want reflect.Type
			}{
				{"TypeOf", fnType, tc.public},
				{"public Elem", tc.public.Elem(), tc.public},
				{"qualified Elem", tc.qualified.Elem(), tc.qualified},
			} {
				if got := tc.build(element.typ); got != element.want {
					t.Errorf("%s rebuilt %v with a different type identity", element.name, element.want)
				}
			}
		})
	}
}

//llgo:cdecl
type reflectAssignmentAlias = func(int64) int64

func TestReflectHigherOrderNamedFuncAssignment(t *testing.T) {
	type Named func(func(int64) int64) int64
	named := reflect.TypeFor[Named]()
	qualified := reflect.TypeFor[[]func(reflectAssignmentAlias) int64]().Elem()
	for _, tc := range []struct {
		name     string
		src, dst reflect.Type
	}{
		{"named to qualified", named, qualified},
		{"qualified to named", qualified, named},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.src.AssignableTo(tc.dst) {
				t.Fatalf("%v must be assignable to %v", tc.src, tc.dst)
			}
			dst := reflect.New(tc.dst).Elem()
			dst.Set(reflect.Zero(tc.src))
			if !dst.IsNil() {
				t.Fatal("assignment did not preserve a nil function")
			}
		})
	}
}

func TestReflectHigherOrderFuncAssignmentMismatch(t *testing.T) {
	type First func(reflectCallbackAlias) int64
	type Second func(reflectCallbackAlias) int64
	first, second := reflect.TypeFor[First](), reflect.TypeFor[Second]()
	if first.AssignableTo(second) {
		t.Fatal("distinct named function types are assignable")
	}
	t.Run("named Set", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("Set accepted distinct named function types")
			}
		}()
		reflect.New(second).Elem().Set(reflect.Zero(first))
	})
	variadic := reflect.TypeOf((func(reflectCallbackAlias, ...byte) int64)(nil))
	slice := reflect.TypeOf((func(func(reflectCallbackArg) int64, []byte) int64)(nil))
	if variadic.AssignableTo(slice) || slice.AssignableTo(variadic) {
		t.Fatal("variadic and slice signatures are assignable")
	}
}

func TestReflectZeroFuncAssignment(t *testing.T) {
	type namedFunc func()
	n := 1
	plain := func() { n++ }
	named := namedFunc(plain)
	public := func(f func(reflectCallbackArg) int64) int64 { return f(reflectCallbackArg{}) }
	qualified := func(f reflectCallbackAlias) int64 { return f(reflectCallbackArg{}) }
	qualifiedType := reflect.TypeOf([1]func(reflectCallbackAlias) int64{}).Elem()
	for _, tc := range []struct {
		name string
		from reflect.Type
		dst  any
	}{
		{"named to unnamed", reflect.TypeOf(named), &plain},
		{"unnamed to named", reflect.TypeOf(plain), &named},
		{"public to qualified", reflect.TypeOf(public), &qualified},
		{"qualified to public", qualifiedType, &public},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := reflect.ValueOf(tc.dst).Elem()
			if dst.IsNil() || !tc.from.AssignableTo(dst.Type()) {
				t.Fatalf("expected an initialized destination assignable from %v", tc.from)
			}
			dst.Set(reflect.Zero(tc.from))
			if !dst.IsNil() {
				t.Fatal("assigning a zero function did not clear the destination")
			}
		})
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

package main

import (
	"reflect"
	"unsafe"
)

func reflectedBase(n int, s string) (bool, int) { return true, n + len(s) }

//llgo:cdecl
type reflectedCFunc func(int, string) (bool, int)

type reflectedFunc func(int, string) (bool, int)

func (f reflectedFunc) Call(s string) (bool, int) { return f(100, s) }
func (f reflectedFunc) Demo() int                 { return 100 }

type reflectedHolder struct{ Fn reflectedFunc }

func checkCall(value reflect.Value) {
	result := value.Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf("hello")})
	if !result[0].Bool() || result[1].Int() != 105 {
		panic("reflected function call")
	}
}

func testFunctionMetadata() {
	typ := reflect.TypeOf((*reflectedFunc)(nil)).Elem()
	if typ.Kind() != reflect.Func || typ.NumIn() != 2 || typ.NumOut() != 2 || typ.NumMethod() != 2 {
		panic("named function metadata")
	}
	if typ.In(0).Kind() != reflect.Int || typ.In(1).Kind() != reflect.String ||
		typ.Out(0).Kind() != reflect.Bool || typ.Out(1).Kind() != reflect.Int || typ.IsVariadic() {
		panic("named function input/output metadata")
	}
	if reflect.TypeOf(reflectedHolder{}).Field(0).Type.Kind() != reflect.Func {
		panic("function field metadata")
	}
	intType := reflect.TypeOf(0)
	if intType.Kind() != reflect.Int || intType.Name() != "int" {
		panic("named int metadata #1412")
	}

	base := reflect.ValueOf(reflectedBase)
	named := reflect.New(typ).Elem()
	named.Set(base)
	checkCall(named)
	method := named.MethodByName("Call").Call([]reflect.Value{reflect.ValueOf("hello")})
	if !method[0].Bool() || method[1].Int() != 105 {
		panic("named function method")
	}
	if result := named.MethodByName("Demo").Call(nil); len(result) != 1 || result[0].Int() != 100 {
		panic("zero-argument named function method")
	}

	closure := func(n int, s string) (bool, int) { return true, n + len(s) }
	newBase := reflect.New(base.Type()).Elem()
	newBase.Set(base)
	checkCall(newBase)
	newClosure := reflect.New(reflect.TypeOf(closure)).Elem()
	newClosure.Set(reflect.ValueOf(closure))
	checkCall(newClosure)
	baseFunction := reflectedBase
	newAtBase := reflect.NewAt(base.Type(), unsafe.Pointer(&baseFunction)).Elem()
	checkCall(newAtBase)
	newAt := reflect.NewAt(reflect.TypeOf(closure), unsafe.Pointer(&closure)).Elem()
	checkCall(newAt)

	noArguments := func() {}
	if reflect.ValueOf(noArguments).Kind() != reflect.Func || reflect.TypeOf(noArguments).NumIn() != 0 || reflect.TypeOf(noArguments).NumOut() != 0 {
		panic("no-argument function metadata")
	}
	returnsClosure := func(offset int) func(int) int {
		return func(value int) int { return value + offset }
	}
	returned := reflect.ValueOf(returnsClosure).Call([]reflect.Value{reflect.ValueOf(40)})[0]
	if returned.Kind() != reflect.Func || returned.Call([]reflect.Value{reflect.ValueOf(2)})[0].Int() != 42 {
		panic("function-returning-closure metadata")
	}

	cTyp := reflect.TypeOf((*reflectedCFunc)(nil)).Elem()
	cValue := reflect.New(cTyp).Elem()
	// Construct the native callable explicitly: a Go descriptor and a C
	// entry pointer have distinct callable identities despite equal widths.
	cValue.Set(reflect.ValueOf(reflectedCFunc(reflectedBase)))
	checkCall(cValue)
}

package main

import "reflect"

type sumMethod interface {
	Sum(int, int, int, int, int, int, int, int, int) int
}

type variadicReceiver int

func (r variadicReceiver) Add(values ...int) int {
	sum := int(r)
	for _, value := range values {
		sum += value
	}
	return sum
}

type functionReceiver func(int) int

func (f functionReceiver) Apply(value int) int {
	return f(value)
}

func (r receiver) Transform(fn func(int) int) func(int) int {
	return func(value int) int { return fn(value) + r.base }
}

func checkMethodTypes() {
	value := receiver{base: 10}
	var iface sumMethod = value
	expected := reflect.TypeOf((func(int, int, int, int, int, int, int, int, int) int)(nil))
	methods := []reflect.Value{
		reflect.ValueOf(value).MethodByName("Sum"),
		reflect.ValueOf(&value).MethodByName("Sum"),
		reflect.ValueOf(&iface).Elem().MethodByName("Sum"),
	}
	for _, method := range methods {
		if method.Type() != expected || reflect.TypeOf(method.Interface()) != expected {
			panic("bound method lost its Go function type")
		}
		bound := method.Interface().(func(int, int, int, int, int, int, int, int, int) int)
		if bound(1, 2, 3, 4, 5, 6, 7, 8, 9) != 55 {
			panic("bound method lost its receiver or arguments")
		}
	}
	interfaceType := reflect.TypeOf((*sumMethod)(nil)).Elem()
	if interfaceType.Method(0).Type != expected {
		panic("interface method lost its Go function type")
	}

	variadic := reflect.ValueOf(variadicReceiver(10)).MethodByName("Add")
	variadicType := reflect.TypeOf((func(...int) int)(nil))
	if variadic.Type() != variadicType || !variadic.Type().IsVariadic() {
		panic("bound method lost its variadic function type")
	}
	if variadic.Interface().(func(...int) int)(1, 2, 3) != 16 {
		panic("bound variadic method lost its slice arguments")
	}

	function := functionReceiver(func(value int) int { return value + 10 })
	method := reflect.ValueOf(function).MethodByName("Apply")
	if method.Type() != reflect.TypeOf((func(int) int)(nil)) {
		panic("function receiver replaced its method signature")
	}
	if method.Interface().(func(int) int)(5) != 15 {
		panic("function receiver lost its descriptor")
	}

	transform := reflect.ValueOf(value).MethodByName("Transform")
	transformType := reflect.TypeOf((func(func(int) int) func(int) int)(nil))
	if transform.Type() != transformType {
		panic("method changed a function parameter or result type")
	}
	boundTransform := transform.Interface().(func(func(int) int) func(int) int)
	result := boundTransform(func(value int) int { return value * 2 })
	if result(5) != 20 {
		panic("method lost a function parameter or result descriptor")
	}
}

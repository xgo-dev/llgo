// LITTEST
package main

import (
	"reflect"
	"strings"
)

// CHECK-LABEL: define void @"g{{.*}}/cl/_testgo/reflectmkfn.main"(){{.*}} {
// CHECK: call %"g{{.*}}/runtime/internal/runtime.iface" @reflect.FuncOf(
// CHECK: call %reflect.Value @reflect.MakeFunc(
// CHECK: call %"g{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(
// CHECK: call i1 @"g{{.*}}/runtime/internal/runtime.MatchesClosure"(
// CHECK: call %"g{{.*}}/runtime/internal/runtime.String" %{{.*}}(ptr %{{.*}}, %"g{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 3 }, i64 2)
// CHECK: call i1 @"g{{.*}}/runtime/internal/runtime.StringEqual"(
func main() {
	typ := reflect.FuncOf([]reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)}, []reflect.Type{reflect.TypeOf("")}, false)
	fn := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		r := strings.Repeat(args[0].String(), int(args[1].Int()))
		return []reflect.Value{reflect.ValueOf(r)}
	})
	r := fn.Interface().(func(string, int) string)("abc", 2)
	if r != "abcabc" {
		panic("error")
	}
}

// CHECK-LABEL: define %"g{{.*}}/runtime/internal/runtime.Slice" @"g{{.*}}/cl/_testgo/reflectmkfn.main$1"(%"g{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK: call %"g{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(
// CHECK: call i64 @reflect.Value.Int(
// CHECK: call %"g{{.*}}/runtime/internal/runtime.String" @strings.Repeat(
// CHECK: call %reflect.Value @reflect.ValueOf(
// CHECK: ret %"g{{.*}}/runtime/internal/runtime.Slice"
// CHECK: }

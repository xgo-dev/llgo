// LITTEST
// Scope: common
package main

import "go/constant"

type myInt int64

func boxMyInt(x int64) any { return myInt(x) }

// CHECK-LABEL: define {{.*}} @main.boxed(){{.*}} {
// CHECK-NOT: MakeInt64
// CHECK-NOT: MakeBool
// CHECK-NOT: boxMyInt
// CHECK-NOT: AllocU
// CHECK: _llgo_itab$
// CHECK: _llgo_ifacebox$
func boxed() (constant.Value, constant.Value, any) {
	return constant.MakeInt64(42), constant.MakeBool(true), boxMyInt(7)
}

// Convert changes the value; do not fold the raw constant as the result type.
func boxRune(x int32) any { return any(string(x)) }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call {{.*}} @main.boxRune
func main() {
	i, b, m := boxed()
	n, ok := constant.Int64Val(i)
	if !ok || n != 42 || constant.BoolVal(b) != true {
		panic("const iface")
	}
	if m.(myInt) != 7 {
		panic(m)
	}
	s := constant.MakeString("hi")
	if constant.StringVal(s) != "hi" {
		panic(s)
	}
	e := constant.MakeString("")
	if constant.StringVal(e) != "" {
		panic("empty")
	}
	if constant.MakeInt64(42) != i {
		panic("int64 equality")
	}
	if boxRune(65).(string) != "A" {
		panic("convert fold")
	}
	println("ok")
}

// LITTEST darwin/arm64 linux/amd64
package main

// ARM64-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.eface" @"main.boxFuncs$1[int]"(ptr swiftself
// AMD64-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.eface" @"main.boxFuncs$1[int]"(ptr nest
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr [[INT_BOX:@"_llgo_main\.box\[int\]"]], ptr undef }
// CHECK-LABEL: define linkonce i1 @"main.boxFuncs$2[int]"(
// CHECK: icmp eq ptr %{{.*}}, [[INT_BOX]]
// ARM64-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.eface" @"main.boxFuncs$1[string]"(ptr swiftself
// AMD64-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.eface" @"main.boxFuncs$1[string]"(ptr nest
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr [[STRING_BOX:@"_llgo_main\.box\[string\]"]], ptr undef }
// CHECK-LABEL: define linkonce i1 @"main.boxFuncs$2[string]"(
// CHECK: icmp eq ptr %{{.*}}, [[STRING_BOX]]

func boxFuncs[T any](value T) (func() any, func(any) bool) {
	type box struct {
		value T
	}
	b := box{value: value}
	makeValue := func() any {
		return b
	}
	isBox := func(v any) bool {
		_, ok := v.(box)
		return ok
	}
	return makeValue, isBox
}

func main() {
	makeInt, isIntBox := boxFuncs(123456789)
	makeString, isStringBox := boxFuncs("closure-env-ok")
	intBox := makeInt()
	stringBox := makeString()
	println(isIntBox(intBox))
	println(isStringBox(stringBox))
	println(isIntBox(stringBox))
	println(isStringBox(intBox))
}

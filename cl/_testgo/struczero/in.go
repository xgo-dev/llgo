// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/foo"

type bar struct {
	pb *byte
	f  float32
}

// A comma-ok assertion must pair the asserted value with its success bit and
// use an all-zero pair on failure, preserving zero values across packages.
// CHECK-LABEL: define { %"{{.*}}foo.Foo", i1 } @main.Bar(%"{{.*}}eface" %0){{.*}} {
// CHECK: [[BAR_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" %0, 0
// CHECK-NEXT: [[BAR_MATCH:%[0-9]+]] = icmp eq ptr [[BAR_TYPE]], @"_llgo_{{.*}}foo.Foo"
// CHECK-NEXT: br i1 [[BAR_MATCH]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: [[BAR_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" %0, 1
// CHECK-NEXT: [[BAR_VALUE:%[0-9]+]] = load %"{{.*}}foo.Foo", ptr [[BAR_DATA]]
// CHECK-NEXT: [[BAR_OK0:%[0-9]+]] = insertvalue { %"{{.*}}foo.Foo", i1 } undef, %"{{.*}}foo.Foo" [[BAR_VALUE]], 0
// CHECK-NEXT: [[BAR_OK:%[0-9]+]] = insertvalue { %"{{.*}}foo.Foo", i1 } [[BAR_OK0]], i1 true, 1
// CHECK: [[BAR_PAIR:%[0-9]+]] = phi { %"{{.*}}foo.Foo", i1 } [ [[BAR_OK]], %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK-NEXT: [[BAR_RET_VALUE:%[0-9]+]] = extractvalue { %"{{.*}}foo.Foo", i1 } [[BAR_PAIR]], 0
// CHECK-NEXT: [[BAR_RET_OK:%[0-9]+]] = extractvalue { %"{{.*}}foo.Foo", i1 } [[BAR_PAIR]], 1
// CHECK-NEXT: [[BAR_RET0:%[0-9]+]] = insertvalue { %"{{.*}}foo.Foo", i1 } undef, %"{{.*}}foo.Foo" [[BAR_RET_VALUE]], 0
// CHECK-NEXT: [[BAR_RET:%[0-9]+]] = insertvalue { %"{{.*}}foo.Foo", i1 } [[BAR_RET0]], i1 [[BAR_RET_OK]], 1
// CHECK-NEXT: ret { %"{{.*}}foo.Foo", i1 } [[BAR_RET]]
func Bar(v any) (ret foo.Foo, ok bool) {
	ret, ok = v.(foo.Foo)
	return
}

// CHECK-LABEL: define { %main.bar, i1 } @main.Foo(%"{{.*}}eface" %0){{.*}} {
// CHECK: [[FOO_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" %0, 0
// CHECK-NEXT: [[FOO_MATCH:%[0-9]+]] = icmp eq ptr [[FOO_TYPE]], @_llgo_main.bar
// CHECK: [[FOO_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" %0, 1
// CHECK-NEXT: [[FOO_VALUE:%[0-9]+]] = load %main.bar, ptr [[FOO_DATA]]
// CHECK: [[FOO_PAIR:%[0-9]+]] = phi { %main.bar, i1 } [ {{.*}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK-NEXT: [[FOO_RET_VALUE:%[0-9]+]] = extractvalue { %main.bar, i1 } [[FOO_PAIR]], 0
// CHECK-NEXT: [[FOO_RET_OK:%[0-9]+]] = extractvalue { %main.bar, i1 } [[FOO_PAIR]], 1
func Foo(v any) (ret bar, ok bool) {
	ret, ok = v.(bar)
	return
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: [[NIL_ASSERT:%[0-9]+]] = call { %main.bar, i1 } @main.Foo(%"{{.*}}eface" zeroinitializer)
	// CHECK-NEXT: [[NIL_VALUE:%[0-9]+]] = extractvalue { %main.bar, i1 } [[NIL_ASSERT]], 0
	// CHECK: [[NIL_OK:%[0-9]+]] = extractvalue { %main.bar, i1 } [[NIL_ASSERT]], 1
	// CHECK: [[NOT_OK:%[0-9]+]] = xor i1 [[NIL_OK]], true
	// CHECK: call void @"{{.*}}PrintBool"(i1 [[NOT_OK]])
	ret, ok := Foo(nil)
	println(ret.pb, ret.f, "notOk:", !ok)

	// CHECK: [[BOX:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 16)
	// CHECK-NEXT: store %"{{.*}}foo.Foo" zeroinitializer, ptr [[BOX]]
	// CHECK-NEXT: [[BOXED:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @"_llgo_{{.*}}foo.Foo", ptr undef }, ptr [[BOX]], 1
	// CHECK-NEXT: [[FOO_ASSERT:%[0-9]+]] = call { %"{{.*}}foo.Foo", i1 } @main.Bar(%"{{.*}}eface" [[BOXED]])
	// CHECK: [[ASSERTED_FOO:%[0-9]+]] = load %"{{.*}}foo.Foo", ptr [[ASSERTED_ADDR:%[0-9]+]]
	// CHECK-NEXT: [[PB:%[0-9]+]] = call ptr @"{{.*}}foo.Foo.Pb"(%"{{.*}}foo.Foo" [[ASSERTED_FOO]])
	// CHECK: call void @"{{.*}}PrintPointer"(ptr [[PB]])
	ret2, ok2 := Bar(foo.Foo{})
	println(ret2.Pb(), ret2.F, ok2)
}

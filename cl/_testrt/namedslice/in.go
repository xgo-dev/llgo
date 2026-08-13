// LITTEST
package main

type MyBytes []byte

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 24)
// CHECK-NEXT: store %"{{.*}}Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr [[DATA]]
// CHECK-NEXT: [[BOXED:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr [[DATA]], 1
// CHECK-NEXT: [[TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[BOXED]], 0
// CHECK-NEXT: [[MATCH:%[0-9]+]] = icmp eq ptr [[TYPE]], @_llgo_main.MyBytes
// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" {{.*}})
// CHECK: [[ASSERT_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[BOXED]], 1
// CHECK-NEXT: [[ASSERT_VALUE:%[0-9]+]] = load %"{{.*}}Slice", ptr [[ASSERT_DATA]]
// CHECK: [[ASSERT_PAIR:%[0-9]+]] = phi { %"{{.*}}Slice", i1 } [ {{.*}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK-NEXT: extractvalue { %"{{.*}}Slice", i1 } [[ASSERT_PAIR]], 0
// CHECK-NEXT: [[OK:%[0-9]+]] = extractvalue { %"{{.*}}Slice", i1 } [[ASSERT_PAIR]], 1
// CHECK-NEXT: br i1 [[OK]], label %{{[^,]+}}, label %{{[^ ]+}}
func main() {
	var i any = MyBytes{}
	_, ok := i.(MyBytes)
	if !ok {
		panic("bad slice")
	}
}

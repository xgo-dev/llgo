// LITTEST
package main

import "github.com/goplus/lib/c"

// Generic instantiation keeps the concrete type in interface metadata and
// preserves both value and pointer method ABIs. llgo.advance must lower to the
// same array GEP for its C helper and linked method spellings.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ANY_VALUE:%[0-9]+]] = load %"main.T[string,int]", ptr
// CHECK: [[ANY_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 24)
// CHECK-NEXT: store %"main.T[string,int]" [[ANY_VALUE]], ptr [[ANY_DATA]]
// CHECK-NEXT: [[ANY:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @"_llgo_main.T[string,int]", ptr undef }, ptr [[ANY_DATA]], 1
// CHECK-NEXT: [[ANY_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[ANY]], 0
// CHECK-NEXT: [[ASSERT_OK:%[0-9]+]] = icmp eq ptr [[ANY_TYPE]], @"_llgo_main.T[string,int]"
// CHECK-NEXT: br i1 [[ASSERT_OK]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: {{^_llgo_[0-9]+:}}
// CHECK: [[ASSERT_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[ANY]], 1
// CHECK-NEXT: [[ASSERT_VALUE:%[0-9]+]] = load %"main.T[string,int]", ptr [[ASSERT_DATA]]
// CHECK-NEXT: [[ASSERT_M:%[0-9]+]] = extractvalue %"main.T[string,int]" [[ASSERT_VALUE]], 0
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[ASSERT_M]])
// CHECK: [[METHOD_VALUE:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 24)
// CHECK: [[METHOD_M:%[0-9]+]] = getelementptr inbounds %"main.T[string,int]", ptr [[METHOD_VALUE]], i32 0, i32 0
// CHECK-NEXT: [[METHOD_N:%[0-9]+]] = getelementptr inbounds %"main.T[string,int]", ptr [[METHOD_VALUE]], i32 0, i32 1
// CHECK: store %"{{.*}}String" { ptr {{.*}}, i64 5 }, ptr [[METHOD_M]]
// CHECK-NEXT: store i64 100, ptr [[METHOD_N]]
// CHECK-NEXT: [[ITAB:%[0-9]+]] = call ptr @"{{.*}}NewItab"(ptr {{.*}}, ptr @"*_llgo_main.T[string,int]")
// CHECK-NEXT: [[IFACE0:%[0-9]+]] = insertvalue %"{{.*}}iface" undef, ptr [[ITAB]], 0
// CHECK-NEXT: [[IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" [[IFACE0]], ptr [[METHOD_VALUE]], 1
// CHECK-NEXT: [[RECEIVER:%[0-9]+]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" [[IFACE]])
// CHECK-NEXT: [[METHODS:%[0-9]+]] = extractvalue %"{{.*}}iface" [[IFACE]], 0
// CHECK-NEXT: [[DEMO_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[METHODS]], i64 3
// CHECK-NEXT: [[DEMO_CODE:%[0-9]+]] = load ptr, ptr [[DEMO_SLOT]]
// CHECK-NEXT: [[METHOD_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[DEMO_CODE]], 0
// CHECK-NEXT: [[METHOD_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[METHOD_PAIR0]], ptr [[RECEIVER]], 1
// CHECK-NEXT: [[CALL_RECEIVER:%[0-9]+]] = extractvalue { ptr, ptr } [[METHOD_PAIR]], 1
// CHECK-NEXT: [[CALL_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[METHOD_PAIR]], 0
// CHECK-NEXT: call void [[CALL_CODE]](ptr [[CALL_RECEIVER]])
// CHECK: [[K_VALUE:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 32)
// CHECK: [[C_ADVANCE:%[0-9]+]] = getelementptr [4 x i64], ptr [[K_VALUE]], i64 1
// CHECK-NEXT: call void @"{{.*}}PrintPointer"(ptr [[C_ADVANCE]])
// CHECK: [[METHOD_ADVANCE:%[0-9]+]] = getelementptr [4 x i64], ptr [[K_VALUE]], i64 1
// CHECK-NEXT: call void @"{{.*}}PrintPointer"(ptr [[METHOD_ADVANCE]])
// CHECK: {{^_llgo_[0-9]+:}}
// CHECK-NEXT: call void @"{{.*}}PanicTypeAssert"(ptr null, ptr [[ANY_TYPE]], ptr @"_llgo_main.T[string,int]")
// CHECK-LABEL: define linkonce void @"main.T[string,int].Info"(%"main.T[string,int]"
// CHECK: store %"main.T[string,int]" [[INFO_VALUE:%[0-9]+]], ptr [[INFO_ADDR:%[0-9]+]]
// CHECK: [[INFO_M_FIELD:%[0-9]+]] = getelementptr inbounds %"main.T[string,int]", ptr [[INFO_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[INFO_M:%[0-9]+]] = load %"{{.*}}String", ptr [[INFO_M_FIELD]]
// CHECK-NEXT: [[INFO_N_FIELD:%[0-9]+]] = getelementptr inbounds %"main.T[string,int]", ptr [[INFO_ADDR]], i32 0, i32 1
// CHECK-NEXT: [[INFO_N:%[0-9]+]] = load i64, ptr [[INFO_N_FIELD]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[INFO_M]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[INFO_N]])
// CHECK-LABEL: define linkonce void @"main.(*T[string,int]).Demo"(ptr
// CHECK: [[DEMO_M_FIELD:%[0-9]+]] = getelementptr inbounds %"main.T[string,int]", ptr [[DEMO_RECEIVER:%[0-9]+]], i32 0, i32 0
// CHECK-NEXT: [[DEMO_M:%[0-9]+]] = load %"{{.*}}String", ptr [[DEMO_M_FIELD]]
// CHECK-NEXT: [[DEMO_N_FIELD:%[0-9]+]] = getelementptr inbounds %"main.T[string,int]", ptr [[DEMO_RECEIVER]], i32 0, i32 1
// CHECK-NEXT: [[DEMO_N:%[0-9]+]] = load i64, ptr [[DEMO_N_FIELD]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[DEMO_M]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[DEMO_N]])
// CHECK-LABEL: define linkonce void @"main.(*T[string,int]).Info"(ptr
// CHECK: [[PTR_INFO_NIL:%[0-9]+]] = icmp eq ptr [[PTR_INFO_RECEIVER:%[0-9]+]], null
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[PTR_INFO_NIL]],{{.*}})
// CHECK-NEXT: [[PTR_INFO_VALUE:%[0-9]+]] = load %"main.T[string,int]", ptr [[PTR_INFO_RECEIVER]]
// CHECK-NEXT: call void @"main.T[string,int].Info"(%"main.T[string,int]" [[PTR_INFO_VALUE]])

type T[M, N any] struct {
	m M
	n N
}

func (t *T[M, N]) Demo() {
	println(t.m, t.n)
}

func (t T[M, N]) Info() {
	println(t.m, t.n)
}

type I interface {
	Demo()
}

type K[N any] [4]N

//llgo:link (*K).Advance llgo.advance
func (t *K[N]) Advance(n int) *K[N] {
	return nil
}

func main() {
	var a any = T[string, int]{"a", 1}
	println(a.(T[string, int]).m)
	var i I = &T[string, int]{"hello", 100}
	i.Demo()

	k := &K[int]{1, 2, 3, 4}
	println(c.Advance(k, 1))
	println(k.Advance(1))
}

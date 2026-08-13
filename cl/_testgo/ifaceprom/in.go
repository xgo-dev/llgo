// LITTEST
package main

// Test of promotion of methods of an interface embedded within a
// struct.  In particular, this test exercises that the correct
// method is called.

type I interface {
	one() int
	two() string
}

type S struct {
	I
}

type impl struct{}

func (impl) one() int {
	return 1
}

func (impl) two() string {
	return "two"
}

func main() {
	var s S
	s.I = impl{}
	if one := s.I.one(); one != 1 {
		panic(one)
	}
	if one := s.one(); one != 1 {
		panic(one)
	}
	closOne := s.I.one
	if one := closOne(); one != 1 {
		panic(one)
	}
	closOne = s.one
	if one := closOne(); one != 1 {
		panic(one)
	}

	if two := s.I.two(); two != "two" {
		panic(two)
	}
	if two := s.two(); two != "two" {
		panic(two)
	}
	closTwo := s.I.two
	if two := closTwo(); two != "two" {
		panic(two)
	}
	closTwo = s.two
	if two := closTwo(); two != "two" {
		panic(two)
	}

	println("pass")
}

// CHECK-LABEL: define i64 @main.S.one(%main.S %0){{.*}} {
// CHECK: [[S_ONE_ADDR:%[0-9]+]] = alloca %main.S
// CHECK: store %main.S %0, ptr [[S_ONE_ADDR]]
// CHECK-NEXT: [[S_ONE_FIELD:%[0-9]+]] = getelementptr inbounds %main.S, ptr [[S_ONE_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[S_ONE_IFACE:%[0-9]+]] = load %"{{.*}}iface", ptr [[S_ONE_FIELD]]
// CHECK: [[S_ONE_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[S_ONE_IFACE]])
// CHECK: [[S_ONE_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[S_ONE_IFACE]], 0
// CHECK: [[S_ONE_SLOT:%.*]] = getelementptr ptr, ptr [[S_ONE_ITAB]], i64 3
// CHECK: [[S_ONE_CODE:%.*]] = load ptr, ptr [[S_ONE_SLOT]]
// CHECK: [[S_ONE_PAIR_0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[S_ONE_CODE]], 0
// CHECK: [[S_ONE_PAIR:%.*]] = insertvalue { ptr, ptr } [[S_ONE_PAIR_0]], ptr [[S_ONE_DATA]], 1
// CHECK: [[S_ONE_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[S_ONE_PAIR]], 1
// CHECK: [[S_ONE_CALL_CODE:%.*]] = extractvalue { ptr, ptr } [[S_ONE_PAIR]], 0
// CHECK: [[S_ONE_RESULT:%.*]] = call i64 [[S_ONE_CALL_CODE]](ptr [[S_ONE_CALL_DATA]])
// CHECK: ret i64 [[S_ONE_RESULT]]

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.S.two(%main.S %0){{.*}} {
// CHECK: [[S_TWO_ADDR:%[0-9]+]] = alloca %main.S
// CHECK: store %main.S %0, ptr [[S_TWO_ADDR]]
// CHECK-NEXT: [[S_TWO_FIELD:%[0-9]+]] = getelementptr inbounds %main.S, ptr [[S_TWO_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[S_TWO_IFACE:%[0-9]+]] = load %"{{.*}}iface", ptr [[S_TWO_FIELD]]
// CHECK: [[S_TWO_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[S_TWO_IFACE]])
// CHECK: [[S_TWO_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[S_TWO_IFACE]], 0
// CHECK: [[S_TWO_SLOT:%.*]] = getelementptr ptr, ptr [[S_TWO_ITAB]], i64 4
// CHECK: [[S_TWO_CODE:%.*]] = load ptr, ptr [[S_TWO_SLOT]]
// CHECK: [[S_TWO_PAIR_0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[S_TWO_CODE]], 0
// CHECK: [[S_TWO_PAIR:%.*]] = insertvalue { ptr, ptr } [[S_TWO_PAIR_0]], ptr [[S_TWO_DATA]], 1
// CHECK: [[S_TWO_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[S_TWO_PAIR]], 1
// CHECK: [[S_TWO_CALL_CODE:%.*]] = extractvalue { ptr, ptr } [[S_TWO_PAIR]], 0
// CHECK: [[S_TWO_RESULT:%.*]] = call %"{{.*}}String" [[S_TWO_CALL_CODE]](ptr [[S_TWO_CALL_DATA]])
// CHECK: ret %"{{.*}}String" [[S_TWO_RESULT]]

// CHECK-LABEL: define i64 @main.impl.one(%main.impl %0){{.*}} {
// CHECK: ret i64 1

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.impl.two(%main.impl %0){{.*}} {
// CHECK: ret %"{{.*}}String" { ptr @{{.*}}, i64 3 }

// CHECK-LABEL: define i64 @"main.(*impl).one"(ptr %0){{.*}} {
// CHECK: [[IMPL_ONE_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[IMPL_ONE_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" {{.*}})
// CHECK: [[IMPL_ONE_RESULT:%.*]] = call i64 @main.impl.one(%main.impl zeroinitializer)
// CHECK: ret i64 [[IMPL_ONE_RESULT]]

// CHECK-LABEL: define %"{{.*}}String" @"main.(*impl).two"(ptr %0){{.*}} {
// CHECK: [[IMPL_TWO_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[IMPL_TWO_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" {{.*}})
// CHECK: [[IMPL_TWO_RESULT:%.*]] = call %"{{.*}}String" @main.impl.two(%main.impl zeroinitializer)
// CHECK: ret %"{{.*}}String" [[IMPL_TWO_RESULT]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[IMPL_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK: store %main.impl zeroinitializer, ptr [[IMPL_DATA]]
// CHECK: [[IMPL_ITAB:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceprom.iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.impl)
// CHECK: [[MAIN_IFACE_0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[IMPL_ITAB]], 0
// CHECK: [[MAIN_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[MAIN_IFACE_0]], ptr [[IMPL_DATA]], 1
// CHECK: store %"{{.*}}iface" [[MAIN_IFACE]], ptr [[S_IFACE_FIELD:%.*]]
// Direct s.I.one uses slot 3 and compares that call result with 1.
// CHECK: [[DIRECT_ONE_IFACE:%.*]] = load %"{{.*}}iface", ptr %{{.*}}
// CHECK: [[DIRECT_ONE_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[DIRECT_ONE_IFACE]])
// CHECK: [[DIRECT_ONE_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[DIRECT_ONE_IFACE]], 0
// CHECK: [[DIRECT_ONE_SLOT:%.*]] = getelementptr ptr, ptr [[DIRECT_ONE_ITAB]], i64 3
// CHECK: [[DIRECT_ONE_CODE:%.*]] = load ptr, ptr [[DIRECT_ONE_SLOT]]
// CHECK: [[DIRECT_ONE_PAIR_0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[DIRECT_ONE_CODE]], 0
// CHECK: [[DIRECT_ONE_PAIR:%.*]] = insertvalue { ptr, ptr } [[DIRECT_ONE_PAIR_0]], ptr [[DIRECT_ONE_DATA]], 1
// CHECK: [[DIRECT_ONE_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[DIRECT_ONE_PAIR]], 1
// CHECK: [[DIRECT_ONE_CALL_CODE:%.*]] = extractvalue { ptr, ptr } [[DIRECT_ONE_PAIR]], 0
// CHECK: [[DIRECT_ONE_RESULT:%.*]] = call i64 [[DIRECT_ONE_CALL_CODE]](ptr [[DIRECT_ONE_CALL_DATA]])
// CHECK: [[DIRECT_ONE_BAD:%.*]] = icmp ne i64 [[DIRECT_ONE_RESULT]], 1
// CHECK: br i1 [[DIRECT_ONE_BAD]], label %{{.*}}, label %{{.*}}
// Promoted s.one extracts the embedded interface and still uses slot 3.
// CHECK: [[PROMOTED_ONE_S:%.*]] = load %main.S, ptr %{{.*}}
// CHECK: [[PROMOTED_ONE_IFACE:%.*]] = extractvalue %main.S [[PROMOTED_ONE_S]], 0
// CHECK: [[PROMOTED_ONE_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[PROMOTED_ONE_IFACE]])
// CHECK: [[PROMOTED_ONE_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[PROMOTED_ONE_IFACE]], 0
// CHECK: [[PROMOTED_ONE_SLOT:%.*]] = getelementptr ptr, ptr [[PROMOTED_ONE_ITAB]], i64 3
// CHECK: [[PROMOTED_ONE_CODE:%.*]] = load ptr, ptr [[PROMOTED_ONE_SLOT]]
// CHECK: [[PROMOTED_ONE_PAIR_0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[PROMOTED_ONE_CODE]], 0
// CHECK: [[PROMOTED_ONE_PAIR:%.*]] = insertvalue { ptr, ptr } [[PROMOTED_ONE_PAIR_0]], ptr [[PROMOTED_ONE_DATA]], 1
// CHECK: [[PROMOTED_ONE_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_ONE_PAIR]], 1
// CHECK: [[PROMOTED_ONE_CALL_CODE:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_ONE_PAIR]], 0
// CHECK: [[PROMOTED_ONE_RESULT:%.*]] = call i64 [[PROMOTED_ONE_CALL_CODE]](ptr [[PROMOTED_ONE_CALL_DATA]])
// CHECK: [[PROMOTED_ONE_BAD:%.*]] = icmp ne i64 [[PROMOTED_ONE_RESULT]], 1
// CHECK: br i1 [[PROMOTED_ONE_BAD]], label %{{.*}}, label %{{.*}}
// Both one method values first validate and retain their originating interfaces.
// CHECK: [[ONE_CLOSURE_IFACE:%.*]] = load %"{{.*}}iface", ptr %{{.*}}
// CHECK: [[ONE_CLOSURE_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[ONE_CLOSURE_IFACE]])
// CHECK: [[ONE_CLOSURE_OK:%.*]] = icmp ne ptr [[ONE_CLOSURE_TYPE]], null
// CHECK: br i1 [[ONE_CLOSURE_OK]], label %{{.*}}, label %{{.*}}
// CHECK: [[PROMOTED_ONE_CLOSURE_S:%.*]] = load %main.S, ptr %{{.*}}
// CHECK: [[PROMOTED_ONE_CLOSURE_IFACE:%.*]] = extractvalue %main.S [[PROMOTED_ONE_CLOSURE_S]], 0
// CHECK: [[PROMOTED_ONE_CLOSURE_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[PROMOTED_ONE_CLOSURE_IFACE]])
// CHECK: [[PROMOTED_ONE_CLOSURE_OK:%.*]] = icmp ne ptr [[PROMOTED_ONE_CLOSURE_TYPE]], null
// CHECK: br i1 [[PROMOTED_ONE_CLOSURE_OK]], label %{{.*}}, label %{{.*}}
// Direct and promoted two calls both use slot 4 and compare the returned string with "two".
// CHECK: [[DIRECT_TWO_IFACE:%.*]] = load %"{{.*}}iface", ptr %{{.*}}
// CHECK: [[DIRECT_TWO_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[DIRECT_TWO_IFACE]])
// CHECK: [[DIRECT_TWO_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[DIRECT_TWO_IFACE]], 0
// CHECK: [[DIRECT_TWO_SLOT:%.*]] = getelementptr ptr, ptr [[DIRECT_TWO_ITAB]], i64 4
// CHECK: [[DIRECT_TWO_CODE:%.*]] = load ptr, ptr [[DIRECT_TWO_SLOT]]
// CHECK: [[DIRECT_TWO_PAIR_0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[DIRECT_TWO_CODE]], 0
// CHECK: [[DIRECT_TWO_PAIR:%.*]] = insertvalue { ptr, ptr } [[DIRECT_TWO_PAIR_0]], ptr [[DIRECT_TWO_DATA]], 1
// CHECK: [[DIRECT_TWO_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[DIRECT_TWO_PAIR]], 1
// CHECK: [[DIRECT_TWO_CALL_CODE:%.*]] = extractvalue { ptr, ptr } [[DIRECT_TWO_PAIR]], 0
// CHECK: [[DIRECT_TWO_RESULT:%.*]] = call %"{{.*}}String" [[DIRECT_TWO_CALL_CODE]](ptr [[DIRECT_TWO_CALL_DATA]])
// CHECK: [[DIRECT_TWO_EQ:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}String" [[DIRECT_TWO_RESULT]], %"{{.*}}String" { ptr @{{.*}}, i64 3 })
// CHECK: [[DIRECT_TWO_BAD:%.*]] = xor i1 [[DIRECT_TWO_EQ]], true
// CHECK: br i1 [[DIRECT_TWO_BAD]], label %{{.*}}, label %{{.*}}
// CHECK: [[PROMOTED_TWO_S:%.*]] = load %main.S, ptr %{{.*}}
// CHECK: [[PROMOTED_TWO_IFACE:%.*]] = extractvalue %main.S [[PROMOTED_TWO_S]], 0
// CHECK: [[PROMOTED_TWO_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[PROMOTED_TWO_IFACE]])
// CHECK: [[PROMOTED_TWO_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[PROMOTED_TWO_IFACE]], 0
// CHECK: [[PROMOTED_TWO_SLOT:%.*]] = getelementptr ptr, ptr [[PROMOTED_TWO_ITAB]], i64 4
// CHECK: [[PROMOTED_TWO_CODE:%.*]] = load ptr, ptr [[PROMOTED_TWO_SLOT]]
// CHECK: [[PROMOTED_TWO_PAIR_0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[PROMOTED_TWO_CODE]], 0
// CHECK: [[PROMOTED_TWO_PAIR:%.*]] = insertvalue { ptr, ptr } [[PROMOTED_TWO_PAIR_0]], ptr [[PROMOTED_TWO_DATA]], 1
// CHECK: [[PROMOTED_TWO_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_TWO_PAIR]], 1
// CHECK: [[PROMOTED_TWO_CALL_CODE:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_TWO_PAIR]], 0
// CHECK: [[PROMOTED_TWO_RESULT:%.*]] = call %"{{.*}}String" [[PROMOTED_TWO_CALL_CODE]](ptr [[PROMOTED_TWO_CALL_DATA]])
// CHECK: [[PROMOTED_TWO_EQ:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}String" [[PROMOTED_TWO_RESULT]], %"{{.*}}String" { ptr @{{.*}}, i64 3 })
// CHECK: [[PROMOTED_TWO_BAD:%.*]] = xor i1 [[PROMOTED_TWO_EQ]], true
// CHECK: br i1 [[PROMOTED_TWO_BAD]], label %{{.*}}, label %{{.*}}
// CHECK: [[TWO_CLOSURE_IFACE:%.*]] = load %"{{.*}}iface", ptr %{{.*}}
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[TWO_CLOSURE_IFACE]])
// CHECK: [[PROMOTED_TWO_CLOSURE_S:%.*]] = load %main.S, ptr %{{.*}}
// CHECK: [[PROMOTED_TWO_CLOSURE_IFACE:%.*]] = extractvalue %main.S [[PROMOTED_TWO_CLOSURE_S]], 0
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[PROMOTED_TWO_CLOSURE_IFACE]])
// The four late closure paths store those same interfaces into their bound environments.
// CHECK: store %"{{.*}}iface" [[ONE_CLOSURE_IFACE]], ptr %{{.*}}
// CHECK: [[ONE_BOUND:%.*]] = insertvalue { ptr, ptr } { ptr @"main.I.one$bound", ptr undef }, ptr [[ONE_BOUND_ENV:%.*]], 1
// CHECK: [[ONE_BOUND_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[ONE_BOUND]], 1
// CHECK: [[ONE_BOUND_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[ONE_BOUND]], 0
// CHECK: [[ONE_BOUND_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[ONE_BOUND_CODE_RAW]])
// CHECK: [[ONE_BOUND_RESULT:%.*]] = call i64 [[ONE_BOUND_CODE]](ptr {{(nest|swiftself)}} [[ONE_BOUND_CALL_ENV]])
// CHECK: [[ONE_BOUND_BAD:%.*]] = icmp ne i64 [[ONE_BOUND_RESULT]], 1
// CHECK: store %"{{.*}}iface" [[PROMOTED_ONE_CLOSURE_IFACE]], ptr %{{.*}}
// CHECK: [[PROMOTED_ONE_BOUND:%.*]] = insertvalue { ptr, ptr } { ptr @"main.I.one$bound", ptr undef }, ptr %{{.*}}, 1
// CHECK: [[PROMOTED_ONE_BOUND_ENV:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_ONE_BOUND]], 1
// CHECK: [[PROMOTED_ONE_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_ONE_BOUND]], 0
// CHECK: [[PROMOTED_ONE_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[PROMOTED_ONE_CODE_RAW]])
// CHECK: [[PROMOTED_ONE_BOUND_RESULT:%.*]] = call i64 [[PROMOTED_ONE_CODE]](ptr {{(nest|swiftself)}} [[PROMOTED_ONE_BOUND_ENV]])
// CHECK: [[PROMOTED_ONE_BOUND_BAD:%.*]] = icmp ne i64 [[PROMOTED_ONE_BOUND_RESULT]], 1
// CHECK: store %"{{.*}}iface" [[TWO_CLOSURE_IFACE]], ptr %{{.*}}
// CHECK: [[TWO_BOUND:%.*]] = insertvalue { ptr, ptr } { ptr @"main.I.two$bound", ptr undef }, ptr %{{.*}}, 1
// CHECK: [[TWO_BOUND_ENV:%.*]] = extractvalue { ptr, ptr } [[TWO_BOUND]], 1
// CHECK: [[TWO_BOUND_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[TWO_BOUND]], 0
// CHECK: [[TWO_BOUND_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[TWO_BOUND_CODE_RAW]])
// CHECK: [[TWO_BOUND_RESULT:%.*]] = call %"{{.*}}String" [[TWO_BOUND_CODE]](ptr {{(nest|swiftself)}} [[TWO_BOUND_ENV]])
// CHECK: [[TWO_BOUND_EQ:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}String" [[TWO_BOUND_RESULT]], %"{{.*}}String" { ptr @{{.*}}, i64 3 })
// CHECK: store %"{{.*}}iface" [[PROMOTED_TWO_CLOSURE_IFACE]], ptr %{{.*}}
// CHECK: [[PROMOTED_TWO_BOUND:%.*]] = insertvalue { ptr, ptr } { ptr @"main.I.two$bound", ptr undef }, ptr %{{.*}}, 1
// CHECK: [[PROMOTED_TWO_BOUND_ENV:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_TWO_BOUND]], 1
// CHECK: [[PROMOTED_TWO_BOUND_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[PROMOTED_TWO_BOUND]], 0
// CHECK: [[PROMOTED_TWO_BOUND_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[PROMOTED_TWO_BOUND_CODE_RAW]])
// CHECK: [[PROMOTED_TWO_BOUND_RESULT:%.*]] = call %"{{.*}}String" [[PROMOTED_TWO_BOUND_CODE]](ptr {{(nest|swiftself)}} [[PROMOTED_TWO_BOUND_ENV]])
// CHECK: [[PROMOTED_TWO_BOUND_EQ:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}String" [[PROMOTED_TWO_BOUND_RESULT]], %"{{.*}}String" { ptr @{{.*}}, i64 3 })

// CHECK-LABEL: define i64 @"main.I.one$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[BOUND_ONE_ENV:%[0-9]+]] = load { %"{{.*}}iface" }, ptr %0
// CHECK: [[BOUND_ONE_IFACE:%.*]] = extractvalue { %"{{.*}}iface" } [[BOUND_ONE_ENV]], 0
// CHECK: [[BOUND_ONE_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[BOUND_ONE_IFACE]])
// CHECK: [[BOUND_ONE_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[BOUND_ONE_IFACE]], 0
// CHECK: [[BOUND_ONE_SLOT:%.*]] = getelementptr ptr, ptr [[BOUND_ONE_ITAB]], i64 3
// CHECK: [[BOUND_ONE_CODE:%.*]] = load ptr, ptr [[BOUND_ONE_SLOT]]
// CHECK-NEXT: [[BOUND_ONE_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[BOUND_ONE_CODE]], 0
// CHECK-NEXT: [[BOUND_ONE_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[BOUND_ONE_PAIR0]], ptr [[BOUND_ONE_DATA]], 1
// CHECK-NEXT: [[BOUND_ONE_RECOVER_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[BOUND_ONE_PAIR]], 0
// CHECK-NEXT: [[BOUND_ONE_RECOVER:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrameAlias"(ptr @"main.I.one$bound", ptr [[BOUND_ONE_RECOVER_CODE]])
// CHECK-NEXT: [[BOUND_ONE_CALL_DATA:%[0-9]+]] = extractvalue { ptr, ptr } [[BOUND_ONE_PAIR]], 1
// CHECK-NEXT: [[BOUND_ONE_CALL_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[BOUND_ONE_PAIR]], 0
// CHECK-NEXT: [[BOUND_ONE_RESULT:%[0-9]+]] = call i64 [[BOUND_ONE_CALL_CODE]](ptr [[BOUND_ONE_CALL_DATA]])
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrameAlias"(ptr [[BOUND_ONE_RECOVER]])
// CHECK: ret i64 [[BOUND_ONE_RESULT]]

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"main.I.two$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[BOUND_TWO_ENV:%[0-9]+]] = load { %"{{.*}}iface" }, ptr %0
// CHECK: [[BOUND_TWO_IFACE:%.*]] = extractvalue { %"{{.*}}iface" } [[BOUND_TWO_ENV]], 0
// CHECK: [[BOUND_TWO_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[BOUND_TWO_IFACE]])
// CHECK: [[BOUND_TWO_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[BOUND_TWO_IFACE]], 0
// CHECK: [[BOUND_TWO_SLOT:%.*]] = getelementptr ptr, ptr [[BOUND_TWO_ITAB]], i64 4
// CHECK: [[BOUND_TWO_CODE:%.*]] = load ptr, ptr [[BOUND_TWO_SLOT]]
// CHECK-NEXT: [[BOUND_TWO_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[BOUND_TWO_CODE]], 0
// CHECK-NEXT: [[BOUND_TWO_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[BOUND_TWO_PAIR0]], ptr [[BOUND_TWO_DATA]], 1
// CHECK-NEXT: [[BOUND_TWO_RECOVER_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[BOUND_TWO_PAIR]], 0
// CHECK-NEXT: [[BOUND_TWO_RECOVER:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrameAlias"(ptr @"main.I.two$bound", ptr [[BOUND_TWO_RECOVER_CODE]])
// CHECK-NEXT: [[BOUND_TWO_CALL_DATA:%[0-9]+]] = extractvalue { ptr, ptr } [[BOUND_TWO_PAIR]], 1
// CHECK-NEXT: [[BOUND_TWO_CALL_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[BOUND_TWO_PAIR]], 0
// CHECK-NEXT: [[BOUND_TWO_RESULT:%[0-9]+]] = call %"{{.*}}String" [[BOUND_TWO_CALL_CODE]](ptr [[BOUND_TWO_CALL_DATA]])
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrameAlias"(ptr [[BOUND_TWO_RECOVER]])
// CHECK: ret %"{{.*}}String" [[BOUND_TWO_RESULT]]

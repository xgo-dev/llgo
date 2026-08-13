// LITTEST
package main

// Method expressions use signature-specific thunks. In particular, the two
// M methods below must not become interchangeable closure types.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: store { ptr, ptr } { ptr @"main.(*outer).M$thunk", ptr null }, ptr [[VOID_BOX:%[0-9]+]]
// CHECK-NEXT: [[VOID_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr [[VOID_CLOSURE_TYPE:@"_llgo_closure\$[^"]+"]], ptr undef }, ptr [[VOID_BOX]], 1
// CHECK: store { ptr, ptr } { ptr @"main.(*InnerInt).M$thunk", ptr null }, ptr [[INT_BOX:%[0-9]+]]
// CHECK-NEXT: [[INT_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr [[INT_CLOSURE_TYPE:@"_llgo_closure\$[^"]+"]], ptr undef }, ptr [[INT_BOX]], 1
// CHECK-NEXT: [[VOID_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[VOID_EFACE]], 0
// CHECK-NEXT: [[VOID_MATCH:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr [[VOID_CLOSURE_TYPE]], ptr [[VOID_TYPE]])
// CHECK-NEXT: br i1 [[VOID_MATCH]], label %{{.*}}, label %{{.*}}
// CHECK: [[VOID_RESULT:%[0-9]+]] = phi { { ptr, ptr }, i1 } [ %{{[0-9]+}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK-NEXT: [[VOID_FN:%[0-9]+]] = extractvalue { { ptr, ptr }, i1 } [[VOID_RESULT]], 0
// CHECK-NEXT: [[VOID_OK:%[0-9]+]] = extractvalue { { ptr, ptr }, i1 } [[VOID_RESULT]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 [[VOID_OK]])
// CHECK: [[INT_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[INT_EFACE]], 0
// CHECK-NEXT: [[INT_MATCH:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr [[VOID_CLOSURE_TYPE]], ptr [[INT_TYPE]])
// CHECK-NEXT: br i1 [[INT_MATCH]], label %{{.*}}, label %{{.*}}
// CHECK: [[INT_RESULT:%[0-9]+]] = phi { { ptr, ptr }, i1 } [ %{{[0-9]+}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK-NEXT: [[ASSERTED_FN:%[0-9]+]] = extractvalue { { ptr, ptr }, i1 } [[INT_RESULT]], 0
// CHECK-NEXT: [[INT_OK:%[0-9]+]] = extractvalue { { ptr, ptr }, i1 } [[INT_RESULT]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 [[INT_OK]])
// CHECK: br i1 [[INT_OK]], label %{{.*}}, label %{{.*}}
// CHECK-LABEL: define void @"main.(*outer).M$thunk"(ptr %0){{.*}} {
// CHECK: call void @"main.(*outer).M"(ptr %0)
// CHECK-NEXT: ret void
// CHECK-LABEL: define i64 @"main.(*InnerInt).M$thunk"(ptr %0){{.*}} {
// CHECK: [[THUNK_RESULT:%[0-9]+]] = call i64 @"main.(*InnerInt).M"(ptr %0)
// CHECK-NEXT: ret i64 [[THUNK_RESULT]]

type inner struct {
	x int
}

type outer struct {
	y int
	inner
}

func (*inner) M() {}

type InnerInt struct {
	X int
}

type OuterInt struct {
	Y int
	InnerInt
}

func (i *InnerInt) M() int {
	return i.X
}

func main() {
	var v1 any = (*outer).M
	var v2 any = (*InnerInt).M
	f1, ok := v1.(func(*outer))
	println(f1, ok)
	f2, ok := v2.(func(*outer))
	println(f2, ok)
	if ok {
		panic("type assertion should have failed but succeeded")
	}
}

func (m *outer) M() {}

// LITTEST darwin/arm64 linux/amd64 windows/arm64 windows/amd64
// Scope: common
package main

// This is the focused regression owner for e2fd92233. Range-over-function SSA
// creates synthetic closure free variables whose names are empty. More than one
// such variable must become distinct blank fields in the physical environment.
type Seq func(yield func(int) bool)

func values(yield func(int) bool) {
	if !yield(1) {
		return
	}
	yield(2)
}

// CHECK-LABEL: define i64 @main.sum(ptr %0){{.*}} {
// CHECK: [[RANGE_ALLOCATION:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK-NEXT: [[RANGE_CLOSURE:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr, ptr } }, ptr [[RANGE_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[ENV:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr, ptr } }, ptr [[RANGE_ALLOCATION]], i32 0, i32 1
// CHECK-NEXT: [[RANGE_CAPTURE_0_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[RANGE_CAPTURE_0:%[0-9]+]], ptr [[RANGE_CAPTURE_0_SLOT]]
// CHECK-NEXT: [[RANGE_CAPTURE_1_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[ENV]], i32 0, i32 1
// CHECK-NEXT: store ptr [[RANGE_CAPTURE_1:%[0-9]+]], ptr [[RANGE_CAPTURE_1_SLOT]]
// CHECK: [[RANGE_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[RANGE_CLOSURE]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.sum$1", ptr [[RANGE_CODE_SLOT]]
// CHECK-NEXT: [[RANGE_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[RANGE_CLOSURE]], i32 0, i32 1
// CHECK-NEXT: store ptr [[ENV]], ptr [[RANGE_ENV_SLOT]]
// CHECK: [[SEQ_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %0
// CHECK-NEXT: [[SEQ_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[SEQ_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[SEQ_ENV:%[0-9]+]] = load ptr, ptr [[SEQ_ENV_FIELD]]
// CHECK: [[SEQ_CODE_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %0
// CHECK-NEXT: [[SEQ_CODE_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[SEQ_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[SEQ_CODE:%[0-9]+]] = load ptr, ptr [[SEQ_CODE_FIELD]]
// ARM64: call void %{{.*}}(ptr swiftself [[SEQ_ENV]], ptr [[RANGE_CLOSURE]])
// AMD64: call void %{{.*}}(ptr nest [[SEQ_ENV]], ptr [[RANGE_CLOSURE]])
func sum(seq Seq) (total int) {
	for value := range seq {
		total += value
	}
	return
}

// The generated callback reads both the synthetic range state and the named
// result from the same environment, which is the original duplicate-empty-name
// failure shape.
// ARM64-LABEL: define i1 @"main.sum$1"(ptr swiftself %0, i64 %1){{.*}} {
// AMD64-LABEL: define i1 @"main.sum$1"(ptr nest %0, i64 %1){{.*}} {
// CHECK: [[CAPTURE:%[0-9]+]] = load { ptr, ptr }, ptr %0
// CHECK: [[RANGE_STATE_PTR:%[0-9]+]] = extractvalue { ptr, ptr } [[CAPTURE]], 0
// CHECK: [[RANGE_TOTAL_PTR:%[0-9]+]] = extractvalue { ptr, ptr } [[CAPTURE]], 1
// CHECK: [[OLD_TOTAL:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[NEW_TOTAL:%[0-9]+]] = add i64 {{%[0-9]+}}, %1
// CHECK: store i64 [[NEW_TOTAL]], ptr %{{[0-9]+}}
// CHECK: ret i1 true

func main() {
	println(sum(values))
}

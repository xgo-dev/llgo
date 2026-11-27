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

// CHECK-LABEL: define i64 @main.sum(%main.Seq %0){{.*}} {
// CHECK: [[ENV:%[0-9]+]] = call ptr @"{{.*}}AllocU"
// CHECK: [[RANGE_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.sum$1", ptr undef }, ptr [[ENV]], 1
// CHECK: [[SEQ_ENV:%[0-9]+]] = extractvalue %main.Seq %0, 1
// CHECK: [[SEQ_CODE:%[0-9]+]] = extractvalue %main.Seq %0, 0
// ARM64: call void %{{.*}}(ptr swiftself [[SEQ_ENV]], { ptr, ptr } [[RANGE_CLOSURE]])
// AMD64: call void %{{.*}}(ptr nest [[SEQ_ENV]], { ptr, ptr } [[RANGE_CLOSURE]])
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
// CHECK: [[CAPTURE:%[0-9]+]] = load {{.*}}, ptr %0
// CHECK: [[OLD_TOTAL:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK: [[NEW_TOTAL:%[0-9]+]] = add i64 {{%[0-9]+}}, %1
// CHECK: store i64 [[NEW_TOTAL]], ptr %{{[0-9]+}}
// CHECK: ret i1 true

func main() {
	println(sum(values))
}

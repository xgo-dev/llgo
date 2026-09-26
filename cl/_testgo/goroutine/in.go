// LITTEST darwin/arm64 linux/amd64
package main

// Goroutine arguments live in a runtime-owned root until the entry call has
// returned. Generated wrappers pass the closure environment through the hidden
// ABI without releasing that root early.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$1"
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$2"
// ARM64-LABEL: define void @"main.main$1"(ptr swiftself
// AMD64-LABEL: define void @"main.main$1"(ptr nest
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[GOROUTINE_TEXT:%[0-9]+]])
// CHECK: [[GOROUTINE_ENV:%[0-9]+]] = load { ptr }, ptr %{{[0-9]+}}
// CHECK-NEXT: [[GOROUTINE_DONE:%[0-9]+]] = extractvalue { ptr } [[GOROUTINE_ENV]], 0
// CHECK-NEXT: store i8 1, ptr [[GOROUTINE_DONE]]
// CHECK-LABEL: define ptr @"main._llgo_routine$1"(ptr
// CHECK: [[ROUTINE1_ARGS:%[0-9]+]] = load { %"{{.*}}String" }, ptr [[ROUTINE1_ROOT:%[0-9]+]]
// CHECK-NEXT: [[ROUTINE1_TEXT:%[0-9]+]] = extractvalue { %"{{.*}}String" } [[ROUTINE1_ARGS]], 0
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[ROUTINE1_TEXT]])
// CHECK: ret ptr null
// CHECK-LABEL: define ptr @"main._llgo_routine$2"(ptr
// CHECK: [[ROUTINE2_ARGS:%[0-9]+]] = load { { ptr, ptr }, %"{{.*}}String" }, ptr [[ROUTINE2_ROOT:%[0-9]+]]
// CHECK-NEXT: [[ROUTINE2_CLOSURE:%[0-9]+]] = extractvalue { { ptr, ptr }, %"{{.*}}String" } [[ROUTINE2_ARGS]], 0
// CHECK-NEXT: [[ROUTINE2_TEXT:%[0-9]+]] = extractvalue { { ptr, ptr }, %"{{.*}}String" } [[ROUTINE2_ARGS]], 1
// CHECK-NEXT: [[ROUTINE2_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[ROUTINE2_CLOSURE]], 1
// CHECK-NEXT: [[ROUTINE2_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[ROUTINE2_CLOSURE]], 0
// ARM64: call void %{{.*}}(ptr swiftself [[ROUTINE2_ENV]], %"{{.*}}String" [[ROUTINE2_TEXT]])
// AMD64: call void %{{.*}}(ptr nest [[ROUTINE2_ENV]], %"{{.*}}String" [[ROUTINE2_TEXT]])
// CHECK-NEXT: ret ptr null

func main() {
	done := false
	go println("hello")
	go func(s string) {
		println(s)
		done = true
	}("Hello, goroutine")
	for !done {
		print(".")
	}
}

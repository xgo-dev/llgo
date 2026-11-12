// LITTEST darwin/arm64 linux/amd64
package main

type Void = [0]byte
type Future[T any] func() T

type IO[T any] func() Future[T]

// Named generic function types keep their identity while their values use one
// descriptor pointer, including a function whose result is another function.
// CHECK-DAG: [[WRITE_IO:@"_llgo_funcval\$[0-9a-f]+"]] = linkonce_odr constant { ptr, ptr } { ptr @"main.WriteFile$1", ptr null }
// CHECK-DAG: [[WRITE_FUTURE:@"_llgo_funcval\$[0-9a-f]+"]] = linkonce_odr constant { ptr, ptr } { ptr @"main.WriteFile$1$1", ptr null }
// CHECK-DAG: [[MAIN_IO:@"_llgo_funcval\$[0-9a-f]+"]] = linkonce_odr constant { ptr, ptr } { ptr @"main.main$1", ptr null }
// CHECK-LABEL: define ptr @main.WriteFile(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK: ret ptr [[WRITE_IO]]

func WriteFile(fileName string) IO[error] {

	// CHECK-LABEL: define ptr @"main.WriteFile$1"(){{.*}} {
	// CHECK: ret ptr [[WRITE_FUTURE]]

	return func() Future[error] {

		// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.WriteFile$1$1"(){{.*}} {

		return func() error {
			return nil
		}
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call [0 x i8] @"main.RunIO{{\[\[0\]uint8\]}}"(ptr [[MAIN_IO]])
// CHECK-NEXT: ret void

func main() {
	RunIO[Void](func() Future[Void] {

		return func() (ret Void) {
			return
		}
	})
}

func RunIO[T any](call IO[T]) T {
	return call()()
}

// CHECK-LABEL: define linkonce [0 x i8] @"main.RunIO{{\[\[0\]uint8\]}}"(ptr %0){{.*}} {
// CHECK: [[IO_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %0
// CHECK-NEXT: [[IO_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[IO_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[IO_ENV:%[0-9]+]] = load ptr, ptr [[IO_ENV_SLOT]]
// CHECK: [[IO_CODE_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %0
// CHECK-NEXT: [[IO_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[IO_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[IO_CODE:%[0-9]+]] = load ptr, ptr [[IO_CODE_SLOT]]
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[IO_CODE]])
// ARM64-NEXT: [[FUTURE:%[0-9]+]] = call ptr %__llgo_funcval_code(ptr swiftself [[IO_ENV]])
// AMD64-NEXT: [[FUTURE:%[0-9]+]] = call ptr %__llgo_funcval_code(ptr nest [[IO_ENV]])
// CHECK-NEXT: [[FUTURE_NIL:%[0-9]+]] = icmp eq ptr [[FUTURE]], null
// CHECK-NEXT: br i1 [[FUTURE_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[FUTURE_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[FUTURE]]
// CHECK-NEXT: [[FUTURE_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[FUTURE_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[FUTURE_ENV:%[0-9]+]] = load ptr, ptr [[FUTURE_ENV_SLOT]]
// CHECK: [[FUTURE_CODE_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[FUTURE]]
// CHECK-NEXT: [[FUTURE_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[FUTURE_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[FUTURE_CODE:%[0-9]+]] = load ptr, ptr [[FUTURE_CODE_SLOT]]
// CHECK: %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr [[FUTURE_CODE]])
// ARM64: [[IO_RESULT:%[0-9]+]] = call [0 x i8] %__llgo_funcval_code1(ptr swiftself [[FUTURE_ENV]])
// AMD64: [[IO_RESULT:%[0-9]+]] = call [0 x i8] %__llgo_funcval_code1(ptr nest [[FUTURE_ENV]])
// CHECK-NEXT: ret [0 x i8] [[IO_RESULT]]

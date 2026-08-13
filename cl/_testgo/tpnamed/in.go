// LITTEST
package main

type Void = [0]byte
type Future[T any] func() T

type IO[T any] func() Future[T]

// CHECK-LABEL: define %"main.IO[error]" @main.WriteFile(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK:   ret %"main.IO[error]" { ptr @"main.WriteFile$1", ptr null }

func WriteFile(fileName string) IO[error] {

	// CHECK-LABEL: define %"main.Future[error]" @"main.WriteFile$1"(){{.*}} {
	// CHECK:   ret %"main.Future[error]" { ptr @"main.WriteFile$1$1", ptr null }

	return func() Future[error] {

		// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.WriteFile$1$1"(){{.*}} {

		return func() error {
			return nil
		}
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call [0 x i8] @"main.RunIO{{\[\[0\]byte\]}}"(%"main.IO{{\[\[0\]byte\]}}" { ptr @"main.main$1", ptr null })
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

// CHECK-LABEL: define linkonce [0 x i8] @"main.RunIO{{\[\[0\]byte\]}}"(%"main.IO{{\[\[0\]byte\]}}" %0){{.*}} {
// CHECK: [[IO_ENV:%[0-9]+]] = extractvalue %"main.IO{{\[\[0\]byte\]}}" %0, 1
// CHECK-NEXT: [[IO_CODE:%[0-9]+]] = extractvalue %"main.IO{{\[\[0\]byte\]}}" %0, 0
// CHECK: [[FUTURE:%[0-9]+]] = call %"main.Future{{\[\[0\]byte\]}}" %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[IO_ENV]])
// CHECK-NEXT: [[FUTURE_ENV:%[0-9]+]] = extractvalue %"main.Future{{\[\[0\]byte\]}}" [[FUTURE]], 1
// CHECK-NEXT: [[FUTURE_CODE:%[0-9]+]] = extractvalue %"main.Future{{\[\[0\]byte\]}}" [[FUTURE]], 0
// CHECK-NEXT: [[FUTURE_NIL:%[0-9]+]] = icmp eq ptr [[FUTURE_CODE]], null
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 [[FUTURE_NIL]])
// CHECK: [[IO_RESULT:%[0-9]+]] = call [0 x i8] %__llgo_funcval_code1(ptr {{(nest|swiftself)}} [[FUTURE_ENV]])
// CHECK-NEXT: ret [0 x i8] [[IO_RESULT]]

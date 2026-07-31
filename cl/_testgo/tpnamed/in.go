// LITTEST
package main

type Void = [0]byte
type Future[T any] func() T

type IO[T any] func() Future[T]

// CHECK-LABEL: define %"main.IO[error]" @main.WriteFile(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"main.IO[error]" { ptr @"main.WriteFile$1", ptr null }
// CHECK-NEXT: }

func WriteFile(fileName string) IO[error] {

	// CHECK-LABEL: define %"main.Future[error]" @"main.WriteFile$1"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret %"main.Future[error]" { ptr @"main.WriteFile$1$1", ptr null }
	// CHECK-NEXT: }

	return func() Future[error] {

		// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"main.WriteFile$1$1"(){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
		// CHECK-NEXT: }

		// CHECK-LABEL: define void @main.init(){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
		// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
		// CHECK-EMPTY:
		// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
		// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
		// CHECK-NEXT:   br label %_llgo_2
		// CHECK-EMPTY:
		// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
		// CHECK-NEXT:   ret void
		// CHECK-NEXT: }

		return func() error {
			return nil
		}
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:  %0 = call [0 x i8] @"main.RunIO{{\[\[0\]byte\]}}"(%"main.IO{{\[\[0\]byte\]}}" { ptr @"main.main$1", ptr null })
// CHECK-NEXT:  ret void
// CHECK-NEXT: }

func main() {

	// CHECK-LABEL: define %"main.Future{{\[\[0\]byte\]}}" @"main.main$1"()
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret %"main.Future{{\[\[0\]byte\]}}" { ptr @"main.main$1$1", ptr null }
	// CHECK-NEXT: }

	RunIO[Void](func() Future[Void] {

		// CHECK-LABEL: define [0 x i8] @"main.main$1$1"(){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   ret [0 x i8] zeroinitializer
		// CHECK-NEXT: }

		return func() (ret Void) {
			return
		}
	})
}

func RunIO[T any](call IO[T]) T {
	return call()()
}

// CHECK-LABEL: define linkonce [0 x i8] @"main.RunIO{{\[\[0\]byte\]}}"(%"main.IO{{\[\[0\]byte\]}}" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"main.IO{{\[\[0\]byte\]}}" %0, 1
// CHECK-NEXT:   %2 = extractvalue %"main.IO{{\[\[0\]byte\]}}" %0, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %2)
// CHECK-NEXT:   %3 = call %"main.Future{{\[\[0\]byte\]}}" %__llgo_funcval_code(ptr {{(nest|swiftself)}} %1)
// CHECK-NEXT:   %4 = extractvalue %"main.Future{{\[\[0\]byte\]}}" %3, 1
// CHECK-NEXT:   %5 = extractvalue %"main.Future{{\[\[0\]byte\]}}" %3, 0
// CHECK-NEXT:   %6 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr %5)
// CHECK-NEXT:   %7 = call [0 x i8] %__llgo_funcval_code1(ptr {{(nest|swiftself)}} %4)
// CHECK-NEXT:   ret [0 x i8] %7
// CHECK-NEXT: }

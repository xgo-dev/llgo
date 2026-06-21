// LITTEST
package main

type Void = [0]byte
type Future[T any] func() T

type IO[T any] func() Future[T]

// CHECK-LABEL: define %"{{.*}}/cl/_testgo/tpnamed.IO[error]" @"{{.*}}/cl/_testgo/tpnamed.WriteFile"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/tpnamed.IO[error]" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.WriteFile$1", ptr null }
// CHECK-NEXT: }

func WriteFile(fileName string) IO[error] {

	// CHECK-LABEL: define %"{{.*}}/cl/_testgo/tpnamed.Future[error]" @"{{.*}}/cl/_testgo/tpnamed.WriteFile$1"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/tpnamed.Future[error]" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.WriteFile$1$1", ptr null }
	// CHECK-NEXT: }

	return func() Future[error] {

		// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/tpnamed.WriteFile$1$1"(){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
		// CHECK-NEXT: }

		// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpnamed.init"(){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/tpnamed.init$guard", align 1
		// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
		// CHECK-EMPTY:
		// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
		// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/tpnamed.init$guard", align 1
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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpnamed.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:  %0 = call [0 x i8] @"{{.*}}/cl/_testgo/tpnamed.RunIO{{\[\[0\]byte\]}}"(%"{{.*}}/cl/_testgo/tpnamed.IO{{\[\[0\]byte\]}}" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.main$1", ptr null })
// CHECK-NEXT:  ret void
// CHECK-NEXT: }

func main() {

	// CHECK-LABEL: define %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" @"{{.*}}/cl/_testgo/tpnamed.main$1"()
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.main$1$1", ptr null }
	// CHECK-NEXT: }

	RunIO[Void](func() Future[Void] {

		// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/tpnamed.main$1$1"(){{.*}} {
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

// CHECK-LABEL: define linkonce %"{{.*}}/cl/_testgo/tpnamed.Future[error]" @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.WriteFile$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = tail call %"{{.*}}/cl/_testgo/tpnamed.Future[error]" @"{{.*}}/cl/_testgo/tpnamed.WriteFile$1"()
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/tpnamed.Future[error]" %1
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.WriteFile$1$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/tpnamed.WriteFile$1$1"()
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %1
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.main$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = tail call %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" @"{{.*}}/cl/_testgo/tpnamed.main$1"()
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" %1
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce [0 x i8] @"{{.*}}/cl/_testgo/tpnamed.RunIO{{\[\[0\]byte\]}}"(%"{{.*}}/cl/_testgo/tpnamed.IO{{\[\[0\]byte\]}}" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/cl/_testgo/tpnamed.IO{{\[\[0\]byte\]}}" %0, 1
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/cl/_testgo/tpnamed.IO{{\[\[0\]byte\]}}" %0, 0
// CHECK-NEXT:   %3 = call %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" %2(ptr %1)
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" %3, 1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/cl/_testgo/tpnamed.Future{{\[\[0\]byte\]}}" %3, 0
// CHECK-NEXT:   %6 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = call [0 x i8] %5(ptr %4)
// CHECK-NEXT:   ret [0 x i8] %7
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce [0 x i8] @"__llgo_stub.{{.*}}/cl/_testgo/tpnamed.main$1$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = tail call [0 x i8] @"{{.*}}/cl/_testgo/tpnamed.main$1$1"()
// CHECK-NEXT:   ret [0 x i8] %1
// CHECK-NEXT: }

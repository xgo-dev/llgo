// LITTEST
package main

type Void = [0]byte
type Future[T any] func() T

type IO[T any] func() Future[T]

// CHECK-LABEL: define %"{{.*}}tpnamed.IO[error]" @"{{.*}}tpnamed.WriteFile"(%"{{.*}}String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}tpnamed.IO[error]" { ptr @"__llgo_stub.{{.*}}tpnamed.WriteFile$1", ptr null }
func WriteFile(fileName string) IO[error] {
	return func() Future[error] {
		// CHECK-LABEL: define %"{{.*}}tpnamed.Future[error]" @"{{.*}}tpnamed.WriteFile$1"(){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   ret %"{{.*}}tpnamed.Future[error]" { ptr @"__llgo_stub.{{.*}}tpnamed.WriteFile$1$1", ptr null }
		return func() error {
			// CHECK-LABEL: define %"{{.*}}iface" @"{{.*}}tpnamed.WriteFile$1$1"(){{.*}} {
			// CHECK-NEXT: _llgo_0:
			// CHECK-NEXT:   ret %"{{.*}}iface" zeroinitializer
			return nil
		}
	}
}

// CHECK-LABEL: define void @"{{.*}}tpnamed.main"(){{.*}} {
func main() {
	// CHECK: call [0 x i8] @"{{.*}}tpnamed.RunIO[[0]byte]"(%"{{.*}}tpnamed.IO[[0]byte]" { ptr @"__llgo_stub.{{.*}}tpnamed.main$1", ptr null })
	// CHECK: ret void
	RunIO[Void](func() Future[Void] {
		// CHECK-LABEL: define %"{{.*}}tpnamed.Future[[0]byte]" @"{{.*}}tpnamed.main$1"(){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   ret %"{{.*}}tpnamed.Future[[0]byte]" { ptr @"__llgo_stub.{{.*}}tpnamed.main$1$1", ptr null }
		return func() (ret Void) {
			// CHECK-LABEL: define [0 x i8] @"{{.*}}tpnamed.main$1$1"(){{.*}} {
			// CHECK-NEXT: _llgo_0:
			// CHECK-NEXT:   ret [0 x i8] zeroinitializer
			return
		}
	})
}

// CHECK-LABEL: define linkonce [0 x i8] @"{{.*}}tpnamed.RunIO[[0]byte]"(%"{{.*}}tpnamed.IO[[0]byte]" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}tpnamed.IO[[0]byte]" %0, 1
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}tpnamed.IO[[0]byte]" %0, 0
// CHECK-NEXT:   %3 = call %"{{.*}}tpnamed.Future[[0]byte]" %2(ptr %1)
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}tpnamed.Future[[0]byte]" %3, 1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}tpnamed.Future[[0]byte]" %3, 0
// CHECK-NEXT:   %6 = call [0 x i8] %5(ptr %4)
// CHECK-NEXT:   ret [0 x i8] %6
func RunIO[T any](call IO[T]) T {
	return call()()
}

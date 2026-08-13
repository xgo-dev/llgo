// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/pthread/sync"
)

// The C-backed Once implementation must lower to pthread_once with a concrete
// callback, rather than duplicating the closure body at each call site.
// CHECK-LABEL: define void @main.f(){{.*}} {
// CHECK: call i32 @pthread_once(ptr @main.once, ptr @"main.f$1")
// CHECK-NEXT: ret void
// CHECK-LABEL: define void @"main.f$1"(){{.*}} {
// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}})
// CHECK-NEXT: ret void
// CHECK-LABEL: define void @main.init(){{.*}} {
// pthread_once_t is a named aggregate on Darwin and i32 on Linux. In both
// cases, preserve the association from the runtime initializer to main.once.
// CHECK: [[ONCE_INIT:%[0-9]+]] = load [[ONCE_TYPE:(i32|%"[^"]*Once")]], ptr @llgoSyncOnceInitVal
// CHECK-NEXT: store [[ONCE_TYPE]] [[ONCE_INIT]], ptr @main.once
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[PREFIX:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}StringFrom"(ptr @{{[0-9]+}}, i64 9)
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[PREFIX]])
// CHECK: [[WHOLE:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}StringFromCStr"(ptr @{{[0-9]+}})
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[WHOLE]])
// CHECK: call void @main.f()
// CHECK-NEXT: call void @main.f()

var once sync.Once = sync.OnceInit

func f() {
	once.Do(func() {
		c.Printf(c.Str("Do once\n"))
	})
}

func main() {
	println(c.GoString(c.Str("sync.Once demo\n"), 9))
	println(c.GoString(c.Str("sync.Once demo\n")))
	f()
	f()
}

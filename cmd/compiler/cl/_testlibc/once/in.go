// LITTEST
// Scope: os (Darwin/Linux pthread_once ABI)
package main

import "github.com/xgo-dev/llgo/cmd/compiler/cl/_testlibc/once/local"

// CHECK-LABEL: define void @main.init(){{.*}} {
// The system header owns pthread_once_t's exact shape. Preserve whichever
// loaded type and value it defines when initializing main.once.
// CHECK: [[ONCE_INIT:%[0-9]+]] = load [[ONCE_TYPE:[^,]+]], ptr @llgoSyncOnceInitVal
// CHECK-NEXT: store [[ONCE_TYPE]] [[ONCE_INIT]], ptr @main.once
// pthread_once accepts a bare C callback on POSIX hosts.
// CHECK-LABEL: define void @main.runOnce(){{.*}} {
// DARWIN: call i32 @pthread_once(ptr @main.once, ptr @"main.runOnce$1")
// LINUX: call i32 @pthread_once(ptr @main.once, ptr @"main.runOnce$1")

var once local.Once = local.OnceInit

func runOnce() {
	once.Do(func() {
		println("Do once")
	})
}

func main() {
	runOnce()
	runOnce()
}

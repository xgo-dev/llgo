// LITTEST
package main

// Package locality has one TLS scalar and one context-local block. Pointer
// values themselves must not become TLS globals or registered global roots.
// CHECK-DAG: @main.scalar = thread_local global i64 0
// CHECK-DAG: @main.__llgo_local_cache = thread_local global i64 0
// CHECK-DAG: @"main.__llgo_tls_init$guard" = thread_local global i8 0
// CHECK-DAG: @"main.__llgo_tls_init$failure_cache" = thread_local global i64 0
// CHECK-NOT: RegisterLocalRoot
// CHECK-NOT: localitycodegen.pointer" = thread_local
// CHECK-NOT: localitycodegen.initialized" = thread_local

// CHECK-LABEL: define ptr @main.__llgo_local_block()
// CHECK: call ptr @"{{.*}}runtime.LocalPackage"(ptr @main.__llgo_local_cache, i64 16, i64 8)

// CHECK-LABEL: define void @main.__llgo_tls_init()
// CHECK: call void @main.__llgo_local_init_0()

// CHECK-LABEL: define void @"main.__llgo_tls_init$ensure"()
// CHECK: call void @"{{.*}}runtime.EnsureLocalInitializer"(ptr @"main.__llgo_tls_init$guard", ptr @"main.__llgo_tls_init$failure_cache"

// CHECK-LABEL: define ptr @{{"?ExportedLocality"?}}()
// CHECK: call i64 @"{{.*}}EnterLocalContext"
// CHECK: call ptr @main.__llgo_local_block()
// CHECK: call void @"{{.*}}LeaveLocalContext"

// CHECK-LABEL: define void @main.init()
// CHECK: call ptr @main.newPointer()
// CHECK: call void @"main.__llgo_tls_init$ensure"()

// CHECK-LABEL: define { i64, ptr, ptr } @main.values()
// CHECK: call void @"main.__llgo_tls_init$ensure"()
// CHECK: load i64, ptr @main.scalar
// CHECK: call ptr @main.__llgo_local_block()

// CHECK-LABEL: define ptr @"main._llgo_routine$1"(ptr
// CHECK: alloca %"{{.*}}LocalContext"
// CHECK: call i64 @"{{.*}}EnterLocalContext"
// CHECK: call void @"{{.*}}LeaveLocalContext"

var backing int

func newPointer() *int {
	return &backing
}

//llgo:tls
var scalar int

//llgo:gls
var pointer *int

//llgo:tls
var initialized = newPointer()

func values() (int, *int, *int) {
	return scalar, pointer, initialized
}

//export ExportedLocality
func ExportedLocality() *int {
	return pointer
}

func main() {
	_, _, _ = values()
	go values()
}

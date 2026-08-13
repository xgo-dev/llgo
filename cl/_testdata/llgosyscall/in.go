// LITTEST
package llgosyscall

import _ "unsafe"

// llgo.syscall lowers to a typed indirect call and translates -1 through
// errno. Cover the ordinary, six-argument, floating-point, and raw forms.
// CHECK-LABEL: define i64 @"{{.*}}.Use"(){{.*}} {
// CHECK: [[USE_R1:%[0-9]+]] = call i64 null(i64 1, i64 2, i64 3)
// CHECK-NEXT: [[USE_FAILED:%[0-9]+]] = icmp eq i64 [[USE_R1]], -1
// CHECK-NEXT: [[USE_ERRNO32:%[0-9]+]] = call i32 @cliteErrno()
// CHECK-NEXT: [[USE_ERRNO:%[0-9]+]] = sext i32 [[USE_ERRNO32]] to i64
// CHECK-NEXT: [[USE_ERR:%[0-9]+]] = select i1 [[USE_FAILED]], i64 [[USE_ERRNO]], i64 0
// CHECK: [[USE_TRIPLE:%[0-9]+]] = insertvalue { i64, i64, i64 } %{{[0-9]+}}, i64 [[USE_ERR]], 2
// CHECK-NEXT: [[USE_RESULT:%[0-9]+]] = extractvalue { i64, i64, i64 } [[USE_TRIPLE]], 0
// CHECK: ret i64 [[USE_RESULT]]
// CHECK-LABEL: define i64 @"{{.*}}.Use5F64"(i64 %0, double %1){{.*}} {
// CHECK: [[USE5_FN:%[0-9]+]] = inttoptr i64 %0 to ptr
// CHECK-NEXT: [[USE5_R1:%[0-9]+]] = call i64 [[USE5_FN]](i64 1, i64 2, i64 3, i64 4, i64 5, double %1)
// CHECK-NEXT: [[USE5_FAILED:%[0-9]+]] = icmp eq i64 [[USE5_R1]], -1
// CHECK-NEXT: [[USE5_ERRNO32:%[0-9]+]] = call i32 @cliteErrno()
// CHECK-NEXT: [[USE5_ERRNO:%[0-9]+]] = sext i32 [[USE5_ERRNO32]] to i64
// CHECK-NEXT: [[USE5_ERR:%[0-9]+]] = select i1 [[USE5_FAILED]], i64 [[USE5_ERRNO]], i64 0
// CHECK: [[USE5_TRIPLE:%[0-9]+]] = insertvalue { i64, i64, i64 } %{{[0-9]+}}, i64 [[USE5_ERR]], 2
// CHECK-NEXT: [[USE5_RESULT:%[0-9]+]] = extractvalue { i64, i64, i64 } [[USE5_TRIPLE]], 0
// CHECK: ret i64 [[USE5_RESULT]]
// CHECK-LABEL: define i64 @"{{.*}}.Use6"(){{.*}} {
// CHECK: [[USE6_R1:%[0-9]+]] = call i64 null(i64 1, i64 2, i64 3, i64 4, i64 5, i64 6)
// CHECK-NEXT: [[USE6_FAILED:%[0-9]+]] = icmp eq i64 [[USE6_R1]], -1
// CHECK-NEXT: [[USE6_ERRNO32:%[0-9]+]] = call i32 @cliteErrno()
// CHECK-NEXT: [[USE6_ERRNO:%[0-9]+]] = sext i32 [[USE6_ERRNO32]] to i64
// CHECK-NEXT: [[USE6_ERR:%[0-9]+]] = select i1 [[USE6_FAILED]], i64 [[USE6_ERRNO]], i64 0
// CHECK: [[USE6_TRIPLE:%[0-9]+]] = insertvalue { i64, i64, i64 } %{{[0-9]+}}, i64 [[USE6_ERR]], 2
// CHECK-NEXT: [[USE6_RESULT:%[0-9]+]] = extractvalue { i64, i64, i64 } [[USE6_TRIPLE]], 0
// CHECK: ret i64 [[USE6_RESULT]]
// CHECK-LABEL: define i64 @"{{.*}}.Use6X"(){{.*}} {
// CHECK: [[USE6X_R1:%[0-9]+]] = call i64 null(i64 1, i64 2, i64 3, i64 4, i64 5, i64 6)
// CHECK-NEXT: [[USE6X_FAILED:%[0-9]+]] = icmp eq i64 [[USE6X_R1]], -1
// CHECK-NEXT: [[USE6X_ERRNO32:%[0-9]+]] = call i32 @cliteErrno()
// CHECK-NEXT: [[USE6X_ERRNO:%[0-9]+]] = sext i32 [[USE6X_ERRNO32]] to i64
// CHECK-NEXT: [[USE6X_ERR:%[0-9]+]] = select i1 [[USE6X_FAILED]], i64 [[USE6X_ERRNO]], i64 0
// CHECK: [[USE6X_TRIPLE:%[0-9]+]] = insertvalue { i64, i64, i64 } %{{[0-9]+}}, i64 [[USE6X_ERR]], 2
// CHECK-NEXT: [[USE6X_RESULT:%[0-9]+]] = extractvalue { i64, i64, i64 } [[USE6X_TRIPLE]], 0
// CHECK: ret i64 [[USE6X_RESULT]]
// CHECK-LABEL: define i64 @"{{.*}}.UsePtr"(){{.*}} {
// CHECK: [[USEPTR_R1:%[0-9]+]] = call i64 null(i64 1, i64 2, i64 3)
// CHECK-NEXT: [[USEPTR_FAILED:%[0-9]+]] = icmp eq i64 [[USEPTR_R1]], -1
// CHECK-NEXT: [[USEPTR_ERRNO32:%[0-9]+]] = call i32 @cliteErrno()
// CHECK-NEXT: [[USEPTR_ERRNO:%[0-9]+]] = sext i32 [[USEPTR_ERRNO32]] to i64
// CHECK-NEXT: [[USEPTR_ERR:%[0-9]+]] = select i1 [[USEPTR_FAILED]], i64 [[USEPTR_ERRNO]], i64 0
// CHECK: [[USEPTR_TRIPLE:%[0-9]+]] = insertvalue { i64, i64, i64 } %{{[0-9]+}}, i64 [[USEPTR_ERR]], 2
// CHECK-NEXT: [[USEPTR_RESULT:%[0-9]+]] = extractvalue { i64, i64, i64 } [[USEPTR_TRIPLE]], 0
// CHECK: ret i64 [[USEPTR_RESULT]]
// CHECK-LABEL: define i64 @"{{.*}}.UseRaw"(){{.*}} {
// CHECK: [[USERAW_R1:%[0-9]+]] = call i64 null(i64 1, i64 2, i64 3)
// CHECK-NEXT: [[USERAW_FAILED:%[0-9]+]] = icmp eq i64 [[USERAW_R1]], -1
// CHECK-NEXT: [[USERAW_ERRNO32:%[0-9]+]] = call i32 @cliteErrno()
// CHECK-NEXT: [[USERAW_ERRNO:%[0-9]+]] = sext i32 [[USERAW_ERRNO32]] to i64
// CHECK-NEXT: [[USERAW_ERR:%[0-9]+]] = select i1 [[USERAW_FAILED]], i64 [[USERAW_ERRNO]], i64 0
// CHECK: [[USERAW_TRIPLE:%[0-9]+]] = insertvalue { i64, i64, i64 } %{{[0-9]+}}, i64 [[USERAW_ERR]], 2
// CHECK-NEXT: [[USERAW_RESULT:%[0-9]+]] = extractvalue { i64, i64, i64 } [[USERAW_TRIPLE]], 0
// CHECK: ret i64 [[USERAW_RESULT]]
// CHECK-LABEL: define i64 @"{{.*}}.UseRaw6"(){{.*}} {
// CHECK: [[USERAW6_R1:%[0-9]+]] = call i64 null(i64 1, i64 2, i64 3, i64 4, i64 5, i64 6)
// CHECK-NEXT: [[USERAW6_FAILED:%[0-9]+]] = icmp eq i64 [[USERAW6_R1]], -1
// CHECK-NEXT: [[USERAW6_ERRNO32:%[0-9]+]] = call i32 @cliteErrno()
// CHECK-NEXT: [[USERAW6_ERRNO:%[0-9]+]] = sext i32 [[USERAW6_ERRNO32]] to i64
// CHECK-NEXT: [[USERAW6_ERR:%[0-9]+]] = select i1 [[USERAW6_FAILED]], i64 [[USERAW6_ERRNO]], i64 0
// CHECK: [[USERAW6_TRIPLE:%[0-9]+]] = insertvalue { i64, i64, i64 } %{{[0-9]+}}, i64 [[USERAW6_ERR]], 2
// CHECK-NEXT: [[USERAW6_RESULT:%[0-9]+]] = extractvalue { i64, i64, i64 } [[USERAW6_TRIPLE]], 0
// CHECK: ret i64 [[USERAW6_RESULT]]

//go:linkname syscall llgo.syscall
func syscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname syscall6 llgo.syscall
func syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname syscall6X llgo.syscall
func syscall6X(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname syscall5f64 llgo.syscall
func syscall5f64(fn, a1, a2, a3, a4, a5 uintptr, f1 float64) (r1, r2, err uintptr)

//go:linkname syscallPtr llgo.syscall
func syscallPtr(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname rawSyscall llgo.syscall
func rawSyscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname rawSyscall6 llgo.syscall
func rawSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

func Use() uintptr {
	r1, _, _ := syscall(0, 1, 2, 3)
	return r1
}

func Use5F64(fn uintptr, f float64) uintptr {
	r1, _, _ := syscall5f64(fn, 1, 2, 3, 4, 5, f)
	return r1
}

func Use6() uintptr {
	r1, _, _ := syscall6(0, 1, 2, 3, 4, 5, 6)
	return r1
}

func Use6X() uintptr {
	r1, _, _ := syscall6X(0, 1, 2, 3, 4, 5, 6)
	return r1
}

func UsePtr() uintptr {
	r1, _, _ := syscallPtr(0, 1, 2, 3)
	return r1
}

func UseRaw() uintptr {
	r1, _, _ := rawSyscall(0, 1, 2, 3)
	return r1
}

func UseRaw6() uintptr {
	r1, _, _ := rawSyscall6(0, 1, 2, 3, 4, 5, 6)
	return r1
}

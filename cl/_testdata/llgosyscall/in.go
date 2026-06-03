// LITTEST
package llgosyscall

import _ "unsafe"

//go:linkname syscall llgo.syscall
func syscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname syscall6 llgo.syscall
func syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname syscall6X llgo.syscall
func syscall6X(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname syscallPtr llgo.syscall
func syscallPtr(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname rawSyscall llgo.syscall
func rawSyscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname rawSyscall6 llgo.syscall
func rawSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

// CHECK-LABEL: define i64 @"{{.*}}.Use"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 null(i64 1, i64 2, i64 3)
// CHECK-NEXT:   %1 = icmp eq i64 %0, -1
// CHECK-NEXT:   %2 = call i32 @cliteErrno()
// CHECK-NEXT:   %3 = sext i32 %2 to i64
// CHECK:   ret i64 %8
func Use() uintptr {
	r1, _, _ := syscall(0, 1, 2, 3)
	return r1
}

// CHECK-LABEL: define i64 @"{{.*}}.Use6"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 null(i64 1, i64 2, i64 3, i64 4, i64 5, i64 6)
// CHECK-NEXT:   %1 = icmp eq i64 %0, -1
// CHECK-NEXT:   %2 = call i32 @cliteErrno()
// CHECK-NEXT:   %3 = sext i32 %2 to i64
// CHECK:   ret i64 %8
func Use6() uintptr {
	r1, _, _ := syscall6(0, 1, 2, 3, 4, 5, 6)
	return r1
}

// CHECK-LABEL: define i64 @"{{.*}}/llgosyscall.Use6X"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 null(i64 1, i64 2, i64 3, i64 4, i64 5, i64 6)
// CHECK-NEXT:   %1 = icmp eq i64 %0, -1
// CHECK-NEXT:   %2 = call i32 @cliteErrno()
// CHECK-NEXT:   %3 = sext i32 %2 to i64
// CHECK-NEXT:   %4 = select i1 %1, i64 %3, i64 0
// CHECK-NEXT:   %5 = insertvalue { i64, i64, i64 } undef, i64 %0, 0
// CHECK-NEXT:   %6 = insertvalue { i64, i64, i64 } %5, i64 0, 1
// CHECK-NEXT:   %7 = insertvalue { i64, i64, i64 } %6, i64 %4, 2
// CHECK-NEXT:   %8 = extractvalue { i64, i64, i64 } %7, 0
// CHECK-NEXT:   %9 = extractvalue { i64, i64, i64 } %7, 1
// CHECK-NEXT:   %10 = extractvalue { i64, i64, i64 } %7, 2
// CHECK-NEXT:   ret i64 %8
// CHECK-NEXT: }
func Use6X() uintptr {
	r1, _, _ := syscall6X(0, 1, 2, 3, 4, 5, 6)
	return r1
}

// CHECK-LABEL: define i64 @"{{.*}}/llgosyscall.UsePtr"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 null(i64 1, i64 2, i64 3)
// CHECK-NEXT:   %1 = icmp eq i64 %0, -1
// CHECK-NEXT:   %2 = call i32 @cliteErrno()
// CHECK-NEXT:   %3 = sext i32 %2 to i64
// CHECK-NEXT:   %4 = select i1 %1, i64 %3, i64 0
// CHECK-NEXT:   %5 = insertvalue { i64, i64, i64 } undef, i64 %0, 0
// CHECK-NEXT:   %6 = insertvalue { i64, i64, i64 } %5, i64 0, 1
// CHECK-NEXT:   %7 = insertvalue { i64, i64, i64 } %6, i64 %4, 2
// CHECK-NEXT:   %8 = extractvalue { i64, i64, i64 } %7, 0
// CHECK-NEXT:   %9 = extractvalue { i64, i64, i64 } %7, 1
// CHECK-NEXT:   %10 = extractvalue { i64, i64, i64 } %7, 2
// CHECK-NEXT:   ret i64 %8
// CHECK-NEXT: }
func UsePtr() uintptr {
	r1, _, _ := syscallPtr(0, 1, 2, 3)
	return r1
}

// CHECK-LABEL: define i64 @"{{.*}}/llgosyscall.UseRaw"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 null(i64 1, i64 2, i64 3)
// CHECK-NEXT:   %1 = icmp eq i64 %0, -1
// CHECK-NEXT:   %2 = call i32 @cliteErrno()
// CHECK-NEXT:   %3 = sext i32 %2 to i64
// CHECK-NEXT:   %4 = select i1 %1, i64 %3, i64 0
// CHECK-NEXT:   %5 = insertvalue { i64, i64, i64 } undef, i64 %0, 0
// CHECK-NEXT:   %6 = insertvalue { i64, i64, i64 } %5, i64 0, 1
// CHECK-NEXT:   %7 = insertvalue { i64, i64, i64 } %6, i64 %4, 2
// CHECK-NEXT:   %8 = extractvalue { i64, i64, i64 } %7, 0
// CHECK-NEXT:   %9 = extractvalue { i64, i64, i64 } %7, 1
// CHECK-NEXT:   %10 = extractvalue { i64, i64, i64 } %7, 2
// CHECK-NEXT:   ret i64 %8
// CHECK-NEXT: }
func UseRaw() uintptr {
	r1, _, _ := rawSyscall(0, 1, 2, 3)
	return r1
}

// CHECK-LABEL: define i64 @"{{.*}}.UseRaw6"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 null(i64 1, i64 2, i64 3, i64 4, i64 5, i64 6)
// CHECK-NEXT:   %1 = icmp eq i64 %0, -1
// CHECK-NEXT:   %2 = call i32 @cliteErrno()
// CHECK-NEXT:   %3 = sext i32 %2 to i64
// CHECK:   ret i64 %8
func UseRaw6() uintptr {
	r1, _, _ := rawSyscall6(0, 1, 2, 3, 4, 5, 6)
	return r1
}

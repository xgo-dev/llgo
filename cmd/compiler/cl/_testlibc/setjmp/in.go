// LITTEST darwin/arm64 linux/amd64 windows/386 windows/amd64 windows/arm64
// Scope: os+arch (setjmp symbols, buffer layout, and dispatch ABI)
package main

import "unsafe"

//go:linkname allocaSigjmpBuf llgo.sigjmpbuf
func allocaSigjmpBuf() unsafe.Pointer

//go:linkname sigsetjmp llgo.sigsetjmp
func sigsetjmp(jb unsafe.Pointer, savemask int32) int32

//go:linkname siglongjmp llgo.siglongjmp
func siglongjmp(jb unsafe.Pointer, retval int32)

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname fprintf C.fprintf
func fprintf(fp unsafe.Pointer, format *int8, __llgo_va_list ...any) int32

// CHECK-LABEL: define void @main.main(){{.*}} {
// The libc ABI controls the sigjmp_buf size and libc symbol spellings.
// DARWIN-ARM64: [[JMPBUF:%[0-9]+]] = alloca i8, i64 196
// LINUX-AMD64: [[JMPBUF:%[0-9]+]] = alloca i8, i64 200
// WINDOWS-386: [[JMPBUF:%[0-9]+]] = alloca i8, i32 64, align 16
// WINDOWS-AMD64: [[JMPBUF:%[0-9]+]] = alloca i8, i64 256, align 16
// WINDOWS-ARM64: [[JMPBUF:%[0-9]+]] = alloca i8, i64 192, align 16
// DARWIN-ARM64-NEXT: [[RET:%[0-9]+]] = call i32 @sigsetjmp(ptr [[JMPBUF]], i32 0)
// LINUX-AMD64-NEXT: [[RET:%[0-9]+]] = call i32 @__sigsetjmp(ptr [[JMPBUF]], i32 0)
// WINDOWS-386-NEXT: [[RET:%[0-9]+]] = call i32 @_setjmp3(ptr [[JMPBUF]], i32 0)
// WINDOWS-AMD64-NEXT: [[RET:%[0-9]+]] = call i32 @{{_setjmpex|__intrinsic_setjmpex}}(ptr [[JMPBUF]], ptr null)
// WINDOWS-ARM64-NEXT: [[RET:%[0-9]+]] = call i32 @llgo_setjmp(ptr [[JMPBUF]])
// CHECK-NEXT: [[FIRST:%[0-9]+]] = icmp eq i32 [[RET]], 0
// CHECK-NEXT: br i1 [[FIRST]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: {{^_llgo_[0-9]+:}}
// DARWIN-ARM64: [[STDERR:%[0-9]+]] = load ptr, ptr @__stderrp
// LINUX-AMD64: [[STDERR:%[0-9]+]] = load ptr, ptr @stderr
// WINDOWS: [[STDERR:%[0-9]+]] = load ptr, ptr @main.stderr
// CHECK-NEXT: call i32 (ptr, ptr, ...) @fprintf(ptr [[STDERR]], ptr @{{[0-9]+}}, ptr getelementptr (i8, ptr @{{[0-9]+}}, i{{32|64}} 2))
// DARWIN-NEXT: call void @siglongjmp(ptr [[JMPBUF]], i32 1)
// LINUX-NEXT: call void @siglongjmp(ptr [[JMPBUF]], i32 1)
// WINDOWS-386-NEXT: call void @longjmp(ptr [[JMPBUF]], i32 1)
// WINDOWS-AMD64-NEXT: call void @llgo_longjmp(ptr [[JMPBUF]], i32 1)
// WINDOWS-ARM64-NEXT: call void @llgo_longjmp(ptr [[JMPBUF]], i32 1)
// CHECK: {{^_llgo_[0-9]+:}}
// CHECK: [[PRINT_RET:%[0-9]+]] = sext i32 [[RET]] to i64
// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[PRINT_RET]])

func main() {
	jb := allocaSigjmpBuf()
	switch ret := sigsetjmp(jb, 0); ret {
	case 0:
		message := cstr("??Hello, setjmp!\n")
		fprintf(stderr, cstr("%s"), unsafe.Add(unsafe.Pointer(message), 2))
		siglongjmp(jb, 1)
	default:
		println("exception:", ret)
	}
}

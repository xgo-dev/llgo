// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// The libc ABI controls the sigjmp_buf size and libc symbol spellings.
// CHECK: [[JMPBUF:%[0-9]+]] = alloca i8, i64 {{(196|200)}}
// CHECK-NEXT: [[RET:%[0-9]+]] = call i32 @{{(__)?sigsetjmp}}(ptr [[JMPBUF]], i32 0)
// CHECK-NEXT: [[FIRST:%[0-9]+]] = icmp eq i32 [[RET]], 0
// CHECK-NEXT: br i1 [[FIRST]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: {{^_llgo_[0-9]+:}}
// CHECK: [[STDERR:%[0-9]+]] = load ptr, ptr @{{(__stderrp|stderr)}}
// CHECK-NEXT: call i32 (ptr, ptr, ...) @fprintf(ptr [[STDERR]], ptr @{{[0-9]+}}, ptr getelementptr (i8, ptr getelementptr (i8, ptr @{{[0-9]+}}, i64 1), i64 1))
// CHECK-NEXT: call void @siglongjmp(ptr [[JMPBUF]], i32 1)
// CHECK: {{^_llgo_[0-9]+:}}
// CHECK: [[PRINT_RET:%[0-9]+]] = sext i32 [[RET]] to i64
// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[PRINT_RET]])

func main() {
	jb := c.AllocaSigjmpBuf()
	switch ret := c.Sigsetjmp(jb, 0); ret {
	case 0:
		cstr := c.Str("??Hello, setjmp!\n")
		c.Fprintf(c.Stderr, c.Str("%s"), c.Advance(c.Pointer(c.Advance(cstr, 1)), 1))
		c.Siglongjmp(jb, 1)
	default:
		println("exception:", ret)
	}
}

// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [18 x i8] c"??Hello, setjmp!\0A\00", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"%s\00", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [10 x i8] c"exception:", align 1{{$}}

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

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca i8
// CHECK-NEXT:   %1 = call i32 @{{(__)?}}sigsetjmp(ptr %0, i32 0)
// CHECK-NEXT:   %2 = icmp eq i32 %1, 0
// CHECK-NEXT:   br i1 %2, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = load ptr, ptr @{{(__stderrp|stderr)}}, align 8
// CHECK-NEXT:   %4 = call i32 (ptr, ptr, ...) @fprintf(ptr %3, ptr @{{[0-9]+}}, ptr getelementptr (i8, ptr getelementptr (i8, ptr @{{[0-9]+}}, i64 1), i64 1))
// CHECK-NEXT:   call void @siglongjmp(ptr %0, i32 1)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 10 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %5 = sext i32 %1 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-NEXT: }

// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LINE: @0 = private unnamed_addr constant [7 x i8] c"Hello\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [10 x i8] c"callback\0A\00", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/callback.callback"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT:   call void %3(ptr %2, ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func callback(msg *c.Char, f func(*c.Char)) {
	f(msg)
}

func main() {
	callback(c.Str("Hello\n"), print)
	callback(c.Str("callback\n"), print)
}

func print(msg *c.Char) {
	c.Printf(msg)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/callback.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/callback.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/callback.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/callback.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/callback.callback"(ptr @0, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testrt/callback.print", ptr null })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/callback.callback"(ptr @1, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testrt/callback.print", ptr null })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/callback.print"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call i32 (ptr, ...) @printf(ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/callback.print"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/callback.print"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

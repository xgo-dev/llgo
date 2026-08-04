// LITTEST
package main

import (
	"fmt"
	"runtime"

	dep "github.com/goplus/llgo/cl/_testgo/rewrite/dep"
)

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [12 x i8] c"main.VarName", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [13 x i8] c"main.VarPlain", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [16 x i8] c"runtime.GOROOT()", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [17 x i8] c"runtime.Version()", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [7 x i8] c"%s: %s\0A", align 1{{$}}

var VarName = "main-default"
var VarPlain string

func printLine(label, value string) {
	fmt.Printf("%s: %s\n", label, value)
}

func main() {
	printLine("main.VarName", VarName)
	printLine("main.VarPlain", VarPlain)
	dep.PrintVar()
	printLine("runtime.GOROOT()", runtime.GOROOT())
	printLine("runtime.Version()", runtime.Version())
}

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/rewrite/dep.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load %"{{.*}}/runtime/internal/runtime.String", ptr @main.VarName, align 8
// CHECK-NEXT:   call void @main.printLine(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 12 }, %"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   %1 = load %"{{.*}}/runtime/internal/runtime.String", ptr @main.VarPlain, align 8
// CHECK-NEXT:   call void @main.printLine(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 13 }, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/rewrite/dep.PrintVar"()
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.String" @runtime.GOROOT()
// CHECK-NEXT:   call void @main.printLine(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 16 }, %"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @runtime.Version()
// CHECK-NEXT:   call void @main.printLine(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 }, %"{{.*}}/runtime/internal/runtime.String" %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.printLine(%"{{.*}}/runtime/internal/runtime.String" %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %2, i64 0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %0, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %5, ptr %3, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %2, i64 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %1, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %7, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %8, ptr %6, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %2, 0
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, i64 2, 1
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %10, i64 2, 2
// CHECK-NEXT:   %12 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Printf(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 7 }, %"{{.*}}/runtime/internal/runtime.Slice" %11)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

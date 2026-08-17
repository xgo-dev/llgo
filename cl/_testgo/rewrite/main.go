// LITTEST
package main

import (
	"fmt"
	"runtime"

	dep "github.com/xgo-dev/llgo/cl/_testgo/rewrite/dep"
)

// Keep the package globals and imported/runtime calls distinct after name
// rewriting.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[VAR_NAME:%[0-9]+]] = load %"{{.*}}String", ptr @main.VarName
// CHECK-NEXT: call void @main.printLine(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 12 }, %"{{.*}}String" [[VAR_NAME]])
// CHECK-NEXT: [[VAR_PLAIN:%[0-9]+]] = load %"{{.*}}String", ptr @main.VarPlain
// CHECK-NEXT: call void @main.printLine(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 13 }, %"{{.*}}String" [[VAR_PLAIN]])
// CHECK-NEXT: call void @"{{.*}}/rewrite/dep.PrintVar"()
// CHECK-NEXT: [[GOROOT:%[0-9]+]] = call %"{{.*}}String" @runtime.GOROOT()
// CHECK-NEXT: call void @main.printLine(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 16 }, %"{{.*}}String" [[GOROOT]])
// CHECK-NEXT: [[VERSION:%[0-9]+]] = call %"{{.*}}String" @runtime.Version()
// CHECK-NEXT: call void @main.printLine(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 17 }, %"{{.*}}String" [[VERSION]])
// CHECK-LABEL: define void @main.printLine(%"{{.*}}String" %0, %"{{.*}}String" %1){{.*}} {
// CHECK: store %"{{.*}}String" %0, ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" %1, ptr %{{[0-9]+}}
// CHECK: [[PRINT_ARGS:%[0-9]+]] = insertvalue %"{{.*}}Slice" %{{[0-9]+}}, i64 2, 2
// CHECK-NEXT: call { i64, %"{{.*}}iface" } @fmt.Printf(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 7 }, %"{{.*}}Slice" [[PRINT_ARGS]])

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

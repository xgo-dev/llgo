// LITTEST
package main

import "os"

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/os.main"(){{.*}} {
func main() {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %0 = call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @os.Getwd()
	// CHECK-NEXT:   %1 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %0, 0
	// CHECK-NEXT:   %2 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %0, 1
	// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %2)
	// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %2, 1
	// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %3, 0
	// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %5, ptr %4, 1
	// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
	// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %7, 0
	// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %8, ptr null, 1
	// CHECK-NEXT:   %10 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %6, %"{{.*}}/runtime/internal/runtime.eface" %9)
	// CHECK-NEXT:   %11 = xor i1 %10, true
	// CHECK-NEXT:   br i1 %11, label %_llgo_1, label %_llgo_2
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	println("cwd:", wd)
}

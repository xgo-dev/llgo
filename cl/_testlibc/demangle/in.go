// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/cpp/llvm"
)

// CHECK: [[MANGLED:@[0-9]+]] = private unnamed_addr constant [29 x i8] c"__ZNK9INIReader10ParseErrorEv", align 1
// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	mangledName := "__ZNK9INIReader10ParseErrorEv"
	// CHECK: [[DEMANGLED:%[0-9]+]] = call ptr @_ZN4llvm15itaniumDemangleENSt3__117basic_string_viewIcNS0_11char_traitsIcEEEEb(%"{{.*}}/runtime/internal/runtime.String" { ptr [[MANGLED]], i64 29 }, i1 true)
	// CHECK-NEXT: [[OK:%[0-9]+]] = icmp ne ptr [[DEMANGLED]], null
	// CHECK-NEXT: br i1 [[OK]], label %{{[^,]+}}, label %{{[^ ]+}}
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[DEMANGLED]])
	if name := llvm.ItaniumDemangle(mangledName, true); name != nil {
		c.Printf(c.Str("%s\n"), name)
	} else {
		println("Failed to demangle")
	}
}

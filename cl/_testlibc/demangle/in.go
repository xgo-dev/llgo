// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/cpp/llvm"
)

// CHECK: @0 = private unnamed_addr constant [29 x i8] c"__ZNK9INIReader10ParseErrorEv", align 1
// CHECK-LABEL: define void @"{{.*}}/cl/_testlibc/demangle.main"(){{.*}} {
func main() {
	mangledName := "__ZNK9INIReader10ParseErrorEv"
	// CHECK:  %0 = call ptr @_ZN4llvm15itaniumDemangleENSt3__117basic_string_viewIcNS0_11char_traitsIcEEEEb(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 29 }, i1 true)
	if name := llvm.ItaniumDemangle(mangledName, true); name != nil {
		c.Printf(c.Str("%s\n"), name)
	} else {
		println("Failed to demangle")
	}
}

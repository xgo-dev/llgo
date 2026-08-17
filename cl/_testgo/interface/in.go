// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdata/foo"
)

type Game1 struct {
	*foo.Game
}

type Game2 struct {
}

func (p *Game2) initGame() {
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[G1_EFACE:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.Game1", ptr undef }, ptr [[G1_DATA:%[0-9]+]], 1
// CHECK-NEXT: [[G1_TYPE:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" [[G1_EFACE]], 0
// CHECK-NEXT: [[G1_IMPLEMENTS:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}foo.Gamer", ptr [[G1_TYPE]])
// CHECK-NEXT: br i1 [[G1_IMPLEMENTS]], label %{{.*}}, label %{{.*}}
// CHECK: [[G1_CALL_DATA:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" [[G1_IFACE:%[0-9]+]])
// CHECK: [[G1_METHOD:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: [[G1_CALL_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[G1_METHOD]], 0
// CHECK-NEXT: [[G1_CALL_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[G1_CALL_PAIR0]], ptr [[G1_CALL_DATA]], 1
// CHECK: call void %{{[0-9]+}}(ptr %{{[0-9]+}})
// CHECK: [[G2_IMPLEMENTS:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}foo.Gamer", ptr @"*_llgo_main.Game2")
// CHECK-NEXT: br i1 [[G2_IMPLEMENTS]], label %{{.*}}, label %{{.*}}
// CHECK: [[G1_EFACE_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" [[G1_EFACE]], 1
// CHECK-NEXT: [[G1_ITAB:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}foo.iface${{[-A-Za-z0-9_]+}}", ptr [[G1_TYPE]])
// CHECK: [[G1_PAIR:%[0-9]+]] = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %{{[0-9]+}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK-NEXT: [[G1_IFACE]] = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } [[G1_PAIR]], 0
// CHECK-NEXT: [[G1_OK:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } [[G1_PAIR]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" [[G1_IFACE]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 [[G1_OK]])
// CHECK: br i1 [[G1_OK]], label %{{.*}}, label %{{.*}}
// CHECK: [[G2_PAIR:%[0-9]+]] = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %{{[0-9]+}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK-NEXT: [[G2_IFACE:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } [[G2_PAIR]], 0
// CHECK-NEXT: [[G2_OK:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } [[G2_PAIR]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" [[G2_IFACE]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 [[G2_OK]])
func main() {
	var g1 any = &Game1{&foo.Game{}}

	var g2 any = &Game2{}

	v1, ok := g1.(foo.Gamer)

	println("OK", v1, ok)

	if ok {
		v1.Load()
	}

	v2, ok := g2.(foo.Gamer)

	println("FAIL", v2, ok)
}

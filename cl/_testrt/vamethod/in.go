// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

// Variadic C-linked methods must forward their receiver-derived format pointer
// and preserve the variadic arguments for direct and interface calls.
// CHECK-LABEL: define i32 @main.CFmt.Printf(%main.CFmt %0, ...){{.*}} {
// CHECK: store %main.CFmt %0, ptr %[[VALUE_COPY:[0-9]+]]
// CHECK: %[[VALUE_FMT_SLOT:[0-9]+]] = getelementptr inbounds %main.CFmt, ptr %[[VALUE_COPY]], i32 0, i32 0
// CHECK-NEXT: %[[VALUE_FMT:[0-9]+]] = load ptr, ptr %[[VALUE_FMT_SLOT]]
// CHECK-NEXT: %[[VALUE_RET:[0-9]+]] = call i32 (ptr, ...) @printf(ptr %[[VALUE_FMT]])
// CHECK-NEXT: ret i32 %[[VALUE_RET]]
// CHECK-LABEL: define i32 @"main.(*CFmt).Printf"(ptr %0, ...){{.*}} {
// CHECK: %[[PTR_FMT_SLOT:[0-9]+]] = getelementptr inbounds %main.CFmt, ptr %0, i32 0, i32 0
// CHECK-NEXT: %[[PTR_FMT:[0-9]+]] = load ptr, ptr %[[PTR_FMT_SLOT]]
// CHECK-NEXT: %[[PTR_RET:[0-9]+]] = call i32 (ptr, ...) @printf(ptr %[[PTR_FMT]])
// CHECK-NEXT: ret i32 %[[PTR_RET]]
// CHECK-LABEL: define void @"main.(*CFmt).SetFormat"(ptr %0, ptr %1){{.*}} {
// CHECK: %[[SET_SLOT:[0-9]+]] = getelementptr inbounds %main.CFmt, ptr %0, i32 0, i32 0
// CHECK-NEXT: store ptr %1, ptr %[[SET_SLOT]]
// CHECK-NEXT: ret void
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[CFMT:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: call void @"main.(*CFmt).SetFormat"(ptr %[[CFMT]], ptr @{{[0-9]+}})
// CHECK: %[[DIRECT_FMT1:[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr %[[DIRECT_FMT1]], ptr @{{[0-9]+}}, i64 100)
// CHECK-NEXT: call void @"main.(*CFmt).SetFormat"(ptr %[[CFMT]], ptr @{{[0-9]+}})
// CHECK: %[[DIRECT_FMT2:[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr %[[DIRECT_FMT2]], i64 200, ptr @{{[0-9]+}})
// CHECK: %[[SET_DATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}runtime.iface" %[[IFACE:[0-9]+]])
// CHECK: %[[SET_METHOD:[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK-NEXT: %[[SET_PAIR_CODE:[0-9]+]] = insertvalue { ptr, ptr } undef, ptr %[[SET_METHOD]], 0
// CHECK-NEXT: %[[SET_PAIR:[0-9]+]] = insertvalue { ptr, ptr } %[[SET_PAIR_CODE]], ptr %[[SET_DATA]], 1
// CHECK-NEXT: %[[SET_CALL_DATA:[0-9]+]] = extractvalue { ptr, ptr } %[[SET_PAIR]], 1
// CHECK-NEXT: %[[SET_CALL_CODE:[0-9]+]] = extractvalue { ptr, ptr } %[[SET_PAIR]], 0
// CHECK-NEXT: call void %[[SET_CALL_CODE]](ptr %[[SET_CALL_DATA]], ptr @{{[0-9]+}})
// CHECK-NEXT: %[[PRINTF_DATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}runtime.iface" %[[IFACE]])
// CHECK: %[[PRINTF_METHOD:[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK-NEXT: %[[PRINTF_PAIR_CODE:[0-9]+]] = insertvalue { ptr, ptr } undef, ptr %[[PRINTF_METHOD]], 0
// CHECK-NEXT: %[[PRINTF_PAIR:[0-9]+]] = insertvalue { ptr, ptr } %[[PRINTF_PAIR_CODE]], ptr %[[PRINTF_DATA]], 1
// CHECK-NEXT: %[[PRINTF_CALL_DATA:[0-9]+]] = extractvalue { ptr, ptr } %[[PRINTF_PAIR]], 1
// CHECK-NEXT: %[[PRINTF_CALL_CODE:[0-9]+]] = extractvalue { ptr, ptr } %[[PRINTF_PAIR]], 0
// CHECK-NEXT: call i32 (ptr, ...) %[[PRINTF_CALL_CODE]](ptr %[[PRINTF_CALL_DATA]], ptr @{{[0-9]+}}, i64 100, i64 200)

//llgo:link (*T).Printf C.printf
func (*T) Printf(__llgo_va_list ...any) c.Int { return 0 }

type T c.Char

//go:linkname Printf C.printf
func Printf(format *c.Char, __llgo_va_list ...any) c.Int

type CFmt struct {
	*T
}

func (f *CFmt) SetFormat(fmt *c.Char) {
	f.T = (*T)(unsafe.Pointer(fmt))
}

type IFmt interface {
	SetFormat(fmt *c.Char)
	Printf(__llgo_va_list ...any) c.Int
}

func main() {
	cfmt := &CFmt{}
	cfmt.SetFormat(c.Str("%s (%d)\n"))
	cfmt.Printf(c.Str("hello"), 100)
	cfmt.SetFormat(c.Str("(%d) %s\n"))
	cfmt.Printf(200, c.Str("world"))

	var i any = &CFmt{}
	ifmt, ok := i.(IFmt)
	if !ok {
		panic("error")
	}
	ifmt.SetFormat(c.Str("%s (%d,%d)\n"))
	ifmt.Printf(c.Str("ifmt"), 100, 200)
}

// LITTEST
// Scope: common
package main

import (
	"unsafe"
)

//go:linkname cstr llgo.cstr
func cstr(string) *int8

// Variadic C-linked methods must forward their receiver-derived format pointer
// and preserve the variadic arguments for direct and interface calls.

//llgo:link (*T).Printf C.printf
func (*T) Printf(__llgo_va_list ...any) int32 { return 0 }

type T int8

//go:linkname Printf C.printf
func Printf(format *int8, __llgo_va_list ...any) int32

type CFmt struct {
	*T
}

func (f *CFmt) SetFormat(fmt *int8) {
	f.T = (*T)(unsafe.Pointer(fmt))
}

type IFmt interface {
	SetFormat(fmt *int8)
	Printf(__llgo_va_list ...any) int32
}

func main() {
	cfmt := &CFmt{}
	cfmt.SetFormat(cstr("%s (%d)\n"))
	cfmt.Printf(cstr("hello"), 100)
	cfmt.SetFormat(cstr("(%d) %s\n"))
	cfmt.Printf(200, cstr("world"))

	var i any = &CFmt{}
	ifmt, ok := i.(IFmt)
	if !ok {
		panic("error")
	}
	ifmt.SetFormat(cstr("%s (%d,%d)\n"))
	ifmt.Printf(cstr("ifmt"), 100, 200)
}

// The linked value receiver materializes its value before loading the embedded
// format pointer; the pointer receiver loads the same field directly.
// CHECK-LABEL: define i32 @main.CFmt.Printf(
// CHECK-SAME: %main.CFmt %[[VALUE:[0-9]+]], ...){{.*}} {
// CHECK: %[[VALUESLOT:[0-9]+]] = alloca %main.CFmt, align 8
// CHECK: store %main.CFmt %[[VALUE]], ptr %[[VALUESLOT]], align 8
// CHECK: %[[VALUEFIELD:[0-9]+]] = getelementptr inbounds nuw %main.CFmt, ptr %[[VALUESLOT]], i32 0, i32 0
// CHECK: %[[VALUEFMT:[0-9]+]] = load ptr, ptr %[[VALUEFIELD]], align 8
// CHECK: %[[VALUERET:[0-9]+]] = call i32 (ptr, ...) @printf(ptr %[[VALUEFMT]])
// CHECK: ret i32 %[[VALUERET]]

// CHECK-LABEL: define i32 @"main.(*CFmt).Printf"(
// CHECK-SAME: ptr %[[PTR:[0-9]+]], ...){{.*}} {
// CHECK: %[[PTRFIELD:[0-9]+]] = getelementptr inbounds nuw %main.CFmt, ptr %[[PTR]], i32 0, i32 0
// CHECK: %[[PTRNIL:[0-9]+]] = icmp eq ptr %[[PTR]], null
// CHECK: br i1 %[[PTRNIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: %[[PTRFMT:[0-9]+]] = load ptr, ptr %[[PTRFIELD]], align 8
// CHECK: %[[PTRRET:[0-9]+]] = call i32 (ptr, ...) @printf(ptr %[[PTRFMT]])
// CHECK: ret i32 %[[PTRRET]]

// CHECK-LABEL: define void @"main.(*CFmt).SetFormat"(
// CHECK-SAME: ptr %[[SELF:[0-9]+]], ptr %[[FORMAT:[0-9]+]]){{.*}} {
// CHECK: %[[FORMATFIELD:[0-9]+]] = getelementptr inbounds nuw %main.CFmt, ptr %[[SELF]], i32 0, i32 0
// CHECK: store ptr %[[FORMAT]], ptr %[[FORMATFIELD]], align 8

// Direct calls pass the receiver's loaded format pointer. Interface calls use
// slots 4 (SetFormat) and 3 (Printf) with the same receiver data as environment.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[CFMT:[0-9]+]] = call noalias dereferenceable(8) ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: call void @"main.(*CFmt).SetFormat"(ptr %[[CFMT]], ptr @{{[0-9]+}})
// CHECK: %[[DIRECTFIELD1:[0-9]+]] = getelementptr inbounds nuw %main.CFmt, ptr %[[CFMT]], i32 0, i32 0
// CHECK: %[[DIRECTFMT1:[0-9]+]] = load ptr, ptr %[[DIRECTFIELD1]], align 8
// CHECK: call i32 (ptr, ...) @printf(ptr %[[DIRECTFMT1]], ptr @{{[0-9]+}}, i64 100)
// CHECK: call void @"main.(*CFmt).SetFormat"(ptr %[[CFMT]], ptr @{{[0-9]+}})
// CHECK: %[[DIRECTFIELD2:[0-9]+]] = getelementptr inbounds nuw %main.CFmt, ptr %[[CFMT]], i32 0, i32 0
// CHECK: %[[DIRECTFMT2:[0-9]+]] = load ptr, ptr %[[DIRECTFIELD2]], align 8
// CHECK: call i32 (ptr, ...) @printf(ptr %[[DIRECTFMT2]], i64 200, ptr @{{[0-9]+}})
// CHECK: %[[EBOX:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.CFmt", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK: %[[TYPE:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %[[EBOX]], 0
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.IFmt, ptr %[[TYPE]])
// CHECK: %[[SETDATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[DYNIFACE:[0-9]+]])
// CHECK: %[[SETVTAB:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %[[DYNIFACE]], 0
// CHECK: %[[SETSLOT:[0-9]+]] = getelementptr ptr, ptr %[[SETVTAB]], i64 4
// CHECK: %[[SETCODE:[0-9]+]] = load ptr, ptr %[[SETSLOT]], align 8
// CHECK: %[[SETPAIR0:[0-9]+]] = insertvalue { ptr, ptr } undef, ptr %[[SETCODE]], 0
// CHECK: %[[SETPAIR:[0-9]+]] = insertvalue { ptr, ptr } %[[SETPAIR0]], ptr %[[SETDATA]], 1
// CHECK: %[[SETENV:[0-9]+]] = extractvalue { ptr, ptr } %[[SETPAIR]], 1
// CHECK: %[[SETCALL:[0-9]+]] = extractvalue { ptr, ptr } %[[SETPAIR]], 0
// CHECK: call void %[[SETCALL]](ptr %[[SETENV]], ptr @{{[0-9]+}})
// CHECK: %[[PRINTDATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[DYNIFACE]])
// CHECK: %[[PRINTVTAB:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %[[DYNIFACE]], 0
// CHECK: %[[PRINTSLOT:[0-9]+]] = getelementptr ptr, ptr %[[PRINTVTAB]], i64 3
// CHECK: %[[PRINTCODE:[0-9]+]] = load ptr, ptr %[[PRINTSLOT]], align 8
// CHECK: %[[PRINTPAIR0:[0-9]+]] = insertvalue { ptr, ptr } undef, ptr %[[PRINTCODE]], 0
// CHECK: %[[PRINTPAIR:[0-9]+]] = insertvalue { ptr, ptr } %[[PRINTPAIR0]], ptr %[[PRINTDATA]], 1
// CHECK: %[[PRINTENV:[0-9]+]] = extractvalue { ptr, ptr } %[[PRINTPAIR]], 1
// CHECK: %[[PRINTCALL:[0-9]+]] = extractvalue { ptr, ptr } %[[PRINTPAIR]], 0
// CHECK: call i32 (ptr, ...) %[[PRINTCALL]](ptr %[[PRINTENV]], ptr @{{[0-9]+}}, i64 100, i64 200)
// CHECK: %[[NEWITAB:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr {{.*}}, ptr %[[TYPE]])
// CHECK: %[[NEWIFACE0:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %[[NEWITAB]], 0
// CHECK: %[[NEWIFACE:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %[[NEWIFACE0]], ptr %{{[0-9]+}}, 1
// CHECK: insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %[[NEWIFACE]], 0
// CHECK: %[[DYNIFACE]] = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %{{[0-9]+}}, 0

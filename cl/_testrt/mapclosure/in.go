// LITTEST
package main

type Type interface {
	String() string
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/mapclosure.demo"(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %3 = getelementptr ptr, ptr %2, i64 3
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } %5, ptr %1, 1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %6, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %6, 0
// CHECK-NEXT:   %9 = call %"{{.*}}/runtime/internal/runtime.String" %8(ptr %7)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %9
// CHECK-NEXT: }
func demo(t Type) string {
	return t.String()
}

type typ struct {
	s string
}

var (
	op = map[string]func(Type) string{
		"demo": demo,
	}
	list = []func(Type) string{demo}
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/mapclosure.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/mapclosure.typ", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 5 }, ptr %1, align 8
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testrt/mapclosure.op", align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 4 }, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_string]_llgo_closure$vc5ZLfKV4flbpeFUtiJWFVJOxWgjZ8JlkoV1ZmTbVIQ", ptr %2, ptr %3)
// CHECK-NEXT:   %5 = load { ptr, ptr }, ptr %4, align 8
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr @"{{.*}}/cl/_testrt/mapclosure.list", align 8
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, 0
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, 1
// CHECK-NEXT:   %9 = icmp uge i64 0, %8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds { ptr, ptr }, ptr %7, i64 0
// CHECK-NEXT:   %11 = load { ptr, ptr }, ptr %10, align 8
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U", ptr @"*_llgo_github.com/goplus/llgo/cl/_testrt/mapclosure.typ")
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %12, 0
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %13, ptr %0, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %5, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %5, 0
// CHECK-NEXT:   %17 = call %"{{.*}}/runtime/internal/runtime.String" %16(ptr %15, %"{{.*}}/runtime/internal/runtime.iface" %14)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U", ptr @"*_llgo_github.com/goplus/llgo/cl/_testrt/mapclosure.typ")
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %18, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %19, ptr %0, 1
// CHECK-NEXT:   %21 = extractvalue { ptr, ptr } %11, 1
// CHECK-NEXT:   %22 = extractvalue { ptr, ptr } %11, 0
// CHECK-NEXT:   %23 = call %"{{.*}}/runtime/internal/runtime.String" %22(ptr %21, %"{{.*}}/runtime/internal/runtime.iface" %20)
// CHECK-NEXT:   %24 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %17, %"{{.*}}/runtime/internal/runtime.String" %23)
// CHECK-NEXT:   %25 = xor i1 %24, true
// CHECK-NEXT:   br i1 %25, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 5 }, ptr %26, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %26, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	t := &typ{"hello"}
	fn1 := op["demo"]
	fn2 := list[0]
	if fn1(t) != fn2(t) {
		panic("error")
	}
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/mapclosure.(*typ).String"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/mapclosure.typ", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.String", ptr %1, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

func (t *typ) String() string {
	return t.s
}

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/mapclosure.demo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/mapclosure.demo"(%"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

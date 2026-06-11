// LITTEST
package main

// CHECK-LINE: @20 = private unnamed_addr constant [4 x i8] c"demo", align 1
// CHECK-LINE: @21 = private unnamed_addr constant [5 x i8] c"hello", align 1
// CHECK-LINE: @25 = private unnamed_addr constant [5 x i8] c"error", align 1

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

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/mapclosure.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/mapclosure.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/mapclosure.init$guard", align 1
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_closure$vc5ZLfKV4flbpeFUtiJWFVJOxWgjZ8JlkoV1ZmTbVIQ", i64 1)
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 4 }, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_closure$vc5ZLfKV4flbpeFUtiJWFVJOxWgjZ8JlkoV1ZmTbVIQ", ptr %1, ptr %2)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testrt/mapclosure.demo", ptr null }, ptr %3, align 8
// CHECK-NEXT:   store ptr %1, ptr @"{{.*}}/cl/_testrt/mapclosure.op", align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5 = getelementptr inbounds { ptr, ptr }, ptr %4, i64 0
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testrt/mapclosure.demo", ptr null }, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4, 0
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, i64 1, 1
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, i64 1, 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %8, ptr @"{{.*}}/cl/_testrt/mapclosure.list", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
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
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/mapclosure.typ", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = load ptr, ptr @"{{.*}}/cl/_testrt/mapclosure.op", align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 4 }, ptr %4, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_string]_llgo_closure$vc5ZLfKV4flbpeFUtiJWFVJOxWgjZ8JlkoV1ZmTbVIQ", ptr %3, ptr %4)
// CHECK-NEXT:   %6 = load { ptr, ptr }, ptr %5, align 8
// CHECK-NEXT:   %7 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr @"{{.*}}/cl/_testrt/mapclosure.list", align 8
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 1
// CHECK-NEXT:   %10 = icmp uge i64 0, %9
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %10, i64 0, i1 true, i64 %9)
// CHECK-NEXT:   %11 = getelementptr inbounds { ptr, ptr }, ptr %8, i64 0
// CHECK-NEXT:   %12 = load { ptr, ptr }, ptr %11, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U", ptr @"*_llgo_{{.*}}/cl/_testrt/mapclosure.typ")
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %14, ptr %0, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %6, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %6, 0
// CHECK-NEXT:   %18 = call %"{{.*}}/runtime/internal/runtime.String" %17(ptr %16, %"{{.*}}/runtime/internal/runtime.iface" %15)
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U", ptr @"*_llgo_{{.*}}/cl/_testrt/mapclosure.typ")
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %19, 0
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %20, ptr %0, 1
// CHECK-NEXT:   %22 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %23 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %24 = call %"{{.*}}/runtime/internal/runtime.String" %23(ptr %22, %"{{.*}}/runtime/internal/runtime.iface" %21)
// CHECK-NEXT:   %25 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %18, %"{{.*}}/runtime/internal/runtime.String" %24)
// CHECK-NEXT:   %26 = xor i1 %25, true
// CHECK-NEXT:   br i1 %26, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 5 }, ptr %27, align 8
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %27, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %28)
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
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/mapclosure.typ", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %4
// CHECK-NEXT: }

func (t *typ) String() string {
	return t.s
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testrt/mapclosure.demo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/mapclosure.demo"(%"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testrt/mapclosure.(*typ).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/mapclosure.(*typ).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

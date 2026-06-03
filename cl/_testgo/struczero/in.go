// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/foo"

type bar struct {
	pb *byte
	f  float32
}

// CHECK-LABEL: define { %"{{.*}}foo.Foo", i1 } @"{{.*}}struczero.Bar"(%"{{.*}}eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}eface" %0, 0
// CHECK-NEXT:   %2 = icmp eq ptr %1, @"_llgo_{{.*}}foo.Foo"
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK: _llgo_1:
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}eface" %0, 1
// CHECK-NEXT:   %4 = load %"{{.*}}foo.Foo", ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { %"{{.*}}foo.Foo", i1 } undef, %"{{.*}}foo.Foo" %4, 0
// CHECK-NEXT:   %6 = insertvalue { %"{{.*}}foo.Foo", i1 } %5, i1 true, 1
// CHECK: _llgo_3:
// CHECK-NEXT:   %7 = phi { %"{{.*}}foo.Foo", i1 } [ %6, %_llgo_1 ], [ zeroinitializer, %_llgo_2 ]
// CHECK-NEXT:   %8 = extractvalue { %"{{.*}}foo.Foo", i1 } %7, 0
// CHECK-NEXT:   %9 = extractvalue { %"{{.*}}foo.Foo", i1 } %7, 1
// CHECK-NEXT:   %10 = insertvalue { %"{{.*}}foo.Foo", i1 } undef, %"{{.*}}foo.Foo" %8, 0
// CHECK-NEXT:   %11 = insertvalue { %"{{.*}}foo.Foo", i1 } %10, i1 %9, 1
// CHECK-NEXT:   ret { %"{{.*}}foo.Foo", i1 } %11
func Bar(v any) (ret foo.Foo, ok bool) {
	ret, ok = v.(foo.Foo)
	return
}

// CHECK-LABEL: define { %"{{.*}}struczero.bar", i1 } @"{{.*}}struczero.Foo"(%"{{.*}}eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}eface" %0, 0
// CHECK-NEXT:   %2 = icmp eq ptr %1, @"_llgo_{{.*}}struczero.bar"
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK: _llgo_1:
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}eface" %0, 1
// CHECK-NEXT:   %4 = load %"{{.*}}struczero.bar", ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { %"{{.*}}struczero.bar", i1 } undef, %"{{.*}}struczero.bar" %4, 0
// CHECK-NEXT:   %6 = insertvalue { %"{{.*}}struczero.bar", i1 } %5, i1 true, 1
// CHECK: _llgo_3:
// CHECK-NEXT:   %7 = phi { %"{{.*}}struczero.bar", i1 } [ %6, %_llgo_1 ], [ zeroinitializer, %_llgo_2 ]
// CHECK-NEXT:   %8 = extractvalue { %"{{.*}}struczero.bar", i1 } %7, 0
// CHECK-NEXT:   %9 = extractvalue { %"{{.*}}struczero.bar", i1 } %7, 1
// CHECK-NEXT:   %10 = insertvalue { %"{{.*}}struczero.bar", i1 } undef, %"{{.*}}struczero.bar" %8, 0
// CHECK-NEXT:   %11 = insertvalue { %"{{.*}}struczero.bar", i1 } %10, i1 %9, 1
// CHECK-NEXT:   ret { %"{{.*}}struczero.bar", i1 } %11
func Foo(v any) (ret bar, ok bool) {
	ret, ok = v.(bar)
	return
}

// CHECK-LABEL: define void @"{{.*}}struczero.main"(){{.*}} {
func main() {
	// CHECK: call { %"{{.*}}struczero.bar", i1 } @"{{.*}}struczero.Foo"(%"{{.*}}eface" zeroinitializer)
	// CHECK: call void @"{{.*}}PrintPointer"(ptr %{{[0-9]+}})
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 32)
	// CHECK-NEXT: %{{[0-9]+}} = fpext float %{{[0-9]+}} to double
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double %{{[0-9]+}})
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 32)
	// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 6 })
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 32)
	// CHECK-NEXT: call void @"{{.*}}PrintBool"(i1 %{{[0-9]+}})
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
	ret, ok := Foo(nil)
	println(ret.pb, ret.f, "notOk:", !ok)

	// CHECK: call ptr @"{{.*}}AllocU"(i64 16)
	// CHECK: store %"{{.*}}foo.Foo" zeroinitializer, ptr %{{[0-9]+}}, align 8
	// CHECK: insertvalue %"{{.*}}eface" { ptr @"_llgo_{{.*}}foo.Foo", ptr undef }, ptr %{{[0-9]+}}, 1
	// CHECK: call { %"{{.*}}foo.Foo", i1 } @"{{.*}}struczero.Bar"(%"{{.*}}eface" %{{[0-9]+}})
	// CHECK: call ptr @"{{.*}}foo.Foo.Pb"(%"{{.*}}foo.Foo" %{{[0-9]+}})
	// CHECK: call void @"{{.*}}PrintPointer"(ptr %{{[0-9]+}})
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 32)
	// CHECK-NEXT: %{{[0-9]+}} = fpext float %{{[0-9]+}} to double
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double %{{[0-9]+}})
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 32)
	// CHECK-NEXT: call void @"{{.*}}PrintBool"(i1 %{{[0-9]+}})
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
	// CHECK-NEXT: ret void
	ret2, ok2 := Bar(foo.Foo{})
	println(ret2.Pb(), ret2.F, ok2)
}

// LITTEST
package main

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/concat.concat"(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %2 = phi %"{{.*}}/runtime/internal/runtime.String" [ zeroinitializer, %_llgo_0 ], [ %13, %_llgo_2 ]
// CHECK-NEXT:   %3 = phi i64 [ -1, %_llgo_0 ], [ %4, %_llgo_2 ]
// CHECK-NEXT:   %4 = add i64 %3, 1
// CHECK-NEXT:   %5 = icmp slt i64 %4, %1
// CHECK-NEXT:   br i1 %5, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %8 = icmp slt i64 %4, 0
// CHECK-NEXT:   %9 = icmp uge i64 %4, %7
// CHECK-NEXT:   %10 = or i1 %9, %8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %6, i64 %4
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.String", ptr %11, align 8
// CHECK-NEXT:   %13 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %2, %"{{.*}}/runtime/internal/runtime.String" %12)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }
func concat(args ...string) (ret string) {
	for _, v := range args {
		ret += v
	}
	return
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/concat.info"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" zeroinitializer, %"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }
func info(s string) string {
	return "" + s + "..."
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/concat.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 48)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %0, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 5 }, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %0, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 1 }, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %0, i64 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 5 }, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4, i64 3, 1
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5, i64 3, 2
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/concat.concat"(%"{{.*}}/runtime/internal/runtime.Slice" %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	result := concat("Hello", " ", "World")
	println(result)
}

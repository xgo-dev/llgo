// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"main.main$1"({ ptr, ptr } { ptr @"__llgo_stub.main.main$2", ptr null })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	// CHECK-LABEL: define void @"main.main$1"({ ptr, ptr } %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
	// CHECK-NEXT:   store { ptr, ptr } %0, ptr %1, align 8
	// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
	// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
	// CHECK-NEXT:   store ptr %1, ptr %3, align 8
	// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"main.main$1$1", ptr undef }, ptr %2, 1
	// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %4, 1
	// CHECK-NEXT:   %6 = extractvalue { ptr, ptr } %4, 0
	// CHECK-NEXT:   call void %6(ptr %5, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
	func(resolve func(error)) {

		// CHECK-LABEL: define void @"main.main$1$1"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
		// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %1)
		// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %1, 1
		// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %3, 0
		// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %5, ptr %4, 1
		// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
		// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %7, 0
		// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %8, ptr null, 1
		// CHECK-NEXT:   %10 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %6, %"{{.*}}/runtime/internal/runtime.eface" %9)
		// CHECK-NEXT:   %11 = xor i1 %10, true
		// CHECK-NEXT:   br i1 %11, label %_llgo_1, label %_llgo_2
		// CHECK-EMPTY:
		// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
		// CHECK-NEXT:   %12 = extractvalue { ptr } %2, 0
		// CHECK-NEXT:   %13 = icmp eq ptr %12, null
		// CHECK-NEXT:   br i1 %13, label %16, label %17
		// CHECK-EMPTY:
		// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
		// CHECK-NEXT:   %14 = extractvalue { ptr } %2, 0
		// CHECK-NEXT:   %15 = icmp eq ptr %14, null
		// CHECK-NEXT:   br i1 %15, label %21, label %22
		// CHECK-EMPTY:
		// CHECK-NEXT: 16:                                               ; preds = %_llgo_1
		// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
		// CHECK-NEXT:   unreachable
		// CHECK-EMPTY:
		// CHECK-NEXT: 17:                                               ; preds = %_llgo_1
		// CHECK-NEXT:   %18 = load { ptr, ptr }, ptr %12, align 8
		// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %18, 1
		// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %18, 0
		// CHECK-NEXT:   call void %20(ptr %19, %"{{.*}}/runtime/internal/runtime.iface" %1)
		// CHECK-NEXT:   ret void
		// CHECK-EMPTY:
		// CHECK-NEXT: 21:                                               ; preds = %_llgo_2
		// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
		// CHECK-NEXT:   unreachable
		// CHECK-EMPTY:
		// CHECK-NEXT: 22:                                               ; preds = %_llgo_2
		// CHECK-NEXT:   %23 = load { ptr, ptr }, ptr %14, align 8
		// CHECK-NEXT:   %24 = extractvalue { ptr, ptr } %23, 1
		// CHECK-NEXT:   %25 = extractvalue { ptr, ptr } %23, 0
		// CHECK-NEXT:   call void %25(ptr %24, %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
		// CHECK-NEXT:   ret void
		// CHECK-NEXT: }

		// CHECK-LABEL: define void @"main.main$2"(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   ret void
		// CHECK-NEXT: }
		func(err error) {
			if err != nil {
				resolve(err)
				return
			}
			resolve(nil)
		}(nil)
	}(func(err error) {
	})
}

// CHECK-LABEL: define linkonce void @"__llgo_stub.main.main$2"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"main.main$2"(%"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

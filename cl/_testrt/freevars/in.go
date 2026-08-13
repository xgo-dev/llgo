// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"main.main$1"({ ptr, ptr } { ptr @"main.main$2", ptr null })
func main() {
	// CHECK-LABEL: define void @"main.main$1"({ ptr, ptr } %0){{.*}} {
	// CHECK: [[RESOLVE_ADDR:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
	// CHECK-NEXT: store { ptr, ptr } %0, ptr [[RESOLVE_ADDR]]
	// CHECK-NEXT: [[ENV:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
	// CHECK-NEXT: [[CAPTURE:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[ENV]], i32 0, i32 0
	// CHECK-NEXT: store ptr [[RESOLVE_ADDR]], ptr [[CAPTURE]]
	// CHECK-NEXT: [[INNER:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.main$1$1", ptr undef }, ptr [[ENV]], 1
	// CHECK-NEXT: [[INNER_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[INNER]], 1
	// CHECK-NEXT: [[INNER_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[INNER]], 0
	// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[INNER_RAW_CODE]])
	// CHECK-NEXT: call void %__llgo_funcval_code(ptr swiftself [[INNER_ENV]], %"{{.*}}iface" zeroinitializer)
	func(resolve func(error)) {

		// CHECK-LABEL: define void @"main.main$1$1"(ptr swiftself %0, %"{{.*}}iface" %1){{.*}} {
		// CHECK: [[CAPTURED:%[0-9]+]] = load { ptr }, ptr %0
		// CHECK-NEXT: [[ERR_TYPE:%[0-9]+]] = call ptr @"{{.*}}IfaceType"(%"{{.*}}iface" %1)
		// CHECK-NEXT: [[ERR_DATA:%[0-9]+]] = extractvalue %"{{.*}}iface" %1, 1
		// CHECK-NEXT: [[ERR_EFACE0:%[0-9]+]] = insertvalue %"{{.*}}eface" undef, ptr [[ERR_TYPE]], 0
		// CHECK-NEXT: [[ERR_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" [[ERR_EFACE0]], ptr [[ERR_DATA]], 1
		// CHECK: [[NIL_TYPE:%[0-9]+]] = call ptr @"{{.*}}IfaceType"(%"{{.*}}iface" zeroinitializer)
		// CHECK-NEXT: [[NIL_EFACE0:%[0-9]+]] = insertvalue %"{{.*}}eface" undef, ptr [[NIL_TYPE]], 0
		// CHECK-NEXT: [[NIL_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" [[NIL_EFACE0]], ptr null, 1
		// CHECK-NEXT: [[IS_NIL:%[0-9]+]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[ERR_EFACE]], %"{{.*}}eface" [[NIL_EFACE]])
		// CHECK-NEXT: [[HAS_ERR:%[0-9]+]] = xor i1 [[IS_NIL]], true
		// CHECK-NEXT: br i1 [[HAS_ERR]], label %{{[^,]+}}, label %{{[^ ]+}}
		// CHECK: [[RESOLVE_PTR:%[0-9]+]] = extractvalue { ptr } [[CAPTURED]], 0
		// CHECK-NEXT: [[RESOLVE:%[0-9]+]] = load { ptr, ptr }, ptr [[RESOLVE_PTR]]
		// CHECK-NEXT: [[RESOLVE_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[RESOLVE]], 1
		// CHECK-NEXT: [[RESOLVE_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[RESOLVE]], 0
		// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[RESOLVE_RAW_CODE]])
		// CHECK-NEXT: call void %__llgo_funcval_code(ptr swiftself [[RESOLVE_ENV]], %"{{.*}}iface" %1)
		// CHECK: [[NIL_RESOLVE_PTR:%[0-9]+]] = extractvalue { ptr } [[CAPTURED]], 0
		// CHECK-NEXT: [[NIL_RESOLVE:%[0-9]+]] = load { ptr, ptr }, ptr [[NIL_RESOLVE_PTR]]
		// CHECK-NEXT: [[NIL_RESOLVE_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[NIL_RESOLVE]], 1
		// CHECK-NEXT: [[NIL_RESOLVE_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[NIL_RESOLVE]], 0
		// CHECK-NEXT: %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr [[NIL_RESOLVE_RAW_CODE]])
		// CHECK-NEXT: call void %__llgo_funcval_code1(ptr swiftself [[NIL_RESOLVE_ENV]], %"{{.*}}iface" zeroinitializer)

		// CHECK-LABEL: define void @"main.main$2"(%"{{.*}}iface" %0){{.*}} {
		// CHECK: ret void
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

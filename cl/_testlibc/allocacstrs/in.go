// LITTEST
package main

import "github.com/goplus/lib/c"

// AllocaCStrs builds a stack-resident, null-terminated pointer array and copies
// every Go string into stack storage.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[INPUT_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[INPUT:%[0-9]+]], 1
// CHECK-NEXT: [[CSTRS_LEN:%[0-9]+]] = add i64 [[INPUT_LEN]], 1
// CHECK-NEXT: [[CSTRS:%[0-9]+]] = alloca ptr, i64 [[CSTRS_LEN]], align 8
// CHECK: [[READ_INDEX:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ [[NEXT_READ_INDEX:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[READ_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[CSTRS]], i64 [[READ_INDEX]]
// CHECK-NEXT: [[READ_CSTR:%[0-9]+]] = load ptr, ptr [[READ_SLOT]]
// CHECK-NEXT: [[AT_END:%[0-9]+]] = icmp eq ptr [[READ_CSTR]], null
// CHECK-NEXT: br i1 [[AT_END]], label %{{.*}}, label %{{.*}}
// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[READ_CSTR]])
// CHECK-NEXT: [[NEXT_READ_INDEX]] = add i64 [[READ_INDEX]], 1
// CHECK: [[COPY_INDEX:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ [[NEXT_COPY_INDEX:%[0-9]+]], %{{.*}} ]
// CHECK: [[COPY_MORE:%[0-9]+]] = icmp slt i64 [[COPY_INDEX]], [[INPUT_LEN]]
// CHECK: [[GO_STRING:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr %{{[0-9]+}}
// CHECK: [[CSTR_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[CSTRS]], i64 [[COPY_INDEX]]
// CHECK: [[GO_STRING_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.String" [[GO_STRING]], 1
// CHECK-NEXT: [[CSTR_LEN:%[0-9]+]] = add i64 [[GO_STRING_LEN]], 1
// CHECK-NEXT: [[CSTR_BUF:%[0-9]+]] = alloca i8, i64 [[CSTR_LEN]], align 1
// CHECK-NEXT: [[CSTR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.CStrCopy"(ptr [[CSTR_BUF]], %"{{.*}}/runtime/internal/runtime.String" [[GO_STRING]])
// CHECK-NEXT: store ptr [[CSTR]], ptr [[CSTR_SLOT]]
// CHECK-NEXT: [[NEXT_COPY_INDEX]] = add i64 [[COPY_INDEX]], 1
// CHECK: [[TERMINATOR:%[0-9]+]] = getelementptr ptr, ptr [[CSTRS]], i64 [[INPUT_LEN]]
// CHECK-NEXT: store ptr null, ptr [[TERMINATOR]]

func main() {
	cstrs := c.AllocaCStrs([]string{"a", "b", "c"}, true)
	n := 0
	for {
		cstr := *c.Advance(cstrs, n)
		if cstr == nil {
			break
		}
		c.Printf(c.Str("%s\n"), cstr)
		n++
	}
}

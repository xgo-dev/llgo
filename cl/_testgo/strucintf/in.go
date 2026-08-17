// LITTEST
package main

import "github.com/xgo-dev/llgo/cl/_testdata/foo"

// CHECK-LABEL: define %"{{.*}}eface" @main.Foo(){{.*}} {
// CHECK: store i64 1, ptr %{{[0-9]+}}
// CHECK: [[FOO_STRUCT:%[0-9]+]] = load { i64 }, ptr %{{[0-9]+}}
// CHECK-NEXT: [[FOO_BOX:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: store { i64 } [[FOO_STRUCT]], ptr [[FOO_BOX]]
// CHECK-NEXT: [[FOO_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr [[LOCAL_STRUCT:@"[^"]*/cl/_testgo/strucintf\.struct\$[^"]+"]], ptr undef }, ptr [[FOO_BOX]], 1
// CHECK-NEXT: ret %"{{.*}}eface" [[FOO_EFACE]]
func Foo() any {
	return struct{ v int }{1}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: [[LOCAL_VALUE:%[0-9]+]] = call %"{{.*}}eface" @main.Foo()
	// CHECK: [[LOCAL_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[LOCAL_VALUE]], 0
	// CHECK-NEXT: [[LOCAL_MATCH:%[0-9]+]] = icmp eq ptr [[LOCAL_TYPE]], [[LOCAL_STRUCT]]
	// CHECK-NEXT: br i1 [[LOCAL_MATCH]], label %{{.*}}, label %{{.*}}
	v := Foo()

	if x, ok := v.(struct{ v int }); ok {
		println(x.v)
	} else {
		println("Foo: not ok")
	}

	// CHECK: [[BAR_VALUE:%[0-9]+]] = call %"{{.*}}eface" @"{{.*}}foo.Bar"()
	// CHECK: [[BAR_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[BAR_VALUE]], 0
	// CHECK-NEXT: [[BAR_MATCH:%[0-9]+]] = icmp eq ptr [[BAR_TYPE]], [[BAR_STRUCT:@"_llgo_struct\$[^"]+"]]
	// CHECK-NEXT: br i1 [[BAR_MATCH]], label %{{.*}}, label %{{.*}}
	bar := foo.Bar()

	if x, ok := bar.(struct{ V int }); ok {
		println(x.V)
	} else {
		println("Bar: not ok")
	}

	// CHECK: [[F_VALUE:%[0-9]+]] = call %"{{.*}}eface" @"{{.*}}foo.F"()
	// CHECK-NEXT: [[F_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[F_VALUE]], 0
	// CHECK-NEXT: [[F_MATCH:%[0-9]+]] = icmp eq ptr [[F_TYPE]], [[LOCAL_STRUCT]]
	// CHECK-NEXT: br i1 [[F_MATCH]], label %{{.*}}, label %{{.*}}
	if x, ok := foo.F().(struct{ v int }); ok {
		println(x.v)
	} else {
		println("F: not ok")
	}
}

// CHECK: [[LOCAL_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[LOCAL_VALUE]], 1
// CHECK-NEXT: [[LOCAL_STRUCT_VALUE:%[0-9]+]] = load { i64 }, ptr [[LOCAL_DATA]]
// CHECK: [[LOCAL_OK:%[0-9]+]] = extractvalue { { i64 }, i1 } %{{[0-9]+}}, 1
// CHECK-NEXT: br i1 [[LOCAL_OK]], label %{{.*}}, label %{{.*}}
// CHECK: [[BAR_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[BAR_VALUE]], 1
// CHECK-NEXT: [[BAR_STRUCT_VALUE:%[0-9]+]] = load { i64 }, ptr [[BAR_DATA]]
// CHECK: [[BAR_OK:%[0-9]+]] = extractvalue { { i64 }, i1 } %{{[0-9]+}}, 1
// CHECK-NEXT: br i1 [[BAR_OK]], label %{{.*}}, label %{{.*}}
// CHECK: [[F_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[F_VALUE]], 1
// CHECK-NEXT: [[F_STRUCT_VALUE:%[0-9]+]] = load { i64 }, ptr [[F_DATA]]
// CHECK: [[F_OK:%[0-9]+]] = extractvalue { { i64 }, i1 } %{{[0-9]+}}, 1
// CHECK-NEXT: br i1 [[F_OK]], label %{{.*}}, label %{{.*}}

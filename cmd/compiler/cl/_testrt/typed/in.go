// LITTEST darwin/amd64 darwin/arm64 linux/386 linux/amd64 linux/arm64 windows/386 windows/amd64 windows/arm64
// Scope: common
package main

type T string
type A [2]int
type MyBytes []byte

func main() {
	var v any = T("hello")
	println(v.(T))
	s, ok := v.(string)
	println(s, ok)

	var a any = A{1, 2}
	ar, ok := a.(A)
	println(ar[0], ar[1], ok)

	var bytes any = MyBytes{}
	_, ok = bytes.(MyBytes)
	println(ok)
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// The named string uses an exact assertion folded directly to its boxed value,
// then statically rejects a comma-ok assertion to the underlying string type.
// CHECK: [[T_VALUE:%.*]] = load %"{{.*}}String", ptr @"_llgo_ifacebox${{.*}}"
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[T_VALUE]])
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" zeroinitializer)
// CHECK: call void @"{{.*}}PrintBool"(i1 false)
// The named array is boxed with its named descriptor, copied out, and returned
// together with the comma-ok bit used by the print.
// CHECK: store [2 x {{i[0-9]+}}] {{%.*}}, ptr [[A_BOX:%[0-9]+]]
// CHECK: [[A_EFACE:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_main.A, ptr undef }, ptr [[A_BOX]], 1
// CHECK: [[A_DYN_DATA:%.*]] = extractvalue %"{{.*}}eface" [[A_EFACE]], 1
// CHECK-NEXT: [[A_VALUE:%.*]] = load [2 x {{i[0-9]+}}], ptr [[A_DYN_DATA]]
// CHECK: [[A_OK_RESULT0:%.*]] = insertvalue { [2 x {{i[0-9]+}}], i1 } undef, [2 x {{i[0-9]+}}] [[A_VALUE]], 0
// CHECK-NEXT: [[A_OK_RESULT:%.*]] = insertvalue { [2 x {{i[0-9]+}}], i1 } [[A_OK_RESULT0]], i1 true, 1
// CHECK: [[A_RESULT:%.*]] = extractvalue { [2 x {{i[0-9]+}}], i1 } [[A_OK_RESULT]], 0
// CHECK: [[A_OK:%.*]] = extractvalue { [2 x {{i[0-9]+}}], i1 } [[A_OK_RESULT]], 1
// CHECK: [[A0:%.*]] = load i{{[0-9]+}}, ptr {{%.*}}
// CHECK: [[A1:%.*]] = load i{{[0-9]+}}, ptr {{%.*}}
// CHECK: call void @"{{.*}}PrintInt"(i64 {{.*}})
// CHECK: call void @"{{.*}}PrintInt"(i64 {{.*}})
// CHECK: call void @"{{.*}}PrintBool"(i1 [[A_OK]])
// A named slice uses its named descriptor while the asserted payload retains
// the ordinary three-word slice representation.
// CHECK-DARWIN: load %"{{.*}}Slice", ptr @"_llgo_ifacebox${{.*}}"
// CHECK-LINUX: load %"{{.*}}Slice", ptr @"_llgo_ifacebox${{.*}}"
// CHECK-WINDOWS: [[BYTES_BOX:%.*]] = call ptr @"{{.*}}AllocU"(i{{[0-9]+}} {{[0-9]+}})
// CHECK-WINDOWS-NEXT: store %"{{.*}}Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i{{[0-9]+}} 0, i{{[0-9]+}} 0 }, ptr [[BYTES_BOX]]
// CHECK-WINDOWS: load %"{{.*}}Slice", ptr [[BYTES_BOX]]
// CHECK: [[BYTES_RESULT0:%.*]] = insertvalue { %"{{.*}}Slice", i1 } undef, %"{{.*}}Slice" %{{.*}}, 0
// CHECK-NEXT: [[BYTES_RESULT:%.*]] = insertvalue { %"{{.*}}Slice", i1 } [[BYTES_RESULT0]], i1 true, 1
// CHECK: [[BYTES_OK:%.*]] = extractvalue { %"{{.*}}Slice", i1 } [[BYTES_RESULT]], 1
// CHECK: call void @"{{.*}}PrintBool"(i1 [[BYTES_OK]])

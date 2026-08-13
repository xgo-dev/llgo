// LITTEST
package main

type T string
type A [2]int

func main() {
	var v any = T("hello")
	println(v.(T))
	s, ok := v.(string)
	println(s, ok)

	var a any = A{1, 2}
	ar, ok := a.(A)
	println(ar[0], ar[1], ok)
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// The named string uses an exact (panicking) assertion, then deliberately fails
// a comma-ok assertion to the underlying unnamed string type.
// CHECK: [[T_DATA:%.*]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK: [[T_EFACE:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_main.T, ptr undef }, ptr [[T_DATA]], 1
// CHECK: [[T_DYN_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[T_EFACE]], 0
// CHECK-NEXT: [[IS_T:%.*]] = icmp eq ptr [[T_DYN_TYPE]], @_llgo_main.T
// CHECK: br i1 [[IS_T]], label %{{.*}}, label %{{.*}}
// CHECK: [[T_DYN_DATA:%.*]] = extractvalue %"{{.*}}eface" [[T_EFACE]], 1
// CHECK-NEXT: [[T_VALUE:%.*]] = load %"{{.*}}String", ptr [[T_DYN_DATA]]
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[T_VALUE]])
// CHECK: [[STRING_DYN_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[T_EFACE]], 0
// CHECK-NEXT: [[IS_STRING:%.*]] = icmp eq ptr [[STRING_DYN_TYPE]], @_llgo_string
// CHECK: call void @"{{.*}}PanicTypeAssert"(ptr null, ptr [[T_DYN_TYPE]], ptr @_llgo_main.T)
// CHECK-NEXT: unreachable
// CHECK: [[STRING_OK_RESULT:%.*]] = phi { %"{{.*}}String", i1 } [ {{%.*}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK: [[STRING_VALUE:%.*]] = extractvalue { %"{{.*}}String", i1 } [[STRING_OK_RESULT]], 0
// CHECK-NEXT: [[STRING_OK:%.*]] = extractvalue { %"{{.*}}String", i1 } [[STRING_OK_RESULT]], 1
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[STRING_VALUE]])
// CHECK: call void @"{{.*}}PrintBool"(i1 [[STRING_OK]])
// The named array is boxed, matched by its named type, copied out, and returned
// together with the comma-ok bit used by the print.
// CHECK: [[A_DATA:%.*]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK: [[A_EFACE:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_main.A, ptr undef }, ptr [[A_DATA]], 1
// CHECK: [[A_DYN_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[A_EFACE]], 0
// CHECK-NEXT: [[IS_A:%.*]] = icmp eq ptr [[A_DYN_TYPE]], @_llgo_main.A
// CHECK: [[A_DYN_DATA:%.*]] = extractvalue %"{{.*}}eface" [[A_EFACE]], 1
// CHECK-NEXT: [[A_VALUE:%.*]] = load [2 x i64], ptr [[A_DYN_DATA]]
// CHECK: [[A_OK_RESULT:%.*]] = phi { [2 x i64], i1 } [ {{%.*}}, %{{.*}} ], [ zeroinitializer, %{{.*}} ]
// CHECK: [[A_RESULT:%.*]] = extractvalue { [2 x i64], i1 } [[A_OK_RESULT]], 0
// CHECK: [[A_OK:%.*]] = extractvalue { [2 x i64], i1 } [[A_OK_RESULT]], 1
// CHECK: [[A0:%.*]] = load i64, ptr {{%.*}}
// CHECK: [[A1:%.*]] = load i64, ptr {{%.*}}
// CHECK: call void @"{{.*}}PrintInt"(i64 [[A0]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[A1]])
// CHECK: call void @"{{.*}}PrintBool"(i1 [[A_OK]])

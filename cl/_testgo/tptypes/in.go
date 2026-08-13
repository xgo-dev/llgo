// LITTEST
package main

type Data[T any] struct {
	v T
}

func (p *Data[T]) Set(v T) {
	p.v = v
}

func (p *(Data[T1])) Set2(v T1) {
	p.v = v
}

type sliceOf[E any] interface {
	~[]E
}

type Slice[S sliceOf[T], T any] struct {
	Data S
}

func (p *Slice[S, T]) Append(t ...T) S {
	p.Data = append(p.Data, t...)
	return p.Data
}

func (p *Slice[S1, T1]) Append2(t ...T1) S1 {
	p.Data = append(p.Data, t...)
	return p.Data
}

type (
	DataInt     = Data[int]
	SliceInt    = Slice[[]int, int]
	DataString  = Data[string]
	SliceString = Slice[[]string, string]
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// Type aliases and direct instantiations retain their concrete field types.
// CHECK: store i64 1, ptr {{%.*}}
// CHECK: [[ALIAS_INT:%.*]] = extractvalue %"main.Data[int]" {{%.*}}, 0
// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[ALIAS_INT]])
// CHECK: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 }, ptr {{%.*}}
// CHECK: [[ALIAS_STRING:%.*]] = extractvalue %"main.Data[string]" {{%.*}}, 0
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[ALIAS_STRING]])
// CHECK: store i64 100, ptr {{%.*}}
// CHECK: [[DIRECT_INT:%.*]] = extractvalue %"main.Data[int]" {{%.*}}, 0
// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[DIRECT_INT]])
// CHECK: [[DIRECT_STRING:%.*]] = extractvalue %"main.Data[string]" {{%.*}}, 0
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[DIRECT_STRING]])
// CHECK: call void @"{{.*}}PrintInt"(i64 0)
// SliceInt.Append receives a one-element []int containing 100.
// CHECK: [[V1:%.*]] = call ptr @"{{.*}}AllocZ"(i64 24)
// CHECK: [[V1_ARG_DATA:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: store i64 100, ptr {{%.*}}
// CHECK: [[V1_ARG0:%.*]] = insertvalue %"{{.*}}Slice" undef, ptr [[V1_ARG_DATA]], 0
// CHECK-NEXT: [[V1_ARG1:%.*]] = insertvalue %"{{.*}}Slice" [[V1_ARG0]], i64 1, 1
// CHECK-NEXT: [[V1_ARG:%.*]] = insertvalue %"{{.*}}Slice" [[V1_ARG1]], i64 1, 2
// CHECK-NEXT: call %"{{.*}}Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr [[V1]], %"{{.*}}Slice" [[V1_ARG]])
// SliceString.Append receives one string element with the 16-byte stride.
// CHECK: [[V2:%.*]] = call ptr @"{{.*}}AllocZ"(i64 24)
// CHECK: [[V2_ARG_DATA:%.*]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 }, ptr {{%.*}}
// CHECK: [[V2_ARG0:%.*]] = insertvalue %"{{.*}}Slice" undef, ptr [[V2_ARG_DATA]], 0
// CHECK-NEXT: [[V2_ARG1:%.*]] = insertvalue %"{{.*}}Slice" [[V2_ARG0]], i64 1, 1
// CHECK-NEXT: [[V2_ARG:%.*]] = insertvalue %"{{.*}}Slice" [[V2_ARG1]], i64 1, 2
// CHECK-NEXT: call %"{{.*}}Slice" @"main.(*Slice{{\[\[\]string,string\]}}).Append"(ptr [[V2]], %"{{.*}}Slice" [[V2_ARG]])
// The direct v3 instantiation is updated twice through Append and Append2.
// CHECK: [[V3:%.*]] = call ptr @"{{.*}}AllocZ"(i64 24)
// CHECK: [[V3_APPEND_DATA:%.*]] = call ptr @"{{.*}}AllocZ"(i64 32)
// CHECK: [[V3_APPEND_ARG0:%.*]] = insertvalue %"{{.*}}Slice" undef, ptr [[V3_APPEND_DATA]], 0
// CHECK-NEXT: [[V3_APPEND_ARG1:%.*]] = insertvalue %"{{.*}}Slice" [[V3_APPEND_ARG0]], i64 4, 1
// CHECK-NEXT: [[V3_APPEND_ARG:%.*]] = insertvalue %"{{.*}}Slice" [[V3_APPEND_ARG1]], i64 4, 2
// CHECK-NEXT: call %"{{.*}}Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr [[V3]], %"{{.*}}Slice" [[V3_APPEND_ARG]])
// CHECK: [[V3_APPEND2_DATA:%.*]] = call ptr @"{{.*}}AllocZ"(i64 32)
// CHECK: [[V3_APPEND2_ARG0:%.*]] = insertvalue %"{{.*}}Slice" undef, ptr [[V3_APPEND2_DATA]], 0
// CHECK-NEXT: [[V3_APPEND2_ARG1:%.*]] = insertvalue %"{{.*}}Slice" [[V3_APPEND2_ARG0]], i64 4, 1
// CHECK-NEXT: [[V3_APPEND2_ARG:%.*]] = insertvalue %"{{.*}}Slice" [[V3_APPEND2_ARG1]], i64 4, 2
// CHECK-NEXT: call %"{{.*}}Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append2"(ptr [[V3]], %"{{.*}}Slice" [[V3_APPEND2_ARG]])

func main() {
	println(DataInt{1}.v)
	println(DataString{"hello"}.v)
	println(Data[int]{100}.v)
	println(Data[string]{"hello"}.v)

	// TODO
	println(Data[struct {
		X int
		Y int
	}]{}.v.X)

	v1 := SliceInt{}
	v1.Append(100)
	v2 := SliceString{}
	v2.Append("hello")
	v3 := Slice[[]int, int]{}
	v3.Append([]int{1, 2, 3, 4}...)
	v3.Append2([]int{1, 2, 3, 4}...)

	println(v1.Data, v1.Data[0])
	println(v2.Data, v2.Data[0])
	println(v3.Data, v3.Data[0])
}

// CHECK-LABEL: define linkonce %"{{.*}}Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr %0, %"{{.*}}Slice" %1){{.*}} {
// CHECK: [[INT_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[INT_OLD:%.*]] = load %"{{.*}}Slice", ptr [[INT_FIELD]]
// CHECK-NEXT: [[INT_DATA:%.*]] = extractvalue %"{{.*}}Slice" %1, 0
// CHECK-NEXT: [[INT_LEN:%.*]] = extractvalue %"{{.*}}Slice" %1, 1
// CHECK-NEXT: [[INT_NEW:%.*]] = call %"{{.*}}Slice" @"{{.*}}SliceAppend"(%"{{.*}}Slice" [[INT_OLD]], ptr [[INT_DATA]], i64 [[INT_LEN]], i64 8)
// CHECK: [[INT_STORE_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: store %"{{.*}}Slice" [[INT_NEW]], ptr [[INT_STORE_FIELD]]
// CHECK-NEXT: [[INT_RESULT_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[INT_RESULT:%.*]] = load %"{{.*}}Slice", ptr [[INT_RESULT_FIELD]]
// CHECK-NEXT: ret %"{{.*}}Slice" [[INT_RESULT]]

// CHECK-LABEL: define linkonce %"{{.*}}Slice" @"main.(*Slice{{\[\[\]string,string\]}}).Append"(ptr %0, %"{{.*}}Slice" %1){{.*}} {
// CHECK: [[STRING_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[STRING_OLD:%.*]] = load %"{{.*}}Slice", ptr [[STRING_FIELD]]
// CHECK-NEXT: [[STRING_DATA:%.*]] = extractvalue %"{{.*}}Slice" %1, 0
// CHECK-NEXT: [[STRING_LEN:%.*]] = extractvalue %"{{.*}}Slice" %1, 1
// CHECK-NEXT: [[STRING_NEW:%.*]] = call %"{{.*}}Slice" @"{{.*}}SliceAppend"(%"{{.*}}Slice" [[STRING_OLD]], ptr [[STRING_DATA]], i64 [[STRING_LEN]], i64 16)
// CHECK: [[STRING_STORE_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: store %"{{.*}}Slice" [[STRING_NEW]], ptr [[STRING_STORE_FIELD]]
// CHECK-NEXT: [[STRING_RESULT_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[STRING_RESULT:%.*]] = load %"{{.*}}Slice", ptr [[STRING_RESULT_FIELD]]
// CHECK-NEXT: ret %"{{.*}}Slice" [[STRING_RESULT]]

// CHECK-LABEL: define linkonce %"{{.*}}Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append2"(ptr %0, %"{{.*}}Slice" %1){{.*}} {
// CHECK: [[APPEND2_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[APPEND2_OLD:%.*]] = load %"{{.*}}Slice", ptr [[APPEND2_FIELD]]
// CHECK-NEXT: [[APPEND2_DATA:%.*]] = extractvalue %"{{.*}}Slice" %1, 0
// CHECK-NEXT: [[APPEND2_LEN:%.*]] = extractvalue %"{{.*}}Slice" %1, 1
// CHECK-NEXT: [[APPEND2_NEW:%.*]] = call %"{{.*}}Slice" @"{{.*}}SliceAppend"(%"{{.*}}Slice" [[APPEND2_OLD]], ptr [[APPEND2_DATA]], i64 [[APPEND2_LEN]], i64 8)
// CHECK: [[APPEND2_STORE_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: store %"{{.*}}Slice" [[APPEND2_NEW]], ptr [[APPEND2_STORE_FIELD]]
// CHECK-NEXT: [[APPEND2_RESULT_FIELD:%.*]] = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[APPEND2_RESULT:%.*]] = load %"{{.*}}Slice", ptr [[APPEND2_RESULT_FIELD]]
// CHECK-NEXT: ret %"{{.*}}Slice" [[APPEND2_RESULT]]

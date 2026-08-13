// LITTEST
package main

type Tuple[T any] struct {
	v T
}

func (t Tuple[T]) Get() T {
	return t.v
}

type Future[T any] interface {
	Then(func(T))
}

type future[T any] struct {
	fn func(func(T))
}

func (f *future[T]) Then(callback func(T)) {
	f.fn(callback)
}

func Async[T any](fn func(func(T))) Future[T] {
	return &future[T]{fn: fn}
}

// CHECK-LABEL: define %"{{.*}}iface" @main.ReadFile(%"{{.*}}String" %0){{.*}} {
// CHECK: [[FUTURE:%[0-9]+]] = call %"{{.*}}iface" @"main.Async[main.Tuple[error]]"({ ptr, ptr } { ptr @"main.ReadFile$1", ptr null })
// CHECK-NEXT: ret %"{{.*}}iface" [[FUTURE]]

func ReadFile(fileName string) Future[Tuple[error]] {
	// CHECK-LABEL: define void @"main.ReadFile$1"({ ptr, ptr } %0){{.*}} {
	// CHECK: [[TUPLE_FIELD:%[0-9]+]] = getelementptr inbounds %"main.Tuple[error]", ptr [[TUPLE_ADDR:%[0-9]+]], i32 0, i32 0
	// CHECK-NEXT: store %"{{.*}}iface" zeroinitializer, ptr [[TUPLE_FIELD]]
	// CHECK-NEXT: [[TUPLE:%[0-9]+]] = load %"main.Tuple[error]", ptr [[TUPLE_ADDR]]
	// CHECK-NEXT: [[RESOLVE_ENV:%[0-9]+]] = extractvalue { ptr, ptr } %0, 1
	// CHECK-NEXT: [[RESOLVE_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } %0, 0
	// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[RESOLVE_RAW_CODE]])
	// CHECK-NEXT: call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[RESOLVE_ENV]], %"main.Tuple[error]" [[TUPLE]])

	return Async[Tuple[error]](func(resolve func(Tuple[error])) {
		resolve(Tuple[error]{v: nil})
	})
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[READ:%[0-9]+]] = call %"{{.*}}iface" @main.ReadFile(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 7 })
// CHECK-NEXT: [[READ_DATA:%[0-9]+]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" [[READ]])
// CHECK-NEXT: [[READ_ITAB:%[0-9]+]] = extractvalue %"{{.*}}iface" [[READ]], 0
// CHECK-NEXT: [[THEN_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[READ_ITAB]], i64 3
// CHECK-NEXT: [[THEN_CODE:%[0-9]+]] = load ptr, ptr [[THEN_SLOT]]
// CHECK-NEXT: [[THEN_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[THEN_CODE]], 0
// CHECK-NEXT: [[THEN_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[THEN_PAIR0]], ptr [[READ_DATA]], 1
// CHECK-NEXT: [[THEN_RECEIVER:%[0-9]+]] = extractvalue { ptr, ptr } [[THEN_PAIR]], 1
// CHECK-NEXT: [[THEN_CALL:%[0-9]+]] = extractvalue { ptr, ptr } [[THEN_PAIR]], 0
// CHECK-NEXT: call void [[THEN_CALL]](ptr [[THEN_RECEIVER]], { ptr, ptr } { ptr @"main.main$1", ptr null })

func main() {
	// CHECK-LABEL: define void @"main.main$1"(%"main.Tuple[error]" %0){{.*}} {
	// CHECK: [[ERROR:%[0-9]+]] = call %"{{.*}}iface" @"main.Tuple[error].Get"(%"main.Tuple[error]" %0)
	// CHECK-NEXT: call void @"{{.*}}PrintIface"(%"{{.*}}iface" [[ERROR]])

	ReadFile("foo.txt").Then(func(v Tuple[error]) {
		println(v.Get())
	})
}

// CHECK-LABEL: define linkonce %"{{.*}}iface" @"main.Async[main.Tuple[error]]"({ ptr, ptr } %0){{.*}} {
// CHECK: [[FUTURE_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT: [[FUTURE_FN:%[0-9]+]] = getelementptr inbounds %"main.future[main.Tuple[error]]", ptr [[FUTURE_DATA]], i32 0, i32 0
// CHECK-NEXT: store { ptr, ptr } %0, ptr [[FUTURE_FN]]
// CHECK-NEXT: [[FUTURE_ITAB:%[0-9]+]] = call ptr @"{{.*}}NewItab"(ptr {{.*}}, ptr @"*_llgo_main.future[main.Tuple[error]]")
// CHECK-NEXT: [[FUTURE_IFACE0:%[0-9]+]] = insertvalue %"{{.*}}iface" undef, ptr [[FUTURE_ITAB]], 0
// CHECK-NEXT: [[FUTURE_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" [[FUTURE_IFACE0]], ptr [[FUTURE_DATA]], 1
// CHECK-NEXT: ret %"{{.*}}iface" [[FUTURE_IFACE]]

// CHECK-LABEL: define linkonce %"{{.*}}iface" @"main.Tuple[error].Get"(%"main.Tuple[error]" %0){{.*}} {
// CHECK: store %"main.Tuple[error]" %0, ptr [[GET_ADDR:%[0-9]+]]
// CHECK-NEXT: [[GET_FIELD:%[0-9]+]] = getelementptr inbounds %"main.Tuple[error]", ptr [[GET_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[GET_VALUE:%[0-9]+]] = load %"{{.*}}iface", ptr [[GET_FIELD]]
// CHECK-NEXT: ret %"{{.*}}iface" [[GET_VALUE]]

// CHECK-LABEL: define linkonce void @"main.(*future[main.Tuple[error]]).Then"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK: [[FN_FIELD:%[0-9]+]] = getelementptr inbounds %"main.future[main.Tuple[error]]", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[FN:%[0-9]+]] = load { ptr, ptr }, ptr [[FN_FIELD]]
// CHECK-NEXT: [[FN_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[FN]], 1
// CHECK-NEXT: [[FN_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[FN]], 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[FN_RAW_CODE]])
// CHECK-NEXT: call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[FN_ENV]], { ptr, ptr } %1)

// CHECK-LABEL: define linkonce %"{{.*}}iface" @"main.(*Tuple[error]).Get"(ptr %0){{.*}} {
// CHECK: [[GET_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK-NEXT: call void @"{{.*}}PanicWrapNilPointer"(i1 [[GET_NIL]],{{.*}})
// CHECK-NEXT: [[GET_RECEIVER:%[0-9]+]] = load %"main.Tuple[error]", ptr %0
// CHECK-NEXT: [[GET_RESULT:%[0-9]+]] = call %"{{.*}}iface" @"main.Tuple[error].Get"(%"main.Tuple[error]" [[GET_RECEIVER]])
// CHECK-NEXT: ret %"{{.*}}iface" [[GET_RESULT]]

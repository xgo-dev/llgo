// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [7 x i8] c"foo.txt", align 1{{$}}
// CHECK: {{^}}@6 = private unnamed_addr constant [17 x i8] c"main.Tuple[error]", align 1{{$}}
// CHECK: {{^}}@7 = private unnamed_addr constant [3 x i8] c"Get", align 1{{$}}

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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.ReadFile(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.iface" @"main.Async[main.Tuple[error]]"({ ptr, ptr } { ptr @"main.ReadFile$1", ptr null })
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %1
// CHECK-NEXT: }

func ReadFile(fileName string) Future[Tuple[error]] {
	// CHECK-LABEL: define void @"main.ReadFile$1"({ ptr, ptr } %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = alloca %"main.Tuple[error]", align 8
	// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
	// CHECK-NEXT:   %2 = getelementptr inbounds %"main.Tuple[error]", ptr %1, i32 0, i32 0
	// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, ptr %2, align 8
	// CHECK-NEXT:   %3 = load %"main.Tuple[error]", ptr %1, align 8
	// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %0, 1
	// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %0, 0
	// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %5)
	// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %4, %"main.Tuple[error]" %3)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	// CHECK-LABEL: define void @main.init(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
	// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
	// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
	// CHECK-NEXT:   br label %_llgo_2
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	return Async[Tuple[error]](func(resolve func(Tuple[error])) {
		resolve(Tuple[error]{v: nil})
	})
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.iface" @main.ReadFile(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 7 })
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %3 = getelementptr ptr, ptr %2, i64 3
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } %5, ptr %1, 1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %6, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %6, 0
// CHECK-NEXT:   call void %8(ptr %7, { ptr, ptr } { ptr @"main.main$1", ptr null })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	// CHECK-LABEL: define void @"main.main$1"(%"main.Tuple[error]" %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.iface" @"main.Tuple[error].Get"(%"main.Tuple[error]" %0)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %1)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	ReadFile("foo.txt").Then(func(v Tuple[error]) {
		println(v.Get())
	})
}

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"main.Async[main.Tuple[error]]"({ ptr, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds %"main.future[main.Tuple[error]]", ptr %1, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.future[main.Tuple[error]]")
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %3, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %4, ptr %1, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %5
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"main.Tuple[error].Get"(%"main.Tuple[error]" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"main.Tuple[error]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"main.Tuple[error]" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"main.Tuple[error]", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"main.(*future[main.Tuple[error]]).Then"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"main.future[main.Tuple[error]]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load { ptr, ptr }, ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %3, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %3, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %5)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %4, { ptr, ptr } %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"main.(*Tuple[error]).Get"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 55 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 3 })
// CHECK-NEXT:   %2 = load %"main.Tuple[error]", ptr %0, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"main.Tuple[error].Get"(%"main.Tuple[error]" %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

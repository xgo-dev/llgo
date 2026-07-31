// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [5 x i8] c"reset", align 1{{$}}
// CHECK: {{^}}@8 = private unnamed_addr constant [4 x i8] c"body", align 1{{$}}

type resetter interface {
	Reset()
}

type item struct {
	value int
}

func (p *item) Reset() {
	println("reset", p.value)
}

func run(v resetter) {
	defer v.Reset()
	println("body")
}

func main() {
	run(&item{42})
}

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

// CHECK-LABEL: define void @"main.(*item).Reset"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.item, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %main.item, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 42, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$yjH5fOWhYIH6Pv7ce-kmK-CVKIWOLvKPVzRvjwBotEM", ptr @"*_llgo_main.item")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   call void @main.run(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.run(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %3 = getelementptr ptr, ptr %2, i64 3
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } %5, ptr %1, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %8 = alloca i8, i64 196, align 1
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 0
// CHECK-NEXT:   store ptr %8, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 2
// CHECK-NEXT:   store ptr %7, ptr %12, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.run, %_llgo_2), ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %9)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 1
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 3
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 4
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %17, align 8
// CHECK-NEXT:   %18 = call i32 @sigsetjmp(ptr %8, i32 0)
// CHECK-NEXT:   %19 = icmp eq i32 %18, 0
// CHECK-NEXT:   br i1 %19, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.run, %_llgo_3), ptr %15, align 8
// CHECK-NEXT:   %20 = load i64, ptr %14, align 8
// CHECK-NEXT:   %21 = load ptr, ptr %17, align 8
// CHECK-NEXT:   %22 = icmp ne ptr %21, null
// CHECK-NEXT:   br i1 %22, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %7)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %23 = load ptr, ptr %17, align 8
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %24, i32 0, i32 0
// CHECK-NEXT:   store ptr %23, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %24, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %26, align 8
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %24, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %6, ptr %27, align 8
// CHECK-NEXT:   store ptr %24, ptr %17, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@main.run, %_llgo_6), ptr %16, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.run, %_llgo_3), ptr %16, align 8
// CHECK-NEXT:   %28 = load ptr, ptr %15, align 8
// CHECK-NEXT:   indirectbr ptr %28, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %29 = load ptr, ptr %17, align 8
// CHECK-NEXT:   %30 = load { ptr, i64, { ptr, ptr } }, ptr %29, align 8
// CHECK-NEXT:   %31 = extractvalue { ptr, i64, { ptr, ptr } } %30, 0
// CHECK-NEXT:   store ptr %31, ptr %17, align 8
// CHECK-NEXT:   %32 = extractvalue { ptr, i64, { ptr, ptr } } %30, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %29)
// CHECK-NEXT:   %33 = extractvalue { ptr, ptr } %32, 1
// CHECK-NEXT:   %34 = extractvalue { ptr, ptr } %32, 0
// CHECK-NEXT:   call void %34(ptr %33)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %35 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %9, align 8
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %35, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %36)
// CHECK-NEXT:   %37 = load ptr, ptr %16, align 8
// CHECK-NEXT:   indirectbr ptr %37, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

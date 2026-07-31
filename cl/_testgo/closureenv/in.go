// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [18 x i8] c"zero-sized capture", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [26 x i8] c"zero-sized capture address", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [26 x i8] c"zero-sized pointer capture", align 1{{$}}
// CHECK: {{^}}@6 = private unnamed_addr constant [5 x i8] c"IsNil", align 1{{$}}
// CHECK: {{^}}@10 = private unnamed_addr constant [23 x i8] c"interface{IsNil() bool}", align 1{{$}}
// CHECK: {{^}}@11 = private unnamed_addr constant [25 x i8] c"nil receiver method value", align 1{{$}}
// CHECK: {{^}}@12 = private unnamed_addr constant [2 x i8] c"ok", align 1{{$}}
// CHECK: {{^}}@13 = private unnamed_addr constant [32 x i8] c"typed-nil interface method value", align 1{{$}}

type nilReceiver struct{}

func (p *nilReceiver) IsNil() bool {
	return p == nil
}

func zeroSizedCapture() func() int {
	captured := struct{}{}
	return func() int {
		if captured != (struct{}{}) {
			return 0
		}
		return 42
	}
}

func zeroSizedAddressCapture() (func() *struct{}, *struct{}) {
	captured := struct{}{}
	return func() *struct{} { return &captured }, &captured
}

func zeroSizedPointerCapture(pointer *struct{}) func() bool {
	return func() bool { return pointer == nil }
}

func main() {
	if zeroSizedCapture()() != 42 {
		panic("zero-sized capture")
	}
	addressClosure, address := zeroSizedAddressCapture()
	if addressClosure() != address {
		panic("zero-sized capture address")
	}
	if !zeroSizedPointerCapture(nil)() {
		panic("zero-sized pointer capture")
	}

	var receiver *nilReceiver
	method := receiver.IsNil
	if !method() {
		panic("nil receiver method value")
	}

	var typedNil interface{ IsNil() bool } = receiver
	interfaceMethod := typedNil.IsNil
	if !interfaceMethod() {
		panic("typed-nil interface method value")
	}
	println("ok")
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

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call { ptr, ptr } @main.zeroSizedCapture()
// CHECK-NEXT:   %1 = extractvalue { ptr, ptr } %0, 1
// CHECK-NEXT:   %2 = extractvalue { ptr, ptr } %0, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %2)
// CHECK-NEXT:   %3 = call i64 %__llgo_funcval_code(ptr {{(nest|swiftself)}} %1)
// CHECK-NEXT:   %4 = icmp ne i64 %3, 42
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 18 }, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = call { { ptr, ptr }, ptr } @main.zeroSizedAddressCapture()
// CHECK-NEXT:   %8 = extractvalue { { ptr, ptr }, ptr } %7, 0
// CHECK-NEXT:   %9 = extractvalue { { ptr, ptr }, ptr } %7, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %8, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %8, 0
// CHECK-NEXT:   %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr %11)
// CHECK-NEXT:   %12 = call ptr %__llgo_funcval_code1(ptr {{(nest|swiftself)}} %10)
// CHECK-NEXT:   %13 = icmp ne ptr %12, %9
// CHECK-NEXT:   br i1 %13, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 26 }, ptr %14, align 8
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %14, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %15)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %16 = call { ptr, ptr } @main.zeroSizedPointerCapture(ptr null)
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %16, 1
// CHECK-NEXT:   %18 = extractvalue { ptr, ptr } %16, 0
// CHECK-NEXT:   %__llgo_funcval_code2 = call ptr asm "", "=r,0"(ptr %18)
// CHECK-NEXT:   %19 = call i1 %__llgo_funcval_code2(ptr {{(nest|swiftself)}} %17)
// CHECK-NEXT:   br i1 %19, label %_llgo_6, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 26 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr }, ptr %22, i32 0, i32 0
// CHECK-NEXT:   store ptr null, ptr %23, align 8
// CHECK-NEXT:   %24 = insertvalue { ptr, ptr } { ptr @"main.(*nilReceiver).IsNil$bound", ptr undef }, ptr %22, 1
// CHECK-NEXT:   %25 = extractvalue { ptr, ptr } %24, 1
// CHECK-NEXT:   %26 = extractvalue { ptr, ptr } %24, 0
// CHECK-NEXT:   %__llgo_funcval_code3 = call ptr asm "", "=r,0"(ptr %26)
// CHECK-NEXT:   %27 = call i1 %__llgo_funcval_code3(ptr {{(nest|swiftself)}} %25)
// CHECK-NEXT:   br i1 %27, label %_llgo_8, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 25 }, ptr %28, align 8
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %28, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %29)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$36psrSzSiKQuwmDQNUwPgWt23w6DHhlw0KM1_Hu7IbY", ptr @"*_llgo_main.nilReceiver")
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %30, 0
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %31, ptr null, 1
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %32)
// CHECK-NEXT:   %34 = icmp ne ptr %33, null
// CHECK-NEXT:   br i1 %34, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_11
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 32 }, ptr %35, align 8
// CHECK-NEXT:   %36 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %35, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %36)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_11
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @12, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %38 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %37, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %32, ptr %38, align 8
// CHECK-NEXT:   %39 = insertvalue { ptr, ptr } { ptr @"main.interface{IsNil() bool}.IsNil$bound", ptr undef }, ptr %37, 1
// CHECK-NEXT:   %40 = extractvalue { ptr, ptr } %39, 1
// CHECK-NEXT:   %41 = extractvalue { ptr, ptr } %39, 0
// CHECK-NEXT:   %__llgo_funcval_code4 = call ptr asm "", "=r,0"(ptr %41)
// CHECK-NEXT:   %42 = call i1 %__llgo_funcval_code4(ptr {{(nest|swiftself)}} %40)
// CHECK-NEXT:   br i1 %42, label %_llgo_10, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %33, %"{{.*}}/runtime/internal/runtime.String" { ptr @10, i64 23 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 5 })
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"main.(*nilReceiver).IsNil"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   ret i1 %1
// CHECK-NEXT: }

// CHECK-LABEL: define { { ptr, ptr }, ptr } @main.zeroSizedAddressCapture(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret { { ptr, ptr }, ptr } { { ptr, ptr } { ptr @"main.zeroSizedAddressCapture$1", ptr null }, ptr @"__llgo.moduleZeroSizedAlloc$" }
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"main.zeroSizedAddressCapture$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr @"__llgo.moduleZeroSizedAlloc$"
// CHECK-NEXT: }

// CHECK-LABEL: define { ptr, ptr } @main.zeroSizedCapture(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret { ptr, ptr } { ptr @"main.zeroSizedCapture$1", ptr null }
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.zeroSizedCapture$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br i1 false, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret i64 0
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret i64 42
// CHECK-NEXT: }

// CHECK-LABEL: define { ptr, ptr } @main.zeroSizedPointerCapture(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store ptr %0, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"main.zeroSizedPointerCapture$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   ret { ptr, ptr } %4
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"main.zeroSizedPointerCapture$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = icmp eq ptr %3, null
// CHECK-NEXT:   ret i1 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"main.(*nilReceiver).IsNil$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = call i1 @"main.(*nilReceiver).IsNil"(ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"main.interface{IsNil() bool}.IsNil$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface" } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %2, 0
// CHECK-NEXT:   %5 = getelementptr ptr, ptr %4, i64 3
// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } undef, ptr %6, 0
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } %7, ptr %3, 1
// CHECK-NEXT:   %9 = extractvalue { ptr, ptr } %8, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %8, 0
// CHECK-NEXT:   %11 = call i1 %10(ptr %9)
// CHECK-NEXT:   ret i1 %11
// CHECK-NEXT: }

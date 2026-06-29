// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK: @0 = private unnamed_addr constant [1 x i8] c"a", align 1
// CHECK: @5 = private unnamed_addr constant [4 x i8] c"Info", align 1
// CHECK: @10 = private unnamed_addr constant [54 x i8] c"{{.*}}/cl/_testrt/tpabi.T[string, int]", align 1
// CHECK: @11 = private unnamed_addr constant [5 x i8] c"hello", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpabi.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/tpabi.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/tpabi.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type T[M, N any] struct {
	m M
	n N
}

func (t *T[M, N]) Demo() {
	println(t.m, t.n)
}

func (t T[M, N]) Info() {
	println(t.m, t.n)
}

type I interface {
	Demo()
}

type K[N any] [4]N

//llgo:link (*K).Advance llgo.advance
func (t *K[N]) Advance(n int) *K[N] {
	return nil
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpabi.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/cl/_testrt/tpabi.T[string,int]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 24, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 1 }, ptr %1, align 8
// CHECK-NEXT:   store i64 1, ptr %2, align 8
// CHECK-NEXT:   %3 = load %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/tpabi.T[string,int]" %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr undef }, ptr %4, 1
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %5, 0
// CHECK-NEXT:   %7 = icmp eq ptr %6, @"_llgo_{{.*}}/cl/_testrt/tpabi.T[string,int]"
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %5, 1
// CHECK-NEXT:   %9 = load %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %8, align 8
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/cl/_testrt/tpabi.T[string,int]" %9, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %11, i32 0, i32 0
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %11, i32 0, i32 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 5 }, ptr %12, align 8
// CHECK-NEXT:   store i64 100, ptr %13, align 8
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$BP0p_lUsEd-IbbtJVukGmgrdQkqzcoYzSiwgUvgFvUs", ptr @"*_llgo_{{.*}}/cl/_testrt/tpabi.T[string,int]")
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %14, 0
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %15, ptr %11, 1
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %16)
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %16, 0
// CHECK-NEXT:   %19 = getelementptr ptr, ptr %18, i64 3
// CHECK-NEXT:   %20 = load ptr, ptr %19, align 8
// CHECK-NEXT:   %21 = insertvalue { ptr, ptr } undef, ptr %20, 0
// CHECK-NEXT:   %22 = insertvalue { ptr, ptr } %21, ptr %17, 1
// CHECK-NEXT:   %23 = extractvalue { ptr, ptr } %22, 1
// CHECK-NEXT:   %24 = extractvalue { ptr, ptr } %22, 0
// CHECK-NEXT:   call void %24(ptr %23)
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %26 = getelementptr inbounds i64, ptr %25, i64 0
// CHECK-NEXT:   %27 = getelementptr inbounds i64, ptr %25, i64 1
// CHECK-NEXT:   %28 = getelementptr inbounds i64, ptr %25, i64 2
// CHECK-NEXT:   %29 = getelementptr inbounds i64, ptr %25, i64 3
// CHECK-NEXT:   store i64 1, ptr %26, align 8
// CHECK-NEXT:   store i64 2, ptr %27, align 8
// CHECK-NEXT:   store i64 3, ptr %28, align 8
// CHECK-NEXT:   store i64 4, ptr %29, align 8
// CHECK-NEXT:   %30 = getelementptr [4 x i64], ptr %25, i64 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %30)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %31 = getelementptr [4 x i64], ptr %25, i64 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %31)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %6, %"{{.*}}/runtime/internal/runtime.String" { ptr @10, i64 54 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

func main() {
	var a any = T[string, int]{"a", 1}
	println(a.(T[string, int]).m)
	var i I = &T[string, int]{"hello", 100}
	i.Demo()

	k := &K[int]{1, 2, 3, 4}
	println(c.Advance(k, 1))
	println(k.Advance(1))
}

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testrt/tpabi.T[string,int].Info"(%"{{.*}}/cl/_testrt/tpabi.T[string,int]" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testrt/tpabi.T[string,int]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 24, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/tpabi.T[string,int]" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.String", ptr %2, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %5 = load i64, ptr %4, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testrt/tpabi.(*T[string,int]).Demo"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.String", ptr %1, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testrt/tpabi.(*T[string,int]).Info"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @10, i64 54 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 4 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/tpabi.T[string,int].Info"(%"{{.*}}/cl/_testrt/tpabi.T[string,int]" %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/tpabi.(*T[string,int]).Demo"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/tpabi.(*T[string,int]).Demo"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/tpabi.(*T[string,int]).Info"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/tpabi.(*T[string,int]).Info"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/tpabi.T[string,int].Info"(ptr %0, %"{{.*}}/cl/_testrt/tpabi.T[string,int]" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/tpabi.T[string,int].Info"(%"{{.*}}/cl/_testrt/tpabi.T[string,int]" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

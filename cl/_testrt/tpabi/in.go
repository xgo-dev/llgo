// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK-LINE: @0 = private unnamed_addr constant [1 x i8] c"a", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [4 x i8] c"Info", align 1
// CHECK-LINE: @10 = private unnamed_addr constant [54 x i8] c"{{.*}}/cl/_testrt/tpabi.T[string, int]", align 1
// CHECK-LINE: @11 = private unnamed_addr constant [5 x i8] c"hello", align 1

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
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 24, i1 false)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 1 }, ptr %2, align 8
// CHECK-NEXT:   store i64 1, ptr %4, align 8
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/tpabi.T[string,int]" %5, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr undef }, ptr %6, 1
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %7, 0
// CHECK-NEXT:   %9 = icmp eq ptr %8, @"_llgo_{{.*}}/cl/_testrt/tpabi.T[string,int]"
// CHECK-NEXT:   br i1 %9, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %7, 1
// CHECK-NEXT:   %11 = load %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %10, align 8
// CHECK-NEXT:   %12 = extractvalue %"{{.*}}/cl/_testrt/tpabi.T[string,int]" %11, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %14 = icmp eq ptr %13, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %13, i32 0, i32 0
// CHECK-NEXT:   %16 = icmp eq ptr %13, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %13, i32 0, i32 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 5 }, ptr %15, align 8
// CHECK-NEXT:   store i64 100, ptr %17, align 8
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$BP0p_lUsEd-IbbtJVukGmgrdQkqzcoYzSiwgUvgFvUs", ptr @"*_llgo_{{.*}}/cl/_testrt/tpabi.T[string,int]")
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %18, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %19, ptr %13, 1
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %20)
// CHECK-NEXT:   %22 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %20, 0
// CHECK-NEXT:   %23 = getelementptr ptr, ptr %22, i64 3
// CHECK-NEXT:   %24 = load ptr, ptr %23, align 8
// CHECK-NEXT:   %25 = insertvalue { ptr, ptr } undef, ptr %24, 0
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } %25, ptr %21, 1
// CHECK-NEXT:   %27 = extractvalue { ptr, ptr } %26, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %26, 0
// CHECK-NEXT:   call void %28(ptr %27)
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %30 = getelementptr inbounds i64, ptr %29, i64 0
// CHECK-NEXT:   %31 = getelementptr inbounds i64, ptr %29, i64 1
// CHECK-NEXT:   %32 = getelementptr inbounds i64, ptr %29, i64 2
// CHECK-NEXT:   %33 = getelementptr inbounds i64, ptr %29, i64 3
// CHECK-NEXT:   store i64 1, ptr %30, align 8
// CHECK-NEXT:   store i64 2, ptr %31, align 8
// CHECK-NEXT:   store i64 3, ptr %32, align 8
// CHECK-NEXT:   store i64 4, ptr %33, align 8
// CHECK-NEXT:   %34 = getelementptr [4 x i64], ptr %29, i64 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %34)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %35 = getelementptr [4 x i64], ptr %29, i64 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %35)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %8, %"{{.*}}/runtime/internal/runtime.String" { ptr @10, i64 54 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
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
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 24, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/tpabi.T[string,int]" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %4, align 8
// CHECK-NEXT:   %6 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %9 = load i64, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testrt/tpabi.(*T[string,int]).Demo"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpabi.T[string,int]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = load i64, ptr %7, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
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

// LITTEST
package main

// Test of promotion of methods of an interface embedded within a
// struct.  In particular, this test exercises that the correct
// method is called.

// CHECK-LINE: @0 = private unnamed_addr constant [3 x i8] c"two", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [48 x i8] c"{{.*}}/cl/_testgo/ifaceprom.impl", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [3 x i8] c"one", align 1
// CHECK-LINE: @13 = private unnamed_addr constant [45 x i8] c"{{.*}}/cl/_testgo/ifaceprom.I", align 1
// CHECK-LINE: @14 = private unnamed_addr constant [4 x i8] c"pass", align 1

type I interface {
	one() int
	two() string
}

type S struct {
	I
}

type impl struct{}

func (impl) one() int {
	return 1
}

func (impl) two() string {
	return "two"
}

func main() {
	var s S
	s.I = impl{}
	if one := s.I.one(); one != 1 {
		panic(one)
	}
	if one := s.one(); one != 1 {
		panic(one)
	}
	closOne := s.I.one
	if one := closOne(); one != 1 {
		panic(one)
	}
	closOne = s.one
	if one := closOne(); one != 1 {
		panic(one)
	}

	if two := s.I.two(); two != "two" {
		panic(two)
	}
	if two := s.two(); two != "two" {
		panic(two)
	}
	closTwo := s.I.two
	if two := closTwo(); two != "two" {
		panic(two)
	}
	closTwo = s.two
	if two := closTwo(); two != "two" {
		panic(two)
	}

	println("pass")
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/ifaceprom.S.one"(%"{{.*}}/cl/_testgo/ifaceprom.S" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/ifaceprom.S", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceprom.S" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call i64 %12(ptr %11)
// CHECK-NEXT:   ret i64 %13
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.S.two"(%"{{.*}}/cl/_testgo/ifaceprom.S" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/ifaceprom.S", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceprom.S" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 4
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call %"{{.*}}/runtime/internal/runtime.String" %12(ptr %11)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %13
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/ifaceprom.(*S).one"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %6 = getelementptr ptr, ptr %5, i64 3
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } %8, ptr %4, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %9, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %9, 0
// CHECK-NEXT:   %12 = call i64 %11(ptr %10)
// CHECK-NEXT:   ret i64 %12
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.(*S).two"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %6 = getelementptr ptr, ptr %5, i64 4
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } %8, ptr %4, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %9, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %9, 0
// CHECK-NEXT:   %12 = call %"{{.*}}/runtime/internal/runtime.String" %11(ptr %10)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %12
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/ifaceprom.impl.one"(%"{{.*}}/cl/_testgo/ifaceprom.impl" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 1
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.impl.two"(%"{{.*}}/cl/_testgo/ifaceprom.impl" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 }
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/ifaceprom.(*impl).one"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 48 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/ifaceprom.impl.one"(%"{{.*}}/cl/_testgo/ifaceprom.impl" zeroinitializer)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.(*impl).two"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 48 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.impl.two"(%"{{.*}}/cl/_testgo/ifaceprom.impl" zeroinitializer)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceprom.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/ifaceprom.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/ifaceprom.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceprom.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/cl/_testgo/ifaceprom.S", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceprom.impl" zeroinitializer, ptr %4, align 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceprom.iface$zZ89tENb5h_KNjvpxf1TXPfaWFYn0IZrZwyVf42lRtA", ptr @"_llgo_{{.*}}/cl/_testgo/ifaceprom.impl")
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5, 0
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %6, ptr %4, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %7, ptr %3, align 8
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %10, align 8
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %11)
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %11, 0
// CHECK-NEXT:   %14 = getelementptr ptr, ptr %13, i64 3
// CHECK-NEXT:   %15 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %16 = insertvalue { ptr, ptr } undef, ptr %15, 0
// CHECK-NEXT:   %17 = insertvalue { ptr, ptr } %16, ptr %12, 1
// CHECK-NEXT:   %18 = extractvalue { ptr, ptr } %17, 1
// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %17, 0
// CHECK-NEXT:   %20 = call i64 %19(ptr %18)
// CHECK-NEXT:   %21 = icmp ne i64 %20, 1
// CHECK-NEXT:   br i1 %21, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %20, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %22, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %23)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %24 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %25 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %24, 0
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %25)
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %25, 0
// CHECK-NEXT:   %28 = getelementptr ptr, ptr %27, i64 3
// CHECK-NEXT:   %29 = load ptr, ptr %28, align 8
// CHECK-NEXT:   %30 = insertvalue { ptr, ptr } undef, ptr %29, 0
// CHECK-NEXT:   %31 = insertvalue { ptr, ptr } %30, ptr %26, 1
// CHECK-NEXT:   %32 = extractvalue { ptr, ptr } %31, 1
// CHECK-NEXT:   %33 = extractvalue { ptr, ptr } %31, 0
// CHECK-NEXT:   %34 = call i64 %33(ptr %32)
// CHECK-NEXT:   %35 = icmp ne i64 %34, 1
// CHECK-NEXT:   br i1 %35, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %34, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %36, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %37)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %38 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %38)
// CHECK-NEXT:   %39 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %39)
// CHECK-NEXT:   %40 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %41 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %40, align 8
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %41)
// CHECK-NEXT:   %43 = icmp ne ptr %42, null
// CHECK-NEXT:   br i1 %43, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_17
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %103, ptr %44, align 8
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %44, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %45)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_17
// CHECK-NEXT:   %46 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %47 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %46, 0
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %47)
// CHECK-NEXT:   %49 = icmp ne ptr %48, null
// CHECK-NEXT:   br i1 %49, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_19
// CHECK-NEXT:   %50 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %110, ptr %50, align 8
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %50, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %51)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_19
// CHECK-NEXT:   %52 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %52)
// CHECK-NEXT:   %53 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %53)
// CHECK-NEXT:   %54 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %55 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %54, align 8
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %55)
// CHECK-NEXT:   %57 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %55, 0
// CHECK-NEXT:   %58 = getelementptr ptr, ptr %57, i64 4
// CHECK-NEXT:   %59 = load ptr, ptr %58, align 8
// CHECK-NEXT:   %60 = insertvalue { ptr, ptr } undef, ptr %59, 0
// CHECK-NEXT:   %61 = insertvalue { ptr, ptr } %60, ptr %56, 1
// CHECK-NEXT:   %62 = extractvalue { ptr, ptr } %61, 1
// CHECK-NEXT:   %63 = extractvalue { ptr, ptr } %61, 0
// CHECK-NEXT:   %64 = call %"{{.*}}/runtime/internal/runtime.String" %63(ptr %62)
// CHECK-NEXT:   %65 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %64, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %66 = xor i1 %65, true
// CHECK-NEXT:   br i1 %66, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %67 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %64, ptr %67, align 8
// CHECK-NEXT:   %68 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %67, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %68)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %69 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %70 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %69, 0
// CHECK-NEXT:   %71 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %70)
// CHECK-NEXT:   %72 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %70, 0
// CHECK-NEXT:   %73 = getelementptr ptr, ptr %72, i64 4
// CHECK-NEXT:   %74 = load ptr, ptr %73, align 8
// CHECK-NEXT:   %75 = insertvalue { ptr, ptr } undef, ptr %74, 0
// CHECK-NEXT:   %76 = insertvalue { ptr, ptr } %75, ptr %71, 1
// CHECK-NEXT:   %77 = extractvalue { ptr, ptr } %76, 1
// CHECK-NEXT:   %78 = extractvalue { ptr, ptr } %76, 0
// CHECK-NEXT:   %79 = call %"{{.*}}/runtime/internal/runtime.String" %78(ptr %77)
// CHECK-NEXT:   %80 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %79, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %81 = xor i1 %80, true
// CHECK-NEXT:   br i1 %81, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %82 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %79, ptr %82, align 8
// CHECK-NEXT:   %83 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %82, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %83)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %84 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %84)
// CHECK-NEXT:   %85 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %85)
// CHECK-NEXT:   %86 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %87 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %86, align 8
// CHECK-NEXT:   %88 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %87)
// CHECK-NEXT:   %89 = icmp ne ptr %88, null
// CHECK-NEXT:   br i1 %89, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %90 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %117, ptr %90, align 8
// CHECK-NEXT:   %91 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %90, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %91)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %92 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %93 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %92, 0
// CHECK-NEXT:   %94 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %93)
// CHECK-NEXT:   %95 = icmp ne ptr %94, null
// CHECK-NEXT:   br i1 %95, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_23
// CHECK-NEXT:   %96 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %125, ptr %96, align 8
// CHECK-NEXT:   %97 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %96, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %97)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_23
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_4
// CHECK-NEXT:   %98 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %99 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %98, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %41, ptr %99, align 8
// CHECK-NEXT:   %100 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.one$bound", ptr undef }, ptr %98, 1
// CHECK-NEXT:   %101 = extractvalue { ptr, ptr } %100, 1
// CHECK-NEXT:   %102 = extractvalue { ptr, ptr } %100, 0
// CHECK-NEXT:   %103 = call i64 %102(ptr %101)
// CHECK-NEXT:   %104 = icmp ne i64 %103, 1
// CHECK-NEXT:   br i1 %104, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %42, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %105 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %106 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %105, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %47, ptr %106, align 8
// CHECK-NEXT:   %107 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.one$bound", ptr undef }, ptr %105, 1
// CHECK-NEXT:   %108 = extractvalue { ptr, ptr } %107, 1
// CHECK-NEXT:   %109 = extractvalue { ptr, ptr } %107, 0
// CHECK-NEXT:   %110 = call i64 %109(ptr %108)
// CHECK-NEXT:   %111 = icmp ne i64 %110, 1
// CHECK-NEXT:   br i1 %111, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %48, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %112 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %113 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %112, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %87, ptr %113, align 8
// CHECK-NEXT:   %114 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.two$bound", ptr undef }, ptr %112, 1
// CHECK-NEXT:   %115 = extractvalue { ptr, ptr } %114, 1
// CHECK-NEXT:   %116 = extractvalue { ptr, ptr } %114, 0
// CHECK-NEXT:   %117 = call %"{{.*}}/runtime/internal/runtime.String" %116(ptr %115)
// CHECK-NEXT:   %118 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %117, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %119 = xor i1 %118, true
// CHECK-NEXT:   br i1 %119, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %88, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %120 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %121 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %120, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %93, ptr %121, align 8
// CHECK-NEXT:   %122 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.two$bound", ptr undef }, ptr %120, 1
// CHECK-NEXT:   %123 = extractvalue { ptr, ptr } %122, 1
// CHECK-NEXT:   %124 = extractvalue { ptr, ptr } %122, 0
// CHECK-NEXT:   %125 = call %"{{.*}}/runtime/internal/runtime.String" %124(ptr %123)
// CHECK-NEXT:   %126 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %125, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %127 = xor i1 %126, true
// CHECK-NEXT:   br i1 %127, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_14
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %94, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal0"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal0"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/ifaceprom.(*impl).one"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/ifaceprom.(*impl).one"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/ifaceprom.(*impl).two"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.(*impl).two"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/ifaceprom.impl.one"(ptr %0, %"{{.*}}/cl/_testgo/ifaceprom.impl" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/ifaceprom.impl.one"(%"{{.*}}/cl/_testgo/ifaceprom.impl" %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/ifaceprom.impl.two"(ptr %0, %"{{.*}}/cl/_testgo/ifaceprom.impl" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.impl.two"(%"{{.*}}/cl/_testgo/ifaceprom.impl" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/ifaceprom.I.one$bound"(ptr %0){{.*}} {
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
// CHECK-NEXT:   %11 = call i64 %10(ptr %9)
// CHECK-NEXT:   ret i64 %11
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.I.two$bound"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface" } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %2, 0
// CHECK-NEXT:   %5 = getelementptr ptr, ptr %4, i64 4
// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } undef, ptr %6, 0
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } %7, ptr %3, 1
// CHECK-NEXT:   %9 = extractvalue { ptr, ptr } %8, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %8, 0
// CHECK-NEXT:   %11 = call %"{{.*}}/runtime/internal/runtime.String" %10(ptr %9)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %11
// CHECK-NEXT: }

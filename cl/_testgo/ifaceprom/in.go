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
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceprom.impl" zeroinitializer, ptr %3, align 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceprom.iface$zZ89tENb5h_KNjvpxf1TXPfaWFYn0IZrZwyVf42lRtA", ptr @"_llgo_{{.*}}/cl/_testgo/ifaceprom.impl")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %3, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %6, ptr %2, align 8
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %8, align 8
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 1
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %18, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %22, 0
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %23)
// CHECK-NEXT:   %25 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %23, 0
// CHECK-NEXT:   %26 = getelementptr ptr, ptr %25, i64 3
// CHECK-NEXT:   %27 = load ptr, ptr %26, align 8
// CHECK-NEXT:   %28 = insertvalue { ptr, ptr } undef, ptr %27, 0
// CHECK-NEXT:   %29 = insertvalue { ptr, ptr } %28, ptr %24, 1
// CHECK-NEXT:   %30 = extractvalue { ptr, ptr } %29, 1
// CHECK-NEXT:   %31 = extractvalue { ptr, ptr } %29, 0
// CHECK-NEXT:   %32 = call i64 %31(ptr %30)
// CHECK-NEXT:   %33 = icmp ne i64 %32, 1
// CHECK-NEXT:   br i1 %33, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %32, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %34, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %36 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %36)
// CHECK-NEXT:   %37 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %38 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %37, align 8
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %38)
// CHECK-NEXT:   %40 = icmp ne ptr %39, null
// CHECK-NEXT:   br i1 %40, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_17
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %98, ptr %41, align 8
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %41, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %42)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_17
// CHECK-NEXT:   %43 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %44 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %43, 0
// CHECK-NEXT:   %45 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %44)
// CHECK-NEXT:   %46 = icmp ne ptr %45, null
// CHECK-NEXT:   br i1 %46, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_19
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %105, ptr %47, align 8
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %47, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %48)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_19
// CHECK-NEXT:   %49 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %49)
// CHECK-NEXT:   %50 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %51 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %50, align 8
// CHECK-NEXT:   %52 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %51)
// CHECK-NEXT:   %53 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %51, 0
// CHECK-NEXT:   %54 = getelementptr ptr, ptr %53, i64 4
// CHECK-NEXT:   %55 = load ptr, ptr %54, align 8
// CHECK-NEXT:   %56 = insertvalue { ptr, ptr } undef, ptr %55, 0
// CHECK-NEXT:   %57 = insertvalue { ptr, ptr } %56, ptr %52, 1
// CHECK-NEXT:   %58 = extractvalue { ptr, ptr } %57, 1
// CHECK-NEXT:   %59 = extractvalue { ptr, ptr } %57, 0
// CHECK-NEXT:   %60 = call %"{{.*}}/runtime/internal/runtime.String" %59(ptr %58)
// CHECK-NEXT:   %61 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %60, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %62 = xor i1 %61, true
// CHECK-NEXT:   br i1 %62, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %63 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %60, ptr %63, align 8
// CHECK-NEXT:   %64 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %63, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %64)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %65 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %66 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %65, 0
// CHECK-NEXT:   %67 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %66)
// CHECK-NEXT:   %68 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %66, 0
// CHECK-NEXT:   %69 = getelementptr ptr, ptr %68, i64 4
// CHECK-NEXT:   %70 = load ptr, ptr %69, align 8
// CHECK-NEXT:   %71 = insertvalue { ptr, ptr } undef, ptr %70, 0
// CHECK-NEXT:   %72 = insertvalue { ptr, ptr } %71, ptr %67, 1
// CHECK-NEXT:   %73 = extractvalue { ptr, ptr } %72, 1
// CHECK-NEXT:   %74 = extractvalue { ptr, ptr } %72, 0
// CHECK-NEXT:   %75 = call %"{{.*}}/runtime/internal/runtime.String" %74(ptr %73)
// CHECK-NEXT:   %76 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %75, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %77 = xor i1 %76, true
// CHECK-NEXT:   br i1 %77, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %78 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %75, ptr %78, align 8
// CHECK-NEXT:   %79 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %78, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %79)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %80 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %80)
// CHECK-NEXT:   %81 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %82 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %81, align 8
// CHECK-NEXT:   %83 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %82)
// CHECK-NEXT:   %84 = icmp ne ptr %83, null
// CHECK-NEXT:   br i1 %84, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %85 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %112, ptr %85, align 8
// CHECK-NEXT:   %86 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %85, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %86)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %87 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %88 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %87, 0
// CHECK-NEXT:   %89 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %88)
// CHECK-NEXT:   %90 = icmp ne ptr %89, null
// CHECK-NEXT:   br i1 %90, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_23
// CHECK-NEXT:   %91 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %120, ptr %91, align 8
// CHECK-NEXT:   %92 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %91, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %92)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_23
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_4
// CHECK-NEXT:   %93 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %94 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %93, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %38, ptr %94, align 8
// CHECK-NEXT:   %95 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.one$bound", ptr undef }, ptr %93, 1
// CHECK-NEXT:   %96 = extractvalue { ptr, ptr } %95, 1
// CHECK-NEXT:   %97 = extractvalue { ptr, ptr } %95, 0
// CHECK-NEXT:   %98 = call i64 %97(ptr %96)
// CHECK-NEXT:   %99 = icmp ne i64 %98, 1
// CHECK-NEXT:   br i1 %99, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %39, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %100 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %101 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %100, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %44, ptr %101, align 8
// CHECK-NEXT:   %102 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.one$bound", ptr undef }, ptr %100, 1
// CHECK-NEXT:   %103 = extractvalue { ptr, ptr } %102, 1
// CHECK-NEXT:   %104 = extractvalue { ptr, ptr } %102, 0
// CHECK-NEXT:   %105 = call i64 %104(ptr %103)
// CHECK-NEXT:   %106 = icmp ne i64 %105, 1
// CHECK-NEXT:   br i1 %106, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %45, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %107 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %108 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %107, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %82, ptr %108, align 8
// CHECK-NEXT:   %109 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.two$bound", ptr undef }, ptr %107, 1
// CHECK-NEXT:   %110 = extractvalue { ptr, ptr } %109, 1
// CHECK-NEXT:   %111 = extractvalue { ptr, ptr } %109, 0
// CHECK-NEXT:   %112 = call %"{{.*}}/runtime/internal/runtime.String" %111(ptr %110)
// CHECK-NEXT:   %113 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %112, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %114 = xor i1 %113, true
// CHECK-NEXT:   br i1 %114, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %83, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %115 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %116 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %115, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %88, ptr %116, align 8
// CHECK-NEXT:   %117 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.two$bound", ptr undef }, ptr %115, 1
// CHECK-NEXT:   %118 = extractvalue { ptr, ptr } %117, 1
// CHECK-NEXT:   %119 = extractvalue { ptr, ptr } %117, 0
// CHECK-NEXT:   %120 = call %"{{.*}}/runtime/internal/runtime.String" %119(ptr %118)
// CHECK-NEXT:   %121 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %120, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %122 = xor i1 %121, true
// CHECK-NEXT:   br i1 %122, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_14
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %89, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
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

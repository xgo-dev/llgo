// LITTEST
package main

// Test of promotion of methods of an interface embedded within a
// struct.  In particular, this test exercises that the correct
// method is called.

// CHECK: @0 = private unnamed_addr constant [3 x i8] c"two", align 1
// CHECK: @1 = private unnamed_addr constant [48 x i8] c"{{.*}}/cl/_testgo/ifaceprom.impl", align 1
// CHECK: @2 = private unnamed_addr constant [3 x i8] c"one", align 1
// CHECK: @13 = private unnamed_addr constant [45 x i8] c"{{.*}}/cl/_testgo/ifaceprom.I", align 1
// CHECK: @14 = private unnamed_addr constant [4 x i8] c"pass", align 1

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
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %1, i32 0, i32 0
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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.S.two"(%"{{.*}}/cl/_testgo/ifaceprom.S" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/ifaceprom.S", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceprom.S" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %1, i32 0, i32 0
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

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/ifaceprom.(*S).one"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %1, align 8
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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/ifaceprom.(*S).two"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %1, align 8
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
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceprom.impl" zeroinitializer, ptr %2, align 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceprom.iface$zZ89tENb5h_KNjvpxf1TXPfaWFYn0IZrZwyVf42lRtA", ptr @"_llgo_{{.*}}/cl/_testgo/ifaceprom.impl")
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %3, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %4, ptr %2, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %1, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %7 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %6, align 8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %7)
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %7, 0
// CHECK-NEXT:   %10 = getelementptr ptr, ptr %9, i64 3
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } undef, ptr %11, 0
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } %12, ptr %8, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %13, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %13, 0
// CHECK-NEXT:   %16 = call i64 %15(ptr %14)
// CHECK-NEXT:   %17 = icmp ne i64 %16, 1
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %16, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %21 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %20, 0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %21)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %21, 0
// CHECK-NEXT:   %24 = getelementptr ptr, ptr %23, i64 3
// CHECK-NEXT:   %25 = load ptr, ptr %24, align 8
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } undef, ptr %25, 0
// CHECK-NEXT:   %27 = insertvalue { ptr, ptr } %26, ptr %22, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %27, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   %30 = call i64 %29(ptr %28)
// CHECK-NEXT:   %31 = icmp ne i64 %30, 1
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %30, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %35 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %34, align 8
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %35)
// CHECK-NEXT:   %37 = icmp ne ptr %36, null
// CHECK-NEXT:   br i1 %37, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_17
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %93, ptr %38, align 8
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %38, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %39)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_17
// CHECK-NEXT:   %40 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %41 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %40, 0
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %41)
// CHECK-NEXT:   %43 = icmp ne ptr %42, null
// CHECK-NEXT:   br i1 %43, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_19
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %100, ptr %44, align 8
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %44, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %45)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_19
// CHECK-NEXT:   %46 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %47 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %46, align 8
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %47)
// CHECK-NEXT:   %49 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %47, 0
// CHECK-NEXT:   %50 = getelementptr ptr, ptr %49, i64 4
// CHECK-NEXT:   %51 = load ptr, ptr %50, align 8
// CHECK-NEXT:   %52 = insertvalue { ptr, ptr } undef, ptr %51, 0
// CHECK-NEXT:   %53 = insertvalue { ptr, ptr } %52, ptr %48, 1
// CHECK-NEXT:   %54 = extractvalue { ptr, ptr } %53, 1
// CHECK-NEXT:   %55 = extractvalue { ptr, ptr } %53, 0
// CHECK-NEXT:   %56 = call %"{{.*}}/runtime/internal/runtime.String" %55(ptr %54)
// CHECK-NEXT:   %57 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %56, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %58 = xor i1 %57, true
// CHECK-NEXT:   br i1 %58, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %59 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %56, ptr %59, align 8
// CHECK-NEXT:   %60 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %59, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %60)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %61 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %61, 0
// CHECK-NEXT:   %63 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %62)
// CHECK-NEXT:   %64 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %62, 0
// CHECK-NEXT:   %65 = getelementptr ptr, ptr %64, i64 4
// CHECK-NEXT:   %66 = load ptr, ptr %65, align 8
// CHECK-NEXT:   %67 = insertvalue { ptr, ptr } undef, ptr %66, 0
// CHECK-NEXT:   %68 = insertvalue { ptr, ptr } %67, ptr %63, 1
// CHECK-NEXT:   %69 = extractvalue { ptr, ptr } %68, 1
// CHECK-NEXT:   %70 = extractvalue { ptr, ptr } %68, 0
// CHECK-NEXT:   %71 = call %"{{.*}}/runtime/internal/runtime.String" %70(ptr %69)
// CHECK-NEXT:   %72 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %71, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %73 = xor i1 %72, true
// CHECK-NEXT:   br i1 %73, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %74 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %71, ptr %74, align 8
// CHECK-NEXT:   %75 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %74, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %75)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %76 = getelementptr inbounds %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %77 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %76, align 8
// CHECK-NEXT:   %78 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %77)
// CHECK-NEXT:   %79 = icmp ne ptr %78, null
// CHECK-NEXT:   br i1 %79, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %80 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %107, ptr %80, align 8
// CHECK-NEXT:   %81 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %80, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %81)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %82 = load %"{{.*}}/cl/_testgo/ifaceprom.S", ptr %0, align 8
// CHECK-NEXT:   %83 = extractvalue %"{{.*}}/cl/_testgo/ifaceprom.S" %82, 0
// CHECK-NEXT:   %84 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %83)
// CHECK-NEXT:   %85 = icmp ne ptr %84, null
// CHECK-NEXT:   br i1 %85, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_23
// CHECK-NEXT:   %86 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %115, ptr %86, align 8
// CHECK-NEXT:   %87 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %86, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %87)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_23
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_4
// CHECK-NEXT:   %88 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %89 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %88, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %35, ptr %89, align 8
// CHECK-NEXT:   %90 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.one$bound", ptr undef }, ptr %88, 1
// CHECK-NEXT:   %91 = extractvalue { ptr, ptr } %90, 1
// CHECK-NEXT:   %92 = extractvalue { ptr, ptr } %90, 0
// CHECK-NEXT:   %93 = call i64 %92(ptr %91)
// CHECK-NEXT:   %94 = icmp ne i64 %93, 1
// CHECK-NEXT:   br i1 %94, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %36, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %95 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %96 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %95, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %41, ptr %96, align 8
// CHECK-NEXT:   %97 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.one$bound", ptr undef }, ptr %95, 1
// CHECK-NEXT:   %98 = extractvalue { ptr, ptr } %97, 1
// CHECK-NEXT:   %99 = extractvalue { ptr, ptr } %97, 0
// CHECK-NEXT:   %100 = call i64 %99(ptr %98)
// CHECK-NEXT:   %101 = icmp ne i64 %100, 1
// CHECK-NEXT:   br i1 %101, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %42, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %102 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %103 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %102, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %77, ptr %103, align 8
// CHECK-NEXT:   %104 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.two$bound", ptr undef }, ptr %102, 1
// CHECK-NEXT:   %105 = extractvalue { ptr, ptr } %104, 1
// CHECK-NEXT:   %106 = extractvalue { ptr, ptr } %104, 0
// CHECK-NEXT:   %107 = call %"{{.*}}/runtime/internal/runtime.String" %106(ptr %105)
// CHECK-NEXT:   %108 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %107, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %109 = xor i1 %108, true
// CHECK-NEXT:   br i1 %109, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %78, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %110 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %111 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %110, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %83, ptr %111, align 8
// CHECK-NEXT:   %112 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/ifaceprom.I.two$bound", ptr undef }, ptr %110, 1
// CHECK-NEXT:   %113 = extractvalue { ptr, ptr } %112, 1
// CHECK-NEXT:   %114 = extractvalue { ptr, ptr } %112, 0
// CHECK-NEXT:   %115 = call %"{{.*}}/runtime/internal/runtime.String" %114(ptr %113)
// CHECK-NEXT:   %116 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %115, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 })
// CHECK-NEXT:   %117 = xor i1 %116, true
// CHECK-NEXT:   br i1 %117, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_14
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %84, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
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

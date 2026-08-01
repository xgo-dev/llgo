// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LABEL: define void @"main.(*T).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (t *T) Invoke() {
	println("invoke")
}

// CHECK-LABEL: define void @main.dump(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %main.eface, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %1, null
// CHECK-NEXT:   br i1 %3, label %4, label %5
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %6 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @main.dumpTyp(ptr %6, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func dump(v any) {
	e := (*eface)(unsafe.Pointer(&v))
	dumpTyp(e._type, "")
}

// CHECK-LABEL: define void @main.dumpTyp(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %0)
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/runtime/abi.(*Type).Kind"(ptr %0)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %5, label %15, label %16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %41
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 7 })
// CHECK-NEXT:   call void @main.dumpTyp(ptr %6, %"{{.*}}/runtime/internal/runtime.String" %7)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %41
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   %9 = icmp ne ptr %8, null
// CHECK-NEXT:   br i1 %9, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   %11 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 9 })
// CHECK-NEXT:   call void @main.dumpUncommon(ptr %10, %"{{.*}}/runtime/internal/runtime.String" %11)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %13, label %49, label %50
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %54, %50, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %50
// CHECK-NEXT:   %14 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %14, label %53, label %54
// CHECK-EMPTY:
// CHECK-NEXT: 15:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 16:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %17 = load i64, ptr %4, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %19 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %19, label %20, label %21
// CHECK-EMPTY:
// CHECK-NEXT: 20:                                               ; preds = %16
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %16
// CHECK-NEXT:   %22 = load i64, ptr %18, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %24 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %24, label %25, label %26
// CHECK-EMPTY:
// CHECK-NEXT: 25:                                               ; preds = %21
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 26:                                               ; preds = %21
// CHECK-NEXT:   %27 = load i32, ptr %23, align 4
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 3
// CHECK-NEXT:   %29 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %29, label %30, label %31
// CHECK-EMPTY:
// CHECK-NEXT: 30:                                               ; preds = %26
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 31:                                               ; preds = %26
// CHECK-NEXT:   %32 = load i8, ptr %28, align 1
// CHECK-NEXT:   %33 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 4
// CHECK-NEXT:   %34 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %34, label %35, label %36
// CHECK-EMPTY:
// CHECK-NEXT: 35:                                               ; preds = %31
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 36:                                               ; preds = %31
// CHECK-NEXT:   %37 = load i8, ptr %33, align 1
// CHECK-NEXT:   %38 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %39 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %39, label %40, label %41
// CHECK-EMPTY:
// CHECK-NEXT: 40:                                               ; preds = %36
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 41:                                               ; preds = %36
// CHECK-NEXT:   %42 = load ptr, ptr %38, align 8
// CHECK-NEXT:   %43 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %22)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %44 = zext i32 %27 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %44)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %45 = zext i8 %32 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %45)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %46 = zext i8 %37 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %46)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %42)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %43)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK-NEXT:   %48 = icmp ne ptr %47, null
// CHECK-NEXT:   br i1 %48, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 49:                                               ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 50:                                               ; preds = %_llgo_3
// CHECK-NEXT:   %51 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %52 = icmp ne ptr %51, null
// CHECK-NEXT:   br i1 %52, label %_llgo_5, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 53:                                               ; preds = %_llgo_5
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 54:                                               ; preds = %_llgo_5
// CHECK-NEXT:   %55 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %56 = load ptr, ptr %55, align 8
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %56)
// CHECK-NEXT:   %58 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 9 })
// CHECK-NEXT:   call void @main.dumpUncommon(ptr %57, %"{{.*}}/runtime/internal/runtime.String" %58)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-NEXT: }
func dumpTyp(t *abi.Type, sep string) {
	print(sep)
	println(t.String(), t.Kind(), t.Size_, t.PtrBytes, t.Hash, t.TFlag, t.Align_, t.PtrToThis_, t.Uncommon())
	if t.Elem() != nil {
		dumpTyp(t.Elem(), sep+"\telem: ")
	}
	if t.Uncommon() != nil {
		dumpUncommon(t.Uncommon(), sep+"\tuncomm: ")
		if t.PtrToThis_ != nil {
			dumpUncommon(t.PtrToThis_.Uncommon(), sep+"\tuncomm: ")
		}
	}
}

// CHECK-LABEL: define void @main.dumpUncommon(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %3, label %4, label %5
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.String", ptr %2, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %8, label %9, label %10
// CHECK-EMPTY:
// CHECK-NEXT: 9:                                                ; preds = %5
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %5
// CHECK-NEXT:   %11 = load i16, ptr %7, align 2
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %13, label %14, label %15
// CHECK-EMPTY:
// CHECK-NEXT: 14:                                               ; preds = %10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 15:                                               ; preds = %10
// CHECK-NEXT:   %16 = load i16, ptr %12, align 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %17 = zext i16 %11 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %18 = zext i16 %16 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %18)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func dumpUncommon(u *abi.UncommonType, sep string) {
	print(sep)
	println(u.PkgPath_, u.Mcount, u.Xcount)
}

type T string

type eface struct {
	_type *abi.Type
	data  unsafe.Pointer
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK: store i1 true, ptr {{%[0-9]+}}, align 1
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }
// CHECK: call void @main.dump
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: store i64 0, ptr {{%[0-9]+}}, align 8
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }
// CHECK: call void @main.dump
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 80)
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[10]_llgo_int"
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int"
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testrt/eface.struct
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.T
// CHECK: ret void
func main() {
	dump(true)
	dump(0)
	dump(int8(0))
	dump(int16(0))
	dump(int32(0))
	dump(int64(0))
	dump(uint(0))
	dump(uint8(0))
	dump(uint16(0))
	dump(uint32(0))
	dump(uint64(0))
	dump(uintptr(0))
	dump(float32(0))
	dump(float64(0))
	dump([10]int{})
	dump(func() {})
	// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
	dump((*int)(nil))
	dump([]int{})
	dump("hello")
	dump(struct {
		x int8
		y int
		z int
	}{})
	var t T
	dump(t)
}

// CHECK-LABEL: define linkonce void @"__llgo_stub.main.main$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"main.main$1"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

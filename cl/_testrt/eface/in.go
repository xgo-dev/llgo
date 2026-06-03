// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.(*T).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (t *T) Invoke() {
	println("invoke")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/eface.eface", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpTyp"(ptr %3, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func dump(v any) {
	e := (*eface)(unsafe.Pointer(&v))
	dumpTyp(e._type, "")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.dumpTyp"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %0)
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/runtime/abi.(*Type).Kind"(ptr %0)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load i64, ptr %4, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %9 = load i32, ptr %8, align 4
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 3
// CHECK-NEXT:   %11 = load i8, ptr %10, align 1
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 4
// CHECK-NEXT:   %13 = load i8, ptr %12, align 1
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %15 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %17 = zext i32 %9 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %18 = zext i8 %11 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %18)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %19 = zext i8 %13 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %19)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK-NEXT:   %21 = icmp ne ptr %20, null
// CHECK-NEXT:   br i1 %21, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK-NEXT:   %23 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpTyp"(ptr %22, %"{{.*}}/runtime/internal/runtime.String" %23)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   %25 = icmp ne ptr %24, null
// CHECK-NEXT:   br i1 %25, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   %27 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 9 })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpUncommon"(ptr %26, %"{{.*}}/runtime/internal/runtime.String" %27)
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %29 = load ptr, ptr %28, align 8
// CHECK-NEXT:   %30 = icmp ne ptr %29, null
// CHECK-NEXT:   br i1 %30, label %_llgo_5, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5, %_llgo_3, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %32 = load ptr, ptr %31, align 8
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %32)
// CHECK-NEXT:   %34 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 9 })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpUncommon"(ptr %33, %"{{.*}}/runtime/internal/runtime.String" %34)
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

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.dumpUncommon"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.String", ptr %2, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %5 = load i16, ptr %4, align 2
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %7 = load i16, ptr %6, align 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %8 = zext i16 %5 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %9 = zext i16 %7 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %9)
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

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.main"(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK: store i1 true, ptr {{%[0-9]+}}, align 1
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }
// CHECK: call void @"{{.*}}/cl/_testrt/eface.dump"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: store i64 0, ptr {{%[0-9]+}}, align 8
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }
// CHECK: call void @"{{.*}}/cl/_testrt/eface.dump"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 80)
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[10]_llgo_int"
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int"
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testrt/eface.struct
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/eface.T"
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
	// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.main$1"(){{.*}} {
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

// CHECK-LABEL: define linkonce void @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/eface.main$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/eface.main$1"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

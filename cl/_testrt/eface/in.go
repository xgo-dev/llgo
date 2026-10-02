// LITTEST
package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

func (t *T) Invoke() {
	println("invoke")
}

// CHECK-LABEL: define void @main.dump(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK: store %"{{.*}}eface" %0, ptr [[DUMP_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[DUMP_TYPE_FIELD:%.*]] = getelementptr inbounds nuw %main.eface, ptr [[DUMP_ADDR]], i32 0, i32 0
// CHECK: [[DUMP_TYPE:%.*]] = load ptr, ptr [[DUMP_TYPE_FIELD]]
// CHECK: call void @main.dumpTyp(ptr [[DUMP_TYPE]], %"{{.*}}String" zeroinitializer)
func dump(v any) {
	e := (*eface)(unsafe.Pointer(&v))
	dumpTyp(e._type, "")
}

// CHECK-LABEL: define void @main.dumpTyp(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" %1)
// CHECK: [[TYPE_NAME:%.*]] = call %"{{.*}}String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %0)
// CHECK: [[TYPE_KIND:%.*]] = call i64 @"{{.*}}/runtime/abi.(*Type).Kind"(ptr %0)
// CHECK: [[TYPE_UNCOMMON:%.*]] = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[TYPE_NAME]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 [[TYPE_KIND]])
// CHECK: [[TYPE_ELEM:%.*]] = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK: [[HAS_ELEM:%.*]] = icmp ne ptr [[TYPE_ELEM]], null
// CHECK: br i1 [[HAS_ELEM]], label %{{.*}}, label %{{.*}}
// CHECK: [[RECURSE_ELEM:%.*]] = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK: [[ELEM_SEP:%.*]] = call %"{{.*}}String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}String" %1, %"{{.*}}String" { ptr @{{.*}}, i64 7 })
// CHECK: call void @main.dumpTyp(ptr [[RECURSE_ELEM]], %"{{.*}}String" [[ELEM_SEP]])
// CHECK: [[RECHECK_UNCOMMON:%.*]] = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK: [[HAS_UNCOMMON:%.*]] = icmp ne ptr [[RECHECK_UNCOMMON]], null
// CHECK: br i1 [[HAS_UNCOMMON]], label %{{.*}}, label %{{.*}}
// CHECK: [[DUMP_UNCOMMON:%.*]] = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK: [[UNCOMMON_SEP:%.*]] = call %"{{.*}}String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}String" %1, %"{{.*}}String" { ptr @{{.*}}, i64 9 })
// CHECK: call void @main.dumpUncommon(ptr [[DUMP_UNCOMMON]], %"{{.*}}String" [[UNCOMMON_SEP]])
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
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" %1)
// CHECK: [[PKG_FIELD:%.*]] = getelementptr inbounds nuw %"{{.*}}UncommonType", ptr %0, i32 0, i32 0
// CHECK: [[PKG_PATH:%.*]] = load %"{{.*}}String", ptr [[PKG_FIELD]]
// CHECK: [[MCOUNT_FIELD:%.*]] = getelementptr inbounds nuw %"{{.*}}UncommonType", ptr %0, i32 0, i32 1
// CHECK: [[MCOUNT:%.*]] = load i16, ptr [[MCOUNT_FIELD]]
// CHECK: [[XCOUNT_FIELD:%.*]] = getelementptr inbounds nuw %"{{.*}}UncommonType", ptr %0, i32 0, i32 2
// CHECK: [[XCOUNT:%.*]] = load i16, ptr [[XCOUNT_FIELD]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[PKG_PATH]])
// CHECK: [[MCOUNT64:%.*]] = zext i16 [[MCOUNT]] to i64
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 [[MCOUNT64]])
// CHECK: [[XCOUNT64:%.*]] = zext i16 [[XCOUNT]] to i64
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 [[XCOUNT64]])

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
// Constant non-direct values use read-only iface boxes instead of AllocU.
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_bool, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_int, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_int8, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_int16, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_int32, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_int64, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_uint, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_uint8, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_uint16, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_uint32, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_uint64, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_uintptr, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_float32, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_float64, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @"[10]_llgo_int", ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @"_llgo_closure${{.*}}", ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @"*_llgo_int", ptr null })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @"[]_llgo_int", ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_string, ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @"{{.*}}/cl/_testrt/eface.struct${{.*}}", ptr @"_llgo_ifacebox${{.*}}" })
// CHECK: call void @main.dump(%"{{.*}}eface" { ptr @_llgo_main.T, ptr @"_llgo_ifacebox${{.*}}" })
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

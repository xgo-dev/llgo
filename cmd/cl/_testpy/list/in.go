// LITTEST
// Scope: common
package main

import (
	_ "github.com/xgo-dev/llgo/cmd/cl/_testpy/callpy/pylink"
	_ "unsafe"
)

// This fixture owns Go-to-Python value conversion, list and tuple construction
// and access, and Python object/string access. All declarations are local so
// the compiler test does not inherit coverage from goplus/lib.
type Object struct {
	unused [0]byte
}

//go:linkname list llgo.pyList
func list(__llgo_va_list ...any) *Object

//go:linkname tuple llgo.pyTuple
func tuple(__llgo_va_list ...any) *Object

// llgo:link (*Object).IsTrue C.PyObject_IsTrue
func (*Object) IsTrue() int32 { return -1 }

// llgo:link (*Object).ListLen C.PyList_Size
func (*Object) ListLen() int { return 0 }

// llgo:link (*Object).ListItem C.PyList_GetItem
func (*Object) ListItem(index int) *Object { return nil }

// llgo:link (*Object).TupleLen C.PyTuple_Size
func (*Object) TupleLen() int { return 0 }

// llgo:link (*Object).Str C.PyObject_Str
func (*Object) Str() *Object { return nil }

// llgo:link (*Object).CStr C.PyUnicode_AsUTF8
func (*Object) CStr() *byte { return nil }

func makeValues(pointer *int) *Object {
	return list(
		true,
		int64(-2),
		uint64(3),
		4.5,
		1+2i,
		"go",
		[]byte("bytes"),
		[...]byte{7, 8},
		[...]byte{},
		pointer,
	)
}

func makeTuple() *Object {
	return tuple(1.0, 2.0)
}

func makeNested() *Object {
	return list(list(int64(10)))
}

func main() {
	value := 100
	values := makeValues(&value)
	nested := makeNested()
	items := makeTuple()
	text := values.Str()

	println("list =", values.ListLen(), values.ListItem(0).IsTrue(), values.ListItem(9).IsTrue())
	println("nested =", nested.ListLen(), nested.ListItem(0).ListLen())
	println("tuple =", items.TupleLen())
	println("str =", text.IsTrue(), text.CStr() != nil)
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[TEXT:%[0-9]+]] = call ptr @PyObject_Str(ptr %{{[0-9]+}})
// CHECK: call i64 @PyList_Size(ptr %{{[0-9]+}})
// CHECK: call ptr @PyList_GetItem(ptr %{{[0-9]+}}, i64 0)
// CHECK: call ptr @PyList_GetItem(ptr %{{[0-9]+}}, i64 9)
// CHECK: call i64 @PyList_Size(ptr %{{[0-9]+}})
// CHECK: [[INNER:%[0-9]+]] = call ptr @PyList_GetItem(ptr %{{[0-9]+}}, i64 0)
// CHECK: call i64 @PyList_Size(ptr [[INNER]])
// CHECK: call i64 @PyTuple_Size(ptr %{{[0-9]+}})
// CHECK: call i32 @PyObject_IsTrue(ptr [[TEXT]])
// CHECK: call ptr @PyUnicode_AsUTF8(ptr [[TEXT]])

// CHECK-LABEL: define ptr @main.makeNested(){{.*}} {
// CHECK: [[INNER:%[0-9]+]] = call ptr @PyList_New(i64 1)
// CHECK: [[INNER_ITEM:%[0-9]+]] = call ptr @PyLong_FromLongLong(i64 10)
// CHECK: call i32 @PyList_SetItem(ptr [[INNER]], i64 0, ptr [[INNER_ITEM]])
// CHECK: [[NESTED:%[0-9]+]] = call ptr @PyList_New(i64 1)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[NESTED]], i64 0, ptr [[INNER]])
// CHECK: ret ptr [[NESTED]]

// CHECK-LABEL: define ptr @main.makeTuple(){{.*}} {
// CHECK: [[TUPLE:%[0-9]+]] = call ptr @PyTuple_New(i64 2)
// CHECK: call i32 @PyTuple_SetItem(ptr [[TUPLE]], i64 0, ptr %{{[0-9]+}})
// CHECK: call i32 @PyTuple_SetItem(ptr [[TUPLE]], i64 1, ptr %{{[0-9]+}})

// CHECK-LABEL: define ptr @main.makeValues(ptr %{{[0-9]+}}){{.*}} {
// CHECK: [[VALUES:%[0-9]+]] = call ptr @PyList_New(i64 10)
// CHECK: [[BOOL:%[0-9]+]] = call ptr @PyBool_FromLong(i32 {{-?1}})
// CHECK: call i32 @PyList_SetItem(ptr [[VALUES]], i64 0, ptr [[BOOL]])
// CHECK: call ptr @PyLong_FromLongLong(i64 -2)
// CHECK: call ptr @PyLong_FromUnsignedLongLong(i64 3)
// CHECK: call ptr @PyFloat_FromDouble(double 4.500000e+00)
// CHECK: call ptr @PyComplex_FromDoubles(double 1.000000e+00, double 2.000000e+00)
// CHECK: call ptr @PyUnicode_FromStringAndSize(ptr {{.*}}, i64 2)
// CHECK: call ptr @PyByteArray_FromStringAndSize(ptr %{{[0-9]+}}, i64 %{{[0-9]+}})
// CHECK: call ptr @PyBytes_FromStringAndSize(ptr {{.*}}, i64 2)
// CHECK: call ptr @PyBytes_FromStringAndSize(ptr null, i64 0)
// CHECK: ptrtoint ptr %{{[0-9]+}} to i{{32|64}}
// CHECK: [[POINTER:%[0-9]+]] = call ptr @PyLong_FromUnsignedLongLong(i64 %{{[0-9]+}})
// CHECK: call i32 @PyList_SetItem(ptr [[VALUES]], i64 9, ptr [[POINTER]])

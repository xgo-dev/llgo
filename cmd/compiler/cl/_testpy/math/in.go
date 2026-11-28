// LITTEST
// Scope: common
package math

import _ "unsafe"

// This fixture owns local Python-module classification, import, function
// declaration, module-variable lookup, and storage without importing a wrapper
// library.
const LLGoPackage = "py.math"

type Object struct {
	unused [0]byte
}

//go:linkname Pi py.pi
var Pi *Object

//go:linkname Sqrt py.sqrt
func Sqrt(x *Object) *Object

func ReadPi() *Object {
	return Pi
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testpy/math.ReadPi"(){{.*}} {
// CHECK: [[MATH:%[0-9]+]] = load ptr, ptr @__llgo_py.math
// CHECK: [[PI:%[0-9]+]] = call ptr @PyObject_GetAttrString(ptr [[MATH]], ptr @{{[0-9]+}})
// CHECK: ret ptr [[PI]]

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/math.init"(){{.*}} {
// CHECK: [[MATH:%[0-9]+]] = call ptr @PyImport_ImportModule(ptr @{{[0-9]+}})
// CHECK: store ptr [[MATH]], ptr @__llgo_py.math

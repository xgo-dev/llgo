package builtins

import _ "unsafe"

const LLGoPackage = "py.builtins"

type Object struct {
	unused [0]byte
}

//go:linkname NewFloat C.PyFloat_FromDouble
func NewFloat(v float64) *Object

// llgo:link (*Object).IsTrue C.PyObject_IsTrue
func (*Object) IsTrue() int32 { return -1 }

// llgo:link (*Object).Float64 C.PyFloat_AsDouble
func (*Object) Float64() float64 { return 0 }

// llgo:link (*Object).Long C.PyLong_AsLong
func (*Object) Long() cLong { return 0 }

//go:linkname Bool py.bool
func Bool() *Object

//go:linkname Float py.float
func Float(v *Object) *Object

//go:linkname Pow py.pow
func Pow(base, exp *Object) *Object

//go:linkname Int py.int
func Int(v *Object) *Object

//go:linkname Max py.max
func Max(__llgo_va_list ...any) *Object

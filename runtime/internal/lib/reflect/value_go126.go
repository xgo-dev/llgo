//go:build go1.26
// +build go1.26

package reflect

import (
	"iter"
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

func (v Value) abiType() *abi.Type {
	if v.flag != 0 && v.flag&flagMethod == 0 && !v.typ_.IsClosure() {
		return v.typ()
	}
	return v.abiTypeSlow()
}

func (v Value) abiTypeSlow() *abi.Type {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Type", Invalid})
	}

	typ := v.typ()
	if v.typ_.IsClosure() {
		return &v.closureFunc().Type
	}
	if v.flag&flagMethod == 0 {
		return typ
	}

	i := int(v.flag) >> flagMethodShift
	if typ.Kind() == abi.Interface {
		tt := (*interfaceType)(unsafe.Pointer(typ))
		if uint(i) >= uint(len(tt.Methods)) {
			panic("reflect: internal error: invalid method index")
		}
		return &tt.Methods[i].Typ_.Type
	}

	ms := typ.ExportedMethods()
	if uint(i) >= uint(len(ms)) {
		panic("reflect: internal error: invalid method index")
	}
	return &ms[i].Mtyp_.Type
}

func (v Value) Fields() iter.Seq2[StructField, Value] {
	t := v.Type()
	if t.Kind() != Struct {
		panic("reflect: Fields of non-struct type " + t.String())
	}
	return func(yield func(StructField, Value) bool) {
		for i := 0; i < v.NumField(); i++ {
			if !yield(t.Field(i), v.Field(i)) {
				return
			}
		}
	}
}

func (v Value) Methods() iter.Seq2[Method, Value] {
	return func(yield func(Method, Value) bool) {
		t := v.Type()
		for i := 0; i < v.NumMethod(); i++ {
			if !yield(t.Method(i), v.Method(i)) {
				return
			}
		}
	}
}

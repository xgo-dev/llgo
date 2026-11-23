//go:build llgo

package gotest

import (
	"go/ast"
	"reflect"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

type descriptorMethods interface {
	M() int
}

type descriptorOther interface {
	M() int
}

type descriptorAlias = descriptorMethods

type descriptorValue struct{}

func (*descriptorValue) M() int { return 42 }

func TestInterfaceTargetDescriptor(t *testing.T) {
	type itabHeader struct {
		inter *abi.Type
		typ   *abi.Type
	}
	type ifaceHeader struct {
		tab  *itabHeader
		data unsafe.Pointer
	}
	check := func(name string, storage unsafe.Pointer, target reflect.Type) {
		t.Helper()
		got := (*ifaceHeader)(storage).tab
		want := (*ifaceHeader)(unsafe.Pointer(&target)).data
		if got == nil || unsafe.Pointer(got.inter) != want {
			t.Fatalf("%s: target interface descriptor differs from reflect.Type", name)
		}
	}
	p := &descriptorValue{}
	var direct descriptorMethods = p
	check("direct", unsafe.Pointer(&direct), reflect.TypeOf((*descriptorMethods)(nil)).Elem())
	var other descriptorOther = p
	check("other named interface", unsafe.Pointer(&other), reflect.TypeOf((*descriptorOther)(nil)).Elem())
	var converted descriptorMethods = other
	check("converted", unsafe.Pointer(&converted), reflect.TypeOf((*descriptorMethods)(nil)).Elem())
	fromInterface := other.(descriptorMethods)
	check("interface assertion", unsafe.Pointer(&fromInterface), reflect.TypeOf((*descriptorMethods)(nil)).Elem())
	var nilSource descriptorOther
	var nilTarget descriptorMethods = nilSource
	if nilTarget != nil {
		t.Fatal("nil interface conversion produced a non-nil value")
	}
	var typedNil descriptorOther = (*descriptorValue)(nil)
	var convertedNil descriptorMethods = typedNil
	check("typed nil conversion", unsafe.Pointer(&convertedNil), reflect.TypeOf((*descriptorMethods)(nil)).Elem())
	if convertedNil == nil || (*ifaceHeader)(unsafe.Pointer(&convertedNil)).data != nil {
		t.Fatal("typed nil conversion changed the dynamic value")
	}
	var empty any = p
	asserted := empty.(descriptorMethods)
	check("asserted", unsafe.Pointer(&asserted), reflect.TypeOf((*descriptorMethods)(nil)).Elem())
	comma, ok := empty.(descriptorMethods)
	if !ok {
		t.Fatal("assertion failed")
	}
	check("comma-ok", unsafe.Pointer(&comma), reflect.TypeOf((*descriptorMethods)(nil)).Elem())
	var alias descriptorAlias = p
	check("alias", unsafe.Pointer(&alias), reflect.TypeOf((*descriptorAlias)(nil)).Elem())
	var anonymous interface{ M() int } = p
	check("anonymous", unsafe.Pointer(&anonymous), reflect.TypeOf((*interface{ M() int })(nil)).Elem())
	var reflected descriptorMethods
	reflect.ValueOf(&reflected).Elem().Set(reflect.ValueOf(p))
	check("reflected", unsafe.Pointer(&reflected), reflect.TypeOf((*descriptorMethods)(nil)).Elem())
	for _, v := range []descriptorMethods{direct, converted, asserted, comma, alias, reflected} {
		if v.M() != 42 {
			t.Fatal("method invocation failed")
		}
	}
	node := new(ast.BinaryExpr)
	var expr ast.Expr = node
	check("ast.Expr", unsafe.Pointer(&expr), reflect.TypeOf((*ast.Expr)(nil)).Elem())
}

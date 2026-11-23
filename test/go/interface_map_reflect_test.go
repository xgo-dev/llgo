package gotest

import (
	"go/ast"
	"reflect"
	"testing"
)

func TestInterfaceMapReflectAssignment(t *testing.T) {
	node := reflect.New(reflect.TypeOf(ast.BinaryExpr{})).Interface().(ast.Node)
	parent := new(ast.BinaryExpr)
	// Reflection builds the itab from the declared field type; compiler-built
	// itabs must use the same descriptor.
	reflect.ValueOf(parent).Elem().FieldByName("X").Set(reflect.ValueOf(node))
	reflected := parent.X
	var direct ast.Expr = reflected.(*ast.BinaryExpr)
	if reflected != direct {
		t.Fatal("interface values with the same dynamic pointer differ")
	}

	for _, keys := range [][2]ast.Expr{{reflected, direct}, {direct, reflected}} {
		m := map[ast.Expr]int{keys[0]: 42}
		if got, ok := m[keys[1]]; !ok || got != 42 {
			t.Fatalf("lookup equivalent key = (%d, %v), want (42, true)", got, ok)
		}
		m[keys[1]] = 43
		if len(m) != 1 || m[keys[0]] != 43 {
			t.Fatalf("update equivalent key: len = %d, value = %d", len(m), m[keys[0]])
		}
		delete(m, keys[1])
		if len(m) != 0 {
			t.Fatal("delete equivalent key retained entry")
		}

		type key struct{ Expr ast.Expr }
		composite := map[key]int{{keys[0]}: 7}
		if composite[key{keys[1]}] != 7 {
			t.Fatal("struct interface field lookup failed")
		}
		array := map[[1]ast.Expr]int{{keys[0]}: 9}
		if array[[1]ast.Expr{keys[1]}] != 9 {
			t.Fatal("array interface element lookup failed")
		}
	}
}

func TestInterfaceMapNilAndDynamicTypes(t *testing.T) {
	var binary *ast.BinaryExpr
	var unary *ast.UnaryExpr
	parent := new(ast.BinaryExpr)
	var node ast.Node = binary
	reflect.ValueOf(parent).Elem().FieldByName("X").Set(reflect.ValueOf(node))
	var direct ast.Expr = binary
	m := map[ast.Expr]int{nil: 1, parent.X: 2, unary: 3}
	if len(m) != 3 || m[nil] != 1 || m[direct] != 2 || m[unary] != 3 {
		t.Fatalf("nil and typed nil keys: len = %d, values = (%d, %d, %d)", len(m), m[nil], m[direct], m[unary])
	}
	if _, ok := m[new(ast.BinaryExpr)]; ok {
		t.Fatal("distinct pointer matched a typed nil key")
	}
}

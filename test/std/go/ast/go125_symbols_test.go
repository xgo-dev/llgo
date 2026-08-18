//go:build go1.25

package ast_test

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestPreorderStack(t *testing.T) {
	expr, err := parser.ParseExpr("left + right")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	ast.PreorderStack(expr, nil, func(node ast.Node, stack []ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && ident.Name == "right" {
			found = true
			if len(stack) != 1 || stack[0] != expr {
				t.Fatalf("stack for right identifier = %#v, want expression root", stack)
			}
		}
		return true
	})
	if !found {
		t.Fatal("PreorderStack did not visit the right identifier")
	}
}

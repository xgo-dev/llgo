package midway

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestTypeExprRejectsNonValueType(t *testing.T) {
	fset := token.NewFileSet()
	file := fset.AddFile("initializer.go", -1, 10)
	r := &rewriter{fset: fset}
	fr := &fileRewriter{rewriter: r}
	// A result tuple must be split into individual initializer result types;
	// it cannot itself become the type of a generated Go variable.
	tuple := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))
	expr := fr.typeExpr(tuple, -1, file.Pos(0))
	if r.err == nil || !strings.Contains(r.err.Error(), "initializer.go:1:1: unsupported SIMD initializer type") {
		t.Fatalf("missing diagnostic for a non-value type: %v", r.err)
	}
	if _, ok := expr.(*ast.BadExpr); !ok {
		t.Fatalf("unsupported type produced usable syntax: %T", expr)
	}
}

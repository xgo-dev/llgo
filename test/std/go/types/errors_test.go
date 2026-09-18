package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestTypeErrors(t *testing.T) {
	src := `package broken

func _() {
    var s string = 1
    var _ = undefined + 1
}`

	var errors []types.Error
	conf := types.Config{
		Importer: testImporter(t),
		Error: func(err error) {
			if terr, ok := err.(types.Error); ok {
				errors = append(errors, terr)
			} else {
				t.Fatalf("unexpected error type %T", err)
			}
		},
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "broken.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, err = conf.Check("broken", fset, []*ast.File{file}, nil)
	if err == nil {
		t.Fatal("expected type-check to report errors")
	}

	if len(errors) < 2 {
		t.Fatalf("expected at least 2 errors, got %d", len(errors))
	}

	var sawAssignError, sawUndefined bool
	for _, typeErr := range errors {
		if typeErr.Error() == "" {
			t.Fatal("types.Error.Error returned empty")
		}
		if strings.Contains(typeErr.Msg, "string") {
			sawAssignError = true
		}
		if strings.Contains(typeErr.Msg, "undefined") {
			sawUndefined = true
		}
	}
	if !sawAssignError {
		t.Fatalf("missing string assignment error: %v", errors)
	}
	if !sawUndefined {
		t.Fatalf("missing undefined identifier error: %v", errors)
	}
}

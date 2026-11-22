package llvmattr

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// Check the audited list against declarations, including target-specific files.
// This catches stale names and commented-out upstream runtime implementations.
func TestRuntimeAttributeModelSymbols(t *testing.T) {
	files, err := filepath.Glob("../../runtime/internal/runtime/*.go")
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
				names[fn.Name.Name] = true
			}
		}
	}
	for name := range runtimeFunctionModels {
		if !names[name] {
			t.Errorf("modeled runtime function %s does not exist", name)
		}
	}
}

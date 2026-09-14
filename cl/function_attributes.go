package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/funcattrs"
)

func isFunctionAttributeComment(line string) bool {
	item, ok := directive.Parse(&ast.Comment{Text: line})
	return ok && (item.Name == "llgo:cold" || item.Name == "llgo:noreturn" || funcattrs.IsSourceDirective(item))
}

func validateFunctionAttributes(fset *token.FileSet, file *ast.File) error {
	allowed := make(map[token.Pos]bool)
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Doc != nil {
			for _, c := range fn.Doc.List {
				allowed[c.Pos()] = true
			}
		}
	}
	for _, group := range file.Comments {
		for _, item := range directive.ParseGroup(group) {
			name := item.Name
			if i := strings.IndexAny(name, "(."); i >= 0 {
				name = name[:i]
			}
			if funcattrs.IsSourceDirective(item) && !allowed[item.Pos] {
				return fmt.Errorf("%s: %s requires a named function or method declaration", fset.Position(item.Pos), name)
			}
			switch name {
			case "llgo:cold", "llgo:noreturn":
				if !allowed[item.Pos] {
					return fmt.Errorf("%s: %s requires a named function or method declaration", fset.Position(item.Pos), name)
				}
				if name != item.Name || item.Args != "" {
					return fmt.Errorf("%s: %s takes no arguments; write each function attribute on its own line", fset.Position(item.Pos), name)
				}
			}
		}
	}
	return nil
}

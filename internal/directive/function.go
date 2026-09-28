package directive

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// ValidateFunctionAttributes uses prepared groups; it never reinterprets the
// original comments. Placement errors retain the annotation's source position.
func (f *File) ValidateFunctionAttributes(fset *token.FileSet) error {
	allowed := make(map[*ast.CommentGroup]bool)
	for _, node := range f.Syntax.Decls {
		if d, ok := node.(*ast.FuncDecl); ok && d.Name.Name != "_" && (d.Recv != nil || d.Name.Name != "init") {
			allowed[d.Doc] = true
		}
	}
	// Walk source order so multiple invalid annotations have stable diagnostics.
	for _, doc := range f.Syntax.Comments {
		for _, d := range f.Group(doc).Items {
			name := d.Name
			if i := strings.IndexAny(name, "(."); i >= 0 {
				name = name[:i]
			}
			switch name {
			case "llgo:param", "llgo:result", "llgo:receiver", "llgo:nonnull", "llgo:range", "llgo:nonnegative", "llgo:sameas", "llgo:access", "llgo:noalias":
				if !allowed[doc] {
					return fmt.Errorf("%s: %s requires a named function or method declaration", fset.Position(d.Pos), name)
				}
			case "llgo:cold", "llgo:noreturn":
				if !allowed[doc] {
					return fmt.Errorf("%s: %s requires a named function or method declaration", fset.Position(d.Pos), name)
				}
				if name != d.Name || d.Args != "" {
					return fmt.Errorf("%s: %s takes no arguments", fset.Position(d.Pos), name)
				}
			}
		}
	}
	for _, node := range f.Syntax.Decls {
		if decl, ok := node.(*ast.FuncDecl); ok {
			if err := f.Functions[decl].WithPositions(fset).ContractError; err != nil {
				return err
			}
		}
	}
	return nil
}

package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// Recognition, validation and collection use the same supported directive set.
var functionAttributeDirectives = map[string]llssa.FunctionAttributes{
	"llgo:cold":     llssa.FunctionCold,
	"llgo:noreturn": llssa.FunctionNoReturn,
}

func isFunctionAttributeComment(line string) bool {
	item, ok := directive.Parse(&ast.Comment{Text: line})
	return ok && functionAttributeDirectives[item.Name] != 0
}

func validateFunctionAttributes(fset *token.FileSet, file *ast.File) error {
	functionDocs := make(map[*ast.CommentGroup]bool)
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Doc != nil {
			functionDocs[fn.Doc] = true
		}
	}
	// Inspect all comments so attributes inside bodies or on other declarations
	// are diagnosed as well as attributes attached to functions.
	for _, group := range file.Comments {
		for _, item := range directive.ParseGroup(group) {
			if err := validateFunctionAttribute(item, functionDocs[group]); err != nil {
				return fmt.Errorf("%s: %w", fset.Position(item.Pos), err)
			}
		}
	}
	return nil
}

func validateFunctionAttribute(item directive.Directive, onFunction bool) error {
	name := item.Name
	if i := strings.IndexAny(name, "(."); i >= 0 {
		name = name[:i]
	}
	switch name {
	case "llgo:param", "llgo:result", "llgo:receiver":
		return fmt.Errorf("%s attributes are not yet supported", name)
	}
	if functionAttributeDirectives[name] == 0 {
		return nil
	}
	if !onFunction {
		return fmt.Errorf("%s requires a named function or method declaration", name)
	}
	if name != item.Name || item.Args != "" {
		return fmt.Errorf("%s takes no arguments; write each function attribute on its own line", name)
	}
	return nil
}

func parseFunctionAttributes(doc *ast.CommentGroup) llssa.FunctionAttributes {
	var attrs llssa.FunctionAttributes
	for _, item := range directive.ParseGroup(doc) {
		attrs |= functionAttributeDirectives[item.Name]
	}
	return attrs
}

// Generic instances retain their source declaration through Origin. Read its
// syntax directly, including private functions omitted by export-data preload.
func sourceFunctionAttributes(fn *ssa.Function) llssa.FunctionAttributes {
	if origin := fn.Origin(); origin != nil {
		fn = origin
	}
	if decl, ok := fn.Syntax().(*ast.FuncDecl); ok {
		return parseFunctionAttributes(decl.Doc)
	}
	return 0
}

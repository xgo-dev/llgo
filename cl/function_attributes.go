package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/exportdata"
)

// CollectPackageExports resolves properties from the effective package syntax.
// Combined patch files place replacements last, including declarations that
// remove an attribute. No AST or source-position identity survives export.
func CollectPackageExports(fset *token.FileSet, files []*ast.File) (*exportdata.Package, error) {
	functions := make(map[string]exportdata.Function)
	for _, file := range files {
		if err := validateFunctionAttributes(fset, file); err != nil {
			return nil, err
		}
		for _, node := range file.Decls {
			decl, ok := node.(*ast.FuncDecl)
			if !ok || decl.Recv == nil && decl.Name.Name == "init" {
				continue
			}
			_, name := astFuncName("", decl)
			f := exportdata.Function{Name: name}
			for _, attr := range directive.ParseGroup(decl.Doc) {
				switch attr.Name {
				case "llgo:cold":
					f.Cold = true
				case "llgo:noreturn":
					f.NoReturn = true
				}
			}
			functions[name] = f
		}
	}
	data := &exportdata.Package{Version: exportdata.Version, Functions: []exportdata.Function{}}
	for _, f := range functions {
		if f.Cold || f.NoReturn {
			data.Functions = append(data.Functions, f)
		}
	}
	sort.Slice(data.Functions, func(i, j int) bool { return data.Functions[i].Name < data.Functions[j].Name })
	return data, nil
}

func validateFunctionAttributes(fset *token.FileSet, file *ast.File) error {
	allowed := make(map[token.Pos]*ast.FuncDecl)
	for _, node := range file.Decls {
		if decl, ok := node.(*ast.FuncDecl); ok && decl.Doc != nil {
			for _, comment := range decl.Doc.List {
				allowed[comment.Pos()] = decl
			}
		}
	}
	for _, group := range file.Comments {
		for _, item := range directive.ParseGroup(group) {
			name := item.Name
			if i := strings.IndexAny(name, "(."); i >= 0 {
				name = name[:i]
			}
			switch name {
			case "llgo:param", "llgo:result", "llgo:receiver":
				return fmt.Errorf("%s: %s attributes are not yet supported", fset.Position(item.Pos), name)
			case "llgo:cold", "llgo:noreturn":
				decl := allowed[item.Pos]
				if decl == nil {
					return fmt.Errorf("%s: %s requires a named function or method declaration", fset.Position(item.Pos), name)
				}
				if decl.Recv == nil && (decl.Name.Name == "init" || decl.Name.Name == "_") {
					return fmt.Errorf("%s: %s cannot annotate %s functions", fset.Position(item.Pos), name, decl.Name.Name)
				}
				if name != item.Name || item.Args != "" {
					return fmt.Errorf("%s: %s takes no arguments", fset.Position(item.Pos), name)
				}
			}
		}
	}
	return nil
}

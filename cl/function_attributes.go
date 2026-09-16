package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
)

// FunctionAttributes indexes source comments for imported functions without syntax.
// The loader fills it before starting backend compilation. Attribute values
// live on aFunction, not in this index. The zero value is ready for use.
type FunctionAttributes struct {
	mu       sync.RWMutex
	packages map[*types.Package]map[string]*ast.CommentGroup
}

func (d *FunctionAttributes) lookup(pkg *types.Package, name string) (*ast.CommentGroup, bool) {
	if d == nil {
		return nil, false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	doc, ok := d.packages[pkg][name]
	return doc, ok
}

func (d *FunctionAttributes) collect(pkg *types.Package, files []*ast.File) {
	if d == nil {
		return
	}
	functions := make(map[string]*ast.CommentGroup)
	for _, file := range files {
		for _, node := range file.Decls {
			if decl, ok := node.(*ast.FuncDecl); ok {
				if decl.Recv == nil && decl.Name.Name == "init" {
					continue
				}
				name, _ := astFuncName(llssa.PathOf(pkg), decl)
				// Combined patch files put replacements after original files.
				// A nil doc also overrides attributes on the original function.
				functions[name] = decl.Doc
			}
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.packages == nil {
		d.packages = make(map[*types.Package]map[string]*ast.CommentGroup)
	}
	d.packages[pkg] = functions
}

func validateFunctionAttributes(fset *token.FileSet, file *ast.File) error {
	allowed := make(map[token.Pos]bool)
	for _, node := range file.Decls {
		if decl, ok := node.(*ast.FuncDecl); ok && decl.Doc != nil {
			for _, comment := range decl.Doc.List {
				allowed[comment.Pos()] = true
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
				if !allowed[item.Pos] {
					return fmt.Errorf("%s: %s requires a named function or method declaration", fset.Position(item.Pos), name)
				}
				if name != item.Name || item.Args != "" {
					return fmt.Errorf("%s: %s takes no arguments", fset.Position(item.Pos), name)
				}
			}
		}
	}
	return nil
}

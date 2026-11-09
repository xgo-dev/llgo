package compile

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Package discovery does not compile gc export data for source-patched
// packages. Enforce the requested source-only compiler checks ourselves, so
// LLGo extensions can be used with -complete and -std too.
func validateCompilerSources(files []string, opts *options) error {
	if !opts.complete && !opts.standard {
		return nil
	}
	fset := token.NewFileSet()
	for _, filename := range files {
		file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		linknames := make(map[string]bool)
		for _, group := range file.Comments {
			for _, comment := range group.List {
				fields := strings.Fields(comment.Text)
				if len(fields) >= 2 && fields[0] == "//go:linkname" {
					linknames[fields[1]] = true
				}
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			pragmas := make(map[string]bool)
			if fn.Doc != nil {
				for _, comment := range fn.Doc.List {
					fields := strings.Fields(comment.Text)
					if len(fields) != 0 {
						pragmas[fields[0]] = true
					}
				}
			}
			message := ""
			if opts.complete && fn.Body == nil && !linknames[fn.Name.Name] && !pragmas["//go:wasmimport"] {
				message = "missing function body"
			}
			if opts.standard && fn.Body != nil && pragmas["//go:uintptrkeepalive"] && !pragmas["//go:uintptrescapes"] && !pragmas["//go:nosplit"] {
				message = "go:uintptrkeepalive requires go:nosplit"
			}
			if message != "" {
				pos := fset.Position(fn.Pos())
				if opts.noColumns.value != 0 {
					pos.Column = 0
				}
				return fmt.Errorf("%s: %s", pos, message)
			}
		}
	}
	return nil
}

package build

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
)

// Decide before importing frontend contracts: cached dependency archives must
// use the same allocation-frame policy as newly compiled packages. Cached type
// exports may omit syntax, so inspect only source files mentioning MemProfile.
func (c *context) sourceUsesHeapProfiles(entries []ssaBuildEntry) (bool, error) {
	for _, entry := range entries {
		if entry.pkg == nil || llssa.IsRuntimeSupportPackage(entry.pkg.Pkg.Path()) {
			continue
		}
		if len(entry.syntax) != 0 {
			for _, file := range entry.syntax {
				if fileUsesHeapProfiles(file) {
					return true, nil
				}
			}
			continue
		}
		cached := c.dedup.Check(entry.id)
		if cached == nil {
			continue
		}
		for _, filename := range cached.GoFiles {
			data, err := os.ReadFile(filename)
			if err != nil {
				return false, err
			}
			if !bytes.Contains(data, []byte("MemProfile")) {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filename, data, parser.ParseComments)
			if err != nil {
				return false, err
			}
			if fileUsesHeapProfiles(file) {
				return true, nil
			}
		}
	}
	return false, nil
}

func fileUsesHeapProfiles(file *ast.File) bool {
	aliases := make(map[string]bool)
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != "runtime" && path != "github.com/xgo-dev/llgo/runtime/internal/lib/runtime" {
			continue
		}
		name := "runtime"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		aliases[name] = true
	}
	observed := false
	ast.Inspect(file, func(node ast.Node) bool {
		if observed {
			return false
		}
		switch n := node.(type) {
		case *ast.SelectorExpr:
			id, ok := n.X.(*ast.Ident)
			observed = ok && aliases[id.Name] && n.Sel.Name == "MemProfile"
		case *ast.Ident:
			observed = aliases["."] && n.Name == "MemProfile"
		}
		return true
	})
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:linkname ") && strings.HasSuffix(strings.TrimSpace(comment.Text), " runtime.MemProfile") {
				observed = true
			}
		}
	}
	return observed
}

package build

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const errorTypeCacheFile = "z_llgo_errors_type.go"

const errorTypeCacheSource = `package errors

import (
	"internal/reflectlite"
	"sync/atomic"
)

var llgoErrorTypeCache atomic.Pointer[reflectlite.Type]

func llgoErrorType() reflectlite.Type {
	if cached := llgoErrorTypeCache.Load(); cached != nil {
		return *cached
	}
	// Concurrent first calls may publish the same canonical descriptor.
	typ := reflectlite.TypeOf((*error)(nil)).Elem()
	llgoErrorTypeCache.Store(&typ)
	return typ
}
`

func deferredErrorTypeSource(goroot string, overlay map[string][]byte) (string, []byte) {
	dir := filepath.Join(goroot, "src", "errors")
	path := filepath.Join(dir, "wrap.go")
	for file := range overlay {
		if filepath.Dir(file) == dir {
			return path, nil
		}
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return path, nil
	}
	for _, file := range files {
		if file.Name() == errorTypeCacheFile {
			return path, nil
		}
		if file.IsDir() || file.Name() == "wrap.go" || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil || bytes.Contains(source, []byte("errorType")) || bytes.Contains(source, []byte("llgoErrorType")) {
			return path, nil
		}
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return path, nil
	}
	return path, deferErrorTypeInit(source)
}

// The descriptor is canonical and the only use is the concrete-target
// validation in errors.As. Leave it out of package initialization so programs
// that merely construct or return errors avoid the reflective call graph.
func deferErrorTypeInit(source []byte) []byte {
	if bytes.Contains(source, []byte("llgoErrorType")) {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "wrap.go", source, parser.ParseComments)
	if err != nil || file.Name.Name != "errors" {
		return nil
	}
	if !slices.ContainsFunc(file.Imports, func(spec *ast.ImportSpec) bool {
		return spec.Name == nil && spec.Path.Value == `"internal/reflectlite"`
	}) {
		return nil
	}
	object := file.Scope.Lookup("errorType")
	if object == nil {
		return nil
	}
	value, ok := object.Decl.(*ast.ValueSpec)
	if !ok || len(value.Names) != 1 || len(value.Values) != 1 ||
		types.ExprString(value.Values[0]) != "reflectlite.TypeOf((*error)(nil)).Elem()" {
		return nil
	}
	var decl *ast.GenDecl
	var use *ast.Ident
	uses := 0
	for _, node := range file.Decls {
		if candidate, ok := node.(*ast.GenDecl); ok && candidate.Tok == token.VAR && len(candidate.Specs) == 1 && candidate.Specs[0] == value {
			decl = candidate
		}
		as, isAs := node.(*ast.FuncDecl)
		isAs = isAs && as.Recv == nil && as.Name.Name == "As"
		ast.Inspect(node, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.Ident:
				if node.Obj == object && node != value.Names[0] {
					uses++
				}
			case *ast.CallExpr:
				if fn, ok := node.Fun.(*ast.SelectorExpr); isAs && ok && fn.Sel.Name == "Implements" && len(node.Args) == 1 {
					if arg, ok := node.Args[0].(*ast.Ident); ok && arg.Obj == object {
						use = arg
					}
				}
			}
			return true
		})
	}
	if decl == nil || uses != 1 || use == nil {
		return nil
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.Contains(comment.Text, "errorType") &&
				(strings.HasPrefix(comment.Text, "//go:linkname") || strings.HasPrefix(comment.Text, "//llgo:link")) {
				return nil
			}
		}
	}
	offset := fset.File(file.Pos()).Offset
	result := slices.Clone(source)
	for i := offset(decl.Pos()); i < offset(decl.End()); i++ {
		if result[i] != '\n' && result[i] != '\r' {
			result[i] = ' '
		}
	}
	return slices.Concat(result[:offset(use.Pos())], []byte("llgoErrorType()"), result[offset(use.End()):])
}

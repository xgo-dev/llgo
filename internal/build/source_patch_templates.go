package build

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
)

func templateDocs(group *ast.CommentGroup) []string {
	var docs []string
	for _, item := range directive.ParseGroup(group) {
		switch item.Name {
		case "llgo:template", "llgointernal:template:gofmt":
			docs = append(docs, "//"+item.Name+" "+item.Args)
		}
	}
	return docs
}

func sourceWithoutTemplateStubs(src []byte, spans []sourcePatchSpan) []byte {
	if len(spans) == 0 {
		return src
	}
	// Remove imports referenced only by forwarding declarations as well.
	fset := token.NewFileSet()
	stripped := commentSourcePatchSpans(src, spans)
	file, err := parser.ParseFile(fset, "", stripped, parser.ParseComments)
	if err != nil {
		return stripped
	}
	return commentSourcePatchSpans(stripped, collectUnusedImportSpans(fset.File(file.Pos()), file))
}

func templateNames(docs map[string][]string) map[string]struct{} {
	names := make(map[string]struct{}, len(docs))
	for name := range docs {
		names[name] = struct{}{}
	}
	return names
}

// A bodyless template declaration in a source patch carries its directives to
// the original declaration. The standard library remains the generic body.
func annotateSourceTemplates(src []byte, docs map[string][]string) ([]byte, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, false, err
	}
	pos := fset.File(file.Pos())
	var out []byte
	last := 0
	newline := "\n"
	if first := bytes.IndexByte(src, '\n'); first > 0 && src[first-1] == '\r' {
		newline = "\r\n"
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || len(docs[fn.Name.Name]) == 0 {
			continue
		}
		offset := pos.Offset(fn.Pos())
		out = append(out, src[last:offset]...)
		out = append(out, strings.Join(docs[fn.Name.Name], newline)...)
		out = append(out, newline...)
		last = offset
	}
	if out == nil {
		return src, false, nil
	}
	return slices.Concat(out, src[last:]), true, nil
}

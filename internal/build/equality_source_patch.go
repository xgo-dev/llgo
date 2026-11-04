package build

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// Equality is an explicit runtime implementation selection, not translation
// coverage for the replaced SDK assembly. Both original public roots must be
// present exactly once and have a real selected Go body with the matching ABI.
func validateEqualitySourcePatchOwners(assembly []byte, bodies [][]byte) error {
	want := map[string]int{"runtime.memequal": 3, "runtime.memequal_varlen": 2}
	texts := make(map[string]bool)
	textLine := regexp.MustCompile(`^TEXT\s+([^\s(,]+)\(SB\)(?:\s*,|\s*$)`)
	for _, line := range strings.Split(string(assembly), "\n") {
		code := strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		match := textLine.FindStringSubmatch(code)
		declarations := 0
		for _, word := range strings.FieldsFunc(code, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ';'
		}) {
			if word == "TEXT" {
				declarations++
			}
		}
		// Unknown declaration layout is not silently omitted, including a
		// second semicolon-separated or macro-produced public TEXT owner.
		if declarations > 1 || declarations == 1 && match == nil {
			return fmt.Errorf("unrecognized equality TEXT declaration %q", code)
		}
		if match == nil {
			continue
		}
		name := match[1]
		if strings.HasSuffix(name, "<>") {
			continue // A private helper is owned with its whole original source.
		}
		name = strings.TrimSuffix(strings.TrimSuffix(name, "<ABIInternal>"), "<ABI0>")
		name = strings.ReplaceAll(name, "·", ".")
		if want[name] == 0 || texts[name] {
			return fmt.Errorf("new or duplicate public equality assembly entry %q", name)
		}
		texts[name] = true
	}
	if len(texts) != len(want) {
		return fmt.Errorf("incomplete public equality assembly group")
	}
	owners := make(map[string]bool)
	for _, source := range bodies {
		file, err := parser.ParseFile(token.NewFileSet(), "equality-owner.go", source, parser.ParseComments)
		if err != nil {
			return err
		}
		unsafeName := ""
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if path == "unsafe" {
				unsafeName = "unsafe"
				if imp.Name != nil {
					unsafeName = imp.Name.Name
				}
			}
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Doc == nil {
				continue
			}
			name, env := "", false
			for _, comment := range fn.Doc.List {
				words := strings.Fields(comment.Text)
				if len(words) == 3 && words[0] == "//go:linkname" && words[1] == fn.Name.Name {
					name = words[2]
				}
				env = env || comment.Text == "//llgo:env" || comment.Text == "// llgo:env"
			}
			count := want[name]
			if count == 0 {
				continue
			}
			if owners[name] || fn.Recv != nil || fn.Body == nil || len(fn.Body.List) == 0 ||
				env != (name == "runtime.memequal_varlen") || !equalitySourcePatchSignature(fn.Type, count, unsafeName) {
				return fmt.Errorf("missing, duplicate or incompatible real equality Go body for %s", name)
			}
			owners[name] = true
		}
	}
	if len(owners) != len(want) {
		return fmt.Errorf("both equality public entries require selected real Go bodies")
	}
	return nil
}

func equalitySourcePatchSignature(signature *ast.FuncType, count int, unsafeName string) bool {
	if signature.TypeParams != nil || signature.Params == nil || signature.Results == nil ||
		len(signature.Results.List) != 1 || signature.Results.NumFields() != 1 {
		return false
	}
	result, ok := signature.Results.List[0].Type.(*ast.Ident)
	if !ok || result.Name != "bool" {
		return false
	}
	index := 0
	for _, field := range signature.Params.List {
		parameters := len(field.Names)
		if parameters == 0 {
			parameters = 1
		}
		for i := 0; i < parameters; i++ {
			if index < 2 {
				pointer, ok := field.Type.(*ast.SelectorExpr)
				if !ok {
					return false
				}
				qualifier, ok := pointer.X.(*ast.Ident)
				if !ok || unsafeName == "" || qualifier.Name != unsafeName || pointer.Sel.Name != "Pointer" {
					return false
				}
			} else {
				size, ok := field.Type.(*ast.Ident)
				if !ok || size.Name != "uintptr" {
					return false
				}
			}
			index++
		}
	}
	return index == count
}

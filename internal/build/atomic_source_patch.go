package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strings"
)

// This is a complete runtime implementation selection, not permission to
// discard native ABI0 result bytes. Unknown/new entry groups fail closed.
func validateAtomicSourcePatchOwners(assembly []byte, declarations, bodies [][]byte, arch string, entries int) error {
	want := make(map[string]bool)
	text := regexp.MustCompile(`^TEXT\s+·([A-Za-z0-9_]+)\(SB\)\s*,`)
	for _, line := range strings.Split(string(assembly), "\n") {
		code := strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		count := 0
		for _, word := range strings.FieldsFunc(code, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ';'
		}) {
			if word == "TEXT" {
				count++
			}
		}
		match := text.FindStringSubmatch(code)
		if count > 1 || count == 1 && match == nil {
			return fmt.Errorf("unrecognized %s atomic TEXT owner %q", arch, code)
		}
		if match != nil {
			name := match[1]
			if !ast.IsExported(name) || want[name] {
				return fmt.Errorf("private/duplicate %s atomic owner %q", arch, name)
			}
			want[name] = true
		}
	}
	if len(want) != entries {
		return fmt.Errorf("%s atomic source has %d entries, want the reviewed complete %d", arch, len(want), entries)
	}
	original, err := atomicSourcePatchFunctionsForArch(declarations, arch)
	if err != nil {
		return err
	}
	replacements, err := atomicSourcePatchFunctionsForArch(bodies, arch)
	if err != nil {
		return err
	}
	for name := range want {
		declaration, body := original[name], replacements[name]
		if declaration == nil || declaration.Body != nil || body == nil || body.Body == nil ||
			len(body.Body.List) == 0 || body.Type.TypeParams.NumFields() != 0 {
			return fmt.Errorf("%s atomic %s lacks its original declaration and real replacement body", arch, name)
		}
		before, err := atomicSourcePatchSignature(declaration.Type)
		if err != nil {
			return err
		}
		after, err := atomicSourcePatchSignature(body.Type)
		if err != nil || !reflect.DeepEqual(before, after) {
			return fmt.Errorf("%s atomic %s replacement changed its exact signature", arch, name)
		}
	}
	for name := range replacements {
		if ast.IsExported(name) && !want[name] {
			return fmt.Errorf("unowned public %s atomic replacement %q", arch, name)
		}
	}
	return nil
}

func atomicSourcePatchFunctions(sources [][]byte) (map[string]*ast.FuncDecl, error) {
	return atomicSourcePatchFunctionsForArch(sources, "ARM64")
}

func atomicSourcePatchFunctionsForArch(sources [][]byte, arch string) (map[string]*ast.FuncDecl, error) {
	functions := make(map[string]*ast.FuncDecl)
	for _, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), "owner.go", source, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
				if functions[function.Name.Name] != nil {
					return nil, fmt.Errorf("duplicate %s atomic Go owner %q", arch, function.Name.Name)
				}
				functions[function.Name.Name] = function
			}
		}
	}
	return functions, nil
}

func atomicSourcePatchSignature(signature *ast.FuncType) ([]string, error) {
	var result []string
	for _, fields := range []*ast.FieldList{signature.Params, signature.Results} {
		result = append(result, "|")
		if fields == nil {
			continue
		}
		for _, field := range fields.List {
			var text bytes.Buffer
			if err := format.Node(&text, token.NewFileSet(), field.Type); err != nil {
				return nil, err
			}
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				result = append(result, text.String())
			}
		}
	}
	return result, nil
}

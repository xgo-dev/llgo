package plan9asm

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// AssemblySDKVersion reads the version of the source tree at root.
// Official archives carry VERSION. Distribution SDKs can omit that file but
// still carry cmd/dist's generated buildcfg source. Read only the selected SDK;
// neither the launcher's runtime nor a Go executable on PATH identifies it.
func AssemblySDKVersion(root string) (string, error) {
	data, err := readAssemblyFile(nil, filepath.Join(root, "VERSION"))
	if err == nil {
		line, _, _ := strings.Cut(string(data), "\n")
		return checkedSDKVersion(strings.TrimSpace(line))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	file := filepath.Join(root, "src", "internal", "buildcfg", "zbootstrap.go")
	data, err = readAssemblyFile(nil, file)
	if err != nil {
		return "", err
	}
	syntax, err := parser.ParseFile(token.NewFileSet(), file, data, 0)
	if err != nil {
		return "", err
	}
	if syntax.Name.Name != "buildcfg" {
		return "", fmt.Errorf("selected SDK version source is not package buildcfg")
	}
	var selected string
	for _, declaration := range syntax.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				if name.Name != "version" {
					continue
				}
				if group.Tok != token.CONST || selected != "" || len(value.Names) != 1 || len(value.Values) != 1 {
					return "", fmt.Errorf("selected SDK buildcfg version is not one literal constant")
				}
				literal, ok := value.Values[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return "", fmt.Errorf("selected SDK buildcfg version is not a string literal")
				}
				selected, err = strconv.Unquote(literal.Value)
				if err != nil {
					return "", fmt.Errorf("invalid selected SDK buildcfg version %q", selected)
				}
			}
		}
	}
	if selected == "" {
		return "", fmt.Errorf("selected SDK buildcfg source has no version constant")
	}
	return checkedSDKVersion(selected)
}

var sdkVersion = regexp.MustCompile(`^(go1\.(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*)|(beta|rc)(0|[1-9][0-9]*))?)(-[^[:space:]]+)?( X:[a-zA-Z0-9_]+(,[a-zA-Z0-9_]+)*)?$`)

// SDKReleaseVersion returns the comparable Go version of a source SDK.
// Callers keep the original build identity for cache keys and tool selection.
func SDKReleaseVersion(v string) string {
	if strings.HasPrefix(v, "devel ") {
		v, _, _ = strings.Cut(strings.TrimPrefix(v, "devel "), " ")
	}
	if m := sdkVersion.FindStringSubmatch(v); m != nil {
		return m[1]
	}
	return ""
}

func checkedSDKVersion(v string) (string, error) {
	if SDKReleaseVersion(v) == "" {
		return "", fmt.Errorf("invalid selected SDK version %q", v)
	}
	return v, nil
}

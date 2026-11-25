// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package build

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
)

// These two files form a closed, target-independent subsystem. Check them
// together without the rest of runtime so an upstream dependency cannot bind
// accidentally to an unrelated LLGo declaration with the same name.
func runtimeFloatingPointSources(read func(string) ([]byte, error), goroot string) (map[string][]byte, error) {
	want := map[string]string{
		"inf2one":         "func(f float64) float64",
		"complex128div":   "func(n complex128, m complex128) complex128",
		"isNaN":           "func(f float64) (is bool)",
		"isFinite":        "func(f float64) bool",
		"isInf":           "func(f float64) bool",
		"abs":             "func(x float64) float64",
		"copysign":        "func(x, y float64) float64",
		"float64bits":     "func(f float64) uint64",
		"float64frombits": "func(b uint64) float64",
	}
	// Go 1.25 added these self-contained helpers to float.go. Complex division
	// does not require them, but review their signatures when they are present.
	optional := map[string]string{
		"floor": "func(x float64) float64",
		"ceil":  "func(x float64) float64",
		"modf":  "func(f float64) (int float64, frac float64)",
	}
	fset := token.NewFileSet()
	var files []*ast.File
	out := make(map[string][]byte)
	for _, name := range []string{"complex", "float"} {
		path := filepath.Join(goroot, "src", "runtime", name+".go")
		src, err := read(path)
		if err != nil {
			return nil, fmt.Errorf("reuse GOROOT runtime/%s.go: %w", name, err)
		}
		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "// +build ") {
					return nil, fmt.Errorf("reuse GOROOT runtime/%s.go: unreviewed directive %s", name, comment.Text)
				}
			}
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				signature := want[decl.Name.Name]
				if signature == "" {
					signature = optional[decl.Name.Name]
				}
				if err := checkRuntimeSourceFunction(fset, decl, signature); err != nil {
					return nil, err
				}
				delete(want, decl.Name.Name)
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					valid := false
					if name == "float" {
						switch spec := spec.(type) {
						case *ast.ImportSpec:
							valid = spec.Path.Value == `"unsafe"` && spec.Name == nil
						case *ast.ValueSpec:
							valid = len(spec.Names) == 1 && ((decl.Tok == token.VAR && spec.Names[0].Name == "inf") ||
								(decl.Tok == token.CONST && slices.Contains([]string{"float64Mask", "float64Shift", "float64Bias"}, spec.Names[0].Name)))
						}
					}
					if !valid {
						return nil, fmt.Errorf("reuse GOROOT runtime/%s.go: unreviewed %s declaration at %s", name, decl.Tok, fset.Position(spec.Pos()))
					}
				}
			default:
				return nil, fmt.Errorf("reuse GOROOT runtime/%s.go: unreviewed declaration", name)
			}
		}
		files = append(files, file)
		out[name] = buildInjectedSourcePatchFile(path, src)
	}
	if len(want) != 0 {
		return nil, fmt.Errorf("reuse GOROOT runtime floating-point helpers: missing helpers %v", want)
	}
	config := types.Config{Importer: importer.Default(), Sizes: types.SizesFor("gc", "386")}
	if _, err := config.Check("runtime", fset, files, nil); err != nil {
		return nil, fmt.Errorf("reuse GOROOT runtime floating-point helpers: %w", err)
	}
	return out, nil
}

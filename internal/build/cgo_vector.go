/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	llclang "github.com/xgo-dev/llgo/internal/clang"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

// prepareCVectorTypes runs before Go type checking, when native alignment must
// already be known to unsafe.Sizeof/Alignof/Offsetof. cmd/cgo intentionally
// spells C vectors as Go arrays; recover their identity from the target Clang
// AST without guessing from names, element types, or the size of an array.
func prepareCVectorTypes(ctx *context, prog llssa.Program, pkg *packages.Package) error {
	names := make(map[string]bool)
	possibleVector := false
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typ := spec.(*ast.TypeSpec)
				name, ok := strings.CutPrefix(typ.Name.Name, "_Ctype_")
				if !ok {
					continue
				}
				names[name] = true
				// Keep this gate conservative: smaller/wider vectors and anonymous
				// vector fields also arrive as arrays and must receive diagnostics.
				// Only Clang can distinguish them from ordinary fixed-size arrays.
				ast.Inspect(typ.Type, func(node ast.Node) bool {
					if a, ok := node.(*ast.ArrayType); ok {
						if n, ok := a.Len.(*ast.BasicLit); ok && n.Value != "0" {
							possibleVector = true
						}
					}
					return true
				})
			}
		}
	}
	if !possibleVector {
		return nil
	}
	buildCtx := build.Default
	buildCtx.GOOS, buildCtx.GOARCH = ctx.buildConf.Goos, ctx.buildConf.Goarch
	buildCtx.BuildTags = parseSourcePatchBuildTags(ctx.conf.BuildFlags)
	_, preambles, decls, err := parseCgoWithCommandEnv(ctx.commands, &buildCtx, &aPackage{Package: pkg}, pkg.Syntax)
	if err != nil {
		return err
	}
	cflags, _, _ := collectCgoFlags(decls)
	for _, preamble := range preambles {
		cflags = append(cflags, "-I"+filepath.Dir(preamble.goFile))
	}
	for _, preamble := range preambles {
		vectors, err := clangCVectorTypes(ctx.compiler(), cgoHeader+"\n"+preamble.src, cflags, names)
		if err != nil {
			return fmt.Errorf("C vector types in %s: %w", preamble.goFile, err)
		}
		for name, lanes := range vectors {
			prog.SetCVectorType(pkg.PkgPath+"._Ctype_"+name, lanes)
		}
	}
	return nil
}

func clangCVectorTypes(compiler *llclang.Cmd, source string, flags []string, names map[string]bool) (map[string]int, error) {
	root, err := clangVectorAST(compiler, source, flags, "")
	if err != nil {
		return nil, err
	}
	vectors := make(map[string]int)
	records := make(map[string]string)
	for _, decl := range root.Inner {
		if decl.Kind == "TypedefDecl" && names[decl.Name] {
			if vector := clangVectorType(&decl); vector != nil {
				if len(vector.Inner) != 1 || !clangVectorLaneType(&vector.Inner[0]) {
					return nil, fmt.Errorf("unsupported C vector %s: require integer lanes or float/double lanes", decl.Name)
				}
				vectors[decl.Name] = vector.NumElements
			}
			collectClangRecordTypes(&decl, decl.Name, records)
		}
	}
	if err := validateClangVectorRecords(root, names, records); err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return vectors, nil
	}
	var probe strings.Builder
	probe.WriteString(source)
	probe.WriteString("\nenum __llgo_vector_layout {\n")
	ordered := make([]string, 0, len(vectors))
	for name := range vectors {
		ordered = append(ordered, name)
	}
	slices.Sort(ordered)
	for i, name := range ordered {
		fmt.Fprintf(&probe, "__llgo_vector_size_%d = sizeof(%s), __llgo_vector_align_%d = _Alignof(%s),\n", i, name, i, name)
	}
	probe.WriteString("};\n")
	layout, err := clangVectorAST(compiler, probe.String(), flags, "__llgo_vector_layout")
	if err != nil {
		return nil, err
	}
	constants := make(map[string]int)
	for _, decl := range layout.Inner {
		if decl.Kind == "EnumConstantDecl" && len(decl.Inner) == 1 {
			expr := &decl.Inner[0]
			for expr.Kind == "ImplicitCastExpr" && len(expr.Inner) == 1 {
				expr = &expr.Inner[0]
			}
			var spelling string
			if err := json.Unmarshal(expr.Value, &spelling); err != nil {
				return nil, fmt.Errorf("invalid Clang vector layout constant %s: %w", decl.Name, err)
			}
			value, err := strconv.Atoi(spelling)
			if err != nil {
				return nil, fmt.Errorf("invalid Clang vector layout constant %s", decl.Name)
			}
			constants[decl.Name] = value
		}
	}
	for i, name := range ordered {
		size, alignment := constants[fmt.Sprintf("__llgo_vector_size_%d", i)], constants[fmt.Sprintf("__llgo_vector_align_%d", i)]
		lanes := vectors[name]
		if size != 16 || alignment != 16 || (lanes != 2 && lanes != 4 && lanes != 8 && lanes != 16) {
			return nil, fmt.Errorf("unsupported C vector %s: size %d, alignment %d, lanes %d; require a naturally aligned 128-bit integer or floating vector", name, size, alignment, lanes)
		}
	}
	return vectors, nil
}

func clangVectorSpelling(typ string) bool {
	return strings.Contains(typ, "__attribute__((__vector_size__(") ||
		strings.Contains(typ, "__attribute__((ext_vector_type(") ||
		strings.Contains(typ, "__attribute__((__ext_vector_type__(")
}

func collectClangRecordTypes(node *clangASTNode, name string, records map[string]string) {
	if node.Kind == "RecordType" && node.Decl != nil {
		records[node.Decl.ID] = name
	}
	for i := range node.Inner {
		collectClangRecordTypes(&node.Inner[i], name, records)
	}
}

func clangVectorLaneType(node *clangASTNode) bool {
	if node.Kind == "BuiltinType" {
		switch node.Type.QualType {
		case "char", "signed char", "unsigned char", "short", "unsigned short", "int", "unsigned int", "long", "unsigned long", "long long", "unsigned long long", "float", "double":
			return true
		}
		return false
	}
	switch node.Kind {
	case "TypedefType", "ElaboratedType", "AttributedType", "ParenType", "QualType", "TypeOfType":
		return len(node.Inner) == 1 && clangVectorLaneType(&node.Inner[0])
	}
	return false
}

func clangVectorType(node *clangASTNode) *clangASTNode {
	switch node.Kind {
	case "VectorType", "ExtVectorType":
		return node
	case "TypedefDecl", "TypedefType", "ElaboratedType", "AttributedType", "ParenType", "QualType", "TypeOfType", "TypeOfExprType":
		for i := range node.Inner {
			if vector := clangVectorType(&node.Inner[i]); vector != nil {
				return vector
			}
		}
	}
	// In particular, a pointer or array containing vectors is not a vector.
	return nil
}

func clangVectorAST(compiler *llclang.Cmd, source string, flags []string, filter string) (*clangASTNode, error) {
	file, err := os.CreateTemp("", "llgo-cvector-*.c")
	if err != nil {
		return nil, err
	}
	name := file.Name()
	defer os.Remove(name)
	_, writeErr := file.WriteString(source)
	closeErr := file.Close()
	if writeErr != nil {
		return nil, writeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	args := []string{"-x", "c", "-Xclang", "-ast-dump=json", "-fsyntax-only"}
	if filter != "" {
		args = append(args, "-Xclang", "-ast-dump-filter="+filter)
	}
	args = append(args, flags...)
	args = append(args, name)
	var output, diagnostics bytes.Buffer
	compiler.Stdout, compiler.Stderr = &output, &diagnostics
	if err := compiler.Compile(args...); err != nil {
		return nil, fmt.Errorf("inspect Clang types: %w\n%s", err, diagnostics.String())
	}
	var root clangASTNode
	if err := json.Unmarshal(output.Bytes(), &root); err != nil {
		return nil, fmt.Errorf("read Clang types: %w", err)
	}
	return &root, nil
}

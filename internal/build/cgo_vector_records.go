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
	"fmt"
	"strings"
)

// validateClangVectorRecords follows by-value field types, including arrays and
// nested records. FieldDecl only carries a type spelling in Clang's JSON; resolve
// that spelling back to the actual typedef/record AST rather than treating an
// array's size or element spelling as evidence that it is a vector.
func validateClangVectorRecords(root *clangASTNode, names map[string]bool, selected map[string]string) error {
	decls := make(map[string]*clangASTNode)
	records := make(map[string]*clangASTNode)
	var index func(*clangASTNode)
	index = func(n *clangASTNode) {
		if n.Kind == "FunctionDecl" {
			// Local typedefs/tags can shadow the translation unit's names.
			// They cannot define the imported records being validated here.
			return
		}
		if n.Kind == "TypedefDecl" {
			decls[n.Name] = n
		}
		if n.Kind == "RecordDecl" && len(n.Inner) != 0 {
			records[n.ID] = n
			if n.Name != "" {
				decls[n.TagUsed+" "+n.Name] = n
			}
		}
		for i := range n.Inner {
			index(&n.Inner[i])
		}
	}
	index(root)
	var indexRecordNames func(*clangASTNode)
	indexRecordNames = func(n *clangASTNode) {
		if n.Kind == "FunctionDecl" {
			return
		}
		if n.Kind == "RecordType" && n.Decl != nil && records[n.Decl.ID] != nil {
			decls[n.Type.QualType] = records[n.Decl.ID]
		}
		for i := range n.Inner {
			indexRecordNames(&n.Inner[i])
		}
	}
	indexRecordNames(root)

	checked := make(map[*clangASTNode]bool)
	hasVector := make(map[*clangASTNode]bool)
	var inspect func(*clangASTNode) (bool, error)
	var fieldType func(*clangASTNode, *clangASTNode) (bool, error)
	fieldType = func(field, parent *clangASTNode) (bool, error) {
		spelling := strings.TrimSpace(field.Type.QualType)
		// Pointers (including arrays of pointers and function pointers) do not
		// embed their pointee's layout in the enclosing record.
		if strings.Contains(spelling, "*") && !clangVectorSpelling(spelling) {
			return false, nil
		}
		if clangVectorSpelling(spelling) {
			return false, fmt.Errorf("anonymous vector field %s.%s requires a named C vector typedef", recordLabel(parent, selected), field.Name)
		}
		for strings.HasSuffix(spelling, "]") {
			i := strings.LastIndexByte(spelling, '[')
			if i < 0 {
				break
			}
			spelling = strings.TrimSpace(spelling[:i])
		}
		for {
			word, rest, _ := strings.Cut(spelling, " ")
			if word != "const" && word != "volatile" && word != "restrict" {
				break
			}
			spelling = strings.TrimSpace(rest)
		}
		if decl := decls[spelling]; decl != nil {
			return inspect(decl)
		}
		// Inline anonymous records have no name usable in a typedef lookup.
		// Clang gives the record and its field declarator the same source
		// range start; limit this association to siblings in the same record.
		if strings.Contains(spelling, "(unnamed ") || strings.Contains(spelling, "(anonymous ") {
			for i := range parent.Inner {
				child := &parent.Inner[i]
				if child.Kind == "RecordDecl" && child.Range.Begin.Offset == field.Range.Begin.Offset {
					return inspect(child)
				}
			}
		}
		return clangVectorSpelling(field.Type.DesugaredQualType), nil
	}
	inspect = func(n *clangASTNode) (bool, error) {
		if checked[n] {
			return hasVector[n], nil
		}
		checked[n] = true
		found := false
		switch n.Kind {
		case "PointerType", "BlockPointerType", "FunctionProtoType", "FunctionNoProtoType":
			return false, nil
		case "VectorType", "ExtVectorType":
			found = true
		case "RecordType":
			if n.Decl != nil && records[n.Decl.ID] != nil {
				has, err := inspect(records[n.Decl.ID])
				if err != nil {
					return false, err
				}
				found = has
			}
		case "RecordDecl":
			custom := clangCustomLayout(n)
			for i := range n.Inner {
				field := &n.Inner[i]
				if field.Kind != "FieldDecl" {
					continue
				}
				custom = custom || clangCustomLayout(field)
				has, err := fieldType(field, n)
				if err != nil {
					return false, err
				}
				found = found || has
			}
			if found && (n.TagUsed == "union" || custom) {
				return false, fmt.Errorf("unsupported C vector record %s: unions and custom record or field alignment require explicit bindings", recordLabel(n, selected))
			}
		default:
			for i := range n.Inner {
				has, err := inspect(&n.Inner[i])
				if err != nil {
					return false, err
				}
				found = found || has
			}
		}
		hasVector[n] = found
		return found, nil
	}
	// Source order keeps diagnostics deterministic.
	for i := range root.Inner {
		n := &root.Inner[i]
		if n.Kind == "RecordDecl" && (selected[n.ID] != "" || names[n.TagUsed+"_"+n.Name]) {
			if _, err := inspect(n); err != nil {
				return err
			}
		}
	}
	return nil
}

func clangCustomLayout(n *clangASTNode) bool {
	for _, child := range n.Inner {
		if child.Kind == "PackedAttr" || child.Kind == "AlignedAttr" || child.Kind == "MaxFieldAlignmentAttr" {
			return true
		}
	}
	return false
}

func recordLabel(n *clangASTNode, selected map[string]string) string {
	if name := selected[n.ID]; name != "" {
		return name
	}
	if n.Name != "" {
		return n.Name
	}
	return "<anonymous>"
}

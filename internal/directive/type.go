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

package directive

import (
	"fmt"
	"go/ast"
	"go/token"
)

// typeBackground resolves calling conventions from the prepared comment group.
// Only declaration documentation has meaning; field and TypeSpec comments retain
// main's behavior of being ignored. Warnings are recorded after validation.
func typeBackground(fset *token.FileSet, decl *ast.GenDecl, items []Directive) (background, warning string, err error) {
	var pos token.Pos
	var name, replacement string
	for _, d := range items {
		var bg, spelling, newer string
		switch d.Name {
		case "llgo:cdecl", "llgo:stdcall":
			if d.Args != "" {
				return "", "", fmt.Errorf("%s: //%s does not accept arguments", fset.Position(d.Pos), d.Name)
			}
			bg, spelling = "C", "//"+d.Name
			if d.Name == "llgo:stdcall" {
				bg = "stdcall"
			}
		case "llgo:type":
			switch d.Args {
			case "C":
				bg, spelling, newer = "C", "//llgo:type C", "//llgo:cdecl"
			case "stdcall":
				bg, spelling, newer = "stdcall", "//llgo:type stdcall", "//llgo:stdcall"
			default:
				continue
			}
		default:
			continue
		}
		if background != "" {
			return "", "", fmt.Errorf("%s: conflicting calling-convention directives on one type: %s and %s", fset.Position(d.Pos), name, spelling)
		}
		background, name, replacement, pos = bg, spelling, newer, d.Pos
	}
	if background == "" {
		return "", "", nil
	}
	if len(decl.Specs) != 1 {
		return "", "", fmt.Errorf("%s: %s applies to a single type declaration, not a grouped type block", fset.Position(pos), name)
	}
	if replacement != "" {
		warning = fmt.Sprintf("DEPRECATED: %s is deprecated at %s; use %s instead", name, fset.Position(pos), replacement)
	}
	return background, warning, nil
}

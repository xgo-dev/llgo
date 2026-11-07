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

// Directive names, normalized by the directive package (the "//" / "// "
// prefix and any arguments are stripped). These identify directive spellings;
// they are distinct from the background values ("C" / "stdcall") they map to.
const (
	nameType    = "llgo:type"
	nameCdecl   = "llgo:cdecl"
	nameStdcall = "llgo:stdcall"
)

// TypeCallConv is the calling-convention directive written on a type declaration.
type TypeCallConv struct {
	Background  string    // normalized background value: "C", "stdcall", or ""
	Name        string    // human-readable directive spelling for diagnostics
	Pos         token.Pos // position of the directive comment
	Replacement string    // non-empty when the directive is deprecated: the spelling to use instead
}

// ParseTypeCallConv scans doc for calling-convention directives on a type
// declaration. It recognizes the new //llgo:cdecl and //llgo:stdcall forms as
// well as the deprecated //llgo:type C / //llgo:type stdcall forms. For the
// deprecated forms it records the replacement spelling in the returned
// TypeCallConv; the caller emits the deprecation warning only once placement is
// validated. More than one calling-convention directive on the same
// declaration is reported as an error.
func ParseTypeCallConv(fset *token.FileSet, doc *ast.CommentGroup) (cc TypeCallConv, err error) {
	seen := false
	for _, d := range ParseGroup(doc) {
		var bg, name, replacement string
		switch d.Name {
		case nameCdecl:
			if d.Args != "" {
				return cc, fmt.Errorf("%s: //llgo:cdecl does not accept arguments", fset.Position(d.Pos))
			}
			bg, name = "C", "//llgo:cdecl"
		case nameStdcall:
			if d.Args != "" {
				return cc, fmt.Errorf("%s: //llgo:stdcall does not accept arguments", fset.Position(d.Pos))
			}
			bg, name = "stdcall", "//llgo:stdcall"
		case nameType:
			// //llgo:type has other forms beyond the two calling-convention
			// values; only C and stdcall specify a calling convention.
			switch d.Args {
			case "C":
				bg, name, replacement = "C", "//llgo:type C", "//llgo:cdecl"
			case "stdcall":
				bg, name, replacement = "stdcall", "//llgo:type stdcall", "//llgo:stdcall"
			default:
				continue
			}
		default:
			continue
		}
		if seen {
			return cc, fmt.Errorf("%s: conflicting calling-convention directives on one type: %s and %s", fset.Position(d.Pos), cc.Name, name)
		}
		cc = TypeCallConv{Background: bg, Name: name, Pos: d.Pos, Replacement: replacement}
		seen = true
	}
	return cc, nil
}

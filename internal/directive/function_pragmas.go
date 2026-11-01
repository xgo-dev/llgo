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
	"go/ast"

	"golang.org/x/tools/go/ssa"
)

// FunctionPragmas contains attributes written on a function declaration.
// It does not include attributes inferred by compiler policies.
type FunctionPragmas struct {
	NoInline       bool
	NoSplit        bool
	UintptrEscapes bool
}

// ReadFunctionPragmas reads the function's own source declaration on each call.
// It neither caches the result nor follows generic origins or synthetic
// wrappers; consumers retain their existing inheritance rules.
func ReadFunctionPragmas(fn *ssa.Function) (pragmas FunctionPragmas) {
	decl := functionDecl(fn)
	if decl == nil {
		return
	}
	return ParseFunctionPragmas(decl.Doc)
}

// ParseFunctionPragmas reads attributes from a source declaration comment.
// It is also used during package preparation, before SSA functions exist.
func ParseFunctionPragmas(doc *ast.CommentGroup) (pragmas FunctionPragmas) {
	for _, item := range ParseGroup(doc) {
		switch item.Name {
		case "go:noinline":
			pragmas.NoInline = true
		case "go:nosplit":
			pragmas.NoSplit = true
		case "go:uintptrescapes":
			pragmas.UintptrEscapes = true
		}
	}
	return
}

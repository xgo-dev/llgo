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

package packages

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

const noescapeBodyDiagnostic = "can only use //go:noescape with external func implementations"

// validateCompilerDirectives applies cmd/compile's //go:noescape body check,
// which is outside go/types. The go command may report this check while loading
// export data, but it can stop before reaching it when an earlier compiler error
// is present, so the source frontend must validate it independently.
func validateCompilerDirectives(fset *token.FileSet, files []*ast.File) []types.Error {
	if fset == nil {
		return nil
	}
	var errs []types.Error
	for _, file := range files {
		if file == nil {
			continue
		}
		previousEnd := file.Name.End()
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			directiveEnd := decl.Pos()
			var declarationToken token.Pos
			if ok {
				declarationToken = fn.Pos()
				directiveEnd = compilerFunctionPos(fn)
			}
			hasNoescape := hasCompilerDirectiveBetween(
				fset, file.Comments, previousEnd, directiveEnd, declarationToken, "go:noescape",
			)
			previousEnd = decl.End()
			if !ok || fn.Body == nil || !hasNoescape {
				continue
			}
			errs = append(errs, types.Error{
				Fset: fset,
				Pos:  directiveEnd,
				Msg:  noescapeBodyDiagnostic,
			})
		}
	}
	return errs
}

func compilerFunctionPos(fn *ast.FuncDecl) token.Pos {
	if fn.Recv != nil && fn.Recv.Opening.IsValid() {
		return fn.Recv.Opening
	}
	if fn.Name != nil {
		return fn.Name.Pos()
	}
	return fn.Pos()
}

// hasCompilerDirectiveBetween reports whether a standalone directive occurs
// between two declarations. cmd/compile keeps directives pending across blank
// lines and ordinary comments, then consumes them at the next declaration. A
// function directive may also occur between the func keyword and its name (or
// receiver).
func hasCompilerDirectiveBetween(
	fset *token.FileSet, groups []*ast.CommentGroup, start, end, declarationToken token.Pos, directive string,
) bool {
	previous := start
	for _, group := range groups {
		if group == nil || group.End() <= start || group.Pos() >= end {
			continue
		}
		for _, comment := range group.List {
			if comment == nil || comment.End() <= start || comment.Pos() >= end {
				continue
			}
			if declarationToken > previous && declarationToken < comment.Pos() {
				previous = declarationToken
			}
			standalone := physicalLine(fset, previous) != physicalLine(fset, comment.Pos())
			previous = comment.End()
			if !standalone || !strings.HasPrefix(comment.Text, "//") {
				continue
			}
			text := strings.TrimPrefix(comment.Text, "//")
			if text == directive {
				return true
			}
			if strings.HasPrefix(text, directive) && len(text) > len(directive) {
				next := text[len(directive)]
				if next == ' ' {
					return true
				}
			}
		}
	}
	return false
}

func physicalLine(fset *token.FileSet, pos token.Pos) int {
	return fset.PositionFor(pos, false).Line
}

func packageHasCompilerDiagnostic(errs []Error, want types.Error) bool {
	position := want.Fset.Position(want.Pos)
	for _, err := range errs {
		if err.Msg == want.Msg && sameDiagnosticLine(err.Pos, position) {
			return true
		}
		for _, line := range strings.Split(err.Msg, "\n") {
			pos, msg, ok := strings.Cut(line, ": ")
			if ok && msg == want.Msg && sameDiagnosticLine(pos, position) {
				return true
			}
		}
	}
	return false
}

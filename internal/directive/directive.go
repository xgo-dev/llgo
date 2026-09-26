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

// Package directive discovers Go and LLGo source directives and owns immutable
// source records shared by package preparation, analyses, and lowering.
package directive

import (
	"go/ast"
	"go/token"
	"strings"
)

// Directive is one normalized Go or LLGo source directive.
type Directive struct {
	Name string
	Args string
	Raw  string
	Pos  token.Pos
}

// Parse normalizes comment when it uses a supported directive spelling.
func Parse(comment *ast.Comment) (Directive, bool) {
	if comment == nil {
		return Directive{}, false
	}
	raw := comment.Text
	var namespace, body string
	switch {
	case strings.HasPrefix(raw, "//go:"):
		namespace, body = "go:", raw[len("//go:"):]
	case strings.HasPrefix(raw, "//llgointernal:"):
		namespace, body = "llgointernal:", raw[len("//llgointernal:"):]
	case strings.HasPrefix(raw, "//llgo:"):
		namespace, body = "llgo:", raw[len("//llgo:"):]
	case strings.HasPrefix(raw, "// llgo:"):
		namespace, body = "llgo:", raw[len("// llgo:"):]
	case strings.HasPrefix(raw, "//export "):
		return Directive{Name: "export", Args: strings.TrimSpace(raw[len("//export "):]), Raw: raw, Pos: comment.Pos()}, true
	default:
		return Directive{}, false
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return Directive{}, false
	}
	name, args := body, ""
	if idx := strings.IndexAny(body, " \t"); idx >= 0 {
		name, args = body[:idx], strings.TrimSpace(body[idx+1:])
	}
	return Directive{Name: namespace + name, Args: args, Raw: raw, Pos: comment.Pos()}, true
}

// ParseGroup returns all normalized directives in doc in source order.
func ParseGroup(doc *ast.CommentGroup) []Directive {
	if doc == nil {
		return nil
	}
	ret := make([]Directive, 0)
	for _, comment := range doc.List {
		if parsed, ok := Parse(comment); ok {
			ret = append(ret, parsed)
		}
	}
	return ret
}

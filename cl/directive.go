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

package cl

import (
	"go/ast"
	"go/token"
	"strings"
)

type sourceDirective struct {
	Name string
	Args string
	Raw  string
	Pos  token.Pos
}

func parseSourceDirective(comment *ast.Comment) (sourceDirective, bool) {
	if comment == nil {
		return sourceDirective{}, false
	}
	raw := comment.Text
	var namespace, body string
	switch {
	case strings.HasPrefix(raw, "//go:"):
		namespace, body = "go:", raw[len("//go:"):]
	case strings.HasPrefix(raw, "//llgo:"):
		namespace, body = "llgo:", raw[len("//llgo:"):]
	case strings.HasPrefix(raw, "// llgo:"):
		namespace, body = "llgo:", raw[len("// llgo:"):]
	case strings.HasPrefix(raw, "//export "):
		return sourceDirective{Name: "export", Args: strings.TrimSpace(raw[len("//export "):]), Raw: raw, Pos: comment.Pos()}, true
	default:
		return sourceDirective{}, false
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return sourceDirective{}, false
	}
	name, args := body, ""
	if idx := strings.IndexAny(body, " \t"); idx >= 0 {
		name, args = body[:idx], strings.TrimSpace(body[idx+1:])
	}
	return sourceDirective{Name: namespace + name, Args: args, Raw: raw, Pos: comment.Pos()}, true
}

func sourceDirectives(doc *ast.CommentGroup) []sourceDirective {
	if doc == nil {
		return nil
	}
	ret := make([]sourceDirective, 0, len(doc.List))
	for _, comment := range doc.List {
		if directive, ok := parseSourceDirective(comment); ok {
			ret = append(ret, directive)
		}
	}
	return ret
}

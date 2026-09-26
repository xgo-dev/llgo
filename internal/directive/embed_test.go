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
	"reflect"
	"strings"
	"testing"
)

func TestEmbedRecordPatternsAndDeferredErrors(t *testing.T) {
	for _, tt := range []struct {
		line     string
		present  bool
		patterns []string
		err      string
	}{
		{`//go:embed plain "two words" ` + "`raw path`", true, []string{"plain", "two words", "raw path"}, ""},
		{`// go:embed "a\"b"`, true, []string{`a"b`}, ""},
		{"//go:embed", true, nil, "missing pattern"},
		{`//go:embed "unterminated`, true, nil, "quoted pattern"},
		{`//go:embed "bad\q"`, true, nil, "quoted pattern"},
		{"//go:embedx other", false, nil, ""},
		{"// ordinary", false, nil, ""},
	} {
		t.Run(tt.line, func(t *testing.T) {
			index := new(Index)
			doc := &ast.CommentGroup{List: []*ast.Comment{nil, {Text: tt.line}}}
			record := index.Group(doc)
			doc.List = nil
			index.Freeze()
			patterns, present, err := EmbedPatterns(index.Group(nil), record)
			if present != tt.present || !reflect.DeepEqual(patterns, tt.patterns) {
				t.Fatalf("patterns = %q, %v", patterns, present)
			}
			if tt.err == "" && err != nil || tt.err != "" && (err == nil || !strings.Contains(err.Error(), tt.err)) {
				t.Fatalf("error = %v, want %q", err, tt.err)
			}
		})
	}
	index := new(Index)
	a := index.Group(&ast.CommentGroup{List: []*ast.Comment{{Text: "//go:embed first"}}})
	b := index.Group(&ast.CommentGroup{List: []*ast.Comment{{Text: "//go:embed second"}}})
	if p, has, err := EmbedPatterns(a, b); err != nil || !has || !reflect.DeepEqual(p, []string{"first", "second"}) {
		t.Fatalf("combined patterns = %v, %v, %v", p, has, err)
	}
	if p, has, err := parseEmbedPatterns(nil); err != nil || has || len(p) != 0 {
		t.Fatal("nil document produced an embed directive")
	}
	if args, err := SplitEmbedArgs(" \tfoo  \t"); err != nil || !reflect.DeepEqual(args, []string{"foo"}) {
		t.Fatalf("trailing whitespace = %q, %v", args, err)
	}
	for _, comment := range []*ast.Comment{nil, {Text: "/* go:embed data */"}} {
		if IsEmbedComment(comment) {
			t.Fatal("loader recognized a non-line comment")
		}
	}
}

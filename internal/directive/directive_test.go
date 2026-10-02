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
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		text string
		name string
		args string
		ok   bool
	}{
		{text: "// ordinary"},
		{text: "//go:"},
		{text: "//go:noinline", name: "go:noinline", ok: true},
		{text: "//llgointernal:tls", name: "llgointernal:tls", ok: true},
		{text: "// llgo:type C", name: "llgo:type", args: "C", ok: true},
		{text: "//llgo:link\tF C.f", name: "llgo:link", args: "F C.f", ok: true},
		{text: "//export F", name: "export", args: "F", ok: true},
	}
	if _, ok := Parse(nil); ok {
		t.Fatal("nil comment parsed as a directive")
	}
	if ParseGroup(nil) != nil {
		t.Fatal("nil comment group returned directives")
	}
	for _, test := range tests {
		got, ok := Parse(&ast.Comment{Text: test.text})
		if ok != test.ok || got.Name != test.name || got.Args != test.args || got.Raw != map[bool]string{true: test.text}[test.ok] {
			t.Fatalf("Parse(%q) = %+v, %v", test.text, got, ok)
		}
	}
}

func TestParseGroupPreservesSourceOrder(t *testing.T) {
	doc := &ast.CommentGroup{List: []*ast.Comment{
		{Text: "// ordinary"},
		{Text: "//go:noinline"},
		{Text: "//llgointernal:tls"},
	}}
	got := ParseGroup(doc)
	if len(got) != 2 || got[0].Name != "go:noinline" || got[1].Name != "llgointernal:tls" {
		t.Fatalf("ParseGroup = %+v", got)
	}
}

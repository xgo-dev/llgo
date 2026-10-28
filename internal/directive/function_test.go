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
	"strings"
	"testing"
)

func TestFunctionDirectiveAttachmentRules(t *testing.T) {
	tests := []struct {
		doc  string
		want bool
	}{
		{"//go:nointerface", true},
		{"//go:nointerface\n//go:noinline", true},
		{"//go:nointerface\n// unrelated prose", false},
		{"//go:nointerface ", false},
		{"// unrelated prose\n//go:nointerface", true},
	}
	for _, test := range tests {
		t.Run(test.doc, func(t *testing.T) {
			// Construct comments directly: go/parser may strip trailing spaces.
			doc := new(ast.CommentGroup)
			for _, line := range strings.Split(test.doc, "\n") {
				doc.List = append(doc.List, &ast.Comment{Text: line})
			}
			got, err := ParseFunction(nil, doc, "T.M", true, false)
			if err != nil || got.NoInterface != test.want {
				t.Fatalf("nointerface = %v, error = %v; want %v", got.NoInterface, err, test.want)
			}
		})
	}
}

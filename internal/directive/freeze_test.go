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

func TestFreezeClosesEveryDiscoveryEntry(t *testing.T) {
	for _, tt := range []struct {
		name     string
		discover func(*Index)
	}{
		{"file", func(s *Index) { s.File(&ast.File{}) }},
		{"group", func(s *Index) { s.Group(&ast.CommentGroup{}) }},
		{"function", func(s *Index) { s.Function(&ast.FuncDecl{}) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := new(Index)
			s.Freeze()
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(r.(string), "not prepared before lowering") {
					t.Fatalf("discovery panic = %v", r)
				}
			}()
			tt.discover(s)
		})
	}
}

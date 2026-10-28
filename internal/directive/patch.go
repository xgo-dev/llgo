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
	"bytes"
	"slices"
	"strings"
)

// SanitizeSourcePatchLines neutralizes consumed overlay commands while preserving
// byte offsets and line endings for later parsing and diagnostics.
func SanitizeSourcePatchLines(src []byte) []byte {
	out := slices.Clone(src)
	lines := bytes.SplitAfter(out, []byte{'\n'})
	changed := false
	markDirective := func(line []byte, prefix []byte) bool {
		idx := bytes.Index(line, prefix)
		if idx < 0 {
			return false
		}
		line[idx+len(prefix)-1] = '_'
		return true
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		if _, _, ok := SourcePatch(trimmed); !ok {
			continue
		}
		switch {
		case markDirective(line, []byte("//llgo:")):
			changed = true
		case markDirective(line, []byte("// llgo:")):
			changed = true
		}
	}
	if !changed {
		return src
	}
	return out
}

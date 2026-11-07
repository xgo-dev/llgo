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

package goescape

import (
	"fmt"
	"go/types"
	"io"
	"sort"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// WriteReport prints deterministic frontend storage decisions. level selects
// functions and verbosity: zero disables output, one prints placements, and
// higher values include reasons. LLVM may subsequently remove more storage.
func (r *Result) WriteReport(w io.Writer, level func(*ssa.Function) int) error {
	if r == nil {
		return nil
	}
	var lines []string
	for site, decision := range r.objects {
		fn := site.Parent()
		verbosity := level(fn)
		if verbosity <= 0 {
			continue
		}
		placement, reason := "stack", "does not escape"
		if decision.Heap {
			placement, reason = "heap", decision.Reason
		} else if decision.ReadOnly {
			placement, reason = "borrowed", "read-only view"
		} else {
			switch site.(type) {
			case *ssa.MakeSlice, *ssa.Convert, *ssa.BinOp:
				placement, reason = "stack-buffer", "heap fallback remains available"
				if c, ok := site.(*ssa.Convert); ok && stringType(c.Type()) {
					if src, ok := c.X.Type().Underlying().(*types.Basic); ok && src.Info()&types.IsInteger != 0 {
						reason = "caller-owned rune encoding buffer"
					}
				}
			case *ssa.MakeMap:
				reason = "caller-owned map header; buckets may allocate"
			}
		}
		kind := strings.TrimPrefix(fmt.Sprintf("%T", site), "*ssa.")
		line := fmt.Sprintf("%s: %s: %s %s", fn.Prog.Fset.Position(site.Pos()), fn, kind, placement)
		if verbosity > 1 {
			line += " (" + reason + ")"
		}
		lines = append(lines, line)
	}
	sort.Strings(lines)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

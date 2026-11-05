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
	"io"
	"sort"
	"strings"
)

// WriteReport prints deterministic placement diagnostics. A nonempty package
// prefix limits output to matching functions; it does not affect analysis.
func (r *Result) WriteReport(w io.Writer, packagePrefix string) error {
	if r == nil {
		return nil
	}
	var lines []string
	for site, decision := range r.objects {
		fn := site.Parent()
		if !strings.HasPrefix(strings.TrimLeft(fn.String(), "(*"), packagePrefix) {
			continue
		}
		placement, reason := "stack", "does not escape"
		if decision.Heap {
			placement, reason = "heap", decision.Reason
		}
		kind := strings.TrimPrefix(fmt.Sprintf("%T", site), "*ssa.")
		lines = append(lines, fmt.Sprintf("%s: %s: %s %s (%s)", fn.Prog.Fset.Position(site.Pos()), fn, kind, placement, reason))
	}
	sort.Strings(lines)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

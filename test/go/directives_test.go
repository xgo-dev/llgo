//go:build llgo

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

package gotest

import (
	"testing"
	_ "unsafe"

	"github.com/goplus/lib/c"
)

//go:linkname directiveSqrt C.sqrt
func directiveSqrt(x c.Double) c.Double

//llgo:link directiveAbs C.abs
func directiveAbs(x c.Int) c.Int { return 0 }

// The spaced form is an LLGo extension and intentionally differs from Go's
// compiler directive spelling.
// llgo:link directiveAbsSpaced C.abs
func directiveAbsSpaced(x c.Int) c.Int { return 0 }

// The anchor type keeps the named //llgo:skip directive attached to a declaration
// where LLGo collects skip comments, while the skipped symbol is named explicitly.
//
//llgo:skip DirectiveSkippedBad
type directiveSkipAnchor struct{}

//go:linkname directiveSkippedMissing C.llgo_missing_directive_skip
func directiveSkippedMissing()

func DirectiveSkippedBad() {
	directiveSkippedMissing()
}

func TestLLGoDirectiveLinknameForms(t *testing.T) {
	if got := float64(directiveSqrt(9)); got != 3 {
		t.Fatalf("go:linkname sqrt = %v, want 3", got)
	}
	if got := int(directiveAbs(-4)); got != 4 {
		t.Fatalf("llgo:link abs = %d, want 4", got)
	}
	if got := int(directiveAbsSpaced(-5)); got != 5 {
		t.Fatalf("spaced llgo:link abs = %d, want 5", got)
	}
}

func TestLLGoDirectiveSkip(t *testing.T) {
	_ = t
	// This is a link-time regression check. If //llgo:skip above is ignored,
	// DirectiveSkippedBad is compiled and the package fails to link because
	// C.llgo_missing_directive_skip does not exist.
}

//go:build llgo && directive_skipall_fixture

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

import _ "unsafe"

// This file is a spelling fixture for //llgo:skipall. It is intentionally behind
// an explicit build tag because enabling it in the normal test package skips
// package codegen, including the package init symbol expected by the test main.
// Manual exercise: go run ./cmd/llgo test -tags directive_skipall_fixture ./test/go.
// The command is expected to fail during link if included in the ordinary test
// package; source-patch skipall behavior is covered by internal/build tests.
//
//llgo:skipall
type directiveSkipAllAnchor struct{}

//go:linkname directiveSkipAllMissing C.llgo_missing_directive_skipall
func directiveSkipAllMissing()

func directiveSkipAllBad() {
	directiveSkipAllMissing()
}

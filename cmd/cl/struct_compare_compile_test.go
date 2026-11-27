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
	"strings"
	"testing"
)

func TestLargeStructZeroCompareUsesSourceAddress(t *testing.T) {
	const source = `
package foo

type large struct {
	value [4096]byte
}

type semantic struct {
	value [5]string
}

func immediate(p *large) bool {
	return *p == large{}
}

func snapshot(p *large, mutate func()) bool {
	v := *p
	mutate()
	return v == large{}
}

func semanticZero(p *semantic) bool {
	return *p == semantic{}
}
`
	_, mod := mustCompileLLPkgFromSrc(t, source)
	immediate := mustNamedFunction(t, mod, "foo.immediate").String()
	if !strings.Contains(immediate, ".memequalzero") {
		t.Fatalf("immediate zero comparison did not use memequalzero:\n%s", immediate)
	}
	if strings.Contains(immediate, "load { [4096 x i8] }") ||
		strings.Contains(immediate, "store { [4096 x i8] }") ||
		strings.Contains(immediate, "stacksave") {
		t.Fatalf("immediate zero comparison materialized the aggregate:\n%s", immediate)
	}
	if !strings.Contains(immediate, "AssertNilDeref") {
		t.Fatalf("elided aggregate load lost its nil check:\n%s", immediate)
	}

	snapshot := mustNamedFunction(t, mod, "foo.snapshot").String()
	if strings.Contains(snapshot, ".memequalzero") {
		t.Fatalf("comparison after a call reused mutable source storage:\n%s", snapshot)
	}
	if !strings.Contains(snapshot, ".memequal") || !strings.Contains(snapshot, "load %foo.large") {
		t.Fatalf("comparison after a call did not preserve its value snapshot:\n%s", snapshot)
	}

	semantic := mustNamedFunction(t, mod, "foo.semanticZero").String()
	if strings.Contains(semantic, ".memequalzero") || strings.Contains(semantic, ".structequal") ||
		!strings.Contains(semantic, ".arrayequal") || !strings.Contains(semantic, "load %foo.semantic") {
		t.Fatalf("semantic zero comparison did not remain field-wise:\n%s", semantic)
	}
}

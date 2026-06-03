/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"unsafe"
)

type promotedEmbedded struct {
	B int
}

type promotedGenericOuter[K any] struct {
	A K
	promotedEmbedded
}

func promotedOffsets[K any](v *promotedGenericOuter[K]) (uintptr, uintptr) {
	return unsafe.Offsetof(v.B), unsafe.Offsetof(v.promotedEmbedded)
}

func TestUnsafeOffsetofGenericPromotedFieldIssue53137(t *testing.T) {
	got, want := promotedOffsets(new(promotedGenericOuter[int]))
	if got != want {
		t.Fatalf("unsafe.Offsetof(v.B) = %d, want embedded field offset %d", got, want)
	}
}

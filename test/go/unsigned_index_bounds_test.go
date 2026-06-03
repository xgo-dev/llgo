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
	"fmt"
	"strings"
	"testing"
)

func TestUnsignedIndexBoundsCheck(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "uint64 sign bit slice",
			f: func() {
				s := []int{1}
				// Set the sign bit so a signed upper-bound compare would miss it.
				var idx uint64 = 1 << 63
				_ = s[idx]
			},
		},
		{
			name: "uint slice",
			f: func() {
				s := []int{1}
				var idx uint = ^uint(0)
				_ = s[idx]
			},
		},
		{
			name: "uint8 slice",
			f: func() {
				s := []int{1}
				var idx uint8 = 255
				_ = s[idx]
			},
		},
		{
			name: "uintptr slice",
			f: func() {
				s := []int{1}
				var idx uintptr = ^uintptr(0)
				_ = s[idx]
			},
		},
		{
			name: "uint64 sign bit array",
			f: func() {
				a := [1]int{1}
				// Array indexing uses a separate IndexAddr path from slices.
				var idx uint64 = 1 << 63
				_ = a[idx]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectUnsignedIndexPanicContaining(t, "out of range", tt.f)
		})
	}
}

func expectUnsignedIndexPanicContaining(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if got := panicString(err); !strings.Contains(got, want) {
			t.Fatalf("panic = %q, want contains %q", got, want)
		}
	}()
	f()
}

func panicString(v any) string {
	if err, ok := v.(interface{ Error() string }); ok {
		return err.Error()
	}
	return fmt.Sprint(v)
}
